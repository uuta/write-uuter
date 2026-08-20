.PHONY: build test verify

build:
	go build -trimpath -o bin/write-uuter ./cmd/write-uuter

test:
	go test ./...

verify: test
	go vet ./...

