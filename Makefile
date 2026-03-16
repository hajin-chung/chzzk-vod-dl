.PHONY: build install

build:
	go build -ldflags "-X main.commitHash=$(shell git rev-parse --short HEAD)" -o bin/cvdl cmd/cvdl/main.go

install: build
	cp bin/cvdl ~/.local/bin
