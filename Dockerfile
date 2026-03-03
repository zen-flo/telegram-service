# ── Build stage ──
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/telegram-service ./cmd/server

# ── Runtime stage ──
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -S app && adduser -S app -G app

WORKDIR /app

COPY --from=builder /bin/telegram-service .

RUN mkdir -p sessions && chown -R app:app /app

USER app

EXPOSE 50051

ENTRYPOINT ["./telegram-service"]
