# HemingwayGuard Architecture

## System Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         HemingwayGuard.app                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌──────────────────┐  ┌───────────────────┐  ┌──────────────────────────┐ │
│  │   MenuBarApp     │  │   FocusMonitor    │  │   KeyboardInterceptor    │ │
│  │   (SwiftUI)      │  │   (AXorcist)      │  │   (CGEventSupervisor)    │ │
│  │                  │  │                   │  │                          │ │
│  │  • Enable toggle │  │  • System-wide    │  │  • Enter key detection   │ │
│  │  • Status icon   │  │    focus tracking │  │  • Event swallow/release │ │
│  │  • Settings menu │  │  • Target app     │  │  • Modifier key handling │ │
│  │                  │  │    detection      │  │                          │ │
│  └────────┬─────────┘  └─────────┬─────────┘  └────────────┬─────────────┘ │
│           │                      │                         │               │
│           └──────────────────────┼─────────────────────────┘               │
│                                  │                                         │
│                         ┌────────▼────────┐                                │
│                         │   Coordinator   │                                │
│                         │                 │                                │
│                         │  • State machine│                                │
│                         │  • Flow control │                                │
│                         │  • Error handling│                               │
│                         └────────┬────────┘                                │
│                                  │                                         │
│           ┌──────────────────────┼──────────────────────┐                  │
│           │                      │                      │                  │
│           ▼                      ▼                      ▼                  │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐            │
│  │    Analyzer     │  │   PopoverUI     │  │    Settings     │            │
│  │                 │  │                 │  │                 │            │
│  │  • Claude API   │  │  • Results view │  │  • Target apps  │            │
│  │  • Prompt build │  │  • User actions │  │  • Preferences  │            │
│  │  • Response parse│ │  • Edit mode    │  │  • UserDefaults │            │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘            │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Data Flow

### Normal Flow (Message Approved)

```
┌─────────┐    ┌─────────────┐    ┌─────────────┐    ┌──────────┐
│  User   │───▶│ Text Field  │───▶│ FocusMonitor│───▶│Coordinator│
│ types   │    │ (Slack etc) │    │ detects     │    │ activates │
└─────────┘    └─────────────┘    └─────────────┘    └─────┬────┘
                                                          │
┌─────────┐    ┌─────────────┐    ┌─────────────┐    ┌────▼─────┐
│  User   │───▶│ Enter Key   │───▶│ Interceptor │───▶│Coordinator│
│ presses │    │             │    │ catches     │    │ analyzes  │
└─────────┘    └─────────────┘    └─────────────┘    └─────┬────┘
                                                          │
                                                    ┌─────▼─────┐
                                                    │  Analyzer │
                                                    │ calls API │
                                                    └─────┬─────┘
                                                          │
                                                    ┌─────▼─────┐
                                                    │ approved: │
                                                    │   true    │
                                                    └─────┬─────┘
                                                          │
┌─────────┐    ┌─────────────┐    ┌─────────────┐    ┌────▼─────┐
│ Message │◀───│ Enter Key   │◀───│ Interceptor │◀───│Coordinator│
│  sent   │    │ released    │    │ releases    │    │ approves  │
└─────────┘    └─────────────┘    └─────────────┘    └──────────┘
```

### Review Flow (Issues Found)

```
┌──────────┐    ┌─────────────┐    ┌─────────────┐
│ Analyzer │───▶│ approved:   │───▶│ Coordinator │
│ returns  │    │   false     │    │ shows popup │
└──────────┘    └─────────────┘    └──────┬──────┘
                                          │
                                    ┌─────▼─────┐
                                    │ PopoverUI │
                                    │ displays  │
                                    └─────┬─────┘
                                          │
            ┌─────────────────────────────┼─────────────────────────────┐
            │                             │                             │
            ▼                             ▼                             ▼
   ┌─────────────────┐           ┌─────────────────┐           ┌─────────────────┐
   │  Send Anyway    │           │ Use Suggestion  │           │     Edit        │
   │                 │           │                 │           │                 │
   │  Release Enter  │           │ Update text,    │           │ Close popover,  │
   │  as-is          │           │ release Enter   │           │ user continues  │
   └─────────────────┘           └─────────────────┘           └─────────────────┘
```

