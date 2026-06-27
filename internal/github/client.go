// Package github posts inline PR review comments straight to the GitHub REST
// API over net/http, authenticated with a token (GITHUB_TOKEN). It deliberately
// never shells out to the gh CLI: the package is self-contained so it runs in
// any CI image without that binary installed.
package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Client is a minimal authenticated REST client. The zero value is not usable;
// construct one with New or NewWithToken.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// New returns a Client reading its bearer token from the GITHUB_TOKEN
// environment variable. It errors when the variable is unset so callers fail
// loudly rather than posting unauthenticated requests.
func New(baseURL string) (*Client, error) {
	token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	if token == "" {
		return nil, fmt.Errorf("github: GITHUB_TOKEN is not set")
	}
	return NewWithToken(baseURL, token), nil
}

// NewWithToken returns a Client using the given token. Useful in tests and when
// the token comes from somewhere other than the environment.
func NewWithToken(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// get issues an authenticated GET and decodes the JSON body into out.
func (c *Client) get(ctx context.Context, path string, out any) error {
	return c.do(ctx, http.MethodGet, path, nil, out)
}

// post issues an authenticated POST with a JSON body and decodes the response
// into out (which may be nil to discard it).
func (c *Client) post(ctx context.Context, path string, body, out any) error {
	return c.do(ctx, http.MethodPost, path, body, out)
}

// patch issues an authenticated PATCH with a JSON body and decodes the response
// into out (which may be nil to discard it).
func (c *Client) patch(ctx context.Context, path string, body, out any) error {
	return c.do(ctx, http.MethodPatch, path, body, out)
}

// do performs the request, applying the auth and content headers GitHub
// expects, and treats any non-2xx status as an error carrying the response body.
func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("github: encode %s %s: %w", method, path, err)
		}
		reader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("github: build %s %s: %w", method, path, err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("github: %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("github: read %s %s response: %w", method, path, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("github: %s %s: unexpected status %d: %s", method, path, resp.StatusCode, strings.TrimSpace(string(payload)))
	}

	if out == nil || len(payload) == 0 {
		return nil
	}
	if err := json.Unmarshal(payload, out); err != nil {
		return fmt.Errorf("github: decode %s %s response: %w", method, path, err)
	}
	return nil
}
