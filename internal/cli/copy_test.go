package cli

import "testing"

func TestValidateFilename(t *testing.T) {
	valid := []string{"SKILL.md", "references__tufte-principles.md", "a.txt", "..hidden"}
	for _, n := range valid {
		if err := validateFilename(n); err != nil {
			t.Errorf("validateFilename(%q) unexpected error: %v", n, err)
		}
	}
	invalid := []string{"", ".", "..", "../evil", "a/b", `a\b`, "/etc/passwd"}
	for _, n := range invalid {
		if err := validateFilename(n); err == nil {
			t.Errorf("validateFilename(%q) = nil, want error", n)
		}
	}
}
