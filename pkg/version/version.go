package version

import (
	"fmt"
	"os"
	"strings"
)

var (
	// Version is the application version (set via ldflags)
	Version = "dev"

	// BuildTime is the build timestamp (set via ldflags)
	BuildTime = "unknown"

	// GitCommit is the git commit hash (set via ldflags)
	GitCommit = "unknown"

	// GitTag is the git tag (set via ldflags)
	GitTag = "unknown"
)

// GetVersion returns the application version
// Falls back to VERSION file if version is "dev"
func GetVersion() string {
	if Version == "dev" {
		// Fallback: try to read from VERSION file
		if data, err := os.ReadFile("VERSION"); err == nil {
			return strings.TrimSpace(string(data))
		}
		// Try relative path from bin directory
		if data, err := os.ReadFile("../VERSION"); err == nil {
			return strings.TrimSpace(string(data))
		}
	}
	return Version
}

// GetFullVersion returns full version information as a map
func GetFullVersion() map[string]string {
	return map[string]string{
		"version":   GetVersion(),
		"buildTime": BuildTime,
		"gitCommit": GitCommit,
		"gitTag":    GitTag,
	}
}

// String returns version as a formatted string
func String() string {
	v := GetVersion()
	if GitTag != "unknown" && GitTag != "" {
		return fmt.Sprintf("%s (tag: %s, commit: %s)", v, GitTag, GitCommit)
	}
	if GitCommit != "unknown" && GitCommit != "" {
		return fmt.Sprintf("%s (commit: %s)", v, GitCommit)
	}
	return v
}
