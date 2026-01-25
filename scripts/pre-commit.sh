#!/bin/bash
set -e

echo "Running pre-commit checks..."

echo "Checking formatting..."
gofmt -l . | grep -v vendor || true
UNFORMATTED=$(gofmt -l . | grep -v vendor | wc -l)
if [ "$UNFORMATTED" -gt 0 ]; then
    echo "Error: Code not formatted. Run: gofmt -w ."
    exit 1
fi
echo "Formatting OK"

echo "Running go vet..."
go vet ./...
echo "Vet OK"

echo "Building..."
go build -o bin/kubecrsh ./cmd/kubecrsh
echo "Build OK"

echo "Running tests..."
go test ./... -short -coverprofile=coverage.out
echo "Tests OK"

COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
echo "Coverage: ${COVERAGE}%"

if command -v golangci-lint &> /dev/null; then
    echo "Running linter..."
    golangci-lint run --timeout=3m
    echo "Lint OK"
else
    echo "Warning: golangci-lint not installed, skipping"
fi

echo ""
echo "All checks passed. Ready to commit."
