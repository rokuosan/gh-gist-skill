package main

import (
	"fmt"
	"os"

	"github.com/rokuosan/gh-gist-skill/internal/cli"
)

const usage = `gh gist-skill: install Agent Skills published as GitHub Gists

Usage:
  gh gist-skill add <gist-url|gist-id> [flags]    install as a git submodule (inside a repo)
  gh gist-skill copy <gist-url|gist-id> [flags]   snapshot-copy a gist skill (not tracked)

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
