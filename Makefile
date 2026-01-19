.PHONY: build lint test clean

VERSION ?= dev
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -X github.com/kareemaly/cortex1/pkg/version.Version=$(VERSION) \
           -X github.com/kareemaly/cortex1/pkg/version.Commit=$(COMMIT) \
           -X github.com/kareemaly/cortex1/pkg/version.BuildDate=$(DATE)

build:
	@mkdir -p bin
	go build -ldflags "$(LDFLAGS)" -o bin/cortex ./cmd/cortex
	go build -ldflags "$(LDFLAGS)" -o bin/cortexd ./cmd/cortexd

lint:
	golangci-lint run

test:
	go test -v ./...

clean:
	rm -rf bin
