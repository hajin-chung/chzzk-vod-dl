.PHONY: build install

build:
	go build -ldflags "-linkmode 'external' -extldflags '-static'" -tags netgo,osusergo -o cvdl .

install: build
	cp cvdl /mnt/d/vod
