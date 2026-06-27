package github

import (
	"context"
	"fmt"
	"strings"
)

// commentMarker is a hidden HTML comment stamped into every comment body coderev
// owns. Listing the PR's issue comments and matching this marker is how
// UpsertComment finds the comment it posted previously, so it can edit it in
// place instead of stacking a new one on every run.
const commentMarker = "<!-- coderev-comment -->"

// issueComment is the slice of GitHub's issue-comment payload we care about.
type issueComment struct {
	ID   int64  `json:"id"`
	Body string `json:"body"`
}

// UpsertComment posts coderev's review summary to the pull request as a single,
// stable comment. On the first call it creates the comment; on later calls it
// finds the existing marked comment and PATCHes it, so a PR never accumulates
// duplicate coderev comments. repo is "owner/name" and pr is the PR number.
func (c *Client) UpsertComment(repo string, pr int, body string) error {
	return c.UpsertCommentContext(context.Background(), repo, pr, body)
}

// UpsertCommentContext is UpsertComment with an explicit context for
// cancellation and timeouts.
func (c *Client) UpsertCommentContext(ctx context.Context, repo string, pr int, body string) error {
	owner, name, err := splitRepo(repo)
	if err != nil {
		return err
	}
	marked := ensureMarker(body)

	existing, err := c.findMarkedComment(ctx, owner, name, pr)
	if err != nil {
		return err
	}

	payload := map[string]string{"body": marked}
	if existing != nil {
		path := fmt.Sprintf("/repos/%s/%s/issues/comments/%d", owner, name, existing.ID)
		return c.patch(ctx, path, payload, nil)
	}
	path := fmt.Sprintf("/repos/%s/%s/issues/%d/comments", owner, name, pr)
	return c.post(ctx, path, payload, nil)
}

// findMarkedComment returns the first issue comment on the PR whose body carries
// the coderev marker, or nil when none exists yet. PR comments live on the
// issues endpoint because every pull request is also an issue.
func (c *Client) findMarkedComment(ctx context.Context, owner, name string, pr int) (*issueComment, error) {
	path := fmt.Sprintf("/repos/%s/%s/issues/%d/comments?per_page=100", owner, name, pr)
	var comments []issueComment
	if err := c.get(ctx, path, &comments); err != nil {
		return nil, err
	}
	for i := range comments {
		if strings.Contains(comments[i].Body, commentMarker) {
			return &comments[i], nil
		}
	}
	return nil, nil
}

// ensureMarker guarantees the body carries the hidden marker so the next run can
// recognise and update it.
func ensureMarker(body string) string {
	if strings.Contains(body, commentMarker) {
		return body
	}
	return body + "\n\n" + commentMarker
}

// splitRepo parses an "owner/name" slug into its two parts.
func splitRepo(repo string) (owner, name string, err error) {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("github: invalid repo %q, want \"owner/name\"", repo)
	}
	return parts[0], parts[1], nil
}
