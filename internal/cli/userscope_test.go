package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/rokuosan/gh-gist-skill/internal/agent"
	"github.com/rokuosan/gh-gist-skill/internal/git"
)

// gitRun is a test helper for preparing fixture repositories.
func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// installUserSkill fakes what `add` does in user scope: clone + links.
func installUserSkill(t *testing.T, upstream, name string) string {
	t.Helper()
	store, err := agent.UserStoreDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(store, 0o755); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(store, name)
	if err := git.Clone(upstream, dest); err != nil {
		t.Fatal(err)
	}
	for _, dirFn := range []func() (string, error){agent.AgentsSkillsDir, agent.ClaudeSkillsDir} {
		dir, err := dirFn()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := agent.Link(dir, name, dest); err != nil {
			t.Fatal(err)
		}
	}
	return dest
}

func TestUserScopeUpdateAndRemove(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	// Run outside any git repository so project scope stays empty.
	t.Chdir(t.TempDir())

	upstream := t.TempDir()
	gitRun(t, upstream, "init", "-q")
	gitRun(t, upstream, "config", "user.email", "test@example.com")
	gitRun(t, upstream, "config", "user.name", "test")
	if err := os.WriteFile(filepath.Join(upstream, "SKILL.md"), []byte("---\nname: my-skill\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, upstream, "add", "SKILL.md")
	gitRun(t, upstream, "commit", "-q", "-m", "init")

	dest := installUserSkill(t, upstream, "my-skill")

	// A stray non-git directory in the store is not managed: update ignores
	// it and remove refuses to delete it.
	store, err := agent.UserStoreDir()
	if err != nil {
		t.Fatal(err)
	}
	stray := filepath.Join(store, "not-a-skill")
	if err := os.MkdirAll(stray, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Remove([]string{"not-a-skill"}); err == nil {
		t.Error("Remove(not-a-skill): want error for non-git directory, got nil")
	}

	// A new upstream commit is picked up by update.
	if err := os.WriteFile(filepath.Join(upstream, "extra.md"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, upstream, "add", "extra.md")
	gitRun(t, upstream, "commit", "-q", "-m", "extra")

	if err := Update(nil); err != nil {
		t.Fatalf("Update() error: %v", err)
	}
	upstreamHead, err := git.Head(upstream)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := git.Head(dest); got != upstreamHead {
		t.Errorf("after update: Head() = %s, want %s", got, upstreamHead)
	}

	// Unknown names are an error.
	if err := Update([]string{"no-such-skill"}); err == nil {
		t.Error("Update(no-such-skill): want error, got nil")
	}

	// remove deletes the clone and both symlinks.
	if err := Remove([]string{"my-skill"}); err != nil {
		t.Fatalf("Remove() error: %v", err)
	}
	if _, err := os.Lstat(dest); !os.IsNotExist(err) {
		t.Errorf("clone still exists after remove")
	}
	if _, err := os.Stat(stray); err != nil {
		t.Errorf("stray directory was touched: %v", err)
	}
	for _, sub := range []string{".agents", ".claude"} {
		link := filepath.Join(home, sub, "skills", "my-skill")
		if _, err := os.Lstat(link); !os.IsNotExist(err) {
			t.Errorf("%s still exists after remove", link)
		}
	}

	if err := Remove([]string{"my-skill"}); err == nil {
		t.Error("Remove() of a removed skill: want error, got nil")
	}
}
