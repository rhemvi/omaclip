// Package vcs returns the application version derived from embedded VCS build info.
// If the working tree was modified at build time, "_dirty" is appended.
package vcs

import (
	"fmt"
	"runtime/debug"
)

// Revision and Modified can be set at build time via ldflags:
//
//	-X clipmaster/foundation/vcs.Revision=$(git rev-parse --short HEAD)
//	-X clipmaster/foundation/vcs.Modified=$(git diff --quiet && echo false || echo true)
var Revision string
var Modified string

func Version(appVersion string) string {
	revision, dirty := resolveRevision()
	if revision == "" {
		return appVersion
	}
	if dirty {
		return fmt.Sprintf("v%s+%s.dirty", appVersion, revision)
	}
	return fmt.Sprintf("v%s+%s", appVersion, revision)
}

func resolveRevision() (string, bool) {
	if Revision != "" {
		return Revision, Modified == "true"
	}

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", false
	}

	var revision string
	var dirty bool
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if len(s.Value) > 7 {
				revision = s.Value[:7]
			} else {
				revision = s.Value
			}
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}

	return revision, dirty
}
