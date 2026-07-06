// Package cli implements the gh gist-skill subcommands.
package cli

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"

	"github.com/rokuosan/gh-gist-skill/internal/agent"
	"github.com/rokuosan/gh-gist-skill/internal/gist"
	"github.com/rokuosan/gh-gist-skill/internal/skill"
)

const skillFileName = "SKILL.md"

// Copy implements `gh gist-skill copy <gist-url|gist-id>`: it takes a
// fire-and-forget snapshot of a gist into <path>/<name> and links it into
// the Claude Code skills directory. The copy is not tracked afterwards;
// running copy again overwrites it.
func Copy(args []string) error {
	fs := flag.NewFlagSet("copy", flag.ContinueOnError)
	path := fs.String("path", filepath.Join(".agents", "skills"), "destination directory for the skill snapshot")
	noLink := fs.Bool("no-link", false, "skip creating symlinks into agent skill directories")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: gh gist-skill copy <gist-url|gist-id> [flags]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("expected exactly one gist URL or ID")
	}

	id, err := gist.ParseID(fs.Arg(0))
	if err != nil {
		return err
	}

	restClient, err := api.DefaultRESTClient()
	if err != nil {
		return fmt.Errorf("failed to create GitHub API client (is gh logged in?): %w", err)
	}
	httpClient, err := api.DefaultHTTPClient()
	if err != nil {
		return err
	}

	g, err := gist.Fetch(restClient, id)
	if err != nil {
		return err
	}
	fmt.Printf("✓ Resolved gist: %s\n", g.ID)

	name, err := detectName(httpClient, g)
	if err != nil {
		return err
	}
	fmt.Printf("✓ Detected skill name from %s: %s\n", skillFileName, name)

	dest := filepath.Join(*path, name)
	if err := writeSnapshot(httpClient, g, dest); err != nil {
		return err
	}
	fmt.Printf("✓ Copied snapshot: %s\n", dest)

	if *noLink {
		return nil
	}
	absDest, err := filepath.Abs(dest)
	if err != nil {
		return err
	}
	claudeDir, err := agent.ClaudeSkillsDir()
	if err != nil {
		return err
	}
	if filepath.Clean(claudeDir) == filepath.Dir(absDest) {
		return nil
	}
	link, err := agent.Link(claudeDir, name, absDest)
	if err != nil {
		return err
	}
	fmt.Printf("✓ Linked: %s\n", link)
	return nil
}

func detectName(httpClient *http.Client, g *gist.Gist) (string, error) {
	f, ok := g.Files[skillFileName]
	if !ok {
		return "", fmt.Errorf("gist %s has no %s (see %s)", g.ID, skillFileName, skill.SpecURL)
	}
	content, err := gist.FileContent(httpClient, f)
	if err != nil {
		return "", err
	}
	return skill.ParseName(content)
}

// writeSnapshot downloads all gist files into a temporary directory and only
// replaces dest once every file has been written, so a mid-download failure
// does not destroy an existing copy.
func writeSnapshot(httpClient *http.Client, g *gist.Gist, dest string) error {
	parent := filepath.Dir(dest)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	tmp, err := os.MkdirTemp(parent, ".gist-skill-tmp-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	// MkdirTemp creates the directory with 0700; align with MkdirAll above.
	if err := os.Chmod(tmp, 0o755); err != nil {
		return err
	}

	for _, f := range g.Files {
		if err := validateFilename(f.Filename); err != nil {
			return err
		}
		content, err := gist.FileContent(httpClient, f)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(tmp, f.Filename), []byte(content), 0o644); err != nil {
			return err
		}
	}

	if err := os.RemoveAll(dest); err != nil {
		return err
	}
	return os.Rename(tmp, dest)
}

// validateFilename rejects filenames that could escape the destination
// directory. Gists are flat so legitimate filenames never contain path
// separators, but the API response is not trusted.
func validateFilename(name string) error {
	if name == "" || name == "." || name == ".." ||
		strings.ContainsAny(name, `/\`) || filepath.IsAbs(name) {
		return fmt.Errorf("refusing to write unsafe filename %q from gist", name)
	}
	return nil
}
