# Graft CLI Tool Makefile

.PHONY: build clean test install dev-setup help

# Build the binary
build:
	go build -o graft .

# Build for multiple platforms
build-all:
	GOOS=linux GOARCH=amd64 go build -o graft-linux-amd64 .
	GOOS=darwin GOARCH=amd64 go build -o graft-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -o graft-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build -o graft-windows-amd64.exe .

# Clean build artifacts
clean:
	rm -f graft graft-* 

# Run tests
test:
	go test ./...

# Install dependencies
deps:
	go mod tidy
	go mod download

# Install the binary to GOPATH/bin
install: build
	cp graft $(GOPATH)/bin/

# Development setup
dev-setup:
	go mod tidy
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Start development database
dev-db:
	docker-compose -f examples/docker-compose.yml up -d

# Stop development database
dev-db-stop:
	docker-compose -f examples/docker-compose.yml down

# Initialize development environment
dev-init: dev-db
	@echo "Waiting for database to be ready..."
	@sleep 5
	export DATABASE_URL="postgres://graft_user:graft_password@localhost:5432/graft_db?sslmode=disable" && \
	./graft init

# Run example migration
dev-migrate: build
	export DATABASE_URL="postgres://graft_user:graft_password@localhost:5432/graft_db?sslmode=disable" && \
	./graft migrate "create example tables" && \
	cp examples/example-migration.sql migrations/$(shell ls migrations/ | tail -1) && \
	./graft apply

# Show development status
dev-status: build
	export DATABASE_URL="postgres://graft_user:graft_password@localhost:5432/graft_db?sslmode=disable" && \
	./graft status

# Reset development database
dev-reset: build
	export DATABASE_URL="postgres://graft_user:graft_password@localhost:5432/graft_db?sslmode=disable" && \
	./graft reset --force

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Show help
help:
	@echo "Available commands:"
	@echo "  build      - Build the graft binary"
	@echo "  build-all  - Build for multiple platforms"
	@echo "  clean      - Clean build artifacts"
	@echo "  test       - Run tests"
	@echo "  deps       - Install dependencies"
	@echo "  install    - Install binary to GOPATH/bin"
	@echo "  dev-setup  - Setup development environment"
	@echo "  dev-db     - Start development database"
	@echo "  dev-db-stop- Stop development database"
	@echo "  dev-init   - Initialize development environment"
	@echo "  dev-migrate- Run example migration"
	@echo "  dev-status - Show development status"
	@echo "  dev-reset  - Reset development database"
	@echo "  fmt        - Format code"
	@echo "  lint       - Lint code"
	@echo "  help       - Show this help"
