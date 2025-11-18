FROM golang:1.23-alpine AS builder
WORKDIR /src
ENV CGO_ENABLED=0

# Copy gen/proto first (needed for replace directive in go.mod)
COPY gen/proto/auth ./gen/proto/auth

# Copy auth service go.mod and go.sum
COPY auth/go.mod auth/go.sum* ./auth/
WORKDIR /src/auth

# Cache deps (now replace directive will work)
RUN go mod download

# Copy all auth service source files
COPY auth/ ./

# Build binary
RUN go build -o /app/auth ./cmd/server

FROM alpine:3.19
RUN addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=builder /app/auth /usr/local/bin/auth
USER app
EXPOSE 8080 9090
ENTRYPOINT ["/usr/local/bin/auth"]

