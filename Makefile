# Makefile for building the crd-upgrade-checker Go project

.PHONY: all build clean

# Default target
all: build

# Build the binary
build:
	mkdir -p bin
	go build -o bin/crd-upgrade-checker main.go

# Clean up generated files
clean:
	rm -f crd-upgrade-checker
