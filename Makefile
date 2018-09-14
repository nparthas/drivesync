IMAGE := $(shell basename `pwd`)
CONFIG_FOLDER := $(shell readlink -f ~/.drivesync)
RUNARGS := --name $(IMAGE) -v $(DIRECTORY):$(DIRECTORY) -v $(CONFIG_FOLDER):/home/drivesync/.drivesync

.PHONY: run fast-run $(IMAGE) build

.PHONY: default-target
default-target: build

.PHONY: $(IMAGE)
$(IMAGE): build-alpine
	docker build --rm -t $(IMAGE) .

.PHONY: deps
deps:
	go get -u google.golang.org/api/drive/v3
	go get -u golang.org/x/oauth2/...

.PHONY: build
build:
	mkdir -p dist
	go build -o dist/$(IMAGE) ./cmd/$(IMAGE)

.PHONY: build-alpine
build-alpine:
	mkdir -p dist
	export GOOS=linux; \
	export GOARCH=386; \
	go build -o dist/$(IMAGE) ./cmd/$(IMAGE)

.PHONY: run
run: $(IMAGE) fast-run

.PHONY: fast-run
fast-run: rm
	docker run -d $(RUNARGS) $(IMAGE) --folder $(DIRECTORY)

.PHONY: sh
sh: $(IMAGE) fast-sh

.PHONY: fast-sh
fast-sh: 
	docker run $(RUNARGS) --entrypoint "/bin/sh" -it $(IMAGE)
	docker rm $(IMAGE)

.PHONY: rm
rm:
	-$(shell docker rm $(IMAGE) --force)

.PHONY: all
all: deps build $(IMAGE)
