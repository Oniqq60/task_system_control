FROM golang:1.23-alpine AS builder
WORKDIR /src
ENV CGO_ENABLED=0

# Copy gen/proto first (needed for replace directive in go.mod)
# The replace directive points to ../gen/proto/task relative to task/
# So we need: /src/task/../gen/proto/task = /src/gen/proto/task
COPY gen/proto/task ./gen/proto/task

# Copy task service go.mod and go.sum
COPY task/go.mod task/go.sum* ./task/
WORKDIR /src/task

# Cache deps (now replace directive will work)
RUN go mod download

# Copy all task service source files
COPY task/ ./

# Build binary
RUN go build -o /app/task ./cmd/server

FROM alpine:3.19
RUN addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=builder /app/task /usr/local/bin/task
USER app
EXPOSE 8081 9091
ENTRYPOINT ["/usr/local/bin/task"]

