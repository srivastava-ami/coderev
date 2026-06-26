package treesitter

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	sitter "github.com/smacker/go-tree-sitter"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

const adapterName = "treesitter"

// maxConcurrentFileAnalyses bounds how many files are parsed and walked in
// parallel, capping memory and CPU use on large repositories.
const maxConcurrentFileAnalyses = 8

// Adapter parses source files with tree-sitter and runs all structural checks.
// It is the primary analysis engine — no external binaries required.
type Adapter struct {
	stds analysis.Standards
}

func New(stds analysis.Standards) *Adapter {
	return &Adapter{stds: stds}
}

func (a *Adapter) Name() string      { return adapterName }
func (a *Adapter) IsAvailable() bool { return true } // pure Go, always available
func (a *Adapter) Capabilities() []string {
	return []string{
		"complexity.*",
		"file_structure.file_length",
		"file_structure.class_length",
		"type_safety.no_any",
		"type_safety.null_safety",
		"hardcoding.magic_numbers",
		"hardcoding.urls_and_paths",
		"documentation.comment_quality",
		"documentation.no_comment_tombstones",
		"documentation.todo_format",
		"observability.logging",
		"stability.error_handling",
	}
}

type analyseResult struct {
	findings []analysis.Finding
	err      error
}

func (a *Adapter) Run(ctx context.Context, req analysis.RunRequest) ([]analysis.Finding, error) {
	results := make(chan analyseResult, len(req.Files))
	sem := make(chan struct{}, maxConcurrentFileAnalyses)

	var wg sync.WaitGroup
	for _, fi := range req.Files {
		fi := fi
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			a.runFile(ctx, fi, results)
		}()
	}
	go func() {
		wg.Wait()
		close(results)
	}()

	findings, err := collectAnalyseResults(results)
	dupFindings := DetectDuplication(req.Files)
	magicFindings := checkMagicNumbers(req.Files, a.stds.Hardcoding.MagicNumbers.Exceptions)
	findings = append(findings, dupFindings...)
	findings = append(findings, magicFindings...)
	return findings, err
}

func (a *Adapter) runFile(ctx context.Context, fi analysis.FileInfo, out chan<- analyseResult) {
	select {
	case <-ctx.Done():
		out <- analyseResult{err: ctx.Err()}
		return
	default:
	}
	findings, err := a.analyseFile(fi)
	out <- analyseResult{findings: findings, err: err}
}

func collectAnalyseResults(results <-chan analyseResult) ([]analysis.Finding, error) {
	var all []analysis.Finding
	var errCount int
	for r := range results {
		if r.err != nil {
			errCount++
			continue
		}
		all = append(all, r.findings...)
	}
	if errCount > 0 {
		return all, fmt.Errorf("treesitter: %d files failed to parse", errCount)
	}
	return all, nil
}

func (a *Adapter) analyseFile(fi analysis.FileInfo) ([]analysis.Finding, error) {
	def := langDefForFile(fi)
	if def == nil {
		return nil, nil // unsupported language — skip silently
	}

	parser := sitter.NewParser()
	parser.SetLanguage(def.GetLanguage())

	tree, err := parser.ParseCtx(context.Background(), nil, fi.Content)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", fi.Path, err)
	}
	defer tree.Close()

	walker := newFileWalker(def, fi, a.stds)
	return walker.walk(tree.RootNode()), nil
}

func langDefForFile(fi analysis.FileInfo) *LangDef {
	if filepath.Ext(fi.Path) == ".tsx" {
		return tsxDef
	}
	return defFor(fi.Language)
}
