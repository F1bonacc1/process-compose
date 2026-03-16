package updater

import (
	"strconv"
	"strings"
)

// CompareVersions compares two semantic version strings.
// Returns -1 if current < latest, 0 if equal, 1 if current > latest.
// Non-numeric versions (e.g. "undefined") are treated as older than any numeric version.
func CompareVersions(current, latest string) int {
	currentParts := parseVersion(current)
	latestParts := parseVersion(latest)

	if currentParts == nil && latestParts == nil {
		return 0
	}
	if currentParts == nil {
		return -1
	}
	if latestParts == nil {
		return 1
	}

	for i := 0; i < 3; i++ {
		if currentParts[i] < latestParts[i] {
			return -1
		}
		if currentParts[i] > latestParts[i] {
			return 1
		}
	}
	return 0
}

// parseVersion strips a leading "v" and splits on "." into [major, minor, patch].
// Returns nil if the version string is not valid semver.
func parseVersion(v string) []int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	if len(parts) != 3 {
		return nil
	}
	nums := make([]int, 3)
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil
		}
		nums[i] = n
	}
	return nums
}
