FROM golang:1.23 AS builder

WORKDIR /src
ENV CGO_ENABLED=0

# Copy generated proto files required by the module replace directive
COPY gen/proto/document ./gen/proto/document

# Copy go.mod and go.sum for dependency resolution
COPY document/go.mod document/go.sum* ./document/
WORKDIR /src/document

# Download module dependencies (takes advantage of Docker layer caching)
RUN go mod download

# Copy the rest of the document service source code
COPY document/ ./

# Build the service binary
RUN GOOS=linux GOARCH=amd64 go build -o /app/document-service ./cmd/server

FROM gcr.io/distroless/base-debian12
WORKDIR /
COPY --from=builder /app/document-service /document-service

EXPOSE 8082 9093

ENTRYPOINT ["/document-service"]

