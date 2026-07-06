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

// Remove implements `gh gist-skill remove <name>`: it uninstalls a skill.
// Project scope: submodule deinit -> git rm -> clean .git/modules.
// User scope: delete the clone. Both scopes also remove the symlinks this
// tool created.
func Remove(args []string) error {
	fs := flag.NewFlagSet("remove", flag.ContinueOnError)
	scope := fs.String("scope", "", "restrict to one scope: project or user")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage: gh gist-skill remove <name> [flags]")
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
		return fmt.Errorf("expected exactly one skill name")
	}
	if *scope != "" && *scope != "project" && *scope != "user" {
		return fmt.Errorf("invalid --scope %q (project or user)", *scope)
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

	var matches []installedSkill
	for _, s := range append(project, user...) {
		if s.Name == name && (*scope == "" || s.Scope == *scope) {
			matches = append(matches, s)
		}
	}
	switch len(matches) {
	case 0:
		return fmt.Errorf("skill %q is not installed", name)
	case 1:
	default:
		return fmt.Errorf("skill %q is installed in both project and user scope; pick one with --scope", name)
	}

	s := matches[0]
	if s.Scope == "project" {
		if err := removeSubmodule(s); err != nil {
			return err
		}
		fmt.Printf("✓ Removed submodule: %s (commit the change to finish)\n", s.Path)
		return agent.RemoveLink(filepath.Join(s.repoRoot, ".claude", "skills"), name, projectLinkTarget(name))
	}
	target, err := filepath.Abs(s.dir())
	if err != nil {
		return err
	}
	if err := os.RemoveAll(s.Path); err != nil {
		return err
	}
	fmt.Printf("✓ Removed: %s\n", s.Path)
	return removeLinks(name, target)
}

func removeSubmodule(s installedSkill) error {
	if err := git.SubmoduleDeinit(s.repoRoot, s.Path); err != nil {
		return err
	}
	if err := git.RemovePath(s.repoRoot, s.Path); err != nil {
		return err
	}
	modules, err := git.ModulesDir(s.repoRoot, s.submoduleName)
	if err != nil {
		return err
	}
	return os.RemoveAll(modules)
}

// removeLinks deletes the symlinks this tool may have created for the skill,
// leaving links that point elsewhere untouched.
func removeLinks(name, target string) error {
	for _, dirFn := range []func() (string, error){agent.AgentsSkillsDir, agent.ClaudeSkillsDir} {
		dir, err := dirFn()
		if err != nil {
			return err
		}
		if err := agent.RemoveLink(dir, name, target); err != nil {
			return err
		}
	}
	return nil
}
