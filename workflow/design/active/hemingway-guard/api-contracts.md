# HemingwayGuard API Contracts

## Claude API Integration

### Model Selection

| Model | Use Case | Notes |
|-------|----------|-------|
| `claude-3-5-sonnet` | Default | Best balance of speed and quality |
| `claude-3-5-haiku` | Fast mode | Lower latency, simpler analysis |

### Request Configuration

```swift
let request = MessageRequest(
    model: .claude35Sonnet,
    maxTokens: 512,
    messages: [
        .user(content: prompt)
    ],
    system: systemPrompt
)
```

---

## Analysis Prompt

### System Prompt

```
You are a writing assistant that analyzes messages for clarity and conciseness using the Hemingway method. You evaluate messages being sent in messaging apps (Slack, Discord, iMessage) and provide feedback.

Always respond with valid JSON matching the specified schema. Do not include any other text.
```

### User Prompt Template

```
Analyze this message being sent in {APP_CONTEXT}:

---
{MESSAGE_TEXT}
---

Evaluate for:
1. Conciseness - Is it too long for the context?
2. Clarity - Is the meaning clear?
3. Readability - Is it easy to read quickly?
4. Tone - Is it appropriate for {APP_CONTEXT}?

Respond with JSON:
{
  "approved": boolean,       // true if message is good to send as-is
  "word_count": number,
  "read_time_seconds": number,
  "grade_level": number,     // Flesch-Kincaid grade level estimate
  "issues": string[],        // List of specific issues found
  "suggestion": string       // Improved version if not approved, empty if approved
}

Guidelines:
- Approve messages that are clear, concise, and context-appropriate
- For DMs: flag messages >100 words as potentially too long
- For channels: flag messages >200 words as potentially too long
- Flag passive voice, unnecessary jargon, or unclear references
- Flag potentially confusing or easily misinterpreted phrasing
- If suggesting improvements, make them substantive, not just minor edits
```

### Context Values

| App | Context String |
|-----|----------------|
| Messages | `iMessage (personal messaging)` |
| Slack | `Slack (workplace chat)` |
| Discord | `Discord (community chat)` |

---

## Response Schema

### AnalysisResult

```swift
struct AnalysisResult: Codable, Equatable {
    /// Whether the message is approved to send as-is
    let approved: Bool

    /// Word count of the message
    let wordCount: Int

    /// Estimated reading time in seconds
    let readTimeSeconds: Int

    /// Flesch-Kincaid grade level (1-12+)
    let gradeLevel: Double

    /// List of specific issues found (empty if approved)
    let issues: [String]

    /// Suggested improved version (empty string if approved)
    let suggestion: String

    enum CodingKeys: String, CodingKey {
        case approved
        case wordCount = "word_count"
        case readTimeSeconds = "read_time_seconds"
        case gradeLevel = "grade_level"
        case issues
        case suggestion
    }
}
```

### Example Responses

#### Approved Message

```json
{
  "approved": true,
  "word_count": 12,
  "read_time_seconds": 3,
  "grade_level": 5.2,
  "issues": [],
  "suggestion": ""
}
```

#### Message with Issues

```json
{
  "approved": false,
  "word_count": 87,
  "read_time_seconds": 26,
  "grade_level": 9.4,
  "issues": [
    "Message is longer than typical for Slack DMs",
    "Passive voice in sentence 2: 'was decided by the team'",
    "Vague timeline: 'soon' could mean different things"
  ],
  "suggestion": "Hey! The team chose to delay the launch until March 15th. Can you update the docs by Friday? Let me know if you need help."
}
```

---

## Error Handling

### API Errors

```swift
enum AnalyzerError: Error {
    case emptyResponse
    case invalidJSON(String)
    case timeout
    case networkError(Error)
    case rateLimited
    case invalidAPIKey
}
```

### Error Recovery

| Error | Recovery |
|-------|----------|
| Timeout (>3s) | Allow message, log warning |
| Network error | Allow message, show brief notification |
| Invalid JSON | Allow message, log error for debugging |
| Rate limited | Allow message, show notification |
| Invalid API key | Disable analysis, prompt for key |

---

## Rate Limiting

### Expected Usage

| Scenario | Requests/min |
|----------|--------------|
| Light usage | 2-5 |
| Active conversation | 10-20 |
| Heavy usage | 30-50 |

### Mitigation Strategies

1. **Debouncing**: Don't analyze if user is still typing rapidly
2. **Caching**: Cache analysis for identical messages (within 5 min)
3. **Skip short messages**: Don't analyze messages under 10 words
4. **Backoff**: Exponential backoff on rate limit errors

---

## Local Analysis Fallback

When API is unavailable, use local heuristics:

```swift
struct LocalAnalyzer {
    func analyze(text: String, context: AppContext) -> AnalysisResult {
        let words = text.split(separator: " ")
        let wordCount = words.count

        var issues: [String] = []
        var approved = true

        // Check length
        let maxWords = context == .dm ? 100 : 200
        if wordCount > maxWords {
            issues.append("Message is quite long (\(wordCount) words)")
            approved = false
        }

        // Check for passive voice indicators
        let passiveIndicators = ["was", "were", "been", "being"]
        let lowerText = text.lowercased()
        for indicator in passiveIndicators {
            if lowerText.contains(" \(indicator) ") {
                issues.append("Possible passive voice detected")
                break
            }
        }

        // Estimate reading time (200 WPM)
        let readTime = max(1, wordCount * 60 / 200)

        // Rough grade level
        let gradeLevel = min(12.0, Double(wordCount) / 10.0)

        return AnalysisResult(
            approved: approved,
            wordCount: wordCount,
            readTimeSeconds: readTime,
            gradeLevel: gradeLevel,
            issues: issues,
            suggestion: ""  // No suggestions without LLM
        )
    }
}
```

---

## Prompt Customization (Future)

Allow users to customize analysis criteria:

```swift
struct AnalysisSettings: Codable {
    var maxWordsForDM: Int = 100
    var maxWordsForChannel: Int = 200
    var checkPassiveVoice: Bool = true
    var checkTone: Bool = true
    var customInstructions: String = ""
}
```

Custom instructions would be appended to the prompt:

```
Additional instructions from user:
{CUSTOM_INSTRUCTIONS}
```

---

## Metrics & Logging

### Events to Log

| Event | Data |
|-------|------|
| Analysis requested | app, word_count, timestamp |
| Analysis completed | approved, latency_ms, issues_count |
| User action | action_type (send/edit/suggestion) |
| Error | error_type, message |

### Privacy Considerations

- **Never log message content**
- Only log aggregate statistics
- Store API key securely in Keychain
- Clear analysis cache on quit
