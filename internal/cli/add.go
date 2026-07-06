package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rokuosan/gh-gist-skill/internal/git"
)

// Add implements `gh gist-skill add <gist-url|gist-id>`: it installs a gist
// skill as a git submodule at <path>/<name> inside the current repository
// (project scope) and links it into the Claude Code skills directory.
func Add(args []string) error {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	path := fs.String("path", filepath.Join(".agents", "skills"), "destination directory for the submodule")
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

	if !git.IsInsideWorkTree(".") {
		return fmt.Errorf("add installs a git submodule and must run inside a git repository; use 'gh gist-skill copy' for a plain snapshot (user-scope clone mode is planned)")
	}

	g, name, _, err := resolveGistSkill(fs.Arg(0))
	if err != nil {
		return err
	}

	dest := filepath.Join(*path, name)
	if _, err := os.Lstat(dest); err == nil {
		return fmt.Errorf("%s already exists; remove it first", dest)
	}

	cloneURL := "https://gist.github.com/" + g.ID + ".git"
	if err := git.SubmoduleAdd(".", cloneURL, dest); err != nil {
		return err
	}
	fmt.Printf("✓ Added submodule: %s\n", dest)

	if *noLink {
		return nil
	}
	return linkClaude(dest, name)
}
