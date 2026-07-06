// Package skill parses SKILL.md frontmatter and validates skill names
// against the Agent Skills specification (https://agentskills.io/specification).
package skill

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// SpecURL is shown in error messages so gist authors know what to fix.
const SpecURL = "https://agentskills.io/specification"

var namePattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// ValidateName checks the spec's name constraints: 1-64 characters,
// lowercase letters/digits/hyphens only, no leading/trailing/consecutive hyphens.
func ValidateName(name string) error {
	if name == "" {
		return errors.New("name is empty")
	}
	if len(name) > 64 {
		return fmt.Errorf("name %q exceeds 64 characters", name)
	}
	if !namePattern.MatchString(name) {
		return fmt.Errorf("name %q must contain only lowercase letters, digits, and hyphens, with no leading, trailing, or consecutive hyphens", name)
	}
	return nil
}

// ParseName extracts and validates the `name` field from SKILL.md frontmatter.
func ParseName(content string) (string, error) {
	fm, err := extractFrontmatter(content)
	if err != nil {
		return "", err
	}
	var meta struct {
		Name string `yaml:"name"`
	}
	if err := yaml.Unmarshal([]byte(fm), &meta); err != nil {
		return "", fmt.Errorf("invalid YAML frontmatter: %w", err)
	}
	if meta.Name == "" {
		return "", fmt.Errorf("SKILL.md frontmatter has no `name` field (see %s)", SpecURL)
	}
	if err := ValidateName(meta.Name); err != nil {
		return "", fmt.Errorf("%w (see %s)", err, SpecURL)
	}
	return meta.Name, nil
}

func extractFrontmatter(content string) (string, error) {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	normalized = strings.TrimPrefix(normalized, "\uFEFF")
	lines := strings.Split(normalized, "\n")
	if len(lines) == 0 || strings.TrimRight(lines[0], " \t") != "---" {
		return "", fmt.Errorf("SKILL.md has no YAML frontmatter (see %s)", SpecURL)
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], " \t") == "---" {
			return strings.Join(lines[1:i], "\n"), nil
		}
	}
	return "", fmt.Errorf("SKILL.md frontmatter is not closed with `---` (see %s)", SpecURL)
}
