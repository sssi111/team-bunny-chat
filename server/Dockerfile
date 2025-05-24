FROM golang:1.21-alpine AS builder

WORKDIR /app

# Установка необходимых инструментов для сборки
RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -o server

FROM alpine:latest

RUN apk add --no-cache sqlite-libs

WORKDIR /app
COPY --from=builder /app/server .

VOLUME ["/data"]
EXPOSE 8080

CMD ["./server"] 