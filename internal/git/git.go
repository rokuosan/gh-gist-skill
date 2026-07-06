// Package git wraps the git CLI via os/exec. The CLI is used instead of a
// Go implementation so submodules and the user's git configuration
// (credential helpers, protocol settings) work as they do on the command line.
package git

import (
	"bytes"
	"fmt"
	"os/exec"
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
