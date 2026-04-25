# Optional: on some platforms `go test` needs CGO_ENABLED=0 for stable test binaries.
.PHONY: test build
test:
	CGO_ENABLED=0 go test ./...

build:
	CGO_ENABLED=0 go build -o trailhead-mcp ./cmd/trailhead-mcp
	CGO_ENABLED=0 go build -o trailhead ./cmd/trailhead
