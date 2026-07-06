package cli

import (
	"errors"
	"flag"
	"fmt"

	"github.com/rokuosan/gh-gist-skill/internal/git"
)

// Update implements `gh gist-skill update [name]`: it moves skills to the
// latest upstream commit. Project-scope submodules are updated with
// `git submodule update --remote` (the new pin still needs a commit);
// user-scope clones are fast-forwarded with `git pull`.
func Update(args []string) error {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: gh gist-skill update [name]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if fs.NArg() > 1 {
		fs.Usage()
		return fmt.Errorf("expected at most one skill name")
	}
	name := fs.Arg(0)

	project, err := projectSkills()
	if err != nil {
		return err
	}
	user, err := userSkills()
	if err != nil {
		return err
	}

	matched, updatedSubmodules := 0, 0
	for _, s := range append(project, user...) {
		if name != "" && s.Name != name {
			continue
		}
		matched++
		before, _ := git.Head(s.dir())
		if s.Scope == "project" {
			err = git.SubmoduleUpdateRemote(s.repoRoot, s.Path)
		} else {
			err = git.PullFFOnly(s.dir())
		}
		if err != nil {
			return fmt.Errorf("failed to update %s: %w", s.Name, err)
		}
		after, _ := git.Head(s.dir())
		if before == after {
			fmt.Printf("✓ %s (%s): already up to date\n", s.Name, s.Scope)
			continue
		}
		fmt.Printf("✓ %s (%s): %s -> %s\n", s.Name, s.Scope, short(before), short(after))
		if s.Scope == "project" {
			updatedSubmodules++
		}
	}

	if matched == 0 {
		if name == "" {
			fmt.Println("no skills installed")
			return nil
		}
		return fmt.Errorf("skill %q is not installed", name)
	}
	if updatedSubmodules > 0 {
		fmt.Println("Note: updated submodules are not committed; review and commit to pin the new versions.")
	}
	return nil
}

func short(commit string) string {
	if len(commit) >= 7 {
		return commit[:7]
	}
	return commit
}
