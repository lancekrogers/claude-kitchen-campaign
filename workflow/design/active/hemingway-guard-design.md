# HemingwayGuard - Text Interception for Messaging Apps

## Overview

HemingwayGuard is a macOS menubar application that intercepts messages before sending in iMessage, Discord, Slack, and other messaging apps to validate them for conciseness, readability, and context-appropriateness using the "Hemingway method."

## Architecture

### System Design

```
┌─────────────────────────────────────────────────────────────────────┐
│                    HemingwayGuard (menubar app)                     │
├─────────────────────────────────────────────────────────────────────┤
│  1. AXObserver monitors AXFocusedUIElementChanged system-wide       │
│  2. When focus → text field in target app, activate monitoring     │
│  3. CGEventTap intercepts Enter/Return key or detects Send action  │
│  4. Read text via kAXValueAttribute from focused element           │
│  5. Send to claude-code-go for Hemingway analysis                  │
│  6. Show approval/suggestion popover                                │
│  7. User accepts → release event | User edits → update field       │
└─────────────────────────────────────────────────────────────────────┘
```

### App Compatibility

| App Type | Detection Method |
|----------|------------------|
| **iMessage (native)** | AXTextArea with AXRole detection |
| **Slack (Electron)** | Electron exposes AX hierarchy to macOS |
| **Discord (Electron)** | Same - Electron apps are AX-compliant |
| **Web browsers** | AXWebArea → can detect contenteditable fields |

### Target App Bundle IDs

- `com.apple.MobileSMS` - iMessage
- `com.tinyspeck.slackmacgap` - Slack
- `com.hnc.Discord` - Discord

## Technical Components

### 1. Focus Monitor (AXObserver)

Uses macOS Accessibility API to monitor system-wide focus changes:

```go
// Pseudocode for focus monitoring
systemElement := AXUIElementCreateSystemWide()
observer := AXObserverCreate(pid, callback)
AXObserverAddNotification(observer, element, kAXFocusedUIElementChangedNotification)
```

When a text field gains focus:
- Check if it's in a target app (Messages, Slack, Discord)
- Check AXRole == AXTextField or AXTextArea
- Start keystroke monitoring for that field

### 2. Keystroke Interceptor (CGEventTap)

Intercepts Enter/Return key presses to prevent immediate message sending:

```go
// Filter for Return/Enter key (keyCode 36 and 76)
tap := CGEventTapCreate(kCGSessionEventTap, kCGHeadInsertEventTap, kCGEventTapOptionDefault,
                         CGEventMaskBit(kCGEventKeyDown), callback, nil)
```

When Enter is pressed in monitored context:
- Return `nil` to swallow the event (prevent send)
- Read text content from focused element
- Trigger validation flow

### 3. Text Analyzer (claude-code-go)

Uses claude-code-go SDK for Hemingway-style analysis:

```go
client := claudecodego.NewClient()
result, err := client.Query(ctx, claudecodego.QueryOptions{
    Prompt: buildHemingwayPrompt(messageText, appContext),
    OutputFormat: "json",
})
```

Analysis prompt returns:
```json
{
  "approved": true,
  "word_count": 42,
  "read_time_seconds": 8,
  "grade_level": 6.2,
  "issues": ["passive voice in sentence 2"],
  "suggestion": "shorter version if needed"
}
```

### 4. Approval UI (SwiftUI Popover)

- Non-blocking popover appears near the text field
- Shows analysis: word count, grade level, issues
- Buttons: "Send Anyway" | "Use Suggestion" | "Edit"
- If approved → release the held keystroke
- If edited → update AXValue and release

## Permissions Required

1. **Accessibility** (System Preferences → Privacy & Security → Accessibility)
   - Required for AXUIElement access and AXObserver

2. **Input Monitoring** (System Preferences → Privacy & Security → Input Monitoring)
   - Required for CGEventTap to intercept keystrokes

## Project Structure

