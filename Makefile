APP_NAME := radon-poller
BIN_DIR := bin
APP_PKG := ./cmd/radon-poller

.PHONY: build build-pi test lint fmt tidy

build:
	go build -o $(BIN_DIR)/$(APP_NAME) $(APP_PKG)

build-pi:
	GOOS=linux GOARCH=arm GOARM=6 CGO_ENABLED=0 go build -o $(BIN_DIR)/$(APP_NAME)-linux-armv6 $(APP_PKG)

test:
	go test ./...

lint:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run

fmt:
	gofmt -w ./cmd ./internal

tidy:
	go mod tidy
