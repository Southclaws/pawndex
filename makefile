VERSION := $(shell git describe --always --dirty --tags)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"
-include .env


static:
	CGO_ENABLED=0 GOOS=linux go build -a $(LDFLAGS) -o pawndex .

fast:
	go build $(LDFLAGS) -o pawndex

local: fast
	./pawndex

build:
	docker build -t southclaws/pawndex:$(VERSION) .

push:
	docker push southclaws/pawndex:$(VERSION)
