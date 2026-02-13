# Development utilities for HemingwayGuard

binary_name := "hemingway-guard"
bin_dir := "bin"
main_path := "./cmd/hemingway-guard"

# Show available dev commands
[private]
default:
    @just --list --unsorted --justfile {{source_file()}}

# Check if required macOS permissions are granted
[no-cd]
check-permissions:
    #!/usr/bin/env bash
    echo "Checking macOS permissions..."
    echo ""
    echo "Required permissions:"
    echo "  1. Accessibility (System Preferences → Privacy & Security → Accessibility)"
    echo "  2. Input Monitoring (System Preferences → Privacy & Security → Input Monitoring)"
    echo ""
    echo "To test permissions, the app must be built and launched."
    echo "Run 'just run' to test. If permissions are missing, macOS will prompt."
    echo ""
    echo "To reset permissions for testing:"
    echo "  tccutil reset Accessibility com.hemingway-guard"
    echo "  tccutil reset PostEvent com.hemingway-guard"

# Watch for file changes and rebuild (requires entr)
[no-cd]
watch:
    #!/usr/bin/env bash
    if ! command -v entr &> /dev/null; then
        echo "Error: 'entr' not found. Install with: brew install entr"
        exit 1
    fi
    echo "Watching for changes..."
    find . -name "*.go" | entr -c just build

# Run with debug logging
[no-cd]
debug:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Building with debug symbols..."
    mkdir -p {{bin_dir}}
    go build -gcflags="all=-N -l" -o {{bin_dir}}/{{binary_name}} {{main_path}}
    echo "Running with debug output..."
    HEMINGWAY_DEBUG=1 ./{{bin_dir}}/{{binary_name}}

# Generate Go documentation
[no-cd]
docs:
    #!/usr/bin/env bash
    echo "Starting godoc server at http://localhost:6060"
    echo "View docs at: http://localhost:6060/pkg/github.com/lancekrogers/hemingway-guard/"
    godoc -http=:6060

# Show project structure
[no-cd]
tree:
    #!/usr/bin/env bash
    if command -v tree &> /dev/null; then
        tree -I 'bin|*.app|dist|coverage.*' --dirsfirst
    else
        find . -type f \( -name "*.go" -o -name "*.swift" -o -name "*.m" -o -name "*.just" -o -name "Justfile" \) | sort
    fi

# Count lines of code
[no-cd]
loc:
    #!/usr/bin/env bash
    echo "Lines of code:"
    find . -type f \( -name "*.go" -o -name "*.swift" -o -name "*.m" \) | xargs wc -l | tail -1

# Check for TODO/FIXME comments
[no-cd]
todos:
    #!/usr/bin/env bash
    echo "TODOs and FIXMEs:"
    grep -rn "TODO\|FIXME" --include="*.go" --include="*.swift" . || echo "None found"

# Lint with golangci-lint (if installed)
[no-cd]
lint-full:
    #!/usr/bin/env bash
    if command -v golangci-lint &> /dev/null; then
        golangci-lint run ./...
    else
        echo "golangci-lint not found. Install with: brew install golangci-lint"
        echo "Falling back to go vet..."
        go vet ./...
    fi

# Profile the binary (requires built binary)
[no-cd]
profile:
    #!/usr/bin/env bash
    echo "CPU profiling not yet implemented"
    echo "To add profiling, import _ \"net/http/pprof\" and expose HTTP endpoint"

# Show git status for project
[no-cd]
status:
    @git status --short

# Show recent commits
[no-cd]
log:
    @git log --oneline -10
