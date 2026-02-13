# HemingwayGuard Swift Project Structure

## Package Manifest

```swift
// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "HemingwayGuard",
    platforms: [
        .macOS(.v13)  // Required for MenuBarExtra
    ],
    products: [
        .executable(name: "HemingwayGuard", targets: ["HemingwayGuard"])
    ],
    dependencies: [
        // Accessibility monitoring
        .package(url: "https://github.com/steipete/AXorcist", from: "1.0.0"),

        // Keyboard event interception
        .package(url: "https://github.com/stephancasas/CGEventSupervisor", from: "1.0.0"),

        // Claude API client
        .package(url: "https://github.com/jamesrochabrun/SwiftAnthropic", from: "1.0.0"),
    ],
    targets: [
        .executableTarget(
            name: "HemingwayGuard",
            dependencies: [
                "AXorcist",
                "CGEventSupervisor",
                "SwiftAnthropic",
            ],
            path: "Sources/HemingwayGuard"
        ),
        .testTarget(
            name: "HemingwayGuardTests",
            dependencies: ["HemingwayGuard"],
            path: "Tests/HemingwayGuardTests"
        ),
    ]
)
```

---

## Directory Structure

```
HemingwayGuard/
├── Package.swift
├── Sources/
│   └── HemingwayGuard/
│       ├── HemingwayGuardApp.swift      # @main entry point, MenuBarExtra
│       ├── Coordinator.swift             # State machine, flow control
│       │
│       ├── Accessibility/
│       │   └── FocusMonitor.swift        # AXorcist wrapper, focus tracking
│       │
│       ├── Keyboard/
│       │   └── KeyInterceptor.swift      # CGEventSupervisor wrapper
│       │
│       ├── Analyzer/
│       │   ├── HemingwayAnalyzer.swift   # Claude API integration
│       │   ├── AnalysisResult.swift      # Response model
│       │   └── PromptBuilder.swift       # Prompt construction
│       │
│       ├── UI/
│       │   ├── MenuBarView.swift         # Dropdown menu content
│       │   ├── ApprovalPopover.swift     # Analysis results view
│       │   └── SettingsView.swift        # Preferences window
│       │
│       └── Models/
│           ├── TargetApp.swift           # Bundle ID + metadata
│           ├── Settings.swift            # UserDefaults wrapper
│           └── UserAction.swift          # Popover action enum
│
├── Tests/
│   └── HemingwayGuardTests/
│       ├── AnalyzerTests.swift           # Prompt + parsing tests
│       ├── CoordinatorTests.swift        # State machine tests
│       └── Mocks/
│           ├── MockFocusMonitor.swift
│           └── MockAnalyzer.swift
│
├── Resources/
│   └── Assets.xcassets/
│       └── AppIcon.appiconset/
│
└── Info.plist
```

---

## File Contents Overview

### HemingwayGuardApp.swift

```swift
import SwiftUI

@main
struct HemingwayGuardApp: App {
    @StateObject private var coordinator = Coordinator()

    var body: some Scene {
        MenuBarExtra {
            MenuBarView(coordinator: coordinator)
        } label: {
            Image(systemName: coordinator.statusIcon)
        }
        .menuBarExtraStyle(.menu)

        Settings {
            SettingsView()
        }
    }
}
```

### Coordinator.swift

```swift
import SwiftUI

@MainActor
final class Coordinator: ObservableObject {
    enum State: Equatable {
        case idle
        case monitoring(app: TargetApp)
        case analyzing
        case reviewing(result: AnalysisResult)
        case releasing
        case error(String)
    }

    @Published var state: State = .idle
    @Published var isEnabled: Bool = true

    private let focusMonitor = FocusMonitor()
    private let interceptor = KeyInterceptor()
    private let analyzer = HemingwayAnalyzer()

    var statusIcon: String {
        switch state {
        case .idle: return isEnabled ? "pencil" : "pencil.slash"
        case .monitoring: return "pencil"
        case .analyzing: return "pencil.circle"
        case .reviewing: return "exclamationmark.triangle"
        case .releasing: return "pencil"
        case .error: return "exclamationmark.circle"
        }
    }

    func start() async throws {
        try await focusMonitor.start()
        try interceptor.start()

        focusMonitor.onFocusChange = { [weak self] app in
            self?.handleFocusChange(app)
        }

        interceptor.onEnterPressed = { [weak self] in
            await self?.handleEnterPressed() ?? true
        }
    }

    func stop() {
        focusMonitor.stop()
        interceptor.stop()
        state = .idle
    }

    // ... state transition methods
}
```

### FocusMonitor.swift

