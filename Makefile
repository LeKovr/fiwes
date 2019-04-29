# exam project makefile

SHELL          = /bin/bash

# -----------------------------------------------------------------------------
# Build config

GO            ?= go
VERSION       ?= $(shell git describe --tags)
SOURCES       ?= cmd/*/*.go */*.go

# -----------------------------------------------------------------------------
# Docker image config

# application name, docker-compose prefix
PROJECT_NAME  ?= $(shell basename $$PWD)

# Hardcoded in docker-compose.yml service name
DC_SERVICE    ?= app

# Generated docker image
DC_IMAGE      ?= $(PROJECT_NAME)

# docker/compose version
DC_VER        ?= 1.23.2

# golang image version
GO_VER        ?= 1.12.4

# docker app for change inside containers
DOCKER_BIN    ?= docker

# -----------------------------------------------------------------------------
# App config

# Docker container port
SERVER_PORT   ?= 8080

# -----------------------------------------------------------------------------

.PHONY: all gen doc build-standalone coverage cov-html build test lint fmt vet vendor up down build-docker clean-docker

##
## Available make targets
##

# default: show target list
all: help

# ------------------------------------------------------------------------------
## Sources

## Run from sources
run:
	$(GO) run ./cmd/fiwes/ --html

## Build app with checks
build-all: lint lint-more vet cov build

## Build app
build: 
	go build -ldflags "-X main.version=$(VERSION)" ./cmd/fiwes

## Build app used in docker from scratch
build-standalone: cov vet lint lint-more
	CGO_ENABLED=0 GOOS=linux go build -ldflags "-X main.version=`git describe --tags`" -installsuffix 'static' -a ./cmd/fiwes

## Generate mocks
gen:
	$(GO) generate ./...

## Format go sources
fmt:
	$(GO) fmt ./...

## Run vet
vet:
	$(GO) vet ./...

## Run linter
lint:
	golint ./...

## Run more linters
lint-more:
	golangci-lint run ./...

## Run tests and fill coverage.out
cov: coverage.out

# internal target
coverage.out: $(SOURCES)
	GIN_MODE=release $(GO) test -test.v -test.race -coverprofile=$@ -covermode=atomic ./...

#	GIN_MODE=release $(GO) test -race -coverprofile=$@ -covermode=atomic -v ./...

## Open coverage report in browser
cov-html: cov
	$(GO) tool cover -html=coverage.out

## Clean coverage report
cov-clean:
	rm -f coverage.*

# ------------------------------------------------------------------------------
## Docker

# internal target
datadir:
	mkdir -p -m 777 var/data/{img,preview}

## Start service in container
up: datadir
up: CMD=up -d $(DC_SERVICE)
up: dc

## Stop service
down:
down: CMD=rm -f -s $(DC_SERVICE)
down: dc

## Build docker image
build-docker:
	@$(MAKE) -s dc CMD="build --force-rm $(DC_SERVICE)"

## Build docker image ignoring cache, for timings etc
build-docker-nc:
	@$(MAKE) -s dc CMD="build --no-cache --force-rm $(DC_SERVICE)"

## Remove docker image & temp files
clean-docker: clean-test-docker
	[[ "$$($(DOCKER_BIN) images -q $(DC_IMAGE) 2> /dev/null)" == "" ]] || $(DOCKER_BIN) rmi $(DC_IMAGE)

# ------------------------------------------------------------------------------

# $$PWD используется для того, чтобы текущий каталог был доступен в контейнере по тому же пути
# и относительные тома новых контейнеров могли его использовать
## run docker-compose
dc: docker-compose.yml
	@$(DOCKER_BIN) run --rm  -i \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v $$PWD:$$PWD \
  -w $$PWD \
  --env=GO_VERSION=$(GO_VER) \
  --env=SERVER_PORT=$(SERVER_PORT) \
  --env=DC_IMAGE=$(DC_IMAGE) \
  docker/compose:$(DC_VER) \
  -p $(PROJECT_NAME) \
  $(CMD)


# ------------------------------------------------------------------------------
## Misc

## Count lines of code (including tests) and update LOC.md
cloc: LOC.md

LOC.md: $(SOURCES)
	cloc --by-file --not-match-f='(_moq_test.go|ml|.md|.sh|.json|file)$$' --md . > $@ 2>/dev/null
	cloc --by-file --not-match-f='(_test.go|ml|.md|.sh|.json|file)$$' . 2>/dev/null
	cloc --by-file --not-match-f='_moq_test.go$$' --match-f='_test.go$$' .  2>/dev/null

## List Makefile targets
help:  Makefile
	@grep -A1 "^##" $< | grep -vE '^--$$' | sed -E '/^##/{N;s/^## (.+)\n(.+):(.*)/\t\2:\1/}' | column -t -s ':'
