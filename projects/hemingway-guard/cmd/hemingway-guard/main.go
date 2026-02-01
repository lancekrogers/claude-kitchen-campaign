// HemingwayGuard - Text interception for messaging apps
// Applies the Hemingway method to validate messages before sending
package main

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa

#include <Cocoa/Cocoa.h>

void runApp() {
    @autoreleasepool {
        [NSApplication sharedApplication];
        [NSApp setActivationPolicy:NSApplicationActivationPolicyAccessory];
        [NSApp run];
    }
}

void stopApp() {
    dispatch_async(dispatch_get_main_queue(), ^{
        [NSApp terminate:nil];
    });
}
*/
import "C"

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/lancekrogers/hemingway-guard/internal/accessibility"
	"github.com/lancekrogers/hemingway-guard/internal/analyzer"
	"github.com/lancekrogers/hemingway-guard/internal/keyboard"
	"github.com/lancekrogers/hemingway-guard/internal/ui"
	"github.com/lancekrogers/hemingway-guard/pkg/apps"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("HemingwayGuard starting...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("Shutting down...")
		cancel()
		C.stopApp()
	}()

	// Initialize components
	hemingway := analyzer.NewAnalyzer()
	menuBar := ui.NewMenuBar()
	focusMonitor := accessibility.NewFocusMonitor(apps.TargetBundleIDs())
	interceptor := keyboard.NewInterceptor()

	// Set up menu bar
	ui.SetMenuCallback(func(action ui.MenuAction) {
		switch action {
		case ui.MenuActionToggleEnabled:
			enabled := !menuBar.IsEnabled()
			menuBar.SetEnabled(enabled)
			if enabled {
				menuBar.SetTitle("✍️")
				interceptor.SetMonitoring(focusMonitor.IsMonitoring())
			} else {
				menuBar.SetTitle("✍️ (off)")
				interceptor.SetMonitoring(false)
			}
			log.Printf("HemingwayGuard %s", map[bool]string{true: "enabled", false: "disabled"}[enabled])

		case ui.MenuActionSettings:
			log.Println("Settings clicked (not implemented)")

		case ui.MenuActionQuit:
			cancel()
			C.stopApp()
		}
	})

	// Set up focus monitoring
	focusMonitor.OnTextFieldFocus(func(element *accessibility.Element, bundleID string) {
		if menuBar.IsEnabled() {
			interceptor.SetMonitoring(true)
			log.Printf("Monitoring text field in %s", bundleID)
		}
	})

	focusMonitor.OnTextFieldBlur(func() {
		interceptor.SetMonitoring(false)
		log.Println("Stopped monitoring text field")
	})

	// Set up keyboard interception
	interceptor.SetHandler(func(ctx context.Context) bool {
		text := focusMonitor.CurrentText()
		if text == "" {
			return true // Allow empty messages
		}

		log.Printf("Analyzing message: %q", truncate(text, 50))

		// Get current app context
		elem := focusMonitor.CurrentElement()
		appCtx := analyzer.AppContext{}
		if elem != nil {
			target := apps.FindTarget(elem.BundleID())
			if target != nil {
				appCtx.AppName = target.Name
			}
		}

		// Analyze the message
		analysis, err := hemingway.Analyze(ctx, text, appCtx)
		if err != nil {
			log.Printf("Analysis error: %v", err)
			return true // On error, allow the message
		}

		log.Printf("Analysis: approved=%v, words=%d, issues=%v",
			analysis.Approved, analysis.WordCount, analysis.Issues)

		if analysis.Approved {
			return true // Message is good, allow sending
		}

		// TODO: Show approval popover and wait for user action
		// For now, we log and allow
		log.Printf("Message has issues but allowing (popover not implemented)")
		return true
	})

	// Start components
	if err := focusMonitor.Start(ctx); err != nil {
		log.Fatalf("Failed to start focus monitor: %v", err)
	}
	defer focusMonitor.Stop()

	if err := interceptor.Start(ctx); err != nil {
		log.Fatalf("Failed to start interceptor: %v", err)
	}
	defer interceptor.Stop()

	// Show menu bar
	menuBar.Show("✍️")

	log.Println("HemingwayGuard ready")
	log.Println("Monitoring: Messages, Slack, Discord")

	// Run the app (blocks until quit)
	C.runApp()
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
