# Variables
BINARY_NAME=api
BUILD_DIR=bin
CMD_PATH=./cmd/api

.PHONY: all config build run start up down clean help

## help: Show available commands
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@echo "  config  - Copy environment config from .env.example"
	@echo "  build   - Tidy Go modules and build the binary"
	@echo "  run     - Run the app directly with 'go run'"
	@echo "  start   - Build and run the compiled binary"
	@echo "  up      - Spin up Postgres, and OpenFGA (Docker)"
	@echo "  down    - Stop local development containers"
	@echo "  clean   - Remove the 'bin' directory and Go artifacts"

config:
	cp .env.example .env

build:
	go mod tidy
	@echo "Building binary..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)

run:
	go run $(CMD_PATH)

start: build
	$(BUILD_DIR)/$(BINARY_NAME)

up:
	docker compose up -d

down:
	docker compose down

clean:
	@echo "Cleaning up..."
	go clean
	rm -rf $(BUILD_DIR)
