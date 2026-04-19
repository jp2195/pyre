.PHONY: build test test-race vet lint fix tidy smoke clean

build:
	go build ./cmd/pyre

test:
	go test ./...

test-race:
	go test -race -v ./...

vet:
	go vet ./...

lint:
	golangci-lint run --timeout=5m

fix:
	go fix ./...
	go mod tidy

tidy:
	go mod tidy

smoke: build
	./pyre --version

clean:
	rm -f pyre
	rm -rf dist/
