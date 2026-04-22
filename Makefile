.PHONY: up down test test-concurrency test-race clean build run lint help

APP_NAME=wallet_app
DOCKER_COMPOSE=docker-compose

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

up:
	$(DOCKER_COMPOSE) up -d --build
	@echo "✅ Приложение запущено на http://localhost:8080"

down:
	$(DOCKER_COMPOSE) down
	@echo "✅ Сервисы остановлены"

down-clean:
	$(DOCKER_COMPOSE) down -v
	@echo "✅ Сервисы остановлены и данные удалены"

logs:
	$(DOCKER_COMPOSE) logs -f

test:
	$(DOCKER_COMPOSE) --profile test up --abort-on-container-exit --exit-code-from test test
	@$(DOCKER_COMPOSE) --profile test down

test-concurrency:
	go test -v -run TestConcurrency ./tests/...

test-race:
	go test -v -race -run TestConcurrency ./tests/...

test-short:
	$(DOCKER_COMPOSE) --profile test run --rm -e TEST_SHORT=true test go test -v -short ./...
	@$(DOCKER_COMPOSE) --profile test down

test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Отчет о покрытии сохранен в coverage.html"

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/$(APP_NAME) ./cmd/app
	@echo "✅ Бинарник собран: bin/$(APP_NAME)"

run:
	go run ./cmd/app

lint:
	@which golangci-lint > /dev/null || (echo "Установите golangci-lint: https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run

fmt:
	go fmt ./...

mod-tidy:
	go mod tidy

docker-build:
	docker build -t $(APP_NAME):latest .

docker-run:
	docker run -p 8080:8080 --env-file configs/.env $(APP_NAME):latest

ps:
	$(DOCKER_COMPOSE) ps

shell:
	$(DOCKER_COMPOSE) exec app /bin/sh

db-shell:
	$(DOCKER_COMPOSE) exec postgres psql -U ${POSTGRES_USER:-postgres} -d ${POSTGRES_DBNAME:-wallet}


check-version:
	@echo "Go version: $$(go version)"
	@echo "Docker version: $$(docker --version)"
	@echo "Docker Compose version: $$(docker-compose --version)"