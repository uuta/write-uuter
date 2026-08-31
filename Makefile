.PHONY: build test verify diff-check

build:
	go build -trimpath -o bin/write-uuter ./cmd/write-uuter
	go build -trimpath -o bin/write-uuter-cloudflare-capture ./cmd/write-uuter-cloudflare-capture

# The black-box suite drives real tmux sessions and now runs a Visual Editor
# and a Writer assembly pass per candidate, so it needs more than the default
# 10-minute per-package limit.
test:
	go test -timeout 30m ./...

verify: test
	go vet ./...
	git diff --check origin/main

diff-check:
	git diff --check origin/main...HEAD
