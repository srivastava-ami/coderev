package ghpr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/srivastava-ami/coderev/internal/analysis"
	"github.com/srivastava-ami/coderev/internal/report"
)

// AnnotateRequest bundles the parameters for Annotate.
type AnnotateRequest struct {
	Report    report.Report
	RepoSlug  string // "owner/repo"
	PRNumber  int
	Target    string // repo root path — used to make file paths relative
}

// Annotate posts inline review comments for every blocker/major finding on a
// GitHub PR using the `gh` CLI. It requires GH_TOKEN in the environment.
func Annotate(req AnnotateRequest) error {
	if err := checkGHAvailable(); err != nil {
		return err
	}
	comments := buildComments(req.Report.Findings, req.Target)
	if len(comments) == 0 {
		return nil
	}
	sha, err := headSHA(req.Target)
	if err != nil {
		return fmt.Errorf("ghpr: resolving HEAD SHA: %w", err)
	}
	poster := &commentPoster{repoSlug: req.RepoSlug, prNumber: req.PRNumber, sha: sha}
	return poster.postAll(comments)
}

type commentPoster struct {
	repoSlug string
	prNumber int
	sha      string
}

func (p *commentPoster) postAll(comments []prComment) error {
	var errs []string
	for _, c := range comments {
		if err := postComment(prPost{repoSlug: p.repoSlug, prNumber: p.prNumber, sha: p.sha, comment: c}); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("ghpr: %d comment(s) failed: %s", len(errs), strings.Join(errs, "; "))
	}
	fmt.Printf("ghpr: posted %d inline comment(s) on PR #%d\n", len(comments), p.prNumber)
	return nil
}

type prComment struct {
	path string
	line int
	body string
}

func buildComments(findings []analysis.Finding, target string) []prComment {
	var out []prComment
	for _, f := range findings {
		if f.Severity != analysis.SeverityBlocker && f.Severity != analysis.SeverityMajor {
			continue
		}
		rel, err := filepath.Rel(target, f.File)
		if err != nil {
			rel = f.File
		}
		body := fmt.Sprintf("**coderev [%s]** `%s`\n\n%s", strings.ToUpper(string(f.Severity)), f.Rule, f.Message)
		if f.Remediation != "" {
			body += "\n\n> **Fix:** " + f.Remediation
		}
		out = append(out, prComment{path: rel, line: f.Line, body: body})
	}
	return out
}

type ghCommentPayload struct {
	Body     string `json:"body"`
	CommitID string `json:"commit_id"`
	Path     string `json:"path"`
	Line     int    `json:"line"`
	Side     string `json:"side"`
}

type prPost struct {
	repoSlug string
	prNumber int
	sha      string
	comment  prComment
}

func postComment(p prPost) error {
	payload := ghCommentPayload{
		Body:     p.comment.body,
		CommitID: p.sha,
		Path:     p.comment.path,
		Line:     p.comment.line,
		Side:     "RIGHT",
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	endpoint := fmt.Sprintf("repos/%s/pulls/%d/comments", p.repoSlug, p.prNumber)
	cmd := exec.Command("gh", "api", "--method", "POST", endpoint, "--input", "-")
	cmd.Stdin = bytes.NewReader(data)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gh api %s: %s", endpoint, strings.TrimSpace(string(out)))
	}
	return nil
}

func headSHA(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func checkGHAvailable() error {
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("ghpr: gh CLI not found — install from https://cli.github.com")
	}
	return nil
}
