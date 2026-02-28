GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOLINT=golangci-lint

BINARY_NAME=icecast-ripper
BINARY_PATH=./bin/$(BINARY_NAME)
MAIN_GO=./cmd/icecast-ripper

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS=-ldflags="-w -s -X main.version=$(VERSION)"

all: build

build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_PATH) $(MAIN_GO)

run: build
	$(BINARY_PATH)

lint:
	$(GOLINT) run ./...

test:
	$(GOTEST) -v ./...

clean:
	$(GOCLEAN)
	rm -f $(BINARY_PATH)

.PHONY: all build run lint test clean
