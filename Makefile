.PHONY: build build-auth build-list-labels build-export auth list-labels export clean test fmt vet mod-download mod-tidy check install build-all

# Binary names
AUTH_BINARY=auth
LIST_LABELS_BINARY=list-labels
EXPORT_BINARY=export

# Paths
AUTH_PATH=./cmd/auth
LIST_LABELS_PATH=./cmd/list-labels
EXPORT_PATH=./cmd/export

# Build all applications
build: build-auth build-list-labels build-export

# Build individual commands
build-auth:
	go build -o $(AUTH_BINARY) $(AUTH_PATH)

build-list-labels:
	go build -o $(LIST_LABELS_BINARY) $(LIST_LABELS_PATH)

build-export:
	go build -o $(EXPORT_BINARY) $(EXPORT_PATH)

# Run commands
auth: build-auth
	./$(AUTH_BINARY)

list-labels: build-list-labels
	./$(LIST_LABELS_BINARY)

export: build-export
	./$(EXPORT_BINARY)

# Clean build artifacts
clean:
	go clean
	rm -f $(AUTH_BINARY) $(LIST_LABELS_BINARY) $(EXPORT_BINARY)
	rm -f $(AUTH_BINARY)-* $(LIST_LABELS_BINARY)-* $(EXPORT_BINARY)-*
	rm -f token.json

# Run tests
test:
	go test -v ./...

# Format code
fmt:
	go fmt ./...

# Run go vet
vet:
	go vet ./...

# Download dependencies
mod-download:
	go mod download

# Tidy dependencies
mod-tidy:
	go mod tidy

# Run all checks
check: fmt vet test

# Install the binaries to $GOPATH/bin
install:
	go install $(AUTH_PATH)
	go install $(LIST_LABELS_PATH)
	go install $(EXPORT_PATH)

# Build for multiple platforms
build-all:
	# Auth binary
	GOOS=darwin GOARCH=amd64 go build -o $(AUTH_BINARY)-darwin-amd64 $(AUTH_PATH)
	GOOS=darwin GOARCH=arm64 go build -o $(AUTH_BINARY)-darwin-arm64 $(AUTH_PATH)
	GOOS=linux GOARCH=amd64 go build -o $(AUTH_BINARY)-linux-amd64 $(AUTH_PATH)
	GOOS=windows GOARCH=amd64 go build -o $(AUTH_BINARY)-windows-amd64.exe $(AUTH_PATH)
	# List labels binary
	GOOS=darwin GOARCH=amd64 go build -o $(LIST_LABELS_BINARY)-darwin-amd64 $(LIST_LABELS_PATH)
	GOOS=darwin GOARCH=arm64 go build -o $(LIST_LABELS_BINARY)-darwin-arm64 $(LIST_LABELS_PATH)
	GOOS=linux GOARCH=amd64 go build -o $(LIST_LABELS_BINARY)-linux-amd64 $(LIST_LABELS_PATH)
	GOOS=windows GOARCH=amd64 go build -o $(LIST_LABELS_BINARY)-windows-amd64.exe $(LIST_LABELS_PATH)
	# Export binary
	GOOS=darwin GOARCH=amd64 go build -o $(EXPORT_BINARY)-darwin-amd64 $(EXPORT_PATH)
	GOOS=darwin GOARCH=arm64 go build -o $(EXPORT_BINARY)-darwin-arm64 $(EXPORT_PATH)
	GOOS=linux GOARCH=amd64 go build -o $(EXPORT_BINARY)-linux-amd64 $(EXPORT_PATH)
	GOOS=windows GOARCH=amd64 go build -o $(EXPORT_BINARY)-windows-amd64.exe $(EXPORT_PATH)

# Show help
help:
	@echo "Gmail CLI Tools - Makefile Commands"
	@echo ""
	@echo "Usage:"
	@echo "  make build        Build all applications"
	@echo "  make auth         Run authentication"
	@echo "  make list-labels  List Gmail labels"
	@echo "  make export       Export emails to JSONL"
	@echo "  make clean        Remove build artifacts"
	@echo "  make test         Run tests"
	@echo "  make check        Run fmt, vet, and test"
	@echo "  make install      Install to GOPATH/bin"
	@echo "  make build-all    Build for all platforms"