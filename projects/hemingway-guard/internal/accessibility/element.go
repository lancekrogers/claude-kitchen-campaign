// Package accessibility provides wrappers for macOS Accessibility APIs.
package accessibility

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework ApplicationServices -framework AppKit

#include <ApplicationServices/ApplicationServices.h>
#import <AppKit/AppKit.h>

// Get the system-wide accessibility element
AXUIElementRef createSystemWideElement() {
    return AXUIElementCreateSystemWide();
}

// Get the focused element from a given element
AXUIElementRef getFocusedElement(AXUIElementRef element) {
    AXUIElementRef focusedElement = NULL;
    AXError error = AXUIElementCopyAttributeValue(
        element,
        kAXFocusedUIElementAttribute,
        (CFTypeRef *)&focusedElement
    );
    if (error != kAXErrorSuccess) {
        return NULL;
    }
    return focusedElement;
}

// Get string attribute from an element
char* getStringAttribute(AXUIElementRef element, CFStringRef attribute) {
    CFTypeRef value = NULL;
    AXError error = AXUIElementCopyAttributeValue(element, attribute, &value);
    if (error != kAXErrorSuccess || value == NULL) {
        return NULL;
    }

    if (CFGetTypeID(value) != CFStringGetTypeID()) {
        CFRelease(value);
        return NULL;
    }

    CFStringRef stringValue = (CFStringRef)value;
    CFIndex length = CFStringGetLength(stringValue);
    CFIndex maxSize = CFStringGetMaximumSizeForEncoding(length, kCFStringEncodingUTF8) + 1;
    char *buffer = malloc(maxSize);

    if (!CFStringGetCString(stringValue, buffer, maxSize, kCFStringEncodingUTF8)) {
        free(buffer);
        CFRelease(value);
        return NULL;
    }

    CFRelease(value);
    return buffer;
}

// Get the role of an element
char* getRole(AXUIElementRef element) {
    return getStringAttribute(element, kAXRoleAttribute);
}

// Get the value (text content) of an element
char* getValue(AXUIElementRef element) {
    return getStringAttribute(element, kAXValueAttribute);
}

// Set the value of an element
int setValue(AXUIElementRef element, const char* value) {
    CFStringRef cfValue = CFStringCreateWithCString(NULL, value, kCFStringEncodingUTF8);
    if (cfValue == NULL) {
        return -1;
    }

    AXError error = AXUIElementSetAttributeValue(element, kAXValueAttribute, cfValue);
    CFRelease(cfValue);

    return error == kAXErrorSuccess ? 0 : -1;
}

// Get the PID of the process owning the element
pid_t getPID(AXUIElementRef element) {
    pid_t pid = 0;
    AXError error = AXUIElementGetPid(element, &pid);
    if (error != kAXErrorSuccess) {
        return -1;
    }
    return pid;
}

// Get bundle identifier for a PID
char* getBundleIDForPID(pid_t pid) {
    NSRunningApplication *app = [NSRunningApplication runningApplicationWithProcessIdentifier:pid];
    if (app == nil || app.bundleIdentifier == nil) {
        return NULL;
    }

    const char *bundleID = [app.bundleIdentifier UTF8String];
    return strdup(bundleID);
}

// Check if the element is editable
int isEditable(AXUIElementRef element) {
    CFTypeRef value = NULL;
    AXError error = AXUIElementCopyAttributeValue(element, CFSTR("AXEditable"), &value);
    if (error != kAXErrorSuccess || value == NULL) {
        // If we can't determine, assume not editable
        return 0;
    }

    int editable = 0;
    if (CFGetTypeID(value) == CFBooleanGetTypeID()) {
        editable = CFBooleanGetValue((CFBooleanRef)value) ? 1 : 0;
    }

    CFRelease(value);
    return editable;
}

// Release an AXUIElement
void releaseElement(AXUIElementRef element) {
    if (element != NULL) {
        CFRelease(element);
    }
}

// Free a C string
void freeString(char* str) {
    if (str != NULL) {
        free(str);
    }
}
*/
import "C"
import (
	"errors"
	"unsafe"
)

// Element wraps an AXUIElementRef.
type Element struct {
	ref C.AXUIElementRef
}

// ErrAccessibilityNotEnabled indicates accessibility permissions are not granted.
var ErrAccessibilityNotEnabled = errors.New("accessibility permissions not enabled")

// ErrElementNotFound indicates the requested element was not found.
var ErrElementNotFound = errors.New("element not found")

// SystemWideElement returns the system-wide accessibility element.
func SystemWideElement() *Element {
	ref := C.createSystemWideElement()
	if uintptr(ref) == 0 {
		return nil
	}
	return &Element{ref: ref}
}

// FocusedElement returns the currently focused UI element.
func (e *Element) FocusedElement() (*Element, error) {
	focused := C.getFocusedElement(e.ref)
	if uintptr(focused) == 0 {
		return nil, ErrElementNotFound
	}
	return &Element{ref: focused}, nil
}

// Role returns the AX role of the element.
func (e *Element) Role() string {
	cStr := C.getRole(e.ref)
	if cStr == nil {
		return ""
	}
	defer C.freeString(cStr)
	return C.GoString(cStr)
}

// Value returns the text value of the element.
func (e *Element) Value() string {
	cStr := C.getValue(e.ref)
	if cStr == nil {
		return ""
	}
	defer C.freeString(cStr)
	return C.GoString(cStr)
}

// SetValue sets the text value of the element.
func (e *Element) SetValue(value string) error {
	cStr := C.CString(value)
	defer C.free(unsafe.Pointer(cStr))

	result := C.setValue(e.ref, cStr)
	if result != 0 {
		return errors.New("failed to set value")
	}
	return nil
}

// PID returns the process ID of the application owning this element.
func (e *Element) PID() int {
	return int(C.getPID(e.ref))
}

// BundleID returns the bundle identifier of the application owning this element.
func (e *Element) BundleID() string {
	pid := C.getPID(e.ref)
	if pid < 0 {
		return ""
	}

	cStr := C.getBundleIDForPID(pid)
	if cStr == nil {
		return ""
	}
	defer C.freeString(cStr)
	return C.GoString(cStr)
}

// IsEditable returns whether the element is editable.
func (e *Element) IsEditable() bool {
	return C.isEditable(e.ref) == 1
}

// IsTextField returns whether the element is a text field or text area.
func (e *Element) IsTextField() bool {
	role := e.Role()
	return role == "AXTextField" || role == "AXTextArea"
}

// Release frees the underlying AXUIElementRef.
func (e *Element) Release() {
	if uintptr(e.ref) != 0 {
		C.releaseElement(e.ref)
		e.ref = C.AXUIElementRef(uintptr(0))
	}
}
