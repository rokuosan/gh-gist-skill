// Package git wraps the git CLI via os/exec. The CLI is used instead of a
// Go implementation so submodules and the user's git configuration
// (credential helpers, protocol settings) work as they do on the command line.
package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// IsInsideWorkTree reports whether dir is inside a git working tree.
func IsInsideWorkTree(dir string) bool {
	out, err := run(dir, "rev-parse", "--is-inside-work-tree")
	return err == nil && out == "true"
}

// SubmoduleAdd runs `git submodule add <url> <path>` in dir.
func SubmoduleAdd(dir, url, path string) error {
	_, err := run(dir, "submodule", "add", url, path)
	return err
}

// RepoRoot returns the top-level directory of the work tree containing dir.
func RepoRoot(dir string) (string, error) {
	return run(dir, "rev-parse", "--show-toplevel")
}

// Clone clones url into dest.
func Clone(url, dest string) error {
	_, err := run(".", "clone", "-q", url, dest)
	return err
}

// PullFFOnly fast-forwards dir to its upstream.
func PullFFOnly(dir string) error {
	_, err := run(dir, "pull", "-q", "--ff-only")
	return err
}

// Head returns the full commit hash of HEAD in dir.
func Head(dir string) (string, error) {
	return run(dir, "rev-parse", "HEAD")
}

// RemoteHead returns the full commit hash of the default branch on origin.
func RemoteHead(dir string) (string, error) {
	out, err := run(dir, "ls-remote", "origin", "HEAD")
	if err != nil {
		return "", err
	}
	fields := strings.Fields(out)
	if len(fields) == 0 {
		return "", fmt.Errorf("git ls-remote origin HEAD returned nothing in %s", dir)
	}
	return fields[0], nil
}

// Submodule describes one entry in .gitmodules.
type Submodule struct {
	Name string
	Path string
	URL  string
}

// Submodules parses .gitmodules at the root of the repository containing dir.
// A missing .gitmodules yields an empty list.
func Submodules(dir string) ([]Submodule, error) {
	root, err := RepoRoot(dir)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(filepath.Join(root, ".gitmodules")); err != nil {
		return nil, nil
	}
	out, err := run(root, "config", "-f", ".gitmodules", "--get-regexp", `^submodule\.`)
	if err != nil {
		return nil, nil // no submodule entries
	}
	index := map[string]int{}
	var subs []Submodule
	for line := range strings.SplitSeq(out, "\n") {
		key, value, ok := strings.Cut(line, " ")
		if !ok {
			continue
		}
		key = strings.TrimPrefix(key, "submodule.")
		var name, field string
		switch {
		case strings.HasSuffix(key, ".path"):
			name, field = strings.TrimSuffix(key, ".path"), "path"
		case strings.HasSuffix(key, ".url"):
			name, field = strings.TrimSuffix(key, ".url"), "url"
		default:
			continue
		}
		i, ok := index[name]
		if !ok {
			i = len(subs)
			index[name] = i
			subs = append(subs, Submodule{Name: name})
		}
		if field == "path" {
			subs[i].Path = value
		} else {
			subs[i].URL = value
		}
	}
	return subs, nil
}

// SubmoduleUpdateRemote checks out the latest upstream commit of the
// submodule at path.
func SubmoduleUpdateRemote(dir, path string) error {
	_, err := run(dir, "submodule", "update", "--init", "--remote", "--", path)
	return err
}

// SubmoduleDeinit unregisters and empties the submodule at path.
func SubmoduleDeinit(dir, path string) error {
	_, err := run(dir, "submodule", "deinit", "-f", "--", path)
	return err
}

// RemovePath runs `git rm -f <path>` in dir.
func RemovePath(dir, path string) error {
	_, err := run(dir, "rm", "-qf", "--", path)
	return err
}

// ModulesDir returns the .git/modules storage directory for the submodule
// with the given .gitmodules name.
func ModulesDir(dir, name string) (string, error) {
	gitDir, err := run(dir, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return "", err
	}
	return filepath.Join(gitDir, "modules", name), nil
}
