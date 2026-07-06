package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rokuosan/gh-gist-skill/internal/agent"
	"github.com/rokuosan/gh-gist-skill/internal/git"
)

// Add implements `gh gist-skill add <gist-url|gist-id>`.
//
// Inside a git repository (project scope) the skill is installed as a git
// submodule at <path>/<name>. Outside one (user scope) it is cloned into
// $XDG_DATA_HOME/gh-gist-skill/skills/<name>. Either way the result is
// linked into the agent skill directories.
func Add(args []string) error {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	scope := fs.String("scope", "auto", "installation scope: auto, project (submodule), or user (clone)")
	noLink := fs.Bool("no-link", false, "skip creating symlinks into agent skill directories")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: gh gist-skill add <gist-url|gist-id> [flags]")
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

	insideRepo := git.IsInsideWorkTree(".")
	switch *scope {
	case "auto":
	case "project":
		if !insideRepo {
			return fmt.Errorf("--scope project requires a git repository")
		}
	case "user":
		insideRepo = false
	default:
		return fmt.Errorf("invalid --scope %q (auto, project, or user)", *scope)
	}

	g, name, _, err := resolveGistSkill(fs.Arg(0))
	if err != nil {
		return err
	}
	cloneURL := "https://gist.github.com/" + g.ID + ".git"

	if insideRepo {
		return addSubmodule(cloneURL, name, *noLink)
	}
	return addClone(cloneURL, name, *noLink)
}

// addSubmodule installs the skill as a git submodule (project scope).
// The path is fixed to .agents/skills at the repository root so
// list/update/remove always find it; the only link is repo-local.
func addSubmodule(cloneURL, name string, noLink bool) error {
	root, err := git.RepoRoot(".")
	if err != nil {
		return err
	}
	path := filepath.Join(".agents", "skills", name)
	dest := filepath.Join(root, path)
	if _, err := os.Lstat(dest); err == nil {
		return fmt.Errorf("%s already exists; remove it first", dest)
	}
	if err := git.SubmoduleAdd(root, cloneURL, path); err != nil {
		return err
	}
	fmt.Printf("✓ Added submodule: %s\n", path)

	if noLink {
		return nil
	}
	return linkProject(root, name)
}

// addClone installs the skill as an independent clone in the user store
// (user scope) and links it into ~/.agents/skills and ~/.claude/skills.
func addClone(cloneURL, name string, noLink bool) error {
	store, err := agent.UserStoreDir()
	if err != nil {
		return err
	}
	dest := filepath.Join(store, name)
	if _, err := os.Lstat(dest); err == nil {
		return fmt.Errorf("%s already exists; use 'gh gist-skill update %s' to update it", dest, name)
	}
	if err := os.MkdirAll(store, 0o755); err != nil {
		return err
	}
	if err := git.Clone(cloneURL, dest); err != nil {
		return err
	}
	fmt.Printf("✓ Cloned: %s\n", dest)

	if noLink {
		return nil
	}
	if err := linkUser(dest, name); err != nil {
		// Roll back so a failed add leaves nothing half-installed behind.
		os.RemoveAll(dest)
		removeLinks(name, dest)
		return err
	}
	return nil
}

// linkUser links a user-scope skill into ~/.agents/skills and ~/.claude/skills.
func linkUser(dest, name string) error {
	agentsDir, err := agent.AgentsSkillsDir()
	if err != nil {
		return err
	}
	link, err := agent.Link(agentsDir, name, dest)
	if err != nil {
		return err
	}
	fmt.Printf("✓ Linked: %s\n", link)
	return linkClaude(dest, name)
}
