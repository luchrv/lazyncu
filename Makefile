GO      ?= go
PKGS    := ./...
PURE    := ./config/... ./detect/... ./scanner/... ./semver/... ./command/... ./audit/... ./version/...

VERSION := $(shell git describe --tags --always --dirty)
COMMIT  := $(shell git rev-parse --short HEAD)
DATE    := $(shell date -u +%FT%TZ)
LDFLAGS := -X github.com/luchrv/lazyncu/version.version=$(VERSION) \
           -X github.com/luchrv/lazyncu/version.commit=$(COMMIT) \
           -X github.com/luchrv/lazyncu/version.date=$(DATE)

.PHONY: build test race cover vet fmt lint check release-check

build:
	$(GO) build -ldflags "$(LDFLAGS)" -o lazyncu .

release-check:
	goreleaser check
	goreleaser release --snapshot --clean

test:
	$(GO) test $(PKGS)

race:
	$(GO) test -race $(PKGS)

cover:
	$(GO) test -race -coverprofile=coverage.out $(PURE)
	$(GO) tool cover -func=coverage.out | tail -1

vet:
	$(GO) vet $(PKGS)

fmt:
	gofmt -l -w .

lint:
	golangci-lint run

check: fmt vet race cover
