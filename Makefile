.PHONY: build install

build:
	go build -ldflags "-X main.commitHash=$(shell git rev-parse --short HEAD) -linkmode 'external' -extldflags '-static'" -tags netgo,osusergo -o cvdl .

install: build
	cp cvdl /mnt/d/vod
