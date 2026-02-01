// HemingwayGuard Approval Popover
// This SwiftUI component displays analysis results and approval options.
// It communicates with the Go backend via Unix socket IPC.

import SwiftUI
import Cocoa

// MARK: - Data Models

struct AnalysisResult: Codable {
    let approved: Bool
    let wordCount: Int
    let readTimeSeconds: Int
    let gradeLevel: Double
    let issues: [String]
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

enum UserAction: String, Codable {
    case sendAnyway = "send_anyway"
    case useSuggestion = "use_suggestion"
    case edit = "edit"
    case cancel = "cancel"
}

struct ActionResponse: Codable {
    let action: UserAction
    let editedText: String?

    enum CodingKeys: String, CodingKey {
        case action
        case editedText = "edited_text"
    }
}

// MARK: - Views

struct ApprovalPopoverView: View {
    let analysis: AnalysisResult
    let originalText: String
    let onAction: (UserAction, String?) -> Void

    @State private var editedText: String = ""
    @State private var isEditing: Bool = false

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            // Header
            HStack {
                Image(systemName: analysis.approved ? "checkmark.circle.fill" : "exclamationmark.triangle.fill")
                    .foregroundColor(analysis.approved ? .green : .orange)
                Text(analysis.approved ? "Ready to Send" : "Review Suggested")
                    .font(.headline)
                Spacer()
            }

            Divider()

            // Stats
            HStack(spacing: 16) {
                StatView(label: "Words", value: "\(analysis.wordCount)")
                StatView(label: "Read time", value: "\(analysis.readTimeSeconds)s")
                StatView(label: "Grade", value: String(format: "%.1f", analysis.gradeLevel))
            }

            // Issues
            if !analysis.issues.isEmpty {
                VStack(alignment: .leading, spacing: 4) {
                    Text("Issues:")
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                    ForEach(analysis.issues, id: \.self) { issue in
                        HStack {
                            Image(systemName: "circle.fill")
                                .font(.system(size: 4))
                                .foregroundColor(.orange)
                            Text(issue)
                                .font(.caption)
                        }
                    }
                }
            }

            // Suggestion
            if !analysis.suggestion.isEmpty {
                VStack(alignment: .leading, spacing: 4) {
                    Text("Suggested:")
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                    Text(analysis.suggestion)
                        .font(.caption)
                        .padding(8)
                        .background(Color.blue.opacity(0.1))
                        .cornerRadius(4)
                }
            }

            // Edit mode
            if isEditing {
                TextEditor(text: $editedText)
                    .frame(height: 80)
                    .border(Color.gray.opacity(0.3), width: 1)
            }

            Divider()

            // Action buttons
            HStack {
                if isEditing {
                    Button("Cancel") {
                        isEditing = false
                        editedText = originalText
                    }
                    Spacer()
                    Button("Send Edited") {
                        onAction(.edit, editedText)
                    }
                    .buttonStyle(.borderedProminent)
                } else {
                    Button("Edit") {
                        editedText = originalText
                        isEditing = true
                    }

                    if !analysis.suggestion.isEmpty {
                        Button("Use Suggestion") {
                            onAction(.useSuggestion, analysis.suggestion)
                        }
                    }

                    Spacer()

                    Button("Send Anyway") {
                        onAction(.sendAnyway, nil)
                    }
                    .buttonStyle(.borderedProminent)
                }
            }
        }
        .padding()
        .frame(width: 320)
    }
}

struct StatView: View {
    let label: String
    let value: String

    var body: some View {
        VStack {
            Text(value)
                .font(.title2)
                .fontWeight(.semibold)
            Text(label)
                .font(.caption)
                .foregroundColor(.secondary)
        }
    }
}

// MARK: - Popover Controller

class PopoverController: NSObject {
    private var popover: NSPopover?
    private var eventMonitor: Any?

    func show(analysis: AnalysisResult, originalText: String, near point: NSPoint, onAction: @escaping (UserAction, String?) -> Void) {
        let contentView = ApprovalPopoverView(
            analysis: analysis,
            originalText: originalText
        ) { [weak self] action, text in
            onAction(action, text)
            self?.close()
        }

        let hostingController = NSHostingController(rootView: contentView)

        popover = NSPopover()
        popover?.contentViewController = hostingController
        popover?.behavior = .transient

        // Create a temporary window to show the popover near the cursor
        let rect = NSRect(x: point.x, y: point.y, width: 1, height: 1)
        let window = NSWindow(contentRect: rect, styleMask: .borderless, backing: .buffered, defer: false)
        window.backgroundColor = .clear
        window.level = .floating
        window.makeKeyAndOrderFront(nil)

        popover?.show(relativeTo: window.contentView!.bounds, of: window.contentView!, preferredEdge: .maxY)

        // Close on click outside
        eventMonitor = NSEvent.addGlobalMonitorForEvents(matching: [.leftMouseDown, .rightMouseDown]) { [weak self] _ in
            self?.close()
        }
    }

    func close() {
        popover?.close()
        popover = nil

        if let monitor = eventMonitor {
            NSEvent.removeMonitor(monitor)
            eventMonitor = nil
        }
    }
}

// MARK: - IPC Helper (for Go integration)

class IPCClient {
    private let socketPath: String

    init(socketPath: String = "/tmp/hemingway-guard.sock") {
        self.socketPath = socketPath
    }

    func sendAction(_ action: UserAction, editedText: String? = nil) {
        let response = ActionResponse(action: action, editedText: editedText)

        guard let data = try? JSONEncoder().encode(response) else {
            print("Failed to encode action response")
            return
        }

        // TODO: Send via Unix socket to Go backend
        print("Would send to Go: \(String(data: data, encoding: .utf8) ?? "")")
    }
}
