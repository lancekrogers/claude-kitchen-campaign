package keyboard

import (
	"context"
	"errors"
	"log"
	"sync"
)

// ErrInputMonitoringNotEnabled indicates Input Monitoring permissions are not granted.
var ErrInputMonitoringNotEnabled = errors.New("input monitoring permissions not enabled")

// InterceptHandler is called when Enter is pressed in a monitored context.
// Return true to allow the keystroke, false to block it.
type InterceptHandler func(ctx context.Context) bool

// Interceptor manages keystroke interception for the Hemingway workflow.
type Interceptor struct {
	mu         sync.RWMutex
	eventTap   *EventTap
	handler    InterceptHandler
	monitoring bool
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewInterceptor creates a new keystroke interceptor.
func NewInterceptor() *Interceptor {
	return &Interceptor{}
}

// SetHandler sets the handler called when Enter is intercepted.
func (i *Interceptor) SetHandler(h InterceptHandler) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.handler = h
}

// Start initializes the event tap.
func (i *Interceptor) Start(ctx context.Context) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.eventTap != nil {
		return nil // Already started
	}

	tap, err := NewEventTap()
	if err != nil {
		return err
	}

	i.eventTap = tap
	i.ctx, i.cancel = context.WithCancel(ctx)

	SetEventCallback(i.handleKeyEvent)
	tap.Start()

	log.Println("Keyboard interceptor started")
	return nil
}

func (i *Interceptor) handleKeyEvent(keyCode int, modifiers Modifiers) bool {
	// Only intercept plain Enter (no modifiers except Shift for newline)
	if modifiers.Command || modifiers.Control || modifiers.Option {
		return true // Allow modified Enter keys
	}

	// Shift+Enter typically means newline, not send
	if modifiers.Shift {
		return true
	}

	i.mu.RLock()
	monitoring := i.monitoring
	handler := i.handler
	ctx := i.ctx
	i.mu.RUnlock()

	if !monitoring {
		return true // Not monitoring, allow the keystroke
	}

	log.Println("Intercepted Enter key in monitored context")

	if handler != nil {
		// Handler decides whether to allow the keystroke
		return handler(ctx)
	}

	return true
}

// SetMonitoring enables or disables active interception.
// When monitoring is true, Enter keystrokes will be processed by the handler.
func (i *Interceptor) SetMonitoring(monitoring bool) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.monitoring = monitoring
	log.Printf("Monitoring: %v", monitoring)
}

// IsMonitoring returns whether the interceptor is actively monitoring.
func (i *Interceptor) IsMonitoring() bool {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return i.monitoring
}

// ReleaseEnter posts an Enter key event to send the message.
func (i *Interceptor) ReleaseEnter() {
	log.Println("Releasing Enter key")
	PostEnterKey()
}

// Stop shuts down the interceptor.
func (i *Interceptor) Stop() {
	i.mu.Lock()
	defer i.mu.Unlock()

	if i.cancel != nil {
		i.cancel()
	}

	if i.eventTap != nil {
		i.eventTap.Stop()
		i.eventTap = nil
	}

	SetEventCallback(nil)
	log.Println("Keyboard interceptor stopped")
}
