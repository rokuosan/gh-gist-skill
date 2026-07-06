// Package gist resolves Gist URLs/IDs and fetches gist metadata via the
// GitHub API using the gh CLI's authentication.
package gist

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// File is a single file in a gist as returned by GET /gists/<id>.
type File struct {
	Filename  string `json:"filename"`
	Content   string `json:"content"`
	Truncated bool   `json:"truncated"`
	RawURL    string `json:"raw_url"`
}

// Gist is the subset of the API response this tool cares about.
type Gist struct {
	ID    string          `json:"id"`
	Files map[string]File `json:"files"`
}

var idPattern = regexp.MustCompile(`^[0-9a-f]{1,64}$`)

// ParseID extracts a gist ID from a gist URL or a bare ID.
// Accepted forms:
//   - fd287c3133457c4fd8f5601d34aa817d
//   - https://gist.github.com/<user>/<id>
//   - https://gist.github.com/<id>
//   - with optional .git suffix, trailing slash, query, or #file- fragment
func ParseID(input string) (string, error) {
	s := strings.TrimSpace(input)
	if i := strings.IndexAny(s, "#?"); i >= 0 {
		s = s[:i]
	}
	s = strings.TrimSuffix(s, "/")

	if strings.Contains(s, "/") {
		withScheme := s
		if !strings.Contains(withScheme, "://") {
			withScheme = "https://" + withScheme
		}
		u, err := url.Parse(withScheme)
		if err != nil || u.Hostname() != "gist.github.com" {
			return "", fmt.Errorf("not a gist.github.com URL: %s", input)
		}
		s = u.Path[strings.LastIndex(u.Path, "/")+1:]
	}
	s = strings.TrimSuffix(s, ".git")

	if !idPattern.MatchString(s) {
		return "", fmt.Errorf("could not extract a gist ID from %q", input)
	}
	return s, nil
}

// RESTClient is the part of go-gh's REST client this package uses.
type RESTClient interface {
	Get(path string, response any) error
}

// Fetch retrieves gist metadata (including inline file contents) by ID.
func Fetch(client RESTClient, id string) (*Gist, error) {
	var g Gist
	if err := client.Get("gists/"+id, &g); err != nil {
		return nil, fmt.Errorf("failed to fetch gist %s: %w", id, err)
	}
	return &g, nil
}

// FileContent returns the full content of f, fetching raw_url when the API
// response was truncated.
func FileContent(httpClient *http.Client, f File) (string, error) {
	if !f.Truncated {
		return f.Content, nil
	}
	u, err := url.Parse(f.RawURL)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return "", fmt.Errorf("gist file %s is truncated but has no usable raw_url (%q)", f.Filename, f.RawURL)
	}
	resp, err := httpClient.Get(f.RawURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch %s: %w", f.Filename, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch %s: HTTP %d", f.Filename, resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", f.Filename, err)
	}
	return string(b), nil
}
