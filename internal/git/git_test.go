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

// commitFile adds a new commit to dir touching name.
func commitFile(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(name+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := run(dir, "add", name); err != nil {
		t.Fatal(err)
	}
	if _, err := run(dir, "commit", "-q", "-m", "add "+name); err != nil {
		t.Fatal(err)
	}
}

func TestCloneHeadPull(t *testing.T) {
	upstream := t.TempDir()
	initRepo(t, upstream)
	dest := filepath.Join(t.TempDir(), "clone")

	if err := Clone(upstream, dest); err != nil {
		t.Fatalf("Clone() error: %v", err)
	}
	head, err := Head(dest)
	if err != nil {
		t.Fatalf("Head() error: %v", err)
	}
	remote, err := RemoteHead(dest)
	if err != nil {
		t.Fatalf("RemoteHead() error: %v", err)
	}
	if head != remote {
		t.Errorf("fresh clone: Head() = %s, RemoteHead() = %s", head, remote)
	}

	commitFile(t, upstream, "extra.md")
	remote, err = RemoteHead(dest)
	if err != nil {
		t.Fatal(err)
	}
	if head == remote {
		t.Fatal("RemoteHead() did not observe the new upstream commit")
	}
	if err := PullFFOnly(dest); err != nil {
		t.Fatalf("PullFFOnly() error: %v", err)
	}
	if got, _ := Head(dest); got != remote {
		t.Errorf("after pull: Head() = %s, want %s", got, remote)
	}
}

func TestSubmoduleLifecycle(t *testing.T) {
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "protocol.file.allow")
	t.Setenv("GIT_CONFIG_VALUE_0", "always")

	upstream := t.TempDir()
	initRepo(t, upstream)
	repo := t.TempDir()
	initRepo(t, repo)

	dest := filepath.Join(".agents", "skills", "my-skill")
	if err := SubmoduleAdd(repo, upstream, dest); err != nil {
		t.Fatal(err)
	}

	subs, err := Submodules(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(subs) != 1 || filepath.FromSlash(subs[0].Path) != dest || subs[0].URL != upstream {
		t.Fatalf("Submodules() = %+v, want one entry with path %s and url %s", subs, dest, upstream)
	}

	commitFile(t, upstream, "extra.md")
	if err := SubmoduleUpdateRemote(repo, dest); err != nil {
		t.Fatalf("SubmoduleUpdateRemote() error: %v", err)
	}
	upstreamHead, _ := Head(upstream)
	if got, _ := Head(filepath.Join(repo, dest)); got != upstreamHead {
		t.Errorf("after update --remote: Head() = %s, want upstream %s", got, upstreamHead)
	}

	if err := SubmoduleDeinit(repo, dest); err != nil {
		t.Fatalf("SubmoduleDeinit() error: %v", err)
	}
	if err := RemovePath(repo, dest); err != nil {
		t.Fatalf("RemovePath() error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, dest)); !os.IsNotExist(err) {
		t.Errorf("submodule path still exists after removal")
	}
	modules, err := ModulesDir(repo, subs[0].Name)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(modules); err != nil {
		t.Fatal(err)
	}
	if subs, _ := Submodules(repo); len(subs) != 0 {
		t.Errorf("Submodules() after removal = %+v, want empty", subs)
	}
}
