// Package keyboard provides CGEventTap wrappers for keystroke interception.
package keyboard

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework ApplicationServices -framework Foundation

#include <ApplicationServices/ApplicationServices.h>

// Key codes for Enter/Return
#define KEYCODE_RETURN 36
#define KEYCODE_ENTER 76

// Callback function type for Go
extern CGEventRef goEventCallback(CGEventTapProxy proxy, CGEventType type, CGEventRef event);

// C callback that bridges to Go
static CGEventRef eventCallback(
    CGEventTapProxy proxy,
    CGEventType type,
    CGEventRef event,
    void *refcon
) {
    return goEventCallback(proxy, type, event);
}

// Create an event tap for key down events
static inline CFMachPortRef createEventTap() {
    CGEventMask eventMask = CGEventMaskBit(kCGEventKeyDown);

    CFMachPortRef tap = CGEventTapCreate(
        kCGSessionEventTap,
        kCGHeadInsertEventTap,
        kCGEventTapOptionDefault,
        eventMask,
        eventCallback,
        NULL
    );

    return tap;
}

// Get the key code from a keyboard event
static inline int64_t getKeyCode(CGEventRef event) {
    return CGEventGetIntegerValueField(event, kCGKeyboardEventKeycode);
}

// Check if shift is held
static inline int isShiftHeld(CGEventRef event) {
    CGEventFlags flags = CGEventGetFlags(event);
    return (flags & kCGEventFlagMaskShift) ? 1 : 0;
}

// Check if command is held
static inline int isCommandHeld(CGEventRef event) {
    CGEventFlags flags = CGEventGetFlags(event);
    return (flags & kCGEventFlagMaskCommand) ? 1 : 0;
}

// Check if control is held
static inline int isControlHeld(CGEventRef event) {
    CGEventFlags flags = CGEventGetFlags(event);
    return (flags & kCGEventFlagMaskControl) ? 1 : 0;
}

// Check if option/alt is held
static inline int isOptionHeld(CGEventRef event) {
    CGEventFlags flags = CGEventGetFlags(event);
    return (flags & kCGEventFlagMaskAlternate) ? 1 : 0;
}

// Enable the event tap
static inline void enableEventTap(CFMachPortRef tap) {
    CGEventTapEnable(tap, true);
}

// Disable the event tap
static inline void disableEventTap(CFMachPortRef tap) {
    CGEventTapEnable(tap, false);
}

// Add event tap to run loop
static inline void addToRunLoop(CFMachPortRef tap) {
    CFRunLoopSourceRef source = CFMachPortCreateRunLoopSource(kCFAllocatorDefault, tap, 0);
    CFRunLoopAddSource(CFRunLoopGetCurrent(), source, kCFRunLoopCommonModes);
    CFRelease(source);
}

// Post a keyboard event
static inline void postKeyEvent(int64_t keyCode, int keyDown) {
    CGEventRef event = CGEventCreateKeyboardEvent(NULL, (CGKeyCode)keyCode, keyDown ? true : false);
    CGEventPost(kCGHIDEventTap, event);
    CFRelease(event);
}
*/
import "C"

import (
	"sync"
)

const (
	// KeyCodeReturn is the key code for the Return key
	KeyCodeReturn = 36
	// KeyCodeEnter is the key code for the numpad Enter key
	KeyCodeEnter = 76
)

// EventCallback is called when a keyboard event is intercepted.
// Return true to allow the event, false to swallow it.
type EventCallback func(keyCode int, modifiers Modifiers) bool

// Modifiers represents keyboard modifier keys.
type Modifiers struct {
	Shift   bool
	Command bool
	Control bool
	Option  bool
}

var (
	eventCallbackMu sync.RWMutex
	eventCallback   EventCallback
)

// SetEventCallback sets the callback function for keyboard events.
func SetEventCallback(cb EventCallback) {
	eventCallbackMu.Lock()
	defer eventCallbackMu.Unlock()
	eventCallback = cb
}

//export goEventCallback
func goEventCallback(proxy C.CGEventTapProxy, eventType C.CGEventType, event C.CGEventRef) C.CGEventRef {
	keyCode := int(C.getKeyCode(event))

	// Only process Enter/Return keys
	if keyCode != KeyCodeReturn && keyCode != KeyCodeEnter {
		return event
	}

	modifiers := Modifiers{
		Shift:   C.isShiftHeld(event) == 1,
		Command: C.isCommandHeld(event) == 1,
		Control: C.isControlHeld(event) == 1,
		Option:  C.isOptionHeld(event) == 1,
	}

	eventCallbackMu.RLock()
	cb := eventCallback
	eventCallbackMu.RUnlock()

	if cb != nil {
		allow := cb(keyCode, modifiers)
		if !allow {
			// Swallow the event by returning NULL
			return C.CGEventRef(uintptr(0))
		}
	}

	return event
}

// EventTap wraps a CGEventTap for keyboard interception.
type EventTap struct {
	tap     C.CFMachPortRef
	enabled bool
	mu      sync.Mutex
}

// NewEventTap creates a new keyboard event tap.
func NewEventTap() (*EventTap, error) {
	tap := C.createEventTap()
	if uintptr(tap) == 0 {
		return nil, ErrInputMonitoringNotEnabled
	}
	return &EventTap{tap: tap}, nil
}

// Start enables the event tap and adds it to the run loop.
func (t *EventTap) Start() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.enabled {
		return
	}

	C.addToRunLoop(t.tap)
	C.enableEventTap(t.tap)
	t.enabled = true
}

// Stop disables the event tap.
func (t *EventTap) Stop() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.enabled {
		return
	}

	C.disableEventTap(t.tap)
	t.enabled = false
}

// IsEnabled returns whether the event tap is currently enabled.
func (t *EventTap) IsEnabled() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.enabled
}

// PostEnterKey programmatically posts an Enter key event.
func PostEnterKey() {
	C.postKeyEvent(C.int64_t(KeyCodeReturn), 1) // key down
	C.postKeyEvent(C.int64_t(KeyCodeReturn), 0) // key up
}
