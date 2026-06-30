package treesitter

import (
	"fmt"
	"hash/fnv"
	"strings"
	"unicode"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

const dupWindowSize = 25 // token sequence length — ~6-10 lines of typical code

type dupLoc struct {
	file string
	line int
}

// DetectDuplication finds repeated token sequences across files using FNV-64
// sliding window hashing. Only cross-file duplicates are flagged.
func DetectDuplication(files []analysis.FileInfo) []analysis.Finding {
	hashMap := buildHashMap(files)
	return emitDupFindings(hashMap)
}

func buildHashMap(files []analysis.FileInfo) map[uint64][]dupLoc {
	hashMap := make(map[uint64][]dupLoc)
	for _, fi := range files {
		if !dupEligible(fi) {
			continue
		}
		indexFile(fi, hashMap)
	}
	return hashMap
}

func dupEligible(fi analysis.FileInfo) bool {
	if isTestFile(fi.Path) {
		return false
	}
	return fi.Language == analysis.LangTypeScript ||
		fi.Language == analysis.LangJavaScript ||
		fi.Language == analysis.LangGo
}

func indexFile(fi analysis.FileInfo, hashMap map[uint64][]dupLoc) {
	lines := strings.Split(string(fi.Content), "\n")
	tokens, lineNums := dupTokenize(lines)
	if len(tokens) < dupWindowSize {
		return
	}
	for i := 0; i <= len(tokens)-dupWindowSize; i++ {
		h := dupHash(tokens[i : i+dupWindowSize])
		hashMap[h] = append(hashMap[h], dupLoc{file: fi.Path, line: lineNums[i]})
	}
}

func emitDupFindings(hashMap map[uint64][]dupLoc) []analysis.Finding {
	seen := make(map[string]bool)
	var findings []analysis.Finding
	for _, locs := range hashMap {
		findings = append(findings, dupPairs(locs, seen)...)
	}
	return findings
}

func dupPairs(locs []dupLoc, seen map[string]bool) []analysis.Finding {
	byFile := deduplicateByFile(locs)
	if len(byFile) < 2 {
		return nil
	}
	list := locList(byFile)
	var findings []analysis.Finding
	for i := 0; i < len(list); i++ {
		findings = append(findings, dupPairsFrom(list, i, seen)...)
	}
	return findings
}

func dupPairsFrom(list []dupLoc, i int, seen map[string]bool) []analysis.Finding {
	var findings []analysis.Finding
	for j := i + 1; j < len(list); j++ {
		if f, ok := dupFinding(list[i], list[j], seen); ok {
			findings = append(findings, f)
		}
	}
	return findings
}

func deduplicateByFile(locs []dupLoc) map[string]dupLoc {
	byFile := make(map[string]dupLoc)
	for _, loc := range locs {
		if _, exists := byFile[loc.file]; !exists {
			byFile[loc.file] = loc
		}
	}
	return byFile
}

func locList(byFile map[string]dupLoc) []dupLoc {
	list := make([]dupLoc, 0, len(byFile))
	for _, loc := range byFile {
		list = append(list, loc)
	}
	return list
}

func dupFinding(a, b dupLoc, seen map[string]bool) (analysis.Finding, bool) {
	if a.file > b.file {
		a, b = b, a
	}
	key := fmt.Sprintf("%s:%d|%s:%d", a.file, a.line, b.file, b.line)
	if seen[key] {
		return analysis.Finding{}, false
	}
	seen[key] = true
	return analysis.Finding{
		Rule:        "file_structure.duplication",
		Pillar:      "file_structure",
		Severity:    analysis.SeverityMajor,
		File:        a.file,
		Line:        a.line,
		Source:      "treesitter",
		Message:     fmt.Sprintf("~%d-token block duplicated in %s:%d", dupWindowSize, b.file, b.line),
		Remediation: "Extract the shared logic into a shared utility module.",
	}, true
}

func dupTokenize(lines []string) (tokens []string, lineNums []int) {
	for i, line := range lines {
		if dupSkipLine(line) {
			continue
		}
		for _, tok := range dupSplitTokens(line) {
			tokens = append(tokens, tok)
			lineNums = append(lineNums, i+1)
		}
	}
	return
}

func dupSkipLine(line string) bool {
	t := strings.TrimSpace(line)
	if strings.HasPrefix(t, "//") || strings.HasPrefix(t, "*") || t == "" || t == ")" {
		return true
	}
	if strings.HasPrefix(t, "import ") || strings.HasPrefix(t, "from ") || strings.HasPrefix(t, "package ") {
		return true
	}
	if strings.HasPrefix(t, `"`) || strings.HasPrefix(t, "`") {
		return true
	}
	return false
}

func dupSplitTokens(line string) []string {
	raw := strings.FieldsFunc(line, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	out := raw[:0]
	for _, tok := range raw {
		if len(tok) > 1 {
			out = append(out, tok)
		}
	}
	return out
}

func dupHash(tokens []string) uint64 {
	h := fnv.New64a()
	for _, t := range tokens {
		h.Write([]byte(t))
		h.Write([]byte{0})
	}
	return h.Sum64()
}
