// Package apps provides target application detection for HemingwayGuard.
package apps

// TargetApp represents a messaging application to monitor.
type TargetApp struct {
	Name     string
	BundleID string
	// TextFieldRoles are the AX roles to look for in this app
	TextFieldRoles []string
}

// DefaultTargets returns the default list of messaging apps to monitor.
func DefaultTargets() []TargetApp {
	return []TargetApp{
		{
			Name:           "Messages",
			BundleID:       "com.apple.MobileSMS",
			TextFieldRoles: []string{"AXTextArea", "AXTextField"},
		},
		{
			Name:           "Slack",
			BundleID:       "com.tinyspeck.slackmacgap",
			TextFieldRoles: []string{"AXTextArea", "AXTextField"},
		},
		{
			Name:           "Discord",
			BundleID:       "com.hnc.Discord",
			TextFieldRoles: []string{"AXTextArea", "AXTextField"},
		},
	}
}

// TargetBundleIDs returns a set of bundle IDs for quick lookup.
func TargetBundleIDs() map[string]bool {
	targets := DefaultTargets()
	ids := make(map[string]bool, len(targets))
	for _, t := range targets {
		ids[t.BundleID] = true
	}
	return ids
}

// IsTargetApp checks if the given bundle ID is a monitored app.
func IsTargetApp(bundleID string) bool {
	return TargetBundleIDs()[bundleID]
}

// FindTarget returns the TargetApp for a given bundle ID, or nil if not found.
func FindTarget(bundleID string) *TargetApp {
	for _, t := range DefaultTargets() {
		if t.BundleID == bundleID {
			return &t
		}
	}
	return nil
}
