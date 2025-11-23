FROM golang:1.23-alpine AS builder

WORKDIR /src
ENV CGO_ENABLED=0

# === КОПИРУЕМ gen/proto СНАЧАЛА ===
COPY gen/proto/auth ./gen/proto/auth
COPY gen/proto/task ./gen/proto/task
COPY gen/proto/document ./gen/proto/document
# Добавьте другие proto, если используются

# Копируем go.mod и go.sum
COPY api_gateway/go.mod api_gateway/go.sum* ./api_gateway/
WORKDIR /src/api_gateway

# Теперь go mod download сработает — replace-директивы разрешатся
RUN go mod download

# Копируем исходный код
COPY api_gateway/ ./

# Собираем
RUN go build -o /app/api-gateway ./cmd/server


# === Финальный образ ===
FROM alpine:3.19
RUN addgroup -S app && adduser -S app -G app
WORKDIR /app

# Копируем бинарник из сборочного образа
COPY --from=builder /app/api-gateway /usr/local/bin/api-gateway

# Переключаемся на непривилегированного пользователя
USER app

# Открываем порты (настройте под ваш случай)
EXPOSE 8081 9091

# Запуск
ENTRYPOINT ["/usr/local/bin/api-gateway"]
