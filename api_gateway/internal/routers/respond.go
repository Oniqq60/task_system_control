package routers

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const maxBodySize = 1 << 20 // 1MB

var (
	errEmptyBody   = errors.New("request body is empty")
	errDecodeBody  = errors.New("failed to decode request body")
	errUnknownBody = errors.New("request body contains unexpected data")
)

func decodeJSON(r *http.Request, dst interface{}) error {
	if r.Body == nil {
		return errEmptyBody
	}
	defer r.Body.Close()

	limited := io.LimitReader(r.Body, maxBodySize)
	decoder := json.NewDecoder(limited)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return errEmptyBody
		}
		return err
	}

	if decoder.More() {
		return errUnknownBody
	}

	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("write json error: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	if message == "" {
		message = http.StatusText(status)
	}
	writeJSON(w, status, map[string]string{"error": message})
}

func mapRPCError(err error) (int, string) {
	if err == nil {
		return http.StatusOK, ""
	}
	st, ok := status.FromError(err)
	if !ok {
		return http.StatusBadGateway, "upstream service error"
	}
	switch st.Code() {
	case codes.InvalidArgument:
		return http.StatusBadRequest, st.Message()
	case codes.NotFound:
		return http.StatusNotFound, st.Message()
	case codes.PermissionDenied, codes.Unauthenticated:
		return http.StatusUnauthorized, st.Message()
	case codes.ResourceExhausted:
		return http.StatusTooManyRequests, st.Message()
	default:
		return http.StatusBadGateway, st.Message()
	}
}

func handleRPCError(w http.ResponseWriter, err error) {
	statusCode, message := mapRPCError(err)
	writeError(w, statusCode, message)
}

func bearerToken(r *http.Request) (string, error) {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if header == "" {
		return "", errors.New("authorization header missing")
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return "", errors.New("authorization header must be Bearer token")
	}
	return strings.TrimSpace(parts[1]), nil
}
