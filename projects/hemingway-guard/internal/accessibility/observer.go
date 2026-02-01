package accessibility

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework ApplicationServices -framework Foundation

#include <ApplicationServices/ApplicationServices.h>

// Callback function type for Go
extern void goFocusCallback(AXUIElementRef element);

// C callback that bridges to Go
static void focusChangedCallback(
    AXObserverRef observer,
    AXUIElementRef element,
    CFStringRef notification,
    void *refcon
) {
    goFocusCallback(element);
}

// Create an observer for the given process
static inline AXObserverRef createObserver(pid_t pid) {
    AXObserverRef observer = NULL;
    AXError error = AXObserverCreate(pid, focusChangedCallback, &observer);
    if (error != kAXErrorSuccess) {
        return NULL;
    }
    return observer;
}

// Add notification to observer
static inline int addNotification(AXObserverRef observer, AXUIElementRef element, CFStringRef notification) {
    AXError error = AXObserverAddNotification(observer, element, notification, NULL);
    return error == kAXErrorSuccess ? 0 : -1;
}

// Get the run loop source for the observer
static inline CFRunLoopSourceRef getRunLoopSource(AXObserverRef observer) {
    return AXObserverGetRunLoopSource(observer);
}

// Get focused application element
static inline AXUIElementRef getFocusedApplication() {
    AXUIElementRef systemWide = AXUIElementCreateSystemWide();
    AXUIElementRef focusedApp = NULL;
    AXError error = AXUIElementCopyAttributeValue(
        systemWide,
        kAXFocusedApplicationAttribute,
        (CFTypeRef *)&focusedApp
    );
    CFRelease(systemWide);

    if (error != kAXErrorSuccess) {
        return NULL;
    }
    return focusedApp;
}
*/
import "C"

import (
	"errors"
	"sync"
)

// FocusCallback is called when focus changes to a new element.
type FocusCallback func(element *Element)

var (
	focusCallbackMu sync.RWMutex
	focusCallback   FocusCallback
)

// SetFocusCallback sets the callback function for focus changes.
func SetFocusCallback(cb FocusCallback) {
	focusCallbackMu.Lock()
	defer focusCallbackMu.Unlock()
	focusCallback = cb
}

//export goFocusCallback
func goFocusCallback(ref C.AXUIElementRef) {
	focusCallbackMu.RLock()
	cb := focusCallback
	focusCallbackMu.RUnlock()

	if cb != nil && uintptr(ref) != 0 {
		// Note: We don't own this ref, so don't release it
		cb(&Element{ref: ref})
	}
}

// Observer wraps an AXObserverRef for monitoring accessibility events.
type Observer struct {
	ref C.AXObserverRef
	pid int
}

// NewObserver creates a new observer for the given process ID.
func NewObserver(pid int) (*Observer, error) {
	ref := C.createObserver(C.pid_t(pid))
	if uintptr(ref) == 0 {
		return nil, ErrAccessibilityNotEnabled
	}
	return &Observer{ref: ref, pid: pid}, nil
}

// AddFocusNotification registers for focus change notifications on the element.
func (o *Observer) AddFocusNotification(element *Element) error {
	result := C.addNotification(o.ref, element.ref, C.CFStringRef(C.kAXFocusedUIElementChangedNotification))
	if result != 0 {
		return errors.New("failed to add focus notification")
	}
	return nil
}

// Start adds the observer to the current run loop.
func (o *Observer) Start() {
	source := C.getRunLoopSource(o.ref)
	C.CFRunLoopAddSource(C.CFRunLoopGetCurrent(), source, C.kCFRunLoopDefaultMode)
}

// Stop removes the observer from the run loop.
func (o *Observer) Stop() {
	source := C.getRunLoopSource(o.ref)
	C.CFRunLoopRemoveSource(C.CFRunLoopGetCurrent(), source, C.kCFRunLoopDefaultMode)
}

// FocusedApplication returns the currently focused application element.
func FocusedApplication() *Element {
	ref := C.getFocusedApplication()
	if uintptr(ref) == 0 {
		return nil
	}
	return &Element{ref: ref}
}
