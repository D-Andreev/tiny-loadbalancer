# Build the application
all: build

build:
	@echo "Building..."

	@go build -o main main.go

# Run the application
run:
	@go run main.go

# Test the application
test:
	@echo "Testing..."
	mkdir -p ./e2e_tests/log
	@go test ./... -v -count=1

# Test coverage report
coverage:
	@echo "Testing with coverage..."
	@go test ./... -coverprofile=coverage.out
	@go tool cover -html=coverage.out

# Clean the binary
clean:
	@echo "Cleaning..."
	@rm -f main

# Live Reload
watch:
	@if command -v air > /dev/null; then \
	    air; \
	    echo "Watching...";\
	else \
	    read -p "Go's 'air' is not installed on your machine. Do you want to install it? [Y/n] " choice; \
	    if [ "$$choice" != "n" ] && [ "$$choice" != "N" ]; then \
	        go install github.com/cosmtrek/air@latest; \
	        air; \
	        echo "Watching...";\
	    else \
	        echo "You chose not to install air. Exiting..."; \
	        exit 1; \
	    fi; \
	fi

.PHONY: all build run test clean
