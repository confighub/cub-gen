.PHONY: build test test-parity ci

build:
	go build ./cmd/cub-gen

test:
	go test ./...

test-parity:
	go test ./cmd/cub-gen -run '^TestGitOpsParity' -count=1 -v

ci: build test test-parity
