# HemingwayGuard Design Documentation

## Overview

HemingwayGuard is a macOS menubar application that intercepts messages before sending in iMessage, Discord, Slack, and other messaging apps to validate them for conciseness, readability, and context-appropriateness using the "Hemingway method."

## Quick Navigation

| Document | Description |
|----------|-------------|
| [architecture.md](./architecture.md) | System design, component diagram, data flow |
| [components.md](./components.md) | Detailed specifications for each component |
| [swift-project.md](./swift-project.md) | Swift project structure, dependencies, Package.swift |
| [api-contracts.md](./api-contracts.md) | LLM prompts, JSON schemas, response formats |
| [verification.md](./verification.md) | Test plan, manual testing checklist, edge cases |

---

## Decision Log

### 2025-02-01: Go → Swift Migration

**Decision:** Migrate from Go+CGO to pure Swift implementation.

**Rationale:**

| Factor | Go+CGO | Swift |
|--------|--------|-------|
| Native API Access | 3/10 (custom CGO) | 10/10 (AXorcist, CGEventSupervisor) |
| Framework Maturity | 5/10 (rare for this domain) | 10/10 (production apps exist) |
| Code Simplicity | 7/10 (CGO complexity) | 9/10 (no FFI) |
| Build/Distribution | 7/10 (manual bundling) | 9/10 (Xcode integrated) |
| LLM SDK Support | 8/10 (claude-code-go) | 8/10 (SwiftAnthropic) |

**Go Implementation Issues Found:**
- ~450 lines of C/Objective-C bridging code
- Unsafe `uintptr` nil comparisons instead of proper type safety
- Manual string memory management with leak potential
- Global mutable callback state (single instance limitation)
- Reference ownership ambiguity documented only in comments
- Thread/RunLoop complexity mixing Go goroutines with Cocoa patterns

**Swift Advantages:**
- **AXorcist** - Production-grade accessibility wrapper, MainActor-isolated
- **CGEventSupervisor** - Modern async/await keyboard events
- **SwiftUI MenuBarExtra** - Native menubar scene (macOS 13+)
- **SwiftAnthropic** - Claude API SDK with image support
- No FFI, no manual memory management, proper type safety

**Status:** Go prototype exists at `projects/hemingway-guard/` (to be archived when implementation begins)

---

## Target Apps

| App | Bundle ID | Notes |
|-----|-----------|-------|
| Messages | `com.apple.MobileSMS` | Native Cocoa, AXTextArea |
| Slack | `com.tinyspeck.slackmacgap` | Electron, AX-compliant |
| Discord | `com.hnc.Discord` | Electron, AX-compliant |

Future expansion possible for any app exposing text fields via Accessibility API.

---

## Permissions Required

1. **Accessibility** (System Preferences → Privacy & Security → Accessibility)
   - Required for AXUIElement access and focus monitoring

2. **Input Monitoring** (System Preferences → Privacy & Security → Input Monitoring)
   - Required for CGEventTap keyboard interception

Both permissions prompt the user on first launch. macOS Sequoia (15.0+) may require periodic re-authorization.

---

## References

### Swift Libraries
- [AXorcist](https://github.com/steipete/AXorcist) - Swift Accessibility wrapper
- [CGEventSupervisor](https://github.com/stephancasas/CGEventSupervisor) - Keyboard event interception
- [SwiftAnthropic](https://github.com/jamesrochabrun/SwiftAnthropic) - Claude API SDK

### Apple Documentation
- [Accessibility Programming Guide](https://developer.apple.com/library/archive/documentation/Accessibility/Conceptual/AccessibilityMacOSX/OSXAXmodel.html)
- [Quartz Event Services](https://developer.apple.com/documentation/coregraphics/quartz_event_services)
- [MenuBarExtra](https://developer.apple.com/documentation/swiftui/menubarextra)

### Related Projects
- [Hammerspoon](https://www.hammerspoon.org/) - macOS automation with Lua
- [AXSwift](https://github.com/tmandry/AXSwift) - Alternative accessibility wrapper
