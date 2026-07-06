package main

import (
	"fmt"
	"os"

	"github.com/rokuosan/gh-gist-skill/internal/cli"
)

const usage = `gh gist-skill: install Agent Skills published as GitHub Gists

Usage:
  gh gist-skill add <gist-url|gist-id> [flags]    install as a submodule (in a repo) or user-scope clone
  gh gist-skill copy <gist-url|gist-id> [flags]   snapshot-copy a gist skill (not tracked)
  gh gist-skill update [name]                     update all skills or one
  gh gist-skill remove <name>                     uninstall a skill and its symlinks
  gh gist-skill list [flags]                      list installed skills and update status

Run 'gh gist-skill <command> --help' for command flags.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}
	var err error
	switch os.Args[1] {
	case "add":
		err = cli.Add(os.Args[2:])
	case "copy":
		err = cli.Copy(os.Args[2:])
	case "update":
		err = cli.Update(os.Args[2:])
	case "remove":
		err = cli.Remove(os.Args[2:])
	case "list":
		err = cli.List(os.Args[2:])
	case "-h", "--help", "help":
		fmt.Print(usage)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n%s", os.Args[1], usage)
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
