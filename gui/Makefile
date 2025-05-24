# Makefile для Team Bunny Chat GUI

# Переменные
BINARY_NAME=chat-gui
GO=go

# Цели
.PHONY: all build run clean deps

all: deps build

# Установка зависимостей
deps:
	$(GO) mod tidy

# Сборка приложения
build:
	$(GO) build -o $(BINARY_NAME) main.go

# Запуск приложения
run:
	$(GO) run main.go

# Очистка
clean:
	rm -f $(BINARY_NAME)
	rm -rf fyne-cross

# Сборка для разных платформ (требуется fyne-cross)
.PHONY: cross-build
cross-build:
	fyne-cross windows -arch=amd64
	fyne-cross darwin -arch=amd64
	fyne-cross linux -arch=amd64

# Установка fyne-cross
.PHONY: install-cross
install-cross:
	$(GO) install github.com/fyne-io/fyne-cross@latest