package updater

import (
	"runtime"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name     string
		current  string
		latest   string
		expected int
	}{
		{"equal versions", "v1.2.3", "v1.2.3", 0},
		{"current older patch", "v1.2.3", "v1.2.4", -1},
		{"current newer patch", "v1.2.4", "v1.2.3", 1},
		{"current older minor", "v1.2.3", "v1.3.0", -1},
		{"current newer minor", "v1.3.0", "v1.2.9", 1},
		{"current older major", "v1.9.9", "v2.0.0", -1},
		{"current newer major", "v2.0.0", "v1.9.9", 1},
		{"without v prefix", "1.2.3", "1.2.4", -1},
		{"mixed v prefix", "v1.2.3", "1.2.3", 0},
		{"undefined current", "undefined", "v1.0.0", -1},
		{"undefined latest", "v1.0.0", "undefined", 1},
		{"both undefined", "undefined", "undefined", 0},
		{"zero versions", "v0.0.0", "v0.0.1", -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompareVersions(tt.current, tt.latest)
			if result != tt.expected {
				t.Errorf("CompareVersions(%q, %q) = %d, want %d",
					tt.current, tt.latest, result, tt.expected)
			}
		})
	}
}

func TestGetArchiveName(t *testing.T) {
	name := getArchiveName()
	expectedOS := runtime.GOOS
	expectedArch := runtime.GOARCH

	if runtime.GOOS == "windows" {
		if name != "process-compose_"+expectedOS+"_"+expectedArch+".zip" {
			t.Errorf("getArchiveName() = %q, want .zip suffix on Windows", name)
		}
	} else {
		if name != "process-compose_"+expectedOS+"_"+expectedArch+".tar.gz" {
			t.Errorf("getArchiveName() = %q, want .tar.gz suffix on non-Windows", name)
		}
	}
}
