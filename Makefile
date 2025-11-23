#Переменные
APP_NAME := pull-request-service
CMD_PATH := ./cmd/pull-request-service
DOCKER_COMPOSE := docker-compose
E2E_COMPOSE_FILE := tests/e2e/docker-compose.e2e.yaml

.PHONY: all build run-local up down test test-unit test-e2e lint load-test clean generate help

all: build

#Сборка и Локальный запуск
# Сборка бинарника локально
build:
	go build -o $(APP_NAME) $(CMD_PATH)

# Запуск локально (без Докера, но нужна поднятая БД на localhost:5432)
run-local:
	DB_DSN=postgres://pruser:prpassword@localhost:5432/prreviewer?sslmode=disable go run $(CMD_PATH)

#Docker

# Поднять все сервисы (App + DB + Swagger) в фоне
up:
	$(DOCKER_COMPOSE) up --build -d

# Остановить сервисы и удалить тома (очистка БД)
down:
	$(DOCKER_COMPOSE) down -v

# Посмотреть логи
logs:
	$(DOCKER_COMPOSE) logs -f

# Тестирование

# Запуск всех тестов
test: test-unit test-e2e

# Запуск только Unit-тестов
test-unit:
	go test -v ./internal/...

# Запуск E2E тестов
# Запуск E2E тестов
test-e2e:
	$(DOCKER_COMPOSE) up -d --build db_test app_test

	go test -v ./tests/e2e/... -count=1 || (echo "Tests failed" && exit 1)

	$(DOCKER_COMPOSE) rm -s -f -v db_test app_test

#Инструменты

# Линтер
lint:
	golangci-lint run ./...

# Генерация моков
generate:
	docker run -v "$$(pwd)":/src -w /src vektra/mockery --all --dir internal/service --output internal/service/mocks --outpkg mocks
	docker run -v "$$(pwd)":/src -w /src vektra/mockery --all --dir internal/http --output internal/http/mocks --outpkg mocks

# Очистка бинарников
clean:
	rm -f $(APP_NAME)

