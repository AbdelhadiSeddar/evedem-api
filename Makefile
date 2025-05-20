# Simple Makefile for a Go project

# Build the application
all: build test

build:
	@echo "Building..."
	[ -d bin ] || mkdir bin
	
	
	@go build -o bin/main cmd/api/main.go

# Run the application
run: build
	cd bin; \
	./main 
#Debug mode
debug: 
	cd bin; \
	dlv debug ../cmd/api

# Create DB container
docker-run:
	docker compose up --build

# Shutdown DB container
docker-down:
	docker compose down 

# Test the application
test:
	@echo "Testing..."
	@go test ./... -v
# Integrations Tests for the application
itest:
	@echo "Running integration tests..."
	@go test ./internal/database -v

# Clean the binary
clean:
	@echo "Cleaning..."
	@rm -f bin/main

# Live Reload
watch:
	@if command -v air > /dev/null; then \
            air; \
            echo "Watching...";\
        else \
            read -p "Go's 'air' is not installed on your machine. Do you want to install it? [Y/n] " choice; \
            if [ "$$choice" != "n" ] && [ "$$choice" != "N" ]; then \
                go install github.com/air-verse/air@latest; \
                air; \
                echo "Watching...";\
            else \
                echo "You chose not to install air. Exiting..."; \
                exit 1; \
            fi; \
        fi

.PHONY: all build run test clean watch docker-run docker-down itest
