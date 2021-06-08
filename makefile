.PHONY: build
build:
	go run -v ./cmd/apiserver
.DEFAULT_GOAL := build