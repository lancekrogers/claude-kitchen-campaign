# Test recipes for HemingwayGuard

# Show available test commands
[private]
default:
    @just --list --unsorted --justfile {{source_file()}}

# Run all tests
[no-cd]
all:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Running all tests..."
    go test -v ./...
    echo "✅ All tests passed"

# Run tests (short mode)
[no-cd]
unit:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Running unit tests..."
    go test -short -timeout 30s ./...
    echo "✅ Unit tests passed"

# Run tests with verbose output
[no-cd]
unit-verbose:
    go test -v -short -timeout 30s ./...

# Run tests with race detection
[no-cd]
race:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Running tests with race detection..."
    go test -race -timeout 60s ./...
    echo "✅ Race detection tests passed"

# Run tests with coverage
[no-cd]
coverage:
    go test -cover ./...

# Generate HTML coverage report
[no-cd]
coverage-html:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Generating coverage report..."
    go test -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html
    echo "✅ Coverage report: coverage.html"

# Run benchmarks
[no-cd]
bench:
    go test -bench=. -benchmem ./...

# Run specific test by name
[no-cd]
run name:
    go test -v -run "{{name}}" ./...
