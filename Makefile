# Makefile — build, test, and install targets for lumina.

BUILD_DIR := _build
BINARY    := $(BUILD_DIR)/lumina
INSTALL   := $(HOME)/.local/bin/lumina
IMAGE     := lumina-tools:latest

.PHONY: build test vet install image clean-bin

build:
	mkdir -p $(BUILD_DIR)
	go build -o $(BINARY) .

test:
	go test ./...

vet:
	go vet ./...

install: build
	mkdir -p $(dir $(INSTALL))
	cp $(BINARY) $(INSTALL)

image:
	docker build -t $(IMAGE) .

clean-bin:
	rm -rf $(BUILD_DIR)
