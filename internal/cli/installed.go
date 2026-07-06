package cli

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/rokuosan/gh-gist-skill/internal/agent"
	"github.com/rokuosan/gh-gist-skill/internal/git"
)

// installedSkill is one skill this tool manages.
type installedSkill struct {
	Name  string
	Scope string // "project" or "user"
	Path  string // work-tree path (project) or clone directory (user)
	// submodule fields, project scope only
	submoduleName string
	repoRoot      string
}

// managedPrefix is the path prefix under which project-scope skills live.
var managedPrefix = filepath.Join(".agents", "skills") + string(filepath.Separator)

// projectSkills lists gist-skill submodules of the repository containing the
// working directory. Outside a repository it returns nothing.
func projectSkills() ([]installedSkill, error) {
	if !git.IsInsideWorkTree(".") {
		return nil, nil
	}
	root, err := git.RepoRoot(".")
	if err != nil {
		return nil, err
	}
	subs, err := git.Submodules(".")
	if err != nil {
		return nil, err
	}
	var skills []installedSkill
	for _, sm := range subs {
		if !strings.HasPrefix(filepath.FromSlash(sm.Path), managedPrefix) {
			continue
		}
		skills = append(skills, installedSkill{
			Name:          filepath.Base(sm.Path),
			Scope:         "project",
			Path:          sm.Path,
			submoduleName: sm.Name,
			repoRoot:      root,
		})
	}
	return skills, nil
}

// userSkills lists clones in the user store.
func userSkills() ([]installedSkill, error) {
	store, err := agent.UserStoreDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(store)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var skills []installedSkill
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skills = append(skills, installedSkill{
			Name:  e.Name(),
			Scope: "user",
			Path:  filepath.Join(store, e.Name()),
		})
	}
	return skills, nil
}

// dir returns the on-disk directory of the skill.
func (s installedSkill) dir() string {
	if s.Scope == "project" {
		return filepath.Join(s.repoRoot, s.Path)
	}
	return s.Path
}
