# API Gateway — руководство по тестированию сервисов

## 1. Предварительные требования
- Docker + Docker Compose для запуска зависимостей (`postgres`, `redis`, `kafka`, `minio`, `mongo`, `auth`, `task`, `document`).
- Go 1.23.4+ (если gateway запускается локально из исходников).
- Установленные proto-клиенты уже сгенерированы, дополнительных шагов не требуется.

## 2. Подготовка окружения
1. Запустить основные сервисы:
   ```bash
   docker-compose up -d postgres redis kafka zookeeper minio mongo auth task document
   ```
2. Проверить, что gRPC-порты доступны (`9090`, `9091`, `9093`).
3. Создать `.env` в `api_gateway/` (пример):
   ```env
   HTTP_PORT=8084
   AUTH_GRPC_ADDR=localhost:9090
   TASK_GRPC_ADDR=localhost:9091
   DOCUMENT_GRPC_ADDR=localhost:9093
   JWT_SECRET=supersecret_here_supersecret_here
   RATE_LIMIT_REQUESTS=120
   RATE_LIMIT_WINDOW=1m
   ALLOWED_ORIGINS=http://localhost:3000
   FORWARD_RESPONSE_LIMIT=10485760
   ```
4. Запустить gateway:
   ```bash
   cd api_gateway
   go run ./cmd/server
   ```
   Логи `HTTP server listening on :8084` подтверждают готовность.

## 3. Получение JWT токена
1. Зарегистрировать пользователя:
   ```bash
   curl -X POST http://localhost:8084/auth/register \
     -H "Content-Type: application/json" \
     -d '{"name":"Manager","email":"boss@example.com","password":"P@ssw0rd","role":"manager"}'
   ```
2. Авторизоваться и сохранить токен:
   ```bash
   TOKEN=$(curl -s -X POST http://localhost:8084/auth/login \
     -H "Content-Type: application/json" \
     -d '{"email":"boss@example.com","password":"P@ssw0rd"}' | jq -r '.token')
   ```
3. Проверить токен через `/auth/validate`:
   ```bash
   curl -X POST http://localhost:8084/auth/validate \
     -H "Authorization: Bearer $TOKEN"
   ```

## 4. Тестирование Task Service через gateway
### Создание задачи
```bash
curl -X POST http://localhost:8084/task \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message":"Prepare sprint report","worker_id":"<worker_uuid>"}'
```

### Обновление задачи
```bash
TASK_ID=<значение из CreateTaskResponse>
curl -X PATCH http://localhost:8084/task/$TASK_ID \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"status":"DONE","reason":"Reviewed"}'
```

### Получение списка задач
```bash
curl -X GET "http://localhost:8084/task?worker_id=<worker_uuid>" \
  -H "Authorization: Bearer $TOKEN"
```

## 5. Тестирование Document Service через gateway
### Добавление документа
```bash
CONTENT="$(base64 -w0 ./report.pdf)"
curl -X POST http://localhost:8084/document \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{
        \"task_id\":\"$TASK_ID\",
        \"filename\":\"report.pdf\",
        \"content_type\":\"application/pdf\",
        \"file_base64\":\"$CONTENT\",
        \"tags\":[\"report\",\"q1\"]
      }"
```

### Получение по идентификатору
```bash
DOC_ID=<значение из AddDocumentResponse>
curl -X GET http://localhost:8084/document/$DOC_ID \
  -H "Authorization: Bearer $TOKEN"
```
Ответ содержит `file_base64`, который можно декодировать: `echo "<base64>" | base64 -d > report_copy.pdf`.

### Список по задаче или владельцу
```bash
curl -X GET http://localhost:8084/document/task/$TASK_ID -H "Authorization: Bearer $TOKEN"
curl -X GET http://localhost:8084/document/owner -H "Authorization: Bearer $TOKEN"
```

### Удаление
```bash
curl -X DELETE http://localhost:8084/document/$DOC_ID \
  -H "Authorization: Bearer $TOKEN"
```

## 6. Негативные и граничные проверки
- **Отсутствие токена:** любой защищённый маршрут должен вернуть `401` и JSON `{"error":"authorization header missing"}`.
- **Превышение лимита запросов:** выполнить более `RATE_LIMIT_REQUESTS` запросов в пределах `RATE_LIMIT_WINDOW` c одного IP — должен вернуться `429` и заголовок `Retry-After`.
- **Неверный JWT:** подставить случайную строку, ожидать `401 token is invalid`.
- **Размер документа:** попытка загрузить файл > `MAX_FILE_SIZE` будет отклонена downstream-сервисом, а возврат > `FORWARD_RESPONSE_LIMIT` приведёт к `413`.
- **Валидация body:** отправка неизвестных полей или пустого тела вернёт `400 errUnknownBody/errEmptyBody`.

## 7. Автоматизация тестов
- Можно использовать Postman collections (`auth/Auth_Service.postman_collection.json`, `task/Task_Service.postman_collection.json`, `document/Document_Service.postman_collection.json`) и заменить base URL на `http://localhost:8084`.
- Для smoke-тестов подойдёт `newman`:
  ```bash
  newman run Auth_Service.postman_collection.json --env-var "base_url=http://localhost:8084"
  ```

## 8. Завершение работы
- Остановить gateway: `Ctrl+C` (graceful shutdown ≤ `SHUTDOWN_GRACE_PERIOD`).
- Остановить инфраструктуру: `docker-compose down`.

