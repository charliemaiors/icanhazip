SHELL := /bin/bash
BINARY_NAME=icahanzip
VERSION=$(shell git symbolic-ref -q --short HEAD || git describe --tags --exact-match)
BUILD_DIR=build
DIST_DIR=dist
BASE_URL=github.com
REPO_REF=scharliemaiors/icanhazip
WITH_PUSH=false
WITH_EXTRA_ARGS?=
# WITH_PLATFORMS=linux/amd64,linux/arm64
GORELEASER_EXTRA_ARGS ?=

.PHONY: build clean install run test deps

# Build del binario
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-linux-amd64 .
	@echo "Binary built: $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-linux-amd64"

build-linux: 
	@mkdir -p $(BUILD_DIR)
	@echo "Building $(BINARY_NAME) for amd64..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-linux-amd64 .
	@echo "Building $(BINARY_NAME) for arm64..."
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w -X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-linux-arm64 .
	

build-macos:
	@echo "Building $(BINARY_NAME) for macOS..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w -X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-macos-arm64 .
	@echo "Binary built: $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-macos-arm64"

build-windows:
	@echo "Building $(BINARY_NAME) for windows (seriously?)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -X main.version=$(VERSION)" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .
	@echo "Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

# Build per più architetture
build-all:
	@echo "Building for multiple architectures..."
	@mkdir -p $(BUILD_DIR)
	# Linux AMD64
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-linux-amd64 .
	# Linux ARM64
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-linux-arm64 .
	# Windows AMD64
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-windows-amd64.exe .
	# Windows ARM64
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-windows-arm64.exe .
	# macOS ARM64
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-macos-arm64 .
	# macOS AMD64
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-macos-amd64 .
	# FreeBSD AMD64
	CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-freebsd-amd64 .
	# FreeBSD ARM64
	CGO_ENABLED=0 GOOS=freebsd GOARCH=arm64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-$(VERSION)-freebsd-arm64 .

# Installa dipendenze
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Pulizia
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR) $(DIST_DIR)
	go clean

# Installa il binario nel sistema
install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	sudo chmod +x /usr/local/bin/$(BINARY_NAME)
	@echo "Installation complete!"


# testing goreleaser KO
# build-docker-images:
#	docker build --platform $(WITH_PLATFORMS) --build-arg VERSION=$(VERSION) $(if $(WITH_EXTRA_ARGS),$(WITH_EXTRA_ARGS)) -t $(BASE_URL)/$(REPO_REF):$(VERSION) .
# ifeq ($(WITH_PUSH),true)
# 	docker push $(BASE_URL)/$(REPO_REF):$(VERSION)
# endif

release:
	goreleaser release --clean --skip=publish $(GORELEASER_EXTRA_ARGS)

test:
	go test -v ./...

# Help
help:
	@echo "Available targets:"
	@echo "  build      - Build the binary"
	@echo "  build-all  - Build for multiple architectures"
	@echo "  deps       - Install dependencies"
	@echo "  clean      - Clean build artifacts"
	@echo "  install    - Install binary to system"
	@echo "  run        - Run locally for development"
	@echo "  test       - Run tests"
	@echo "  package    - Create distribution package"
	@echo "  help       - Show this help"