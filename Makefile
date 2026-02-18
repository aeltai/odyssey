VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
BINARY  := odyssey
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: build test test-integration lint clean install download-testdata

build:
	go build $(LDFLAGS) -o $(BINARY) .

install:
	go install $(LDFLAGS) .

test:
	go test -race -count=1 ./...

test-integration: download-testdata
	go test -race -count=1 -v -run 'TestDashboard|TestAllDashboards' ./internal/...

lint:
	go vet ./...

clean:
	rm -f $(BINARY)

download-testdata:
	@./scripts/download-testdata.sh
