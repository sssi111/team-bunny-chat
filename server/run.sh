#!/bin/bash

echo "Очистка старых контейнеров..."
docker-compose down -v

echo "Запуск контейнеров..."
docker-compose up -d

echo "Сервисы запущены!"
echo "RabbitMQ UI доступен по адресу: http://localhost:15672"
echo "API сервер доступен по адресу: http://localhost:8080" 
