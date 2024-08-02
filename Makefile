# Build the application
all: build

build:
	@echo "Building..."
	@go build -o tinyloadbalancer main.go

run:
	@go run main.go

test:
	@echo "Testing..."
	@go test ./... -v -count=1

coverage:
	@echo "Testing with coverage..."
	@go test ./... -coverprofile=coverage.out
	@go tool cover -html=coverage.out

clean:
	@echo "Cleaning..."
	@rm -f tinyloadbalancer

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
