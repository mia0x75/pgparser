GOYACC ?= go tool goyacc

.PHONY: all build generate generate-check test testdata vet race lint fmt help

all: build vet test lint

build:
	go build ./...

generate:
	go run ./internal/astgen
	$(GOYACC) -o parser.go -l -p yy parser.y
	go run github.com/mia0x75/parserkit/cmd/yaccspan parser.go
	go run ./internal/astgen parser

generate-check:
	$(MAKE) generate
	git diff --exit-code -- parser.go ast/pg_ext.go ast/*_generated.go

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
	@echo "  generate - Regenerate parser.go and AST from parser.y"
	@echo "  generate-check - Fail if generated files differ from the grammar"
	@echo "  test     - Run tests with race detector and coverage (includes corpus gate)"
	@echo "  testdata - Run the corpus gate with per-statement diagnostics"
	@echo "  vet      - Run go vet"
	@echo "  race     - Run tests with race detector (no coverage overhead)"
	@echo "  lint     - Run golangci-lint (fails if the tool is missing)"
	@echo "  fmt      - Format tracked Go files"
	@echo "  all      - Run build, vet, test, and lint"
