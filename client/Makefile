BIN = feedie-client
PREFIX ?= $(HOME)/.local
DIR := $(realpath .)

.PHONY: all build install clean

all: build

build:
	go mod tidy
	go build -o $(BIN)

install: build
	ln -sf "$(DIR)/$(BIN)" "$(PREFIX)/bin/feedie"

clean:
	rm -f $(BIN)
