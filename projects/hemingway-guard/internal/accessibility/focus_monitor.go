package accessibility

import (
	"context"
	"log"
	"sync"
	"time"
)

// FocusMonitor monitors system-wide focus changes and identifies text fields in target apps.
type FocusMonitor struct {
	mu              sync.RWMutex
	targetBundleIDs map[string]bool
	currentElement  *Element
	onTextFieldFocus func(element *Element, bundleID string)
	onTextFieldBlur  func()

	pollInterval time.Duration
	running      bool
	stopCh       chan struct{}
}

// NewFocusMonitor creates a new focus monitor.
func NewFocusMonitor(targetBundleIDs map[string]bool) *FocusMonitor {
	return &FocusMonitor{
		targetBundleIDs: targetBundleIDs,
		pollInterval:    100 * time.Millisecond,
		stopCh:          make(chan struct{}),
	}
}

// OnTextFieldFocus sets the callback for when a text field in a target app gains focus.
func (m *FocusMonitor) OnTextFieldFocus(cb func(element *Element, bundleID string)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onTextFieldFocus = cb
}

// OnTextFieldBlur sets the callback for when focus leaves a monitored text field.
func (m *FocusMonitor) OnTextFieldBlur(cb func()) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onTextFieldBlur = cb
}

// Start begins monitoring focus changes.
// Uses polling approach as a fallback since observer-based approach requires
// complex run loop integration.
func (m *FocusMonitor) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return nil
	}
	m.running = true
	m.mu.Unlock()

	systemElement := SystemWideElement()
	if systemElement == nil {
		return ErrAccessibilityNotEnabled
	}

	go m.pollLoop(ctx, systemElement)
	return nil
}

func (m *FocusMonitor) pollLoop(ctx context.Context, systemElement *Element) {
	ticker := time.NewTicker(m.pollInterval)
	defer ticker.Stop()
	defer systemElement.Release()

	var lastWasTextField bool

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			focused, err := systemElement.FocusedElement()
			if err != nil {
				continue
			}

			bundleID := focused.BundleID()
			isTarget := m.targetBundleIDs[bundleID]
			isTextField := focused.IsTextField()

			m.mu.RLock()
			onFocus := m.onTextFieldFocus
			onBlur := m.onTextFieldBlur
			m.mu.RUnlock()

			// Transitioned into a monitored text field
			if isTarget && isTextField && !lastWasTextField {
				log.Printf("Focus: text field in %s", bundleID)
				m.mu.Lock()
				m.currentElement = focused
				m.mu.Unlock()

				if onFocus != nil {
					onFocus(focused, bundleID)
				}
			}

			// Transitioned out of a monitored text field
			if lastWasTextField && (!isTarget || !isTextField) {
				log.Printf("Blur: left monitored text field")
				m.mu.Lock()
				if m.currentElement != nil {
					m.currentElement.Release()
					m.currentElement = nil
				}
				m.mu.Unlock()

				if onBlur != nil {
					onBlur()
				}
			}

			lastWasTextField = isTarget && isTextField

			// Release if not storing
			if m.currentElement != focused {
				focused.Release()
			}
		}
	}
}

// CurrentElement returns the currently focused text field element, if any.
func (m *FocusMonitor) CurrentElement() *Element {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentElement
}

// CurrentText returns the text in the currently focused field.
func (m *FocusMonitor) CurrentText() string {
	m.mu.RLock()
	elem := m.currentElement
	m.mu.RUnlock()

	if elem == nil {
		return ""
	}
	return elem.Value()
}

// SetCurrentText sets the text in the currently focused field.
func (m *FocusMonitor) SetCurrentText(text string) error {
	m.mu.RLock()
	elem := m.currentElement
	m.mu.RUnlock()

	if elem == nil {
		return ErrElementNotFound
	}
	return elem.SetValue(text)
}

// Stop stops the focus monitor.
func (m *FocusMonitor) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	close(m.stopCh)
	m.running = false

	if m.currentElement != nil {
		m.currentElement.Release()
		m.currentElement = nil
	}
}

// IsMonitoring returns whether focus is currently on a monitored text field.
func (m *FocusMonitor) IsMonitoring() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentElement != nil
}
