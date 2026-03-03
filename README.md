# Telegram Service

---

gRPC-сервис на Go для управления независимыми Telegram-соединениями через библиотеку **gotd/td**.

Сервис позволяет:
- Создавать независимые Telegram-соединения 
- Авторизовываться через QR + 2FA 
- Отправлять текстовые сообщения 
- Получать входящие сообщения через stream 
- Удалять Telegram сессии
- Сервис корректно завершает работу при получении SIGINT/SIGTERM.

---

## Тулчейн

- Go 1.26
- Телеграм клиент: [gotd/td](https://github.com/gotd/td)
- gRPC + Protocol Buffers
- In-memory хранение сессий
- Docker / Docker Compose


---

## Структура проекта

```shell
.
├── Dockerfile
├── Makefile
├── README.md
├── cmd
│   └── server
│       └── main.go
├── docker-compose.yml
├── go.mod
├── go.sum
├── internal
│   ├── app
│   │   └── app.go
│   ├── broker
│   │   └── dispatcher.go
│   ├── config
│   │   └── config.go
│   ├── grpc
│   │   ├── server.go
│   │   └── telegram_handler.go
│   ├── session
│   │   ├── manager.go
│   │   ├── manager_test.go
│   │   └── session.go
│   └── telegram
│       ├── client.go
│       └── message.go
├── pkg
│   └── api
│       └── proto
│           ├── telegram.pb.go
│           └── telegram_grpc.pb.go
└── proto
    └── telegram.proto

```

---

## Архитектура | Основные компоненты

### gRPC слой (`internal/grpc`)

Реализует методы:

- `CreateSession`
- `DeleteSession`
- `SendMessage`
- `SubscribeMessages` (server streaming)
- `GetSessionStatus`

Обработчики делегируют бизнес-логику менеджеру сессий.

### Менеджер сессий (`internal/session`)

Отвечает за:
- создание сессий, 
- хранение их в потокобезопасной структуре, 
- удаление сессий, 
- изоляцию жизненного цикла каждой сессии 

Каждая сессия запускается в отдельной goroutine со своим context и экземпляром Telegram-клиента.

### Telegram-клиент (`internal/telegram`)

Инкапсулирует работу с gotd:
- авторизация через QR, 
- поддержка 2FA (через `TG_2FA_PASSWORD`), 
- отправка сообщений, 
- получение обновлений, 
- logout и корректное завершение работы.

Telegram-логика изолирована от остальных слоёв приложения.

### Dispatcher (`internal/broker`)

Лёгкий in-memory механизм pub/sub:
- один «топик» на сессию, 
- поддержка нескольких подписчиков, 
- неблокирующая доставка сообщений.

Используется для передачи входящих сообщений в gRPC-стрим.

---

## Особенности реализации

- Каждая Telegram-сессия полностью изолирована
- Используется context для graceful shutdown
- Потокобезопасность обеспечена sync.RWMutex
- Используется in-memory pub/sub для доставки сообщений
- Telegram session storage сохраняется в файл.

---

## Конфигурация

Используются переменные окружения:

| ENV               | Описание                   |
|-------------------|----------------------------|
| TELEGRAM_API_ID   | Telegram API ID            |
| TELEGRAM_API_HASH | Telegram API hash          |
| TG_2FA_PASSWORD   | Опциональный 2FA пароль    |
| GRPC_PORT         | gRPC port (default: 50051) |

---

## Локальный запуск

### 1. Установить зависимости

```shell
go mod download
```

### 2. Запустить сервер

```shell
go run ./cmd/server/main.go
```

### 3. Проверить работоспособность сервера

```shell
grpcurl -plaintext localhost:50051 list
```

Вы должны увидеть:

```shell
pact.telegram.TelegramService
```

---

## Запуск через Docker

### 1. Собрать образ

```shell
docker build -t telegram-service .
```

### 2. Запустить контейнер

```shell
docker run -p 50051:50051 \
  --name telegram-service \
  -e TELEGRAM_API_ID=<your_api_id> \
  -e TELEGRAM_API_HASH=<your_api_hash> \
  -e TG_2FA_PASSWORD=<your_password> \
  -v $(pwd)/sessions:/app/sessions \
  telegram-service
```

---

## Запуск через docker-compose

**docker-compose.yml**

```yaml
services:
  telegram-service:
    build: .
    container_name: telegram-service
    ports:
      - "50051:50051"
    environment:
      TELEGRAM_API_ID: ${TELEGRAM_API_ID}
      TELEGRAM_API_HASH: ${TELEGRAM_API_HASH}
      TG_2FA_PASSWORD: ${TG_2FA_PASSWORD}
    volumes:
      - ./sessions:/app/sessions
    restart: unless-stopped
```

```shell
export TELEGRAM_API_ID=<your_api_id>
export TELEGRAM_API_HASH=<your_api_hash>
export TG_2FA_PASSWORD=<your_password>

docker compose up --build
```

---

## gRPC API

### Примеры gRPC-запросов

#### Создание сессии

```shell
grpcurl -plaintext -d '{}' \
localhost:50051 pact.telegram.TelegramService/CreateSession
```

Ответ:

```json
{
  "sessionId": "<session_id>",
  "qrCode": "tg://login?token=..."
}
```

#### Удаление сессии

```shell
grpcurl -plaintext -d '{
  "sessionId": "<session_id>"
}' \
localhost:50051 pact.telegram.TelegramService/DeleteSession
```

#### Отправка сообщения

```shell
grpcurl -plaintext -d '{
  "sessionId": "<session_id>",
  "peer": "@username",
  "text": "hello"
}' \
localhost:50051 pact.telegram.TelegramService/SendMessage
```

#### Получение входящих сообщений

```shell
grpcurl -plaintext -d '{
  "sessionId": "<session_id>"
}' \
localhost:50051 pact.telegram.TelegramService/SubscribeMessages
```

---

## Авторизация

**Flow авторизации:**
- Вызывается `CreateSession`
- Сервис генерирует QR token 
- Пользователь сканирует QR в мобильном приложении Telegram 
- Если аккаунт защищён 2FA:
  - Пароль берётся из переменной окружения `TG_2FA_PASSWORD`.

---


