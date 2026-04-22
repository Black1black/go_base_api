# Wallet Service API

REST API сервис для управления кошельками с поддержкой конкурентных операций (1000+ RPS).

## 🛠 Технологии

- **Go 1.25.1** - основной язык
- **PostgreSQL 15** - база данных
- **Gin** - веб-фреймворк
- **GORM** - ORM
- **Docker & Docker Compose** - контейнеризация
- **Zap** - логирование

## 📦 Требования

- Docker 20.10+
- Docker Compose 2.0+
- Make (опционально, для удобства)
- Git

## 🚀 Быстрый старт

### 1. Клонирование репозитория

```bash
git clone git@github.com:Black1black/go_base_api.git
cd go_base_api
```

### 2. Настройка переменных окружения

# Скопируйте пример конфигурации
```bash
cp .env.example .env
```

# Отредактируйте при необходимости
```bash
nano configs/.env
```

### 3. Запуск с Docker Compose

# Сборка и запуск всех сервисов
```bash
docker compose up -d --build
```

# Проверка статуса
```bash
docker compose ps
```

# Просмотр логов
```bash
docker compose logs -f app
```

### 4. Проверка работоспособности

# Health check (должен вернуть 404, но это нормально)
curl http://localhost:8080/api/v1/wallets/00000000-0000-0000-0000-000000000000

# Создание кошелька и депозит
curl -X POST http://localhost:8080/api/v1/wallet \
  -H "Content-Type: application/json" \
  -d '{
    "walletId": "123e4567-e89b-12d3-a456-426614174000",
    "operationType": "DEPOSIT",
    "amount": 1000
  }'

# Получение баланса
curl http://localhost:8080/api/v1/wallets/123e4567-e89b-12d3-a456-426614174000

### 5. Остановка

# Остановка сервисов
```bash
docker compose down
```

# Остановка с удалением данных
```bash
docker compose down -v
```


#### Тестирование

# Запуск всех тестов (unit + integration + concurrency)
```bash
docker compose --profile test up --abort-on-container-exit --exit-code-from test test
```




