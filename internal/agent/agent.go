// Package agent links installed skills into agent skill directories.
package agent

import (
	"fmt"
	"os"
	"path/filepath"
)

// ClaudeSkillsDir returns the Claude Code user skills directory.
func ClaudeSkillsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "skills"), nil
}

// AgentsSkillsDir returns the standard user-level Agent Skills directory.
func AgentsSkillsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".agents", "skills"), nil
}

// UserStoreDir returns where user-scope skill clones live:
// $XDG_DATA_HOME/gh-gist-skill/skills (default ~/.local/share).
func UserStoreDir() (string, error) {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "gh-gist-skill", "skills"), nil
}

// RemoveLink deletes dir/<name> if it is a symlink pointing to target.
// Anything else (a real file, a link elsewhere, nothing) is left untouched.
func RemoveLink(dir, name, target string) error {
	link := filepath.Join(dir, name)
	got, err := os.Readlink(link)
	if err != nil || got != target {
		return nil
	}
	return os.Remove(link)
}

// Link creates dir/<name> as a symlink to target. An existing symlink is
// replaced; a real file or directory at the link path is left untouched.
func Link(dir, name, target string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	link := filepath.Join(dir, name)
	if info, err := os.Lstat(link); err == nil {
		if info.Mode()&os.ModeSymlink == 0 {
			return "", fmt.Errorf("%s already exists and is not a symlink; remove it first", link)
		}
		if err := os.Remove(link); err != nil {
			return "", err
		}
	}
	if err := os.Symlink(target, link); err != nil {
		return "", err
	}
	return link, nil
}
