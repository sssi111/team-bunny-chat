#!/bin/bash

if ! command -v go &> /dev/null; then
    echo "Go не установлен. Пожалуйста, установите Go для запуска приложения."
    exit 1
fi

echo "Установка зависимостей..."
go mod tidy

echo "Запуск Team Bunny Chat GUI..."
go run main.go
