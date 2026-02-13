# HemingwayGuard Verification Plan

## Test Categories

| Category | Type | Coverage |
|----------|------|----------|
| [Unit Tests](#unit-tests) | Automated | Core logic, parsing, state machine |
| [Integration Tests](#integration-tests) | Automated | Component interactions |
| [Manual Tests](#manual-testing-checklist) | Manual | Real-world app behavior |
| [Edge Cases](#edge-cases) | Both | Boundary conditions |
| [Permission Tests](#permission-testing) | Manual | macOS security prompts |

---

## Unit Tests

### AnalyzerTests.swift

```swift
import XCTest
@testable import HemingwayGuard

final class AnalyzerTests: XCTestCase {

    // MARK: - JSON Parsing

    func testParseApprovedResponse() throws {
        let json = """
        {
            "approved": true,
            "word_count": 12,
            "read_time_seconds": 3,
            "grade_level": 5.2,
            "issues": [],
            "suggestion": ""
        }
        """

        let result = try JSONDecoder().decode(AnalysisResult.self, from: json.data(using: .utf8)!)

        XCTAssertTrue(result.approved)
        XCTAssertEqual(result.wordCount, 12)
        XCTAssertEqual(result.readTimeSeconds, 3)
        XCTAssertEqual(result.gradeLevel, 5.2, accuracy: 0.01)
        XCTAssertTrue(result.issues.isEmpty)
        XCTAssertEqual(result.suggestion, "")
    }

    func testParseIssuesResponse() throws {
        let json = """
        {
            "approved": false,
            "word_count": 87,
            "read_time_seconds": 26,
            "grade_level": 9.4,
            "issues": ["Too long", "Passive voice"],
            "suggestion": "Shorter version"
        }
        """

        let result = try JSONDecoder().decode(AnalysisResult.self, from: json.data(using: .utf8)!)

        XCTAssertFalse(result.approved)
        XCTAssertEqual(result.issues.count, 2)
        XCTAssertFalse(result.suggestion.isEmpty)
    }

    func testParseInvalidJSON() {
        let json = "not valid json"

        XCTAssertThrowsError(
            try JSONDecoder().decode(AnalysisResult.self, from: json.data(using: .utf8)!)
        )
    }

    // MARK: - Prompt Building

    func testBuildPromptIncludesContext() {
        let builder = PromptBuilder()
        let prompt = builder.build(text: "Hello world", app: .slack)

        XCTAssertTrue(prompt.contains("Slack"))
        XCTAssertTrue(prompt.contains("workplace chat"))
        XCTAssertTrue(prompt.contains("Hello world"))
    }

    func testBuildPromptEscapesSpecialCharacters() {
        let builder = PromptBuilder()
        let prompt = builder.build(text: "Test \"quotes\" and {braces}", app: .messages)

        XCTAssertTrue(prompt.contains("quotes"))
        XCTAssertTrue(prompt.contains("braces"))
    }

    // MARK: - Local Fallback

    func testLocalAnalyzerLongMessage() {
        let analyzer = LocalAnalyzer()
        let longText = String(repeating: "word ", count: 150)

        let result = analyzer.analyze(text: longText, context: .dm)

        XCTAssertFalse(result.approved)
        XCTAssertTrue(result.issues.contains { $0.contains("long") })
    }

    func testLocalAnalyzerPassiveVoice() {
        let analyzer = LocalAnalyzer()
        let text = "The report was written by the team."

        let result = analyzer.analyze(text: text, context: .channel)

        XCTAssertTrue(result.issues.contains { $0.contains("passive") })
    }
}
```

### CoordinatorTests.swift

```swift
import XCTest
@testable import HemingwayGuard

final class CoordinatorTests: XCTestCase {

    // MARK: - State Transitions

    func testIdleToMonitoringOnFocus() async {
        let coordinator = Coordinator()
        let mockFocusMonitor = MockFocusMonitor()
        coordinator.focusMonitor = mockFocusMonitor

        await coordinator.start()
        mockFocusMonitor.simulateFocus(app: .slack)

        XCTAssertEqual(coordinator.state, .monitoring(app: .slack))
    }

    func testMonitoringToAnalyzingOnEnter() async {
        let coordinator = Coordinator()
        coordinator.state = .monitoring(app: .slack)

        let mockAnalyzer = MockAnalyzer()
        mockAnalyzer.mockResult = AnalysisResult(approved: true, ...)
        coordinator.analyzer = mockAnalyzer

        _ = await coordinator.handleEnterPressed()

        // Should have transitioned through analyzing
        XCTAssertEqual(mockAnalyzer.analyzeCallCount, 1)
    }

    func testApprovedMessageReleasesEnter() async {
        let coordinator = Coordinator()
        coordinator.state = .monitoring(app: .slack)

        let mockAnalyzer = MockAnalyzer()
        mockAnalyzer.mockResult = AnalysisResult(approved: true, ...)
        coordinator.analyzer = mockAnalyzer

        let shouldRelease = await coordinator.handleEnterPressed()

        XCTAssertTrue(shouldRelease)
    }

    func testIssuesShowsPopover() async {
        let coordinator = Coordinator()
        coordinator.state = .monitoring(app: .slack)

        let mockAnalyzer = MockAnalyzer()
        mockAnalyzer.mockResult = AnalysisResult(
            approved: false,
            issues: ["Too long"],
            ...
        )
        coordinator.analyzer = mockAnalyzer

        let shouldRelease = await coordinator.handleEnterPressed()

        XCTAssertFalse(shouldRelease)
        if case .reviewing(let result) = coordinator.state {
            XCTAssertFalse(result.approved)
        } else {
            XCTFail("Expected reviewing state")
        }
    }

    // MARK: - Error Handling

    func testTimeoutAllowsMessage() async {
        let coordinator = Coordinator()
        coordinator.state = .monitoring(app: .slack)

        let mockAnalyzer = MockAnalyzer()
        mockAnalyzer.shouldTimeout = true
        coordinator.analyzer = mockAnalyzer

        let shouldRelease = await coordinator.handleEnterPressed()

        XCTAssertTrue(shouldRelease)  // On error, allow message
    }

    // MARK: - User Actions

    func testSendAnywayReleasesEnter() {
        let coordinator = Coordinator()
        let mockInterceptor = MockKeyInterceptor()
        coordinator.interceptor = mockInterceptor
        coordinator.state = .reviewing(result: AnalysisResult(...))

        coordinator.handleUserAction(.sendAnyway)

        XCTAssertEqual(mockInterceptor.releaseEnterCallCount, 1)
    }

    func testUseSuggestionUpdatesText() {
        let coordinator = Coordinator()
        let mockFocusMonitor = MockFocusMonitor()
        coordinator.focusMonitor = mockFocusMonitor
        coordinator.state = .reviewing(result: AnalysisResult(suggestion: "New text", ...))

        coordinator.handleUserAction(.useSuggestion("New text"))

        XCTAssertEqual(mockFocusMonitor.lastSetText, "New text")
    }
}
```

---

## Integration Tests

### Focus + Interceptor Integration

```swift
func testFocusActivatesInterceptor() async throws {
    let coordinator = Coordinator()
    try await coordinator.start()

    // Simulate focus on Slack text field
    // (Requires actual AXorcist integration or sophisticated mocking)

    XCTAssertTrue(coordinator.interceptor.isActive)
}
```

### End-to-End Flow (Mocked API)

```swift
func testEndToEndApprovedFlow() async throws {
    let coordinator = Coordinator()
    coordinator.analyzer = MockAnalyzer(alwaysApprove: true)

    try await coordinator.start()

    // Simulate focus
    coordinator.focusMonitor.simulateFocus(app: .messages)

    // Simulate Enter press
    let released = await coordinator.handleEnterPressed()

    XCTAssertTrue(released)
    XCTAssertEqual(coordinator.state, .releasing)
}
```

---

## Manual Testing Checklist

### Setup

- [ ] App launches without crash
- [ ] Menubar icon appears (pencil icon)
- [ ] Accessibility permission prompt appears on first launch
- [ ] Input Monitoring permission prompt appears on first launch
- [ ] API key can be entered in Settings

### Messages.app (iMessage)

- [ ] Focus on composition field detected
- [ ] Icon changes to indicate monitoring
- [ ] Typing "Hello" and pressing Enter → message sends immediately (short message approved)
- [ ] Typing 150+ words and pressing Enter → popover appears
- [ ] "Send Anyway" → message sends
- [ ] "Use Suggestion" → text replaced, message sends
- [ ] "Edit" → popover closes, user can continue editing
- [ ] Shift+Enter → newline inserted, no interception

### Slack

- [ ] Focus on DM text field detected
- [ ] Focus on channel text field detected
- [ ] Same interception behavior as Messages
- [ ] Thread replies work correctly
- [ ] Slash commands not intercepted (start with /)

### Discord

- [ ] Focus on message input detected
- [ ] Same interception behavior as Messages
- [ ] Emoji picker doesn't interfere
- [ ] Rich embeds don't break text extraction

### Edge Cases (see below)

- [ ] All edge cases verified

### Performance

- [ ] Analysis completes in <3 seconds
- [ ] No noticeable typing lag
- [ ] App uses <50MB memory idle
- [ ] CPU usage minimal when idle

---

## Edge Cases

### Input Variations

| Case | Expected Behavior | Test |
|------|-------------------|------|
| Empty message | Allow (no analysis) | Press Enter with empty field |
| Single character | Allow (too short to analyze) | Type "k" and Enter |
| Very long (>1000 chars) | Analyze, likely flag | Paste long text |
| Unicode/emoji only | Allow | Send "👍👍👍" |
| Code blocks | Analyze but be lenient | Send ``` code ``` |
| URLs only | Allow | Send a URL |
| Mixed content | Analyze normally | Text + URL + emoji |

### Modifier Keys

| Case | Expected Behavior | Test |
|------|-------------------|------|
| Shift+Enter | Newline, no interception | Hold Shift, press Enter |
| Cmd+Enter | Pass through (some apps use this) | Hold Cmd, press Enter |
| Ctrl+Enter | Pass through | Hold Ctrl, press Enter |
| Option+Enter | Pass through | Hold Option, press Enter |

### Focus Changes

| Case | Expected Behavior | Test |
|------|-------------------|------|
| Switch apps during analysis | Cancel analysis, release if held | Cmd+Tab while analyzing |
| Close chat window | Cancel, reset state | Close window while typing |
| Switch conversations | Reset monitoring | Click different chat |
| Background/foreground | Maintain state | Minimize and restore |

### API Failures

| Case | Expected Behavior | Test |
|------|-------------------|------|
| Network offline | Allow message, show notification | Disable WiFi |
| API timeout | Allow message after 3s | (Simulate slow response) |
| Invalid response | Allow message, log error | (Simulate bad JSON) |
| Rate limited | Allow message, show notification | (Simulate 429 response) |

### Secure Input

| Case | Expected Behavior | Test |
|------|-------------------|------|
| Password field | Do NOT intercept | Focus on password input |
| 1Password/etc | Do NOT intercept | Open password manager |

---

## Permission Testing

### First Launch Flow

1. Launch app for first time
2. Verify Accessibility permission dialog appears
3. Click "Open System Preferences"
4. Grant permission in System Preferences
5. Verify Input Monitoring dialog appears (or shows in sequence)
6. Grant Input Monitoring permission
7. Verify app becomes functional

### Permission Denied

1. Launch app
2. Deny Accessibility permission
3. Verify app shows disabled state
4. Verify menubar icon indicates problem
5. Verify clicking icon shows "Grant Permissions" option

### Permission Revoked

1. Launch app with permissions
2. Revoke Accessibility in System Preferences
3. Verify app detects revocation
4. Verify graceful degradation

### macOS Sequoia (15.0+)

- [ ] Test monthly permission re-authorization flow
- [ ] Test post-reboot permission state

---

## Performance Benchmarks

| Metric | Target | Measurement |
|--------|--------|-------------|
| Time to analyze (p50) | <1.5s | API latency + processing |
| Time to analyze (p95) | <3.0s | Including slow responses |
| Memory usage (idle) | <50MB | Activity Monitor |
| CPU usage (idle) | <1% | Activity Monitor |
| CPU usage (typing) | <5% | Activity Monitor |
| Focus detection latency | <100ms | Time from focus to icon change |

---

## Regression Test Suite

When making changes, run:

```bash
# All unit tests
swift test

# Specific test file
swift test --filter AnalyzerTests

# With verbose output
swift test --verbose
```

Before release:
1. Run full test suite
2. Complete manual testing checklist
3. Test on both Intel and Apple Silicon
4. Test on macOS 13, 14, and 15
