VERSION := $(shell cat VERSION)
LDFLAGS := -ldflags "-X main.version=$(VERSION)"
-include .env


fast:
	go build $(LDFLAGS) -o pawndex

static:
	CGO_ENABLED=0 GOOS=linux go build -a $(LDFLAGS) -o pawndex .

version:
	git tag $(VERSION)
	git push
	git push origin $(VERSION)

test: fast
	./pawndex


# Docker

build:
	docker build --no-cache -t southclaws/pawndex:$(VERSION) .

push:
	docker push southclaws/pawndex:$(VERSION)
	
run:
	-docker kill pawndex
	-docker rm pawndex
	docker run \
		--name pawndex \
		--publish 7795:80 \
		--detach \
		-e PAWNDEX_BIND=$(PAWNDEX_BIND) \
		-e PAWNDEX_DOMAIN=$(PAWNDEX_DOMAIN) \
		-e PAWNDEX_GITHUBTOKEN=$(PAWNDEX_GITHUBTOKEN) \
		-e PAWNDEX_SEARCHINTERVAL=$(PAWNDEX_SEARCHINTERVAL) \
		-e DEBUG=$(DEBUG) \
		southclaws/pawndex:$(VERSION)
