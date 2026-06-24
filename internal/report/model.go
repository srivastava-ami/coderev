package report

import (
	"path/filepath"
	"sort"
	"time"

	"github.com/srivastava-ami/coderev/internal/analysis"
	"github.com/srivastava-ami/coderev/internal/architecture"
	"github.com/srivastava-ami/coderev/internal/baseline"
	"github.com/srivastava-ami/coderev/internal/config"
)

// Report is the complete data model serialised into the HTML report.
type Report struct {
	Meta         Meta
	Summary      Summary
	Architecture architecture.Summary
	Pillars      []PillarResult
	Files        []FileResult
	Findings     []analysis.Finding
	Warnings     []analysis.AdapterWarning
	Exceptions   []config.Exception
	Generated    time.Time
}

type Meta struct {
	RepoName         string
	RepoPath         string
	StandardsFile    string
	StandardsVersion string
	Generated        string
}

type Summary struct {
	TotalFiles    int
	TotalFindings int
	BySeverity    map[string]int
	ByPillar      map[string]int
	OverallStatus string           // "PASS" | "FAIL"
	Delta         *baseline.Delta  // nil when no baseline exists
}

type PillarResult struct {
	Name     string
	Status   string
	Rating   string // A–E reliability grade
	Findings []analysis.Finding
	Score    float64
}

type FileResult struct {
	Path      string
	Language  string
	Lines     int
	Findings  []analysis.Finding
	HeatScore float64
}

// BuildRequest bundles all inputs for Build so callers don't need positional args.
type BuildRequest struct {
	Target    string
	Standards config.Standards
	StdFile   string
	Files     []analysis.FileInfo
	Findings  []analysis.Finding
	Warnings  []analysis.AdapterWarning
	Arch      architecture.Summary
	Delta     *baseline.Delta // optional: trend vs saved baseline
}

// Build constructs a Report from raw analysis output.
func Build(req BuildRequest) Report {
	return Report{
		Meta:         buildMeta(req.Target, req.StdFile, req.Standards),
		Summary:      buildSummary(req.Files, req.Findings, req.Delta),
		Architecture: req.Arch,
		Pillars:      buildPillars(req.Findings),
		Files:        buildFileResults(req.Files, req.Findings),
		Findings:     req.Findings,
		Warnings:     req.Warnings,
		Exceptions:   req.Standards.Exceptions,
		Generated:    time.Now(),
	}
}

func buildMeta(target, stdFile string, stds config.Standards) Meta {
	return Meta{
		RepoName:         filepath.Base(target),
		RepoPath:         target,
		StandardsFile:    stdFile,
		StandardsVersion: stds.Meta.Version,
		Generated:        time.Now().Format(time.RFC1123),
	}
}

func buildSummary(files []analysis.FileInfo, findings []analysis.Finding, delta *baseline.Delta) Summary {
	bySeverity := map[string]int{"blocker": 0, "major": 0, "advisory": 0, "info": 0}
	byPillar := map[string]int{}
	for _, f := range findings {
		bySeverity[string(f.Severity)]++
		byPillar[f.Pillar]++
	}
	status := "PASS"
	if bySeverity["blocker"] > 0 || bySeverity["major"] > 0 {
		status = "FAIL"
	}
	return Summary{
		TotalFiles:    len(files),
		TotalFindings: len(findings),
		BySeverity:    bySeverity,
		ByPillar:      byPillar,
		OverallStatus: status,
		Delta:         delta,
	}
}

func pillarRating(blockers, majors int) string {
	switch {
	case blockers >= 5:
		return "E"
	case blockers >= 3 || (blockers >= 1 && majors >= 5):
		return "D"
	case blockers >= 1 || majors >= 5:
		return "C"
	case majors >= 2:
		return "B"
	default:
		return "A"
	}
}

func countBlockers(findings []analysis.Finding) int {
	n := 0
	for _, f := range findings {
		if f.Severity == analysis.SeverityBlocker {
			n++
		}
	}
	return n
}

func buildPillars(findings []analysis.Finding) []PillarResult {
	pillarMap := groupByPillar(findings)
	pillars := make([]PillarResult, 0, len(pillarMap))
	for name, fs := range pillarMap {
		pillars = append(pillars, buildPillar(name, fs))
	}
	sort.Slice(pillars, func(i, j int) bool {
		if pillars[i].Status != pillars[j].Status {
			return pillars[i].Status < pillars[j].Status
		}
		return pillars[i].Name < pillars[j].Name
	})
	return pillars
}

func groupByPillar(findings []analysis.Finding) map[string][]analysis.Finding {
	m := map[string][]analysis.Finding{}
	for _, f := range findings {
		m[f.Pillar] = append(m[f.Pillar], f)
	}
	return m
}

func buildPillar(name string, fs []analysis.Finding) PillarResult {
	blockers, majors := 0, 0
	for _, f := range fs {
		switch f.Severity {
		case analysis.SeverityBlocker:
			blockers++
		case analysis.SeverityMajor:
			majors++
		}
	}
	status := "WARN"
	if blockers > 0 {
		status = "FAIL"
	}
	score := 0.0
	if len(fs) > 0 {
		score = 1.0 - float64(blockers)/float64(len(fs))
	}
	return PillarResult{Name: name, Status: status, Rating: pillarRating(blockers, majors), Findings: fs, Score: score}
}

func buildFileResults(files []analysis.FileInfo, findings []analysis.Finding) []FileResult {
	byFile := map[string][]analysis.Finding{}
	for _, f := range findings {
		byFile[f.File] = append(byFile[f.File], f)
	}

	maxFindings := 1
	for _, fs := range byFile {
		if len(fs) > maxFindings {
			maxFindings = len(fs)
		}
	}

	results := make([]FileResult, 0, len(files))
	for _, fi := range files {
		fs := byFile[fi.Path]
		results = append(results, FileResult{
			Path:      fi.Path,
			Language:  string(fi.Language),
			Lines:     fi.Lines,
			Findings:  fs,
			HeatScore: float64(len(fs)) / float64(maxFindings),
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return len(results[i].Findings) > len(results[j].Findings)
	})
	return results
}
