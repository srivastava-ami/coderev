package ghpr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/srivastava-ami/coderev/internal/analysis"
	"github.com/srivastava-ami/coderev/internal/report"
)

// interCommentDelay paces inline-comment posts to stay under GitHub's inline
// comment rate limit.
const interCommentDelay = 150 * time.Millisecond

// AnnotateRequest bundles the parameters for Annotate.
type AnnotateRequest struct {
	Report   report.Report
	RepoSlug string // "owner/repo"
	PRNumber int
	Target   string // repo root path — used to make file paths relative
}

// Annotate posts inline review comments for every blocker/major finding on a
// GitHub PR using the `gh` CLI. It requires GH_TOKEN in the environment.
func Annotate(req AnnotateRequest) error {
	if err := checkGHAvailable(); err != nil {
		return err
	}
	sha, err := headSHA(req.Target)
	if err != nil {
		return fmt.Errorf("ghpr: resolving HEAD SHA: %w", err)
	}
	// Fetch which lines are actually in the PR diff — the GitHub API rejects
	// comments on lines outside the diff hunk with 422 "could not be resolved".
	diffed, err := fetchDiffLines(req.RepoSlug, req.PRNumber)
	if err != nil {
		return fmt.Errorf("ghpr: fetching PR diff lines: %w", err)
	}
	comments := buildComments(req.Report.Findings, req.Target, diffed)
	if len(comments) == 0 {
		fmt.Fprintf(os.Stderr, "ghpr: no findings matched lines in PR diff — skipping inline annotation\n")
		return nil
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
	for i, c := range comments {
		if i > 0 {
			time.Sleep(interCommentDelay)
		}
		if err := postComment(prPost{repoSlug: p.repoSlug, prNumber: p.prNumber, sha: p.sha, comment: c}); err != nil {
			errs = append(errs, err.Error())
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("ghpr: %d comment(s) failed: %s", len(errs), strings.Join(errs, "; "))
	}
	fmt.Fprintf(os.Stderr, "ghpr: posted %d inline comment(s) on PR #%d\n", len(comments), p.prNumber)
	return nil
}

type prComment struct {
	path string
	line int
	body string
}

// buildComments returns inline comments for blockers/majors whose line falls
// within a diff hunk. Findings on lines outside the diff are silently skipped —
// they appear in the written report but cannot be annotated inline.
func buildComments(findings []analysis.Finding, target string, diffed map[string]map[int]bool) []prComment {
	var out []prComment
	for _, f := range findings {
		if f.Severity != analysis.SeverityBlocker && f.Severity != analysis.SeverityMajor {
			continue
		}
		rel, err := filepath.Rel(target, f.File)
		if err != nil {
			rel = f.File
		}
		rel = filepath.ToSlash(rel)
		if fileLines, ok := diffed[rel]; !ok || !fileLines[f.Line] {
			continue
		}
		body := fmt.Sprintf("**coderev [%s]** `%s`\n\n%s", strings.ToUpper(string(f.Severity)), f.Rule, f.Message)
		if f.Remediation != "" {
			body += "\n\n> **Fix:** " + f.Remediation
		}
		out = append(out, prComment{path: rel, line: f.Line, body: body})
	}
	return out
}

// fetchDiffLines returns map[filepath]map[line]bool for every line that appears
// on the RIGHT (new) side of the PR diff — context lines and added lines both
// count; removed lines are not in the new file and cannot be annotated.
func fetchDiffLines(repoSlug string, prNumber int) (map[string]map[int]bool, error) {
	endpoint := fmt.Sprintf("repos/%s/pulls/%d/files", repoSlug, prNumber)
	cmd := exec.Command("gh", "api", "--paginate", endpoint)
	raw, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gh api %s: %w", endpoint, err)
	}
	var files []struct {
		Filename string `json:"filename"`
		Patch    string `json:"patch"`
	}
	if err := json.Unmarshal(raw, &files); err != nil {
		return nil, fmt.Errorf("parsing PR files response: %w", err)
	}
	result := make(map[string]map[int]bool, len(files))
	for _, f := range files {
		if lines := parsePatchLines(f.Patch); len(lines) > 0 {
			result[f.Filename] = lines
		}
	}
	return result, nil
}

// parsePatchLines extracts new-file line numbers from a unified diff patch string.
// Every context line and added line (i.e. lines visible on the RIGHT side) is included.
func parsePatchLines(patch string) map[int]bool {
	lines := make(map[int]bool)
	if patch == "" {
		return lines
	}
	currentLine := 0
	for _, line := range strings.Split(patch, "\n") {
		if strings.HasPrefix(line, "@@") {
			// "@@ -old,count +new,count @@" — extract new-file start line
			idx := strings.Index(line, "+")
			if idx < 0 {
				continue
			}
			rest := line[idx+1:]
			if i := strings.IndexAny(rest, ", @"); i >= 0 {
				rest = rest[:i]
			}
			n, err := strconv.Atoi(rest)
			if err != nil {
				continue
			}
			currentLine = n - 1
			continue
		}
		if strings.HasPrefix(line, "-") {
			continue // removed line — not in new file, cannot be annotated
		}
		currentLine++
		lines[currentLine] = true
	}
	return lines
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

// PostInlineComment posts a single body as an inline review comment on the first
// file of the PR diff, so it appears inline in the Files Changed tab.
func PostInlineComment(repoSlug string, prNumber int, target, body string) error {
	if err := checkGHAvailable(); err != nil {
		return err
	}
	sha, err := headSHA(target)
	if err != nil {
		return fmt.Errorf("ghpr: resolving HEAD SHA: %w", err)
	}
	files, err := listDiffFiles(repoSlug, prNumber)
	if err != nil || len(files) == 0 {
		return fmt.Errorf("ghpr: no files in PR diff to attach inline comment")
	}
	rel := filepath.ToSlash(files[0])
	if err := postComment(prPost{repoSlug: repoSlug, prNumber: prNumber, sha: sha, comment: prComment{path: rel, line: 1, body: body}}); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "ghpr: posted AI review as inline comment on %s in PR #%d\n", rel, prNumber)
	return nil
}

// listDiffFiles returns the filename of every file changed in the PR.
func listDiffFiles(repoSlug string, prNumber int) ([]string, error) {
	endpoint := fmt.Sprintf("repos/%s/pulls/%d/files", repoSlug, prNumber)
	cmd := exec.Command("gh", "api", "--paginate", "--jq", ".[].filename", endpoint)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gh api %s: %w", endpoint, err)
	}
	var files []string
	for _, f := range strings.Fields(string(out)) {
		f = strings.TrimSpace(f)
		if f != "" {
			files = append(files, f)
		}
	}
	return files, nil
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
