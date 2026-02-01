# HemingwayGuard

A macOS menubar application that intercepts messages before sending in iMessage, Discord, Slack, and other messaging apps to validate them using the "Hemingway method" - checking for conciseness, readability, and context-appropriateness.

## Features

- System-wide text field monitoring using macOS Accessibility APIs
- Keystroke interception to catch messages before they're sent
- LLM-powered analysis using claude-code-go
- Approval popover with suggestions and editing capabilities
- Menubar app with enable/disable toggle

## Supported Apps

- iMessage (com.apple.MobileSMS)
- Slack (com.tinyspeck.slackmacgap)
- Discord (com.hnc.Discord)

## Requirements

- macOS 12.0+
- Go 1.23+
- Accessibility permissions (System Preferences → Privacy & Security → Accessibility)
- Input Monitoring permissions (System Preferences → Privacy & Security → Input Monitoring)

## Installation

```bash
# Clone the repository
git clone https://github.com/lancekrogers/hemingway-guard.git
cd hemingway-guard

# Install dependencies
just deps

# Build
just build

# Run
just run
```

## Usage

1. Launch HemingwayGuard - it will appear as a ✍️ icon in your menubar
2. Grant Accessibility and Input Monitoring permissions when prompted
3. Open Messages, Slack, or Discord and start typing
4. When you press Enter to send a message, HemingwayGuard will:
   - Analyze the message for clarity and conciseness
   - Show an approval popover if issues are found
   - Let you edit, use the suggestion, or send anyway

## Architecture

See [workflow/design/active/hemingway-guard-design.md](../../workflow/design/active/hemingway-guard-design.md) for detailed architecture documentation.

## Development

```bash
# Run tests
just test

# Lint
just lint

# Format code
just fmt

# Build .app bundle
just bundle
```

## License

MIT
