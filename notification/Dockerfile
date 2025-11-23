FROM golang:1.23.4-alpine AS builder

WORKDIR /app

# copy proto modules required by replace directives (place at repo root level)
COPY gen/proto/auth /gen/proto/auth
COPY gen/proto/task /gen/proto/task
COPY gen/proto/document /gen/proto/document

COPY notification/go.mod notification/go.sum ./
RUN go mod download

COPY notification/ ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o notification ./cmd/server

FROM gcr.io/distroless/base-debian12:latest
WORKDIR /root/
COPY --from=builder /app/notification .

ENV HTTP_PORT=8083
EXPOSE 8083

CMD ["/root/notification"]

