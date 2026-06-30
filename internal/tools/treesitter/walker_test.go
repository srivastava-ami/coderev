package treesitter

import (
	"strings"
	"testing"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

// defaultStds returns standards with all thresholds at professional-services defaults.
func defaultStds() analysis.Standards {
	return analysis.Standards{
		Complexity: analysis.ComplexityStd{
			Cyclomatic: analysis.CyclomaticStd{MaxValue: 8, AdvisoryAt: 5, HardBlockAt: 12},
			Cognitive:  analysis.CognitiveStd{MaxValue: 10},
			Function:   analysis.FunctionLengthStd{MaxLines: 30, AdvisoryAt: 20},
			Parameters: analysis.ParameterStd{MaxCount: 3},
			Nesting:    analysis.NestingStd{MaxDepth: 2},
		},
		FileStructure: analysis.FileStructureStd{
			FileLength:  analysis.FileLengthStd{MaxLines: 250, AdvisoryAt: 150},
			ClassLength: analysis.ClassLengthStd{MaxLines: 120},
		},
	}
}

func findingsForSrc(t *testing.T, src string, lang analysis.Language) []analysis.Finding {
	t.Helper()
	// Special handling for Terraform (no tree-sitter grammar)
	if lang == analysis.LangTerraform {
		return findingsForTerraform(t, src)
	}
	adapter := New(defaultStds())
	fi := analysis.FileInfo{Path: "test.ts", Language: lang, Content: []byte(src)}
	findings, err := adapter.analyseFile(fi)
	if err != nil {
		t.Fatalf("analyseFile: %v", err)
	}
	return findings
}

func findingsForTerraform(t *testing.T, src string) []analysis.Finding {
	t.Helper()
	stds := defaultStds()
	stds.TerraformConventions = analysis.TerraformConventionsStd{Severity: "blocker"}

	// Create a minimal walker for Terraform pattern checks (no tree-sitter AST parsing)
	w := &fileWalker{
		src:   []byte(src),
		file:  "test.tf",
		lang:  analysis.LangTerraform,
		stds:  stds,
		findings: []analysis.Finding{},
	}
	// Run Terraform-specific checks directly
	lines := strings.Split(src, "\n")
	w.checkTerraformConventions(lines)
	return w.findings
}

func hasRule(findings []analysis.Finding, rule string) bool {
	for _, f := range findings {
		if f.Rule == rule {
			return true
		}
	}
	return false
}

func countRule(findings []analysis.Finding, rule string) int {
	n := 0
	for _, f := range findings {
		if f.Rule == rule {
			n++
		}
	}
	return n
}

// ── Complexity ────────────────────────────────────────────────────────────────

func TestSimpleFunctionNoViolation(t *testing.T) {
	src := `
function greet(name: string): string {
  return "hello " + name;
}
`
	findings := findingsForSrc(t, src, analysis.LangTypeScript)
	for _, f := range findings {
		if f.Pillar == "complexity" {
			t.Errorf("unexpected complexity finding on simple function: %s — %s", f.Rule, f.Message)
		}
	}
}

func TestCyclomaticComplexityViolation(t *testing.T) {
	src := `
function classify(a: number, b: number, c: number, d: number): string {
  if (a > 0) {
    if (b > 0) {
      return "ab";
    } else if (c > 0) {
      return "ac";
    }
  } else if (d > 0) {
    return "d";
  }
  if (a < 0 && b < 0) {
    return "neg";
  }
  if (c === 0 || d === 0) {
    return "zero";
  }
  return "other";
}
`
	findings := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(findings, "complexity.cyclomatic") {
		t.Error("expected complexity.cyclomatic violation, got none")
	}
}

func TestFunctionTooManyParameters(t *testing.T) {
	src := `
function process(a: string, b: string, c: string, d: string): void {
  console.log(a, b, c, d);
}
`
	findings := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(findings, "complexity.parameter_count") {
		t.Error("expected complexity.parameter_count violation for 4 params (max 3)")
	}
}

func TestFunctionLengthViolation(t *testing.T) {
	lines := "function longFn(): void {\n"
	for i := 0; i < 35; i++ {
		lines += "  const x = 1;\n"
	}
	lines += "}\n"
	findings := findingsForSrc(t, lines, analysis.LangTypeScript)
	if !hasRule(findings, "complexity.function_length") {
		t.Error("expected complexity.function_length violation for 37-line function")
	}
}

func TestNestingDepthViolation(t *testing.T) {
	src := `
function nested(): void {
  if (true) {
    if (true) {
      if (true) {
        const x = 1;
      }
    }
  }
}
`
	findings := findingsForSrc(t, src, analysis.LangTypeScript)
	if !hasRule(findings, "complexity.nesting") {
		t.Error("expected complexity.nesting violation for depth 3")
	}
}
