# Makefile

# Переменные
MIGRATE := migrate

# Каталог с миграциями (в проекте: auth/migration)
MIGRATIONS_DIR := ./migration

# Подтянуть переменные из .env, если файл существует
-include .env

# Параметры БД (можно переопределить через переменные окружения/CLI)
DB_HOST ?= postgres
DB_PORT ?= 5432
DB_USER ?= user
DB_PASSWORD ?= password
DB_NAME ?= task
DB_SSLMODE ?= disable

# Полный URL БД (можно передать готовый DB_URL=...)
DB_URL ?= postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)

# Экспортировать переменные окружения для рецептов
export DB_HOST
export DB_PORT
export DB_USER
export DB_PASSWORD
export DB_NAME
export DB_SSLMODE
export DB_URL

.PHONY: create up down version force drop goto redo create-db

# Создать новую миграцию (две SQL-миграции: up и down)
# Использование: make create name=add_users_table
create:

	$(MIGRATE) create -dir $(MIGRATIONS_DIR) -ext sql -seq $(name)

# Применить все миграции
up:
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database $(DB_URL) up

# Откатить одну миграцию (или N: make down n=2)
down:
	@if [ -n "$(n)" ]; then \
		$(MIGRATE) -path $(MIGRATIONS_DIR) -database $(DB_URL) down $(n); \
	else \
		$(MIGRATE) -path $(MIGRATIONS_DIR) -database $(DB_URL) down 1; \
	fi

# Текущая версия миграций
version:
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database $(DB_URL) version

# Установить принудительно версию (make force version=3)
force:
	@test -n "$(version)" || (echo "[ERR] Укажите версию: make force version=N" && exit 1)
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database $(DB_URL) force $(version)

# Снести все и схему заново (ОСТОРОЖНО)
drop:
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database $(DB_URL) drop -f

# Перекатиться на конкретную версию (make goto version=5)
goto:
	@test -n "$(version)" || (echo "[ERR] Укажите версию: make goto version=N" && exit 1)
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database $(DB_URL) goto $(version)

# Откатить и снова применить последнюю миграцию
redo:
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database $(DB_URL) down 1 || true
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database $(DB_URL) up 1

# Создать базу данных task (если не существует)
# Примечание: команда создаст БД, даже если она уже существует (PostgreSQL выдаст предупреждение, но не ошибку)
# Использует имя контейнера из docker-compose (tasksystemcontrol-postgres-1)
create-db:
	@echo "Создание базы данных $(DB_NAME)..."
	@docker exec -i tasksystemcontrol-postgres-1 psql -U $(DB_USER) -d taskdb -c "CREATE DATABASE $(DB_NAME);" 2>&1 || echo "База данных уже существует или произошла ошибка"
	@echo "База данных $(DB_NAME) готова"