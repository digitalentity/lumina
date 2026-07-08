# Makefile — build, test, and install targets for lumina.

BINARY  := lumina
INSTALL := $(HOME)/.local/bin/$(BINARY)
IMAGE   := lumina-tools:latest

.PHONY: build test vet install image clean-bin

build:
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
	rm -f $(BINARY)
