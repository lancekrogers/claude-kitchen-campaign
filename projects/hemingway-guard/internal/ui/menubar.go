// Package ui provides the user interface components for HemingwayGuard.
package ui

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa

// Function declarations - implemented in menubar_darwin.m
void createStatusItem(const char* title);
void setStatusItemTitle(const char* title);
void setEnabledState(int enabled);
void removeStatusItem(void);
*/
import "C"

import (
	"log"
	"sync"
)

// MenuAction represents menu item actions.
type MenuAction int

const (
	MenuActionToggleEnabled MenuAction = 1
	MenuActionSettings      MenuAction = 2
	MenuActionQuit          MenuAction = 3
)

// MenuCallback is called when a menu item is clicked.
type MenuCallback func(action MenuAction)

var (
	menuCallbackMu sync.RWMutex
	menuCallback   MenuCallback
)

// SetMenuCallback sets the callback for menu item clicks.
func SetMenuCallback(cb MenuCallback) {
	menuCallbackMu.Lock()
	defer menuCallbackMu.Unlock()
	menuCallback = cb
}

//export goMenuItemClicked
func goMenuItemClicked(tag C.int) {
	menuCallbackMu.RLock()
	cb := menuCallback
	menuCallbackMu.RUnlock()

	if cb != nil {
		cb(MenuAction(tag))
	}
}

// MenuBar manages the macOS menu bar status item.
type MenuBar struct {
	enabled bool
	mu      sync.Mutex
}

// NewMenuBar creates a new menu bar manager.
func NewMenuBar() *MenuBar {
	return &MenuBar{enabled: true}
}

// Show displays the status item in the menu bar.
func (m *MenuBar) Show(title string) {
	cTitle := C.CString(title)
	C.createStatusItem(cTitle)
	log.Printf("Menu bar created with title: %s", title)
}

// SetTitle updates the status item title.
func (m *MenuBar) SetTitle(title string) {
	cTitle := C.CString(title)
	C.setStatusItemTitle(cTitle)
}

// SetEnabled updates the enabled checkbox state.
func (m *MenuBar) SetEnabled(enabled bool) {
	m.mu.Lock()
	m.enabled = enabled
	m.mu.Unlock()

	if enabled {
		C.setEnabledState(1)
	} else {
		C.setEnabledState(0)
	}
}

// IsEnabled returns the current enabled state.
func (m *MenuBar) IsEnabled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.enabled
}

// Hide removes the status item from the menu bar.
func (m *MenuBar) Hide() {
	C.removeStatusItem()
}
