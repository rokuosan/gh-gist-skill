// Package cli implements the gh gist-skill subcommands.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"

	"github.com/rokuosan/gh-gist-skill/internal/agent"
	"github.com/rokuosan/gh-gist-skill/internal/gist"
	"github.com/rokuosan/gh-gist-skill/internal/git"
	"github.com/rokuosan/gh-gist-skill/internal/skill"
)

const skillFileName = "SKILL.md"

// Copy implements `gh gist-skill copy <gist-url|gist-id>`: it takes a
// fire-and-forget snapshot of a gist. The copy is not tracked afterwards;
// running copy again overwrites it.
//
// Inside a git repository (project scope) the snapshot goes to
// <root>/.agents/skills/<name> with a repo-local ./.claude/skills link.
// Outside one (user scope) it goes to ~/.agents/skills/<name> with a
// ~/.claude/skills link. An explicit --path just places the files there.
func Copy(args []string) error {
	fs := flag.NewFlagSet("copy", flag.ContinueOnError)
	path := fs.String("path", "", "custom destination directory (skips symlinks)")
	noLink := fs.Bool("no-link", false, "skip creating symlinks into agent skill directories")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: gh gist-skill copy <gist-url|gist-id> [flags]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return fmt.Errorf("expected exactly one gist URL or ID")
	}

	if *path == "" {
		scope, reason, err := resolveScope("auto")
		if err != nil {
			return err
		}
		fmt.Printf("✓ Scope: %s (%s)\n", scope, reason)
	}

	g, name, httpClient, err := resolveGistSkill(fs.Arg(0))
	if err != nil {
		return err
	}

	if *path != "" {
		dest := filepath.Join(*path, name)
		if err := writeSnapshot(httpClient, g, dest); err != nil {
			return err
		}
		fmt.Printf("✓ Copied snapshot: %s\n", dest)
		return nil
	}

	if git.IsInsideWorkTree(".") {
		root, err := git.RepoRoot(".")
		if err != nil {
			return err
		}
		dest := filepath.Join(root, ".agents", "skills", name)
		if err := writeSnapshot(httpClient, g, dest); err != nil {
			return err
		}
		fmt.Printf("✓ Copied snapshot: %s\n", dest)
		if *noLink {
			return nil
		}
		return linkProject(root, name)
	}

	agentsDir, err := agent.AgentsSkillsDir()
	if err != nil {
		return err
	}
	dest := filepath.Join(agentsDir, name)
	if err := writeSnapshot(httpClient, g, dest); err != nil {
		return err
	}
	fmt.Printf("✓ Copied snapshot: %s\n", dest)
	if *noLink {
		return nil
	}
	return linkClaude(dest, name)
}

// resolveGistSkill runs the shared front half of every install: parse the
// gist reference, fetch its metadata, and detect the validated skill name.
func resolveGistSkill(arg string) (*gist.Gist, string, *http.Client, error) {
	id, err := gist.ParseID(arg)
	if err != nil {
		return nil, "", nil, err
	}

	restClient, err := api.DefaultRESTClient()
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to create GitHub API client (is gh logged in?): %w", err)
	}
	httpClient, err := api.DefaultHTTPClient()
	if err != nil {
		return nil, "", nil, err
	}

	g, err := gist.Fetch(restClient, id)
	if err != nil {
		return nil, "", nil, err
	}
	fmt.Printf("✓ Resolved gist: %s\n", g.ID)

	name, err := detectName(httpClient, g)
	if err != nil {
		return nil, "", nil, err
	}
	fmt.Printf("✓ Detected skill name from %s: %s\n", skillFileName, name)
	return g, name, httpClient, nil
}

// projectLinkTarget is the relative symlink target used for repo-local
// ./.claude/skills/<name> links (resolved from inside .claude/skills).
func projectLinkTarget(name string) string {
	return filepath.Join("..", "..", ".agents", "skills", name)
}

// linkProject symlinks <root>/.claude/skills/<name> to the project-scope
// skill via a relative path, so the link is committable and nothing outside
// the repository is touched.
func linkProject(root, name string) error {
	link, err := agent.Link(filepath.Join(root, ".claude", "skills"), name, projectLinkTarget(name))
	if err != nil {
		return err
	}
	fmt.Printf("✓ Linked: %s\n", link)
	return nil
}

// linkClaude symlinks ~/.claude/skills/<name> to dest, skipping the link
// when dest already lives in that directory.
func linkClaude(dest, name string) error {
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
