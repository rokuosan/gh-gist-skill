package skill

import (
	"strings"
	"testing"
)

func TestParseName(t *testing.T) {
	tests := []struct {
		desc    string
		content string
		want    string
		wantErr bool
	}{
		{
			desc:    "basic",
			content: "---\nname: japanese-tech-writing\ndescription: writing\n---\n\n# Body\n",
			want:    "japanese-tech-writing",
		},
		{
			desc:    "CRLF",
			content: "---\r\nname: my-skill\r\n---\r\nbody\r\n",
			want:    "my-skill",
		},
		{
			desc:    "BOM",
			content: "\uFEFF---\nname: my-skill\n---\n",
			want:    "my-skill",
		},
		{
			desc:    "no frontmatter",
			content: "# Just a readme\n",
			wantErr: true,
		},
		{
			desc:    "unclosed frontmatter",
			content: "---\nname: my-skill\n",
			wantErr: true,
		},
		{
			desc:    "no name field",
			content: "---\ndescription: nope\n---\n",
			wantErr: true,
		},
		{
			desc:    "invalid yaml",
			content: "---\nname: [\n---\n",
			wantErr: true,
		},
		{
			desc:    "uppercase name",
			content: "---\nname: MySkill\n---\n",
			wantErr: true,
		},
		{
			desc:    "leading hyphen",
			content: "---\nname: -skill\n---\n",
			wantErr: true,
		},
		{
			desc:    "consecutive hyphens",
			content: "---\nname: my--skill\n---\n",
			wantErr: true,
		},
		{
			desc:    "too long",
			content: "---\nname: " + strings.Repeat("a", 65) + "\n---\n",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		got, err := ParseName(tt.content)
		if tt.wantErr {
			if err == nil {
				t.Errorf("%s: ParseName() = %q, want error", tt.desc, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s: unexpected error: %v", tt.desc, err)
			continue
		}
		if got != tt.want {
			t.Errorf("%s: ParseName() = %q, want %q", tt.desc, got, tt.want)
		}
	}
}

func TestValidateName(t *testing.T) {
	valid := []string{"a", "a1", "my-skill", "a-b-c", strings.Repeat("a", 64)}
	for _, n := range valid {
		if err := ValidateName(n); err != nil {
			t.Errorf("ValidateName(%q) unexpected error: %v", n, err)
		}
	}
	invalid := []string{"", "A", "my_skill", "-a", "a-", "a--b", "日本語", strings.Repeat("a", 65)}
	for _, n := range invalid {
		if err := ValidateName(n); err == nil {
			t.Errorf("ValidateName(%q) = nil, want error", n)
		}
	}
}
