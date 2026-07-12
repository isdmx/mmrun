package version

import (
	"strings"
	"testing"
)

func TestGet_DefaultsPopulated(t *testing.T) {
	info := Get()
	if info.Version == "" {
		t.Error("Version should not be empty")
	}
	if info.GoVersion == "" || info.OS == "" || info.Arch == "" {
		t.Errorf("runtime fields should be populated: %+v", info)
	}
}

func TestString_ContainsVersionAndCommit(t *testing.T) {
	Version = "v1.2.3"
	Commit = "abc1234"
	Date = "2026-07-12T00:00:00Z"
	t.Cleanup(func() {
		Version, Commit, Date = "dev", "none", "unknown"
	})

	s := String()
	for _, want := range []string{"mmrun", "v1.2.3", "abc1234", "2026-07-12T00:00:00Z"} {
		if !strings.Contains(s, want) {
			t.Errorf("String() = %q, missing %q", s, want)
		}
	}
}
