BINARY_NAME=chat-gui
GO=go

.PHONY: all build run clean deps test

all: deps build

deps:
	$(GO) mod tidy

build:
	$(GO) build -o $(BINARY_NAME) main.go

run:
	$(GO) run main.go

clean:
	rm -f $(BINARY_NAME)
	rm -rf fyne-cross

test:
	$(GO) test -v ./...

test-short:
	$(GO) test -v -short ./...

.PHONY: cross-build
cross-build:
	fyne-cross windows -arch=amd64
	fyne-cross darwin -arch=amd64
	fyne-cross linux -arch=amd64

.PHONY: install-cross
install-cross:
	$(GO) install github.com/fyne-io/fyne-cross@latest
