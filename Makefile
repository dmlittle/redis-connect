PKG_NAME ?= main

BFLAGS ?=
LFLAGS ?=
TFLAGS ?=

default: build

.PHONY: build
build: install
	@echo "---> Building"
	go build -ldflags "-w -s" -o bin/redis-connect

.PHONY: install
install:
	@echo "---> Installing dependencies"
	dep ensure

.PHONY: clean
clean:
	@echo "---> Cleaning"
	@rm -rf ./bin
