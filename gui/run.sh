#!/bin/bash

# Скрипт для инициализации и запуска Team Bunny Chat GUI

# Проверяем наличие Go
if ! command -v go &> /dev/null; then
    echo "Go не установлен. Пожалуйста, установите Go для запуска приложения."
    exit 1
fi

# Проверяем наличие необходимых зависимостей
echo "Установка зависимостей..."
go mod tidy

# Запускаем приложение
echo "Запуск Team Bunny Chat GUI..."
go run main.go