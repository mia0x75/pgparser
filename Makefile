GOYACC ?= go run golang.org/x/tools/cmd/goyacc@v0.38.0

.PHONY: all generate test testdata vet race lint fmt help

all: vet test lint

generate:
	$(GOYACC) -o parser.go -l -p yy parser.y

test:
	go test -v -race -count=1 -cover ./...

testdata:
	PGPARSER_TESTDATA=1 go test ./tests -run TestParseTestdataSQL -v

vet:
	go vet ./...

race:
	go test -v -race -count=1 ./...

lint:
	if command -v golangci-lint >/dev/null; then golangci-lint run; else echo "golangci-lint not found"; fi

fmt:
	go fmt ./...

help:
	@echo "Available targets:"
	@echo "  generate - Regenerate parser.go from parser.y"
	@echo "  test     - Run tests with race detector and coverage"
	@echo "  testdata - Run PostgreSQL regression-corpus diagnostic"
	@echo "  vet      - Run go vet"
	@echo "  race     - Run tests with race detector (no coverage overhead)"
	@echo "  lint     - Run golangci-lint"
	@echo "  fmt      - Format tracked Go files"
	@echo "  all      - Run vet, test, and lint"
