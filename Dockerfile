FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY . .

# Скачиваем зависимости и собираем бинарь
RUN go mod download
RUN go build -o app ./cmd/main.go

# Финальный образ — только бинарник
FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/app .

EXPOSE 8080
CMD ["./app"]
