package treesitter

import (
	"regexp"
	"strings"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

func checkMagicNumbers(files []analysis.FileInfo) []analysis.Finding {
	re := regexp.MustCompile(`(?:^|[^a-zA-Z0-9_.])([0-9]+(?:\.[0-9]+)?)(?:[^a-zA-Z0-9_.]|$)`)

	var findings []analysis.Finding
	for _, fi := range files {
		if isTestFile(fi.Path) {
			continue
		}
		if strings.HasSuffix(fi.Path, ".json") || strings.HasSuffix(fi.Path, ".toml") || strings.HasSuffix(fi.Path, ".yaml") || strings.HasSuffix(fi.Path, ".yml") {
			continue
		}
		lines := strings.Split(string(fi.Content), "\n")
		findings = append(findings, scanLinesForMagicNumbers(lines, fi.Path, re)...)
	}
	return findings
}

func scanLinesForMagicNumbers(lines []string, path string, re *regexp.Regexp) []analysis.Finding {
	var findings []analysis.Finding
	for i, line := range lines {
		lineNum := i + 1
		trimmed := strings.TrimSpace(line)
		if isCommentLine(trimmed, path) || skipLine(trimmed) {
			continue
		}
		findings = append(findings, emitMagicMatches(line, trimmed, path, lineNum, re)...)
	}
	return findings
}

func emitMagicMatches(line, trimmed, path string, lineNum int, re *regexp.Regexp) []analysis.Finding {
	var findings []analysis.Finding
	for _, m := range re.FindAllStringSubmatch(line, -1) {
		num := m[1]
		if num == "0" || num == "1" || num == "0.0" || num == "1.0" {
			continue
		}
		if isConstantLike(trimmed) {
			continue
		}
		findings = append(findings, analysis.Finding{
			Rule:        "hardcoding.magic_number",
			Pillar:      "hardcoding",
			Severity:    analysis.SeverityAdvisory,
			File:        path,
			Line:        lineNum,
			Source:      "treesitter",
			Message:     "magic number literal: " + num + " — use a named constant",
			Remediation: "Extract the literal into a const or enum with a descriptive name.",
		})
	}
	return findings
}

func isCommentLine(trimmed string, path string) bool {
	if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "*") {
		return true
	}
	if strings.HasPrefix(trimmed, "--") && strings.HasSuffix(path, ".sql") {
		return true
	}
	return false
}

func skipLine(trimmed string) bool {
	if strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "from ") || strings.HasPrefix(trimmed, "package ") {
		return true
	}
	if strings.HasPrefix(trimmed, "use ") || strings.HasPrefix(trimmed, "mod ") {
		return true
	}
	if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") {
		return true
	}
	if strings.HasPrefix(trimmed, "version") || strings.HasPrefix(trimmed, "http") {
		return true
	}
	return false
}

func isConstantLike(trimmed string) bool {
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "const ") || strings.HasPrefix(lower, "let ") || strings.HasPrefix(lower, "var ") {
		return true
	}
	if strings.Contains(trimmed, "= ") && (strings.Contains(trimmed, " //") || strings.Contains(trimmed, " #")) {
		return true
	}
	return false
}
