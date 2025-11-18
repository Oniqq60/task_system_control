SHELL := /bin/sh

PROTOC := protoc
TOOLS_BIN := $(GOPATH)/bin

.PHONY: tools generate auth task document clean

# Установка инструментов для генерации
tools:
	@echo "Установка инструментов..."
	go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.34.2
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1

# Генерация кода из proto файлов
generate: tools
	@echo "Генерация кода из proto файлов..."
	@if not exist ..\..\gen\proto mkdir ..\..\gen\proto 2>nul || mkdir -p ../../gen/proto 2>/dev/null || true
	$(PROTOC) \
		--go_out=../../gen/proto \
		--go_opt=paths=source_relative \
		--go-grpc_out=../../gen/proto \
		--go-grpc_opt=paths=source_relative \
		-I. \
		auth/v1/auth.proto \
		task/v1/task.proto \
		document/v1/document.proto
	@echo "Генерация завершена"

# Генерация только для Auth Service
auth: tools
	@echo "Генерация кода для auth.proto..."
	@if not exist ..\..\gen\proto mkdir ..\..\gen\proto 2>nul || mkdir -p ../../gen/proto 2>/dev/null || true
	$(PROTOC) \
		--go_out=../../gen/proto \
		--go_opt=paths=source_relative \
		--go-grpc_out=../../gen/proto \
		--go-grpc_opt=paths=source_relative \
		-I. \
		auth/v1/auth.proto
	@echo "Генерация auth.proto завершена"

# Генерация только для Task Service
task: tools
	@echo "Генерация кода для task.proto..."
	@if not exist ..\..\gen\proto mkdir ..\..\gen\proto 2>nul || mkdir -p ../../gen/proto 2>/dev/null || true
	$(PROTOC) \
		--go_out=../../gen/proto \
		--go_opt=paths=source_relative \
		--go-grpc_out=../../gen/proto \
		--go-grpc_opt=paths=source_relative \
		-I. \
		task/v1/task.proto
	@echo "Генерация task.proto завершена"

# Генерация только для Document Service
document: tools
	@echo "Генерация кода для document.proto..."
	@if not exist ..\..\gen\proto mkdir ..\..\gen\proto 2>nul || mkdir -p ../../gen/proto 2>/dev/null || true
	$(PROTOC) \
		--go_out=../../gen/proto \
		--go_opt=paths=source_relative \
		--go-grpc_out=../../gen/proto \
		--go-grpc_opt=paths=source_relative \
		-I. \
		document/v1/document.proto
	@echo "Генерация document.proto завершена"

# Очистка сгенерированных файлов
clean:
	@if exist ..\..\gen rmdir /s /q ..\..\gen 2>nul || rm -rf ../../gen 2>/dev/null || true
	@echo "Очистка завершена"
