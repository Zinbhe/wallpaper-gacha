.PHONY: all build clean install test run help

# Binary name
BINARY_NAME=wallpaper-gacha

# Output directory
OUTPUT_DIR=./bin

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build flags
CGO_ENABLED=1
BUILD_FLAGS=-v

all: build

## build: Build the binary
build:
	@mkdir -p $(OUTPUT_DIR)
	CGO_ENABLED=$(CGO_ENABLED) $(GOBUILD) $(BUILD_FLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME)

## clean: Remove build artifacts
clean:
	$(GOCLEAN)
	rm -rf $(OUTPUT_DIR)

## install: Install dependencies
install:
	$(GOMOD) download
	$(GOMOD) tidy

## test: Run tests
test:
	CGO_ENABLED=$(CGO_ENABLED) $(GOTEST) -v ./...

## run: Build and run the application
run: build
	$(OUTPUT_DIR)/$(BINARY_NAME)

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^## //p' Makefile
