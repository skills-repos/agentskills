GORELEASER ?= go run github.com/goreleaser/goreleaser/v2@v2.18.2

.PHONY: build test lint snapshot
build:
	go build -o skills-ref ./cmd/skills-ref
test:
	go test -race ./...
lint:
	@test -z "$$(gofmt -l .)" || { gofmt -l .; exit 1; }
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@v0.8.1 ./...
# Local archives only; nothing is published. SBOMs need syft on PATH.
snapshot:
	$(GORELEASER) check
	$(GORELEASER) release --snapshot --clean $$(command -v syft >/dev/null || echo --skip=sbom)
