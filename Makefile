# Copy environment config to .env server. Update configuratioin when done
config: 
	cp .env.example .env

# Run server
run: 
	go run ./cmd/api
	