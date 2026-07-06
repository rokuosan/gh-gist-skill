package gist

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseID(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{input: "fd287c3133457c4fd8f5601d34aa817d", want: "fd287c3133457c4fd8f5601d34aa817d"},
		{input: "https://gist.github.com/k16shikano/fd287c3133457c4fd8f5601d34aa817d", want: "fd287c3133457c4fd8f5601d34aa817d"},
		{input: "https://gist.github.com/fd287c3133457c4fd8f5601d34aa817d", want: "fd287c3133457c4fd8f5601d34aa817d"},
		{input: "https://gist.github.com/fd287c3133457c4fd8f5601d34aa817d.git", want: "fd287c3133457c4fd8f5601d34aa817d"},
		{input: "https://gist.github.com/k16shikano/fd287c3133457c4fd8f5601d34aa817d#file-skill-md", want: "fd287c3133457c4fd8f5601d34aa817d"},
		{input: "https://gist.github.com/k16shikano/fd287c3133457c4fd8f5601d34aa817d/", want: "fd287c3133457c4fd8f5601d34aa817d"},
		{input: "  fd287c3133457c4fd8f5601d34aa817d  ", want: "fd287c3133457c4fd8f5601d34aa817d"},
		{input: "gist.github.com/k16shikano/fd287c3133457c4fd8f5601d34aa817d", want: "fd287c3133457c4fd8f5601d34aa817d"},
		{input: "https://example.com/foo/bar", wantErr: true},
		{input: "https://evil.example/gist.github.com/user/fd287c3133457c4fd8f5601d34aa817d", wantErr: true},
		{input: "https://gist.github.com.evil.example/user/fd287c3133457c4fd8f5601d34aa817d", wantErr: true},
		{input: "https://evil.example/fd287c3133457c4fd8f5601d34aa817d?x=gist.github.com", wantErr: true},
		{input: "not a gist", wantErr: true},
		{input: "", wantErr: true},
		{input: "https://gist.github.com/", wantErr: true},
	}
	for _, tt := range tests {
		got, err := ParseID(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseID(%q) = %q, want error", tt.input, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseID(%q) unexpected error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseID(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFileContent(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("full content"))
	}))
	defer srv.Close()

	inline := File{Filename: "SKILL.md", Content: "inline", Truncated: false, RawURL: srv.URL}
	got, err := FileContent(srv.Client(), inline)
	if err != nil || got != "inline" {
		t.Errorf("inline: got (%q, %v), want (\"inline\", nil)", got, err)
	}

	truncated := File{Filename: "SKILL.md", Content: "partial", Truncated: true, RawURL: srv.URL}
	got, err = FileContent(srv.Client(), truncated)
	if err != nil || got != "full content" {
		t.Errorf("truncated: got (%q, %v), want (\"full content\", nil)", got, err)
	}

	for _, raw := range []string{"", "http://example.com/raw", "::bad::"} {
		bad := File{Filename: "SKILL.md", Content: "partial", Truncated: true, RawURL: raw}
		if _, err := FileContent(srv.Client(), bad); err == nil {
			t.Errorf("raw_url %q: want error, got nil", raw)
		}
	}
}
