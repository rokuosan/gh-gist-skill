package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/rokuosan/gh-gist-skill/internal/git"
)

// List implements `gh gist-skill list`: it prints the skills managed by this
// tool (project-scope submodules and user-scope clones) with their pinned
// commit and whether upstream has moved.
func List(args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	noStatus := fs.Bool("no-status", false, "skip the network check for upstream updates")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: gh gist-skill list [flags]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	project, err := projectSkills()
	if err != nil {
		return err
	}
	user, err := userSkills()
	if err != nil {
		return err
	}
	skills := append(project, user...)
	if len(skills) == 0 {
		fmt.Println("no skills installed")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 2, 8, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSCOPE\tCOMMIT\tSTATUS\tPATH")
	for _, s := range skills {
		commit := "-"
		if head, err := git.Head(s.dir()); err == nil {
			commit = head[:7]
		}
		status := ""
		if !*noStatus {
			status = updateStatus(s.dir())
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", s.Name, s.Scope, commit, status, s.Path)
	}
	return w.Flush()
}

// updateStatus compares HEAD with the upstream default branch.
func updateStatus(dir string) string {
	head, err := git.Head(dir)
	if err != nil {
		return "unknown"
	}
	remote, err := git.RemoteHead(dir)
	if err != nil {
		return "unknown"
	}
	if head == remote {
		return "up to date"
	}
	return "update available"
}
