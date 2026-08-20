.PHONY: build test verify diff-check

build:
	go build -trimpath -o bin/write-uuter ./cmd/write-uuter

test:
	go test ./...

verify: test
	go vet ./...
	git diff --check origin/main

diff-check:
	git diff --check origin/main...HEAD
