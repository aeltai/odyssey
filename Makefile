VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
BINARY  := odyssey
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: build test test-integration lint clean install download-testdata web

build: web
	go build $(LDFLAGS) -o $(BINARY) .

install: web
	go install $(LDFLAGS) .

web:
	@if [ ! -d cmd/dist ]; then \
		echo "Building frontend..."; \
		cd web && npm install --silent && npm run build; \
	fi

web-dev:
	cd web && npm run dev

web-rebuild:
	cd web && npm install --silent && npm run build

test:
	go test -race -count=1 ./...

test-integration: download-testdata
	go test -race -count=1 -v -run 'TestDashboard|TestAllDashboards' ./internal/...

lint:
	go vet ./...

clean:
	rm -f $(BINARY)
	rm -rf cmd/dist

download-testdata:
	@./scripts/download-testdata.sh
