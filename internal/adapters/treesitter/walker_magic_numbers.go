package treesitter

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

type magicChecker struct {
	re     *regexp.Regexp
	excSet map[string]bool
}

func newMagicChecker(exceptions []int) *magicChecker {
	excSet := make(map[string]bool, len(exceptions))
	for _, n := range exceptions {
		excSet[fmt.Sprintf("%d", n)] = true
	}
	return &magicChecker{
		re:     regexp.MustCompile(`(?:^|[^a-zA-Z0-9_.])([0-9]+(?:\.[0-9]+)?)(?:[^a-zA-Z0-9_.]|$)`),
		excSet: excSet,
	}
}

func checkMagicNumbers(files []analysis.FileInfo, exceptions []int) []analysis.Finding {
	mc := newMagicChecker(exceptions)
	var findings []analysis.Finding
	for _, fi := range files {
		if isTestFile(fi.Path) {
			continue
		}
		if strings.HasSuffix(fi.Path, ".json") || strings.HasSuffix(fi.Path, ".toml") ||
			strings.HasSuffix(fi.Path, ".yaml") || strings.HasSuffix(fi.Path, ".yml") {
			continue
		}
		lines := strings.Split(string(fi.Content), "\n")
		findings = append(findings, mc.scanLines(lines, fi.Path)...)
	}
	return findings
}

func (mc *magicChecker) scanLines(lines []string, path string) []analysis.Finding {
	var findings []analysis.Finding
	for i, line := range lines {
		lineNum := i + 1
		trimmed := strings.TrimSpace(line)
		if isCommentLine(trimmed, path) || skipLine(trimmed) {
			continue
		}
		findings = append(findings, mc.emitMatches(line, trimmed, path, lineNum)...)
	}
	return findings
}

func (mc *magicChecker) emitMatches(line, trimmed, path string, lineNum int) []analysis.Finding {
	var findings []analysis.Finding
	for _, m := range mc.re.FindAllStringSubmatch(line, -1) {
		num := m[1]
		if num == "0" || num == "1" || num == "0.0" || num == "1.0" {
			continue
		}
		if mc.excSet[num] {
			continue
		}
		if isConstantLike(trimmed) {
			continue
		}
		findings = append(findings, analysis.Finding{
			Rule:        "hardcoding.magic_numbers",
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
