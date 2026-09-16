GO ?= go

.PHONY: all fmt fmt-check vet test race build example sdl-smoke sdl-golden ci

all: ci

fmt:
	gofmt -w $$(find . -name '*.go' -type f)

fmt-check:
	@test -z "$$(gofmt -l $$(find . -name '*.go' -type f))"

vet:
	CGO_ENABLED=0 $(GO) vet ./...

test:
	CGO_ENABLED=0 $(GO) test ./...

race:
	CGO_ENABLED=1 $(GO) test -race ./...

build:
	CGO_ENABLED=0 $(GO) build ./...

example:
	CGO_ENABLED=0 $(GO) build ./examples/...

sdl-smoke:
	CGO_ENABLED=0 $(GO) test -v -tags=sdl_integration .

sdl-golden:
	CGO_ENABLED=0 $(GO) test -v -tags=sdl_golden ./internal/renderer/sdl3

ci: fmt-check vet test build example sdl-smoke
