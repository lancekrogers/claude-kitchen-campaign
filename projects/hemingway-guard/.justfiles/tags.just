# Git tag management recipes

# Show available tag commands
[private]
default:
    @just --list --unsorted --justfile {{source_file()}}

# List all tags (sorted by version)
[no-cd]
list:
    @git tag -l --sort=-version:refname

# Show tag details
[no-cd]
show version:
    git show "{{version}}" --quiet

# Create a git tag with message
[no-cd]
create version message="":
    #!/usr/bin/env bash
    set -euo pipefail
    if [ -z "{{message}}" ]; then
        git tag -a "{{version}}" -m "{{version}}"
    else
        git tag -a "{{version}}" -m "{{message}}"
    fi
    echo "✅ Created tag {{version}}"

# Delete a tag locally
[no-cd]
delete version:
    git tag -d "{{version}}"
    @echo "✅ Deleted local tag {{version}}"

# Push tag to remote
[no-cd]
push version:
    git push origin "{{version}}"
    @echo "✅ Pushed tag {{version}} to origin"

# Push all tags to remote
[no-cd]
push-all:
    git push origin --tags
    @echo "✅ Pushed all tags to origin"

# Delete tag from remote
[no-cd]
delete-remote version:
    git push origin --delete "{{version}}"
    @echo "✅ Deleted tag {{version}} from origin"

# Show latest tag
[no-cd]
latest:
    @git describe --tags --abbrev=0 2>/dev/null || echo "No tags found"

# Bump patch version and create tag
[no-cd]
bump-patch:
    #!/usr/bin/env bash
    set -euo pipefail
    LATEST=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
    # Remove 'v' prefix if present
    VERSION="${LATEST#v}"
    # Parse version components
    IFS='.' read -r MAJOR MINOR PATCH <<< "$VERSION"
    # Bump patch
    NEW_PATCH=$((PATCH + 1))
    NEW_VERSION="v${MAJOR}.${MINOR}.${NEW_PATCH}"
    git tag -a "$NEW_VERSION" -m "$NEW_VERSION"
    echo "✅ Created tag $NEW_VERSION (from $LATEST)"

# Bump minor version and create tag
[no-cd]
bump-minor:
    #!/usr/bin/env bash
    set -euo pipefail
    LATEST=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
    VERSION="${LATEST#v}"
    IFS='.' read -r MAJOR MINOR PATCH <<< "$VERSION"
    NEW_MINOR=$((MINOR + 1))
    NEW_VERSION="v${MAJOR}.${NEW_MINOR}.0"
    git tag -a "$NEW_VERSION" -m "$NEW_VERSION"
    echo "✅ Created tag $NEW_VERSION (from $LATEST)"

# Bump major version and create tag
[no-cd]
bump-major:
    #!/usr/bin/env bash
    set -euo pipefail
    LATEST=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
    VERSION="${LATEST#v}"
    IFS='.' read -r MAJOR MINOR PATCH <<< "$VERSION"
    NEW_MAJOR=$((MAJOR + 1))
    NEW_VERSION="v${NEW_MAJOR}.0.0"
    git tag -a "$NEW_VERSION" -m "$NEW_VERSION"
    echo "✅ Created tag $NEW_VERSION (from $LATEST)"
