# API Gateway — технический обзор

## 1. Назначение и зона ответственности
- Выполняет роль единой входной точки для фронтенда/клиентов, пробрасывая HTTP-запросы в gRPC-сервисы `auth`, `task`, `document`.
- Инкапсулирует детали транспорта (gRPC) и схемы авторизации, предоставляя REST-подобный интерфейс.
- Реализует сквозные политики безопасности (JWT-проверка), ограничения (rate limit, body size) и CORS.

## 2. Поток обработки запроса
1. HTTP-запрос приходит на `net/http` сервер (`cmd/server/main.go`).
2. Глобальные middleware:
   - `RateLimiter` (`internal/middleware/ratelimit.go`) — ограничение по количеству запросов с клиента.
   - `CORS` (`internal/middleware/cors.go`) — контроль Origins/Headers/Methods.
3. Маршрутизатор (`internal/routers/router.go`) распределяет запросы по группам `/auth`, `/task`, `/document`.
4. Роут вызывает gRPC-клиент соответствующего сервиса (`internal/services/*`) и возвращает HTTP-ответ.
5. Ошибки gRPC преобразуются в HTTP-коды (`internal/routers/respond.go`), тела формируются JSON-энкодером.

## 3. Сетевые зависимости и протоколы
- **Auth Service** (`AUTH_GRPC_ADDR`, по умолчанию `auth:9090`) — регистрация, вход и валидация JWT.
- **Task Service** (`TASK_GRPC_ADDR`, `task:9091`) — CRUD-задач.
- **Document Service** (`DOCUMENT_GRPC_ADDR`, `document:9093`) — управление файлами (MinIO+Mongo за кулисами).
- gRPC подключение устанавливается с TLS=off (`credentials/insecure`) и блокировкой до успешного коннекта (таймаут 5 c).

## 4. Конфигурация (`internal/config/config.go`)
| Переменная | Назначение | Значение по умолчанию |
| --- | --- | --- |
| `HTTP_PORT` | Порт HTTP-сервера | `8084` |
| `AUTH_GRPC_ADDR` | Endpoint Auth-сервиса | `auth:9090` |
| `TASK_GRPC_ADDR` | Endpoint Task-сервиса | `task:9091` |
| `DOCUMENT_GRPC_ADDR` | Endpoint Document-сервиса | `document:9093` |
| `JWT_SECRET` | Ключ проверки токенов (min 32 байта) | **обязателен** |
| `RATE_LIMIT_REQUESTS` | Допустимое число запросов на окно | `60` |
| `RATE_LIMIT_WINDOW` | Длительность окна, `time.ParseDuration` | `1m` |
| `ALLOWED_ORIGINS` | CSV-список Origin для CORS | пусто (разрешено всем) |
| `SHUTDOWN_GRACE_PERIOD` | Время на graceful shutdown | `10s` |
| `READ_TIMEOUT` / `WRITE_TIMEOUT` / `IDLE_TIMEOUT` | Таймауты HTTP-сервера | `15s/15s/60s` |
| `FORWARD_RESPONSE_LIMIT` | Макс. размер документа при выдаче | `10 MiB` |

Загрузка конфигурации: `.env` (опционально) → переменные среды → дефолты. Отсутствие `JWT_SECRET` приводит к ошибке запуска.

## 5. Основные пакеты
- `cmd/server/main.go` — bootstrap: загрузка конфигурации, инициализация middleware, конструкторов сервисов и запуск HTTP.
- `internal/services/*` — обёртки над gRPC-клиентами (Auth/Task/Document). Инкапсулируют подключение, закрытие и экспортируют методы, отражающие proto.
- `internal/utils/jwt.go` — верификация JWT HS256: проверка подписи, обязательных claim (`user_id`, `exp`), выдача ошибок `ErrInvalidToken`, `ErrExpiredToken`.
- `internal/routers/*` — HTTP-маршруты и бизнес-логика преобразования payload ↔ gRPC.
- `internal/middleware/*` — инфраструктурные слои (CORS, rate limiting).

## 6. HTTP-эндпоинты
### Auth (`internal/routers/auth.go`)
- `POST /auth/register` — тело `{name,email,password,role,manager_id}` → `Register`. Ответ: `RegisterResponse`.
- `POST /auth/login` — `{email,password}` → `Login`. Ответ содержит JWT и refresh (если реализовано в gRPC).
- `POST /auth/validate` — `{token}` или заголовок `Authorization: Bearer …` → `ValidateToken`.

### Task (`internal/routers/task.go`)
| Метод | Путь | Авторизация | Вход | gRPC | Особенности |
| --- | --- | --- | --- | --- | --- |
| `POST` | `/task` | Bearer обязательный | `{message, worker_id}` | `CreateTask` | `created_by` берётся из JWT |
| `PATCH` | `/task/{id}` | Bearer | `{message?, status?, reason?}` | `UpdateTask` | `id` из URL |
| `GET` | `/task` | Bearer | query `worker_id`, `created_by`, `status` | `TaskList` | Проксирует фильтры |

### Document (`internal/routers/document.go`)
| Метод | Путь | Авторизация | Вход | gRPC | Ограничения |
| --- | --- | --- | --- | --- | --- |
| `POST` | `/document` | Bearer | `{task_id, filename, content_type, file_base64, tags[]}` | `AddDocument` | Тело base64-декодируется; владелец = `user_id` |
| `DELETE` | `/document/{id}` | Bearer | — | `DeleteDocument` | Проверяется владелец |
| `GET` | `/document/{id}` | Bearer | — | `GetDocument` | Ограничение `FORWARD_RESPONSE_LIMIT`; файл кодируется обратно в base64 |
| `GET` | `/document/task/{taskId}` | Bearer | — | `GetDocumentsByTask` | |
| `GET` | `/document/owner` | Bearer | — | `GetDocumentsByOwner` | owner = `user_id` токена |

## 7. Обработка ошибок и ответы
- Декодирование тела (`decodeJSON`) ограничено 1 MiB; неизвестные поля запрещены.
- gRPC-ошибки переводятся в HTTP: `InvalidArgument → 400`, `Unauthenticated/PermissionDenied → 401`, `NotFound → 404`, `ResourceExhausted → 429`, прочее → `502`.
- JWT ошибки (`ErrInvalidToken`, `ErrExpiredToken`) мапятся на 401 и текст из ошибки.
- Для документов превышение `FORWARD_RESPONSE_LIMIT` выдаёт `413`.

## 8. Нефункциональные аспекты
- **Производительность:** rate limiter хранит состояние в памяти процесса; горизонтально масштабируется с sticky IP или внешним стореджем (пока отсутствует).
- **Безопасность:** требуется минимум 32-байтовый секрет; все защищённые маршруты работают только при наличии корректного Bearer токена.
- **Наблюдаемость:** используется стандартный `log.Logger` без структурированного логирования; при необходимости подключить zap/logrus.
- **Завершение работы:** graceful shutdown с таймаутом `SHUTDOWN_GRACE_PERIOD`; gRPC соединения закрываются через `defer`.

## 9. Развёртывание и запуск
```bash
cd api_gateway
HTTP_PORT=8084 \
AUTH_GRPC_ADDR=localhost:9090 \
TASK_GRPC_ADDR=localhost:9091 \
DOCUMENT_GRPC_ADDR=localhost:9093 \
JWT_SECRET='supersecret_here_supersecret_here' \
go run ./cmd/server
```
Сервис безопасно завершается по `SIGINT/SIGTERM`. Для production рекомендуется запускать рядом с системами обнаружения сбоев и проксировать трафик через внешние балансировщики/ingress.

