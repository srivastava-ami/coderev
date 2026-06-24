package report

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

// GenerateSARIF writes a SARIF 2.1.0 report to outputPath.
// GitHub Code Scanning ingests this via the upload-sarif action.
func GenerateSARIF(r Report, outputPath, repoURI string) error {
	log := buildSARIFLog(r, repoURI)
	data, err := json.MarshalIndent(log, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling SARIF: %w", err)
	}
	return os.WriteFile(outputPath, data, 0o644)
}

type sarifLog struct {
	Schema  string      `json:"$schema"`
	Version string      `json:"version"`
	Runs    []sarifRun  `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string      `json:"name"`
	Version        string      `json:"version"`
	InformationURI string      `json:"informationUri"`
	Rules          []sarifRule `json:"rules"`
}

type sarifRule struct {
	ID               string              `json:"id"`
	Name             string              `json:"name"`
	ShortDescription sarifMessage        `json:"shortDescription"`
	Properties       sarifRuleProperties `json:"properties,omitempty"`
}

type sarifRuleProperties struct {
	Tags []string `json:"tags,omitempty"`
}

type sarifResult struct {
	RuleID    string          `json:"ruleId"`
	Level     string          `json:"level"`
	Message   sarifMessage    `json:"message"`
	Locations []sarifLocation `json:"locations"`
}

type sarifMessage struct {
	Text string `json:"text"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysical `json:"physicalLocation"`
}

type sarifPhysical struct {
	ArtifactLocation sarifArtifact `json:"artifactLocation"`
	Region           sarifRegion   `json:"region"`
}

type sarifArtifact struct {
	URI       string `json:"uri"`
	URIBaseID string `json:"uriBaseId,omitempty"`
}

type sarifRegion struct {
	StartLine int `json:"startLine"`
}

func buildSARIFLog(r Report, repoURI string) sarifLog {
	rules := buildSARIFRules(r.Findings)
	results := buildSARIFResults(r.Findings, r.Meta.RepoPath)
	return sarifLog{
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		Version: "2.1.0",
		Runs: []sarifRun{{
			Tool: sarifTool{Driver: sarifDriver{
				Name:           "coderev",
				Version:        "dev",
				InformationURI: "https://github.com/srivastava-ami/coderev",
				Rules:          rules,
			}},
			Results: results,
		}},
	}
}

func buildSARIFRules(findings []analysis.Finding) []sarifRule {
	seen := map[string]bool{}
	var rules []sarifRule
	for _, f := range findings {
		if seen[f.Rule] {
			continue
		}
		seen[f.Rule] = true
		meta := analysis.RuleRegistry[f.Rule]
		rules = append(rules, sarifRule{
			ID:               f.Rule,
			Name:             ruleIDToName(f.Rule),
			ShortDescription: sarifMessage{Text: f.Message},
			Properties:       sarifRuleProperties{Tags: meta.Tags},
		})
	}
	return rules
}

func buildSARIFResults(findings []analysis.Finding, repoPath string) []sarifResult {
	results := make([]sarifResult, 0, len(findings))
	for _, f := range findings {
		uri := toRelativeURI(f.File, repoPath)
		results = append(results, sarifResult{
			RuleID:  f.Rule,
			Level:   sarifLevel(f.Severity),
			Message: sarifMessage{Text: f.Message},
			Locations: []sarifLocation{{
				PhysicalLocation: sarifPhysical{
					ArtifactLocation: sarifArtifact{URI: uri, URIBaseID: "%SRCROOT%"},
					Region:           sarifRegion{StartLine: sarifMax(1, f.Line)},
				},
			}},
		})
	}
	return results
}

func sarifLevel(s analysis.Severity) string {
	switch s {
	case analysis.SeverityBlocker:
		return "error"
	case analysis.SeverityMajor:
		return "warning"
	case analysis.SeverityAdvisory:
		return "note"
	default:
		return "none"
	}
}

func toRelativeURI(filePath, repoPath string) string {
	rel := strings.TrimPrefix(filePath, repoPath)
	rel = strings.TrimPrefix(rel, "/")
	return rel
}

func ruleIDToName(id string) string {
	parts := strings.Split(id, ".")
	var out []string
	for _, p := range parts {
		if len(p) > 0 {
			out = append(out, strings.ToUpper(p[:1])+p[1:])
		}
	}
	return strings.Join(out, "")
}

func sarifMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}
