# Chat History Server

Сервер для хранения и получения истории сообщений чатов.

## Описание

Сервер подписывается на все топики в RabbitMQ с паттерном "bunny.*" и сохраняет сообщения в SQLite. Для каждого чата создается отдельная таблица. Предоставляет HTTP API для получения истории сообщений.

## API

OpenAPI спецификация находится в файле [`openapi.yaml`](./openapi.yaml).

### Получение истории сообщений

```
GET /v1/chats/history?chat={chat_name}&limit=50
```

## Запуск

### Предварительные требования

1. Запустите RabbitMQ:
```bash
cd docker/rabbitmq
docker-compose up -d
```

2. Дождитесь, пока RabbitMQ запустится и создаст сеть `rabbitmq_net`

### Запуск сервера

```bash
cd docker/server
docker-compose up -d
```

## Структура проекта

```
.
├── docker/                # Docker конфигурации
│   ├── rabbitmq/         # Конфигурация RabbitMQ
│   │   ├── docker-compose.yml
│   │   └── README.md
│   └── server/          # Конфигурация сервера
│       ├── docker-compose.yml
│       └── Dockerfile
├── internal/            # Внутренние пакеты
│   ├── api/            # HTTP API
│   ├── models/         # Модели данных
│   └── rabbitmq/       # Работа с RabbitMQ
├── openapi.yaml        # Спецификация API
└── main.go             # Точка входа
```

## Переменные окружения

- `RABBITMQ_URL`: URL для подключения к RabbitMQ (по умолчанию: amqp://guest:guest@rabbitmq:5672/)
- `PORT`: порт HTTP сервера (по умолчанию: 8080)
- `SQLITE_DB_PATH`: путь к файлу базы данных SQLite (по умолчанию: /data/chat.db)

## База данных

- Используется SQLite
- Для каждого чата создается отдельная таблица с именем `chat_{name}`
- Данные хранятся в Docker volume `chat_data`

---

## Лицензия

MIT 