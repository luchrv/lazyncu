GO      ?= go
PKGS    := ./...
PURE    := ./config/... ./detect/... ./scanner/... ./semver/... ./command/... ./audit/... ./version/...

VERSION := $(shell git describe --tags --always --dirty)
COMMIT  := $(shell git rev-parse --short HEAD)
DATE    := $(shell date -u +%FT%TZ)
LDFLAGS := -X github.com/luchrv/lazyncu/version.version=$(VERSION) \
           -X github.com/luchrv/lazyncu/version.commit=$(COMMIT) \
           -X github.com/luchrv/lazyncu/version.date=$(DATE)

.PHONY: build test race cover vet fmt lint check release-check demos

build:
	$(GO) build -ldflags "$(LDFLAGS)" -o lazyncu .

release-check:
	goreleaser check
	goreleaser release --snapshot --clean

demos:
	@command -v vhs >/dev/null 2>&1 || { echo "vhs not found — install with: brew install vhs"; exit 1; }
	./assets/tapes/setup-demo-env.sh
	vhs assets/tapes/hero.tape
	vhs assets/tapes/vulns.tape
	vhs assets/tapes/add-path.tape

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
