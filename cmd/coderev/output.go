package main

import (
	"fmt"

	"github.com/srivastava-ami/coderev/internal/analysis"
	"github.com/srivastava-ami/coderev/internal/report"
)

func printStartup(target, stdFile string, ads []analysis.ToolAdapter) {
	fmt.Printf("coderev™ %s — by Amit Srivastava · https://github.com/srivastava-ami/coderev\n\n", version)
	fmt.Printf("coderev · target: %s\n", target)
	fmt.Printf("          standards: %s\n", stdFile)
	fmt.Printf("          adapters: %s\n", adapterNames(ads))
	fmt.Println("          scanning…")
}

func printSummary(s report.Summary, warnings []analysis.AdapterWarning, outputPath string) {
	fmt.Println()
	fmt.Printf("  files scanned : %d\n", s.TotalFiles)
	fmt.Printf("  total findings: %d\n", s.TotalFindings)
	fmt.Printf("    blockers    : %d\n", s.BySeverity["blocker"])
	fmt.Printf("    major       : %d\n", s.BySeverity["major"])
	fmt.Printf("    advisory    : %d\n", s.BySeverity["advisory"])

	if len(warnings) > 0 {
		fmt.Printf("\n  ⚠  %d adapter(s) skipped (not installed)\n", len(warnings))
	}
	status := "✓ PASS"
	if s.OverallStatus == "FAIL" {
		status = "✗ FAIL"
	}
	fmt.Printf("\n  status: %s\n", status)
	fmt.Printf("  report: %s\n", outputPath)
	fmt.Printf("\n  ⭐ useful? star coderev: https://github.com/srivastava-ami/coderev\n\n")
}