---

## State Machine

```
                              ┌─────────────┐
                              │    IDLE     │
                              │             │
                              └──────┬──────┘
                                     │
                          Focus on target app
                                     │
                              ┌──────▼──────┐
                              │  MONITORING │◀─────────────────────┐
                              │             │                      │
                              └──────┬──────┘                      │
                                     │                             │
                              Enter pressed                        │
                                     │                             │
                              ┌──────▼──────┐                      │
                              │  ANALYZING  │                      │
                              │             │                      │
                              └──────┬──────┘                      │
                                     │                             │
                    ┌────────────────┼────────────────┐            │
                    │                │                │            │
               Approved          Issues found      Error           │
                    │                │                │            │
             ┌──────▼──────┐  ┌──────▼──────┐  ┌─────▼─────┐      │
             │  RELEASING  │  │  REVIEWING  │  │  ERROR    │      │
             │             │  │             │  │           │      │
             └──────┬──────┘  └──────┬──────┘  └─────┬─────┘      │
                    │                │                │            │
                    │         User action             │            │
                    │                │                │            │
                    └────────────────┴────────────────┴────────────┘
```

### State Definitions

| State | Description | Transitions |
|-------|-------------|-------------|
| **IDLE** | App running, not monitoring any text field | → MONITORING (focus detected) |
| **MONITORING** | Focus on target app text field, watching for Enter | → ANALYZING (Enter pressed), → IDLE (focus lost) |
| **ANALYZING** | LLM analysis in progress | → RELEASING (approved), → REVIEWING (issues), → ERROR (timeout/failure) |
| **REVIEWING** | Popover shown, awaiting user action | → RELEASING (any action) |
| **RELEASING** | Re-injecting Enter key | → MONITORING |
| **ERROR** | Analysis failed, allowing message through | → MONITORING |

---

## Component Interactions

### Startup Sequence

```
1. App launches
2. Check Accessibility permission → prompt if needed
3. Check Input Monitoring permission → prompt if needed
4. Initialize FocusMonitor (AXorcist)
5. Initialize KeyboardInterceptor (CGEventSupervisor)
6. Initialize Analyzer (SwiftAnthropic client)
7. Show menubar icon
8. Enter IDLE state
```

### Shutdown Sequence

```
1. User clicks Quit
2. Disable KeyboardInterceptor
3. Stop FocusMonitor
4. Release any held events
5. Terminate app
```

---

## Threading Model

```
┌─────────────────────────────────────────────────────────────────┐
│                        Main Thread (@MainActor)                 │
├─────────────────────────────────────────────────────────────────┤
│  • SwiftUI views (MenuBar, Popover)                             │
│  • Coordinator state machine                                    │
│  • Settings access                                              │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ async/await
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Background Tasks                           │
├─────────────────────────────────────────────────────────────────┤
│  • Analyzer.analyze() - HTTP request to Claude API              │
│  • FocusMonitor callbacks (dispatched to MainActor)             │
│  • KeyboardInterceptor callbacks (dispatched to MainActor)      │
└─────────────────────────────────────────────────────────────────┘
```

All UI and state updates happen on `@MainActor`. Background work (API calls) uses Swift's structured concurrency with `async/await`.

---

## Error Handling Strategy

| Error Type | Handling | User Impact |
|------------|----------|-------------|
| **Accessibility denied** | Show alert, disable monitoring | App non-functional until granted |
| **Input Monitoring denied** | Show alert, disable interception | Messages sent without analysis |
| **API timeout (>3s)** | Auto-release Enter, log warning | Message sent without analysis |
| **API error** | Auto-release Enter, show brief notification | Message sent without analysis |
| **Focus lost during analysis** | Cancel analysis, release if held | Message may be delayed |
| **Invalid JSON response** | Treat as approved, log error | Message sent without analysis |

**Principle:** Never block the user. If anything fails, allow the message to send.
