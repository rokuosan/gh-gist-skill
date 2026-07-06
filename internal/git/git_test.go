package git

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// initRepo creates a git repository with one commit containing SKILL.md,
// standing in for a gist's clone URL.
func initRepo(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "test"},
	} {
		if _, err := run(dir, args...); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: my-skill\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := run(dir, "add", "SKILL.md"); err != nil {
		t.Fatal(err)
	}
	if _, err := run(dir, "commit", "-q", "-m", "init"); err != nil {
		t.Fatal(err)
	}
}

func TestIsInsideWorkTree(t *testing.T) {
	repo := t.TempDir()
	initRepo(t, repo)
	if !IsInsideWorkTree(repo) {
		t.Errorf("IsInsideWorkTree(%q) = false, want true", repo)
	}
	if IsInsideWorkTree(os.TempDir()) {
		t.Errorf("IsInsideWorkTree(%q) = true, want false", os.TempDir())
	}
}

func TestSubmoduleAdd(t *testing.T) {
	// Git blocks file:// submodule clones by default; allow it for the test.
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "protocol.file.allow")
	t.Setenv("GIT_CONFIG_VALUE_0", "always")

	upstream := t.TempDir()
	initRepo(t, upstream)
	repo := t.TempDir()
	initRepo(t, repo)

	dest := filepath.Join(".agents", "skills", "my-skill")
	if err := SubmoduleAdd(repo, upstream, dest); err != nil {
		t.Fatalf("SubmoduleAdd() error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, dest, "SKILL.md")); err != nil {
		t.Errorf("SKILL.md not checked out in submodule: %v", err)
	}
	gitmodules, err := os.ReadFile(filepath.Join(repo, ".gitmodules"))
	if err != nil {
		t.Fatalf(".gitmodules not created: %v", err)
	}
	if want := "path = .agents/skills/my-skill"; !strings.Contains(string(gitmodules), want) {
		t.Errorf(".gitmodules missing %q:\n%s", want, gitmodules)
	}

	// Adding to an occupied path fails with git's own error.
	if err := SubmoduleAdd(repo, upstream, dest); err == nil {
		t.Error("SubmoduleAdd() over existing path: want error, got nil")
	}
}
