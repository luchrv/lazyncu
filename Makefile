GO      ?= go
PKGS    := ./...
PURE    := ./config/... ./detect/... ./scanner/... ./semver/... ./command/... ./audit/...

.PHONY: build test race cover vet fmt lint check

build:
	$(GO) build $(PKGS)

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