```
hemingway-guard/
├── cmd/
│   └── hemingway-guard/
│       └── main.go              # App entry point
├── internal/
│   ├── accessibility/
│   │   ├── observer.go          # AXObserver wrapper
│   │   ├── element.go           # AXUIElement helpers
│   │   └── focus_monitor.go     # Focus change detection
│   ├── keyboard/
│   │   ├── event_tap.go         # CGEventTap wrapper
│   │   └── interceptor.go       # Enter key interception logic
│   ├── analyzer/
│   │   └── hemingway.go         # claude-code-go integration
│   └── ui/
│       ├── menubar.go           # Status bar icon
│       └── popover.swift        # SwiftUI approval popover (bridged)
├── pkg/
│   └── apps/
│       └── targets.go           # App bundle IDs to monitor
├── justfile
└── go.mod
```

## Implementation Phases

### Phase 1: Core Infrastructure
1. Set up Go project with CGO for macOS APIs
2. Implement AXUIElementCreateSystemWide wrapper
3. Implement basic focus monitoring (log focused element info)
4. Test with different apps to verify AX hierarchy visibility

### Phase 2: Keystroke Interception
1. Implement CGEventTap for Enter key
2. Add logic to swallow event conditionally
3. Implement text extraction from focused element
4. Test intercept → extract → log flow

### Phase 3: LLM Integration
1. Integrate claude-code-go SDK
2. Design and test Hemingway analysis prompt
3. Implement JSON response parsing
4. Handle timeouts and errors gracefully

### Phase 4: Approval UI
1. Create SwiftUI popover (bridged to Go)
2. Show analysis results
3. Implement "Send Anyway" / "Use Suggestion" actions
4. Handle text field updates via AXUIElementSetAttributeValue

### Phase 5: Polish
1. Menubar app with enable/disable toggle
2. Settings: target apps, prompt customization
3. Handle edge cases (multi-line messages, keyboard shortcuts)
4. Package as .app bundle

## Technical Challenges & Solutions

### 1. Swift/Go Bridge for UI

**Recommendation**: Separate Swift helper for popover, communicate via XPC or Unix socket.

Options considered:
- cgo + Objective-C runtime - Spawn Objective-C/Swift code from Go
- Separate Swift process - Go daemon + Swift UI helper (IPC via Unix socket)
- Pure Go UI - Use fyne or similar, but less native feel

### 2. Event Timing

- CGEventTap callback must return quickly
- LLM call is async (100-500ms)
- **Solution**: Hold the event reference, complete callback, then re-inject or discard

### 3. App Detection

```go
// Check focused app's bundle ID
pid := focusedApp.ProcessIdentifier()
bundleID := bundleIDForPID(pid)
if isTargetApp(bundleID) { ... }
```

## Verification Plan

1. **Unit tests**: Mock accessibility elements, test focus detection logic
2. **Integration test**: Launch Messages.app, type text, verify interception
3. **Manual testing**:
   - iMessage: compose → press Enter → verify popover
   - Slack: DM → press Enter → verify popover
   - Discord: channel message → press Enter → verify popover
4. **Edge cases**:
   - Shift+Enter (should NOT trigger)
   - Empty message
   - Very long message (>1000 chars)
   - Secure input fields (should be ignored)

## References

- [macOS Accessibility API - System-wide Text Access](https://macdevelopers.wordpress.com/2014/01/31/accessing-text-value-from-any-system-wide-application-via-accessibility-api/)
- [CGEventTap Keyboard Monitoring](https://www.logcg.com/en/archives/2902.html)
- [CGEventSupervisor Swift Library](https://github.com/stephancasas/CGEventSupervisor)
- [AXorcist - Swift Accessibility Wrapper](https://github.com/steipete/AXorcist)
- [Hammerspoon AXUIElement Docs](https://www.hammerspoon.org/docs/hs.axuielement.html)
- [Apple Accessibility Programming Guide](https://developer.apple.com/library/archive/documentation/Accessibility/Conceptual/AccessibilityMacOSX/OSXAXmodel.html)
- [InputMethodKit Documentation](https://developer.apple.com/documentation/inputmethodkit)
- [claude-code-go SDK](https://github.com/lancekrogers/claude-code-go)
