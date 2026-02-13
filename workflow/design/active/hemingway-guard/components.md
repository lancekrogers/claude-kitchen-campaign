# HemingwayGuard Component Specifications

## Component Overview

| Component | Library | Responsibility |
|-----------|---------|----------------|
| [MenuBarApp](#menubarapp) | SwiftUI MenuBarExtra | App entry, status icon, settings menu |
| [FocusMonitor](#focusmonitor) | AXorcist | System-wide focus tracking, target app detection |
| [KeyboardInterceptor](#keyboardinterceptor) | CGEventSupervisor | Enter key interception, event management |
| [Coordinator](#coordinator) | Custom | State machine, flow orchestration |
| [Analyzer](#analyzer) | SwiftAnthropic | Claude API integration, prompt building |
| [PopoverUI](#popoverui) | SwiftUI | Analysis results, user actions |
| [Settings](#settings) | UserDefaults | Preferences, target app configuration |

---

## MenuBarApp

### Purpose
Main app entry point using SwiftUI's `MenuBarExtra` for macOS 13+.

### Interface

```swift
@main
struct HemingwayGuardApp: App {
    @StateObject private var coordinator = Coordinator()

    var body: some Scene {
        MenuBarExtra {
            MenuBarView(coordinator: coordinator)
        } label: {
            Label("HemingwayGuard", systemImage: coordinator.statusIcon)
        }
        .menuBarExtraStyle(.menu)

        Settings {
            SettingsView()
        }
    }
}
```

### Status Icons

| State | Icon | Description |
|-------|------|-------------|
| Idle/Disabled | `pencil.slash` | Not monitoring |
| Monitoring | `pencil` | Watching for Enter key |
| Analyzing | `pencil.circle` | LLM request in progress |
| Issue Found | `exclamationmark.triangle` | Popover shown |

### Menu Items

```
┌─────────────────────────────┐
│ ☑ Enabled                   │
├─────────────────────────────┤
│ Target Apps...              │
│ Preferences...              │
├─────────────────────────────┤
│ About HemingwayGuard        │
│ Quit                        │
└─────────────────────────────┘
```

---

## FocusMonitor

### Purpose
Track system-wide focus changes using AXorcist to detect when user focuses a text field in a target application.

### Interface

```swift
@MainActor
final class FocusMonitor: ObservableObject {
    @Published var currentApp: TargetApp?
    @Published var isTextFieldFocused: Bool = false
    @Published var focusedElement: AXUIElement?

    func start() async throws
    func stop()

    func getCurrentText() -> String?
    func setCurrentText(_ text: String) throws
}
```

### Dependencies

```swift
import AXorcist

// AXorcist provides:
// - AXUIElement wrapper with Swift-native API
// - Focus change notifications
// - Text field detection via AXRole
// - Value reading/writing via AXValue attribute
```

### Detection Logic

```swift
func handleFocusChange(_ element: AXUIElement) {
    // 1. Get owning application's bundle ID
    let bundleID = element.application?.bundleIdentifier

    // 2. Check if it's a target app
    guard let app = Settings.shared.targetApps.first(where: { $0.bundleID == bundleID }) else {
        currentApp = nil
        return
    }

    // 3. Check if focused element is a text field
    let role = element.role
    guard role == .textField || role == .textArea else {
        isTextFieldFocused = false
        return
    }

    // 4. Activate monitoring
    currentApp = app
    isTextFieldFocused = true
    focusedElement = element
}
```

### Target App Detection

| Bundle ID | App Name | Notes |
|-----------|----------|-------|
| `com.apple.MobileSMS` | Messages | Native AXTextArea |
| `com.tinyspeck.slackmacgap` | Slack | Electron, web-based text areas |
| `com.hnc.Discord` | Discord | Electron, contenteditable divs |

---

## KeyboardInterceptor

### Purpose
Intercept Enter/Return key presses using CGEventSupervisor to prevent immediate message sending.

### Interface

```swift
@MainActor
final class KeyboardInterceptor: ObservableObject {
    @Published var isIntercepting: Bool = false

    var onEnterPressed: (() async -> Bool)?  // Return true to allow, false to swallow

    func start() throws
    func stop()
    func releaseEnter()
}
```

### Dependencies

```swift
import CGEventSupervisor

// CGEventSupervisor provides:
// - Async/await event handling
// - Event cancellation (return nil to swallow)
// - Modifier key detection
// - Key code discriminators
```

### Key Handling Logic

```swift
func setupEventTap() {
    CGEventSupervisor.shared.subscribe(to: .keyDown) { [weak self] event in
        guard let self = self else { return event }

        // Only intercept plain Enter (keyCode 36 or 76)
        guard event.keyCode == 36 || event.keyCode == 76 else {
            return event
        }

        // Allow Shift+Enter (newline), Cmd+Enter, etc.
        if event.flags.contains(.shift) ||
           event.flags.contains(.command) ||
           event.flags.contains(.control) {
            return event
        }

        // Check if we should intercept
        guard isIntercepting, let handler = onEnterPressed else {
            return event
        }

        // Swallow the event, trigger async handler
        Task { @MainActor in
            let shouldAllow = await handler()
            if shouldAllow {
                self.releaseEnter()
            }
        }

        return nil  // Swallow the event
    }
}

func releaseEnter() {
    // Post synthetic Enter key event
    let keyDown = CGEvent(keyboardEventSource: nil, virtualKey: 36, keyDown: true)
    let keyUp = CGEvent(keyboardEventSource: nil, virtualKey: 36, keyDown: false)
    keyDown?.post(tap: .cghidEventTap)
    keyUp?.post(tap: .cghidEventTap)
}
```

### Key Codes

| Key | Code | Notes |
|-----|------|-------|
| Return | 36 | Main keyboard |
| Enter | 76 | Numpad |
| Shift | flag | `CGEventFlags.maskShift` |
| Command | flag | `CGEventFlags.maskCommand` |

---

## Coordinator

### Purpose
Central state machine coordinating all components.

### Interface

```swift
@MainActor
final class Coordinator: ObservableObject {
    enum State {
        case idle
        case monitoring(app: TargetApp)
        case analyzing
        case reviewing(result: AnalysisResult)
        case releasing
        case error(Error)
    }

    @Published var state: State = .idle
    @Published var isEnabled: Bool = true

    private let focusMonitor = FocusMonitor()
    private let interceptor = KeyboardInterceptor()
    private let analyzer = Analyzer()

    var statusIcon: String { /* based on state */ }

    func start() async throws
    func stop()
    func handleUserAction(_ action: UserAction)
}
```

### State Transitions

```swift
func handleEnterPressed() async -> Bool {
    guard case .monitoring(let app) = state else { return true }

    state = .analyzing

    do {
        let text = focusMonitor.getCurrentText() ?? ""
        let result = try await analyzer.analyze(text: text, app: app)

        if result.approved {
            state = .releasing
            return true  // Allow Enter
        } else {
            state = .reviewing(result: result)
            showPopover(result: result)
            return false  // Swallow Enter, wait for user action
        }
    } catch {
        state = .error(error)
        return true  // On error, allow message through
    }
}

func handleUserAction(_ action: UserAction) {
    switch action {
    case .sendAnyway:
        state = .releasing
        interceptor.releaseEnter()

    case .useSuggestion(let text):
        try? focusMonitor.setCurrentText(text)
        state = .releasing
        interceptor.releaseEnter()

    case .edit:
        state = .monitoring(app: currentApp!)
        dismissPopover()
    }
}
```

---

## Analyzer

### Purpose
Send messages to Claude API for Hemingway-style analysis.

### Interface

```swift
final class Analyzer {
    func analyze(text: String, app: TargetApp) async throws -> AnalysisResult
}

struct AnalysisResult: Codable {
    let approved: Bool
    let wordCount: Int
    let readTimeSeconds: Int
    let gradeLevel: Double
    let issues: [String]
    let suggestion: String
}
```

### Dependencies

```swift
import SwiftAnthropic

// SwiftAnthropic provides:
// - Claude API client
// - Streaming support
// - Error handling
// - Image support (if needed)
```

### Implementation

```swift
final class Analyzer {
    private let client: Anthropic

    init() {
        client = Anthropic(apiKey: Settings.shared.apiKey)
    }

    func analyze(text: String, app: TargetApp) async throws -> AnalysisResult {
        let prompt = buildPrompt(text: text, context: app.name)

        let response = try await client.messages.create(
            model: .claude3Sonnet,
            maxTokens: 1024,
            messages: [.user(prompt)]
        )

        guard let content = response.content.first?.text else {
            throw AnalyzerError.emptyResponse
        }

        return try JSONDecoder().decode(AnalysisResult.self, from: content.data(using: .utf8)!)
    }

    private func buildPrompt(text: String, context: String) -> String {
        // See api-contracts.md for full prompt
    }
}
```

### Timeout Handling

```swift
func analyzeWithTimeout(text: String, app: TargetApp) async throws -> AnalysisResult {
    try await withThrowingTaskGroup(of: AnalysisResult.self) { group in
        group.addTask {
            try await self.analyze(text: text, app: app)
        }

        group.addTask {
            try await Task.sleep(nanoseconds: 3_000_000_000)  // 3 seconds
            throw AnalyzerError.timeout
        }

        let result = try await group.next()!
        group.cancelAll()
        return result
    }
}
```

---

## PopoverUI

### Purpose
Display analysis results and capture user action.

### Interface

```swift
struct ApprovalPopoverView: View {
    let result: AnalysisResult
    let originalText: String
    let onAction: (UserAction) -> Void

    var body: some View { /* ... */ }
}

enum UserAction {
    case sendAnyway
    case useSuggestion(String)
    case edit
}
```

### Layout

```
┌────────────────────────────────────────┐
│  ⚠️ Review Suggested                   │
├────────────────────────────────────────┤
│                                        │
│  Words: 42    Read: 8s    Grade: 6.2   │
│                                        │
├────────────────────────────────────────┤
│  Issues:                               │
│  • passive voice in sentence 2         │
│  • message could be more concise       │
│                                        │
├────────────────────────────────────────┤
│  Suggested:                            │
│  ┌──────────────────────────────────┐  │
│  │ "Shorter version here..."       │  │
│  └──────────────────────────────────┘  │
│                                        │
├────────────────────────────────────────┤
│  [Edit]  [Use Suggestion]  [Send ▶]   │
└────────────────────────────────────────┘
```

---

## Settings

### Purpose
Persist user preferences using UserDefaults.

### Interface

```swift
final class Settings: ObservableObject {
    static let shared = Settings()

    @AppStorage("isEnabled") var isEnabled: Bool = true
    @AppStorage("apiKey") var apiKey: String = ""
    @AppStorage("targetAppBundleIDs") var targetAppBundleIDs: [String] = [
        "com.apple.MobileSMS",
        "com.tinyspeck.slackmacgap",
        "com.hnc.Discord"
    ]

    var targetApps: [TargetApp] {
        targetAppBundleIDs.compactMap { TargetApp(bundleID: $0) }
    }
}

struct TargetApp: Identifiable {
    let bundleID: String
    let name: String
    let icon: NSImage?

    init?(bundleID: String) {
        // Look up app info from bundle ID
    }
}
```

### Stored Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| `isEnabled` | Bool | true | Global enable/disable |
| `apiKey` | String | "" | Claude API key |
| `targetAppBundleIDs` | [String] | [Messages, Slack, Discord] | Apps to monitor |
| `maxWordCount` | Int | 100 | Threshold for "too long" warning |
| `autoSendOnApproved` | Bool | true | Auto-release Enter if approved |