```swift
import AXorcist
import AppKit

@MainActor
final class FocusMonitor: ObservableObject {
    @Published var currentApp: TargetApp?
    @Published var focusedElement: AXUIElement?

    var onFocusChange: ((TargetApp?) -> Void)?

    private var observer: AXObserver?

    func start() async throws {
        // Request accessibility permission if needed
        guard AXIsProcessTrusted() else {
            throw FocusMonitorError.accessibilityDenied
        }

        // Set up AXorcist observer for focus changes
        // ...
    }

    func stop() {
        observer = nil
        currentApp = nil
        focusedElement = nil
    }

    func getCurrentText() -> String? {
        guard let element = focusedElement else { return nil }
        return element.value as? String
    }

    func setCurrentText(_ text: String) throws {
        guard let element = focusedElement else {
            throw FocusMonitorError.noFocusedElement
        }
        element.setValue(text)
    }
}
```

### KeyInterceptor.swift

```swift
import CGEventSupervisor

@MainActor
final class KeyInterceptor: ObservableObject {
    @Published var isActive: Bool = false

    var onEnterPressed: (() async -> Bool)?

    private var subscription: AnyCancellable?

    func start() throws {
        guard CGPreflightListenEventAccess() else {
            CGRequestListenEventAccess()
            throw KeyInterceptorError.inputMonitoringDenied
        }

        subscription = CGEventSupervisor.shared
            .publisher(for: .keyDown)
            .filter { $0.keyCode == 36 || $0.keyCode == 76 }
            .filter { !$0.flags.contains(.shift) }
            .sink { [weak self] event in
                self?.handleKeyEvent(event)
            }

        isActive = true
    }

    func stop() {
        subscription?.cancel()
        subscription = nil
        isActive = false
    }

    func releaseEnter() {
        CGEvent.postEnterKey()
    }

    private func handleKeyEvent(_ event: CGEvent) {
        // Cancel event, trigger async handler
        event.cancel()

        Task { @MainActor in
            let shouldRelease = await onEnterPressed?() ?? true
            if shouldRelease {
                releaseEnter()
            }
        }
    }
}
```

---

## Info.plist

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleIdentifier</key>
    <string>com.hemingway-guard</string>

    <key>CFBundleName</key>
    <string>HemingwayGuard</string>

    <key>CFBundleDisplayName</key>
    <string>HemingwayGuard</string>

    <key>CFBundleVersion</key>
    <string>1.0.0</string>

    <key>CFBundleShortVersionString</key>
    <string>1.0.0</string>

    <key>LSMinimumSystemVersion</key>
    <string>13.0</string>

    <key>LSUIElement</key>
    <true/>

    <key>NSHighResolutionCapable</key>
    <true/>

    <key>NSAccessibilityUsageDescription</key>
    <string>HemingwayGuard needs Accessibility access to monitor text fields in messaging apps and read message content for analysis.</string>
</dict>
</plist>
```

---

## Build & Run Commands

### Development

```bash
# Build
swift build

# Run
swift run HemingwayGuard

# Build for release
swift build -c release
```

### Testing

```bash
# Run all tests
swift test

# Run specific test
swift test --filter AnalyzerTests
```

### Distribution

```bash
# Build release binary
swift build -c release

# Create .app bundle
mkdir -p HemingwayGuard.app/Contents/MacOS
mkdir -p HemingwayGuard.app/Contents/Resources
cp .build/release/HemingwayGuard HemingwayGuard.app/Contents/MacOS/
cp Info.plist HemingwayGuard.app/Contents/

# Code sign (ad-hoc)
codesign --force --deep --sign - HemingwayGuard.app

# Code sign (for distribution)
codesign --force --deep --sign "Developer ID Application: Your Name" HemingwayGuard.app

# Notarize
xcrun notarytool submit HemingwayGuard.zip --apple-id "you@example.com" --team-id "TEAMID" --password "@keychain:AC_PASSWORD"
```

---

## Xcode Project (Alternative)

If SPM doesn't work well with MenuBarExtra, create an Xcode project:

1. File → New → Project → macOS → App
2. Interface: SwiftUI
3. Life Cycle: SwiftUI App
4. Add package dependencies via File → Add Packages
5. Set LSUIElement = YES in Info.plist

The Xcode approach provides:
- Visual .app bundle management
- Integrated code signing
- Asset catalog support
- Easier debugging

---

## Dependency Notes

### AXorcist
- Requires macOS 14.0+ for full features
- MainActor-isolated for thread safety
- Fuzzy query matching for element discovery

### CGEventSupervisor
- Modern async/await API
- Event cancellation support
- Keyboard discriminators for key code filtering

### SwiftAnthropic
- Claude API client with streaming
- Supports all Claude models
- Image message support (for future features)

---

## Migration from Go Prototype

When implementation begins, the Go prototype at `projects/hemingway-guard/` should be:

1. Moved to `projects/hemingway-guard/archive/go-prototype/`
2. Swift project created in `projects/hemingway-guard/`
3. Go README updated to point to Swift implementation

The Go code serves as a reference for:
- Accessibility API usage patterns
- Event interception timing
- Target app bundle IDs
- Analysis prompt structure
