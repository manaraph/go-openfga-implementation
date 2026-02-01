# Copy environment config to .env
# Update configuration when as required
config: 
	cp .env.example .env

# Build 
build: 
	go mod tidy

# Run server
run: 
	go run ./cmd/api
	
# spin up local development container
up:
	docker compose up -d

# shut down local development container
down:
	docker compose down