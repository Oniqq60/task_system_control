# gRPC Proto Definitions

Эта папка содержит proto файлы для определения gRPC контрактов между микросервисами.

## Структура

```
api/proto/
├── auth/
│   └── v1/
│       └── auth.proto      # Контракты для Auth Service (v1)
├── Makefile
└── README.md
```

После генерации код будет в:
```
gen/proto/
└── auth/
    └── v1/
        ├── auth.pb.go          # Структуры сообщений
        └── auth_grpc.pb.go     # gRPC клиент и сервер
```

## Генерация Go кода из proto файлов

### Требования

1. Установить `protoc` (Protocol Buffers Compiler):
   - Windows: https://github.com/protocolbuffers/protobuf/releases
   - Или через `choco install protoc`

2. Установить Go плагины для protoc (через Makefile):
   ```bash
   make tools
   ```

   Или вручную:
   ```bash
   go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.34.2
   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1
   ```

### Генерация

Из папки `api/proto/`:

```bash
# Генерация всех proto файлов
make generate

# Или только для Auth Service
make auth
```

Это создаст файлы в `gen/proto/`:
- `gen/proto/auth/v1/auth.pb.go` - структуры сообщений
- `gen/proto/auth/v1/auth_grpc.pb.go` - gRPC клиент и сервер

### Очистка

```bash
make clean
```

Удалит все сгенерированные файлы из `gen/proto/`.

## Использование в сервисах

После генерации кода, сервисы могут импортировать:
```go
import "github.com/Oniqq60/task_system_control/gen/proto/auth/v1"
```

## Обновление контрактов

1. Отредактируй `.proto` файл в `api/proto/`
2. Запусти генерацию: `make generate`
3. Обнови реализацию сервиса, если изменились методы
4. Запусти `go mod tidy` в сервисе

## Версионирование

Proto файлы версионируются через папки `v1/`, `v2/` и т.д. Это позволяет:
- Поддерживать несколько версий API одновременно
- Плавно мигрировать между версиями
- Не ломать существующие клиенты
