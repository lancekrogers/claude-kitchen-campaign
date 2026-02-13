# macOS .app bundle recipes for HemingwayGuard

binary_name := "hemingway-guard"
app_name := "HemingwayGuard"
bin_dir := "bin"
main_path := "./cmd/hemingway-guard"

# Show available bundle commands
[private]
default:
    @just --list --unsorted --justfile {{source_file()}}

# Build unsigned .app bundle
[no-cd]
build:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Building {{app_name}}.app bundle..."

    # Build the binary
    mkdir -p {{bin_dir}}
    go build -o {{bin_dir}}/{{binary_name}} {{main_path}}

    # Create bundle structure
    mkdir -p "{{app_name}}.app/Contents/MacOS"
    mkdir -p "{{app_name}}.app/Contents/Resources"

    # Copy binary and Info.plist
    cp {{bin_dir}}/{{binary_name}} "{{app_name}}.app/Contents/MacOS/"
    cp Info.plist "{{app_name}}.app/Contents/"

    echo "✅ Bundle created: {{app_name}}.app"

# Build and sign .app bundle (ad-hoc signing)
[no-cd]
sign: build
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Signing {{app_name}}.app..."
    codesign --force --deep --sign - "{{app_name}}.app"
    echo "✅ Bundle signed (ad-hoc)"

# Build release bundle (optimized, signed)
[no-cd]
release:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Building release bundle..."

    # Build optimized binary
    mkdir -p {{bin_dir}}
    go build -ldflags '-s -w' -o {{bin_dir}}/{{binary_name}} {{main_path}}

    # Create bundle structure
    mkdir -p "{{app_name}}.app/Contents/MacOS"
    mkdir -p "{{app_name}}.app/Contents/Resources"

    # Copy binary and Info.plist
    cp {{bin_dir}}/{{binary_name}} "{{app_name}}.app/Contents/MacOS/"
    cp Info.plist "{{app_name}}.app/Contents/"

    # Sign the bundle
    codesign --force --deep --sign - "{{app_name}}.app"

    echo "✅ Release bundle created: {{app_name}}.app"

# Create distributable DMG
[no-cd]
dmg: release
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Creating DMG..."
    mkdir -p dist

    # Create temporary directory for DMG contents
    DMG_TEMP=$(mktemp -d)
    cp -R "{{app_name}}.app" "$DMG_TEMP/"
    ln -s /Applications "$DMG_TEMP/Applications"

    # Create DMG
    hdiutil create -volname "{{app_name}}" \
        -srcfolder "$DMG_TEMP" \
        -ov -format UDZO \
        "dist/{{app_name}}.dmg"

    rm -rf "$DMG_TEMP"
    echo "✅ DMG created: dist/{{app_name}}.dmg"

# Create distributable ZIP
[no-cd]
zip: release
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Creating ZIP..."
    mkdir -p dist
    zip -r "dist/{{app_name}}.zip" "{{app_name}}.app"
    echo "✅ ZIP created: dist/{{app_name}}.zip"

# Clean bundle artifacts
[no-cd]
clean:
    #!/usr/bin/env bash
    set -euo pipefail
    echo "Cleaning bundle artifacts..."
    rm -rf "{{app_name}}.app"
    rm -rf dist
    echo "✅ Bundle artifacts cleaned"

# Open bundle in Finder
[no-cd]
open:
    open "{{app_name}}.app"

# Show bundle info
[no-cd]
info:
    #!/usr/bin/env bash
    if [ -d "{{app_name}}.app" ]; then
        echo "Bundle: {{app_name}}.app"
        echo "Size: $(du -sh "{{app_name}}.app" | cut -f1)"
        echo ""
        echo "Info.plist:"
        plutil -p "{{app_name}}.app/Contents/Info.plist"
    else
        echo "Bundle not found. Run 'just bundle build' first."
    fi
