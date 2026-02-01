// Package analyzer provides LLM-based text analysis using the Hemingway method.
package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Analysis represents the result of Hemingway analysis on a message.
type Analysis struct {
	Approved        bool     `json:"approved"`
	WordCount       int      `json:"word_count"`
	ReadTimeSeconds int      `json:"read_time_seconds"`
	GradeLevel      float64  `json:"grade_level"`
	Issues          []string `json:"issues"`
	Suggestion      string   `json:"suggestion"`
}

// AppContext provides context about where the message is being sent.
type AppContext struct {
	AppName   string // e.g., "Slack", "Discord", "iMessage"
	ChannelType string // e.g., "DM", "channel", "group"
}

// Analyzer performs Hemingway-style text analysis.
type Analyzer struct {
	// Client will be claude-code-go client when integrated
	// For now, we use a placeholder interface
}

// NewAnalyzer creates a new Hemingway analyzer.
func NewAnalyzer() *Analyzer {
	return &Analyzer{}
}

// Analyze performs Hemingway analysis on the given text.
func (a *Analyzer) Analyze(ctx context.Context, text string, appCtx AppContext) (*Analysis, error) {
	if strings.TrimSpace(text) == "" {
		return &Analysis{
			Approved:  true,
			WordCount: 0,
			Issues:    []string{},
		}, nil
	}

	// TODO: Integrate with claude-code-go
	// prompt := buildPrompt(text, appCtx)
	_ = buildPrompt // silence unused warning until LLM integration

	// For now, return a mock analysis
	return a.mockAnalysis(text)
}

func buildPrompt(text string, appCtx AppContext) string {
	contextDesc := fmt.Sprintf("%s %s", appCtx.AppName, appCtx.ChannelType)
	if contextDesc == " " {
		contextDesc = "messaging app"
	}

	return fmt.Sprintf(`Analyze this message for %s:

"%s"

Apply the Hemingway method: check for conciseness, clarity, and readability.

Return JSON only:
{
  "approved": bool,     // true if message is good to send as-is
  "word_count": int,
  "read_time_seconds": int,
  "grade_level": float, // Flesch-Kincaid grade level
  "issues": [],         // list of issues found (e.g., "too long", "passive voice", "unclear")
  "suggestion": ""      // shorter/clearer version if not approved, empty string if approved
}

Guidelines:
- Approve messages that are clear, concise, and appropriate for the context
- Flag overly long messages (>100 words for DMs, >200 for channels)
- Flag passive voice, jargon, or unclear phrasing
- Flag messages that could be misinterpreted
- Suggest a more concise version if there are issues`, contextDesc, text)
}

// mockAnalysis provides a simple local analysis without LLM.
// This will be replaced with claude-code-go integration.
func (a *Analyzer) mockAnalysis(text string) (*Analysis, error) {
	words := strings.Fields(text)
	wordCount := len(words)

	// Simple heuristics for mock analysis
	issues := []string{}
	approved := true

	// Check length
	if wordCount > 100 {
		issues = append(issues, "message is quite long")
		approved = false
	}

	// Check for passive voice indicators (very basic)
	passiveIndicators := []string{"was", "were", "been", "being", "is being", "are being"}
	lowerText := strings.ToLower(text)
	for _, indicator := range passiveIndicators {
		if strings.Contains(lowerText, " "+indicator+" ") {
			issues = append(issues, "possible passive voice detected")
			break
		}
	}

	// Estimate reading time (average 200 wpm)
	readTime := (wordCount * 60) / 200
	if readTime < 1 {
		readTime = 1
	}

	// Very rough grade level estimate
	gradeLevel := float64(wordCount) / 10.0
	if gradeLevel > 12 {
		gradeLevel = 12
	}

	suggestion := ""
	if !approved && wordCount > 100 {
		// Truncate as a simple "suggestion"
		suggestion = strings.Join(words[:50], " ") + "..."
	}

	return &Analysis{
		Approved:        approved,
		WordCount:       wordCount,
		ReadTimeSeconds: readTime,
		GradeLevel:      gradeLevel,
		Issues:          issues,
		Suggestion:      suggestion,
	}, nil
}

// ParseAnalysis parses JSON response from the LLM into an Analysis struct.
func ParseAnalysis(jsonStr string) (*Analysis, error) {
	var analysis Analysis
	err := json.Unmarshal([]byte(jsonStr), &analysis)
	if err != nil {
		return nil, fmt.Errorf("failed to parse analysis: %w", err)
	}
	return &analysis, nil
}
