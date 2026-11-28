GOYACC ?= go tool goyacc
GRAMMAR_CONFLICTS := conflicts: 416 shift/reduce, 37 reduce/reduce
GRAMMAR_CONFLICT_SIGNATURE := 2b3cfaf2c3f5d0b11c49f6be364e1e69d3431d056f13eb34be7afe2dfec19335

.PHONY: all build generate generate-check test testdata vet race lint fmt help

all: build vet test lint

build:
	go build ./...

generate:
	go run ./internal/astgen
	@set -eu; \
	output=$$(mktemp); \
	tables=$$(mktemp); \
	trap 'rm -f "$$output" "$$tables"' EXIT HUP INT TERM; \
	if ! $(GOYACC) -o parser/parser_generated.go -l -p yy parser/grammar.y >"$$output" 2>&1; then \
		cat "$$output"; exit 1; \
	fi; \
	if ! $(GOYACC) -v "$$tables" -o /dev/null -l -p yy parser/grammar.y >/dev/null 2>&1; then \
		echo 'Failed to generate grammar conflict table.'; exit 1; \
	fi; \
	cat "$$output"; \
	if [ "$$(grep '^conflicts:' "$$output" || true)" != "$(GRAMMAR_CONFLICTS)" ]; then \
		echo 'Grammar conflict baseline changed; review the grammar before updating it.'; exit 1; \
	fi; \
	signature=$$(grep 'conflict (' "$$tables" | sha256sum | cut -d ' ' -f 1); \
	if [ "$$signature" != "$(GRAMMAR_CONFLICT_SIGNATURE)" ]; then \
		echo "Grammar conflict signature changed: $$signature"; exit 1; \
	fi
	go run github.com/mia0x75/parserkit/cmd/yaccspan parser/parser_generated.go
	go run ./internal/astgen parser

generate-check:
	$(MAKE) generate
	@set -eu; \
	before=$$(mktemp); after=$$(mktemp); \
	trap 'rm -f "$$before" "$$after"' EXIT HUP INT TERM; \
	sha256sum parser/parser_generated.go ast/*_generated.go >"$$before"; \
	$(MAKE) generate; \
	sha256sum parser/parser_generated.go ast/*_generated.go >"$$after"; \
	cmp "$$before" "$$after"

test:
	go test -race -count=1 -cover ./...

testdata:
	PGPARSER_TESTDATA=1 go test -v ./tests -run TestParseTestdataSQL

vet:
	go vet ./...

race:
	go test -v -race -count=1 ./...

lint:
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint is required for this target" >&2; exit 1; }
	golangci-lint run

fmt:
	go fmt ./...

help:
	@echo "Available targets:"
	@echo "  build    - Compile all packages"
	@echo "  generate - Regenerate parser/parser_generated.go and AST from parser/grammar.y"
	@echo "  generate-check - Fail if generated files differ from the grammar"
	@echo "  test     - Run tests with race detector and coverage (includes corpus gate)"
	@echo "  testdata - Run the PostgreSQL corpus gate with per-statement diagnostics"
	@echo "  vet      - Run go vet"
	@echo "  race     - Run tests with race detector (no coverage overhead)"
	@echo "  lint     - Run golangci-lint (fails if the tool is missing)"
	@echo "  fmt      - Format tracked Go files"
	@echo "  all      - Run build, vet, test, and lint"
