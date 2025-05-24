# Сервер чата

Серверная часть чата на Go с использованием RabbitMQ и SQLite.

## Компоненты

- REST API (Gin) для получения истории сообщений
- RabbitMQ для обмена сообщениями в реальном времени
- SQLite для хранения истории сообщений
- Docker и Docker Compose для контейнеризации

## Запуск

```bash
./run.sh
```

Скрипт автоматически:
- Остановит старые контейнеры
- Запустит RabbitMQ и сервер в Docker
- Настроит все необходимые подключения

## API

### REST Endpoints

- `GET /v1/chats/history?chat=ChatName`
  - Получение истории сообщений чата
  - Параметры:
    - `chat` - название чата (обязательный)
  - Ответ:
    ```json
    {
      "chat": "ChatName",
      "messages": [
        {
          "username": "user1",
          "body": "message text",
          "timestamp": "2025-05-24T17:51:22.846648+03:00"
        }
      ]
    }
    ```

### RabbitMQ

- Topic Exchange для каждого чата
- Формат сообщения:
  ```json
  {
    "username": "user1",
    "body": "message text",
    "timestamp": "2025-05-24T17:51:22.846648+03:00",
    "chat_name": "ChatName"
  }
  ```

### Структура проекта

```
.
├── internal/           # Внутренняя логика
│   ├── api/           # REST API handlers
│   ├── models/        # Модели данных
│   └── rabbitmq/      # RabbitMQ клиент
├── Dockerfile         # Сборка образа
├── docker-compose.yml # Конфигурация контейнеров
└── run.sh            # Скрипт запуска
```

### Тесты

```bash
go test ./...
```
