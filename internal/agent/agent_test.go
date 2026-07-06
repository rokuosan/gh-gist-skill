package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLink(t *testing.T) {
	tmp := t.TempDir()
	target := filepath.Join(tmp, "skills-store", "my-skill")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(tmp, "claude", "skills")

	link, err := Link(dir, "my-skill", target)
	if err != nil {
		t.Fatalf("Link() error: %v", err)
	}
	got, err := os.Readlink(link)
	if err != nil {
		t.Fatalf("Readlink() error: %v", err)
	}
	if got != target {
		t.Errorf("Readlink() = %q, want %q", got, target)
	}

	// Re-linking replaces an existing symlink.
	other := filepath.Join(tmp, "skills-store", "other")
	if err := os.MkdirAll(other, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Link(dir, "my-skill", other); err != nil {
		t.Fatalf("Link() relink error: %v", err)
	}
	if got, _ := os.Readlink(link); got != other {
		t.Errorf("after relink Readlink() = %q, want %q", got, other)
	}

	// A real directory at the link path is refused.
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(link, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Link(dir, "my-skill", target); err == nil {
		t.Error("Link() over a real directory: want error, got nil")
	}
}
