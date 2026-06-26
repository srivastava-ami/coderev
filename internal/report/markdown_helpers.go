package report

import (
	"fmt"
	"strings"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

func severityIcon(s analysis.Severity) string {
	switch s {
	case analysis.SeverityBlocker:
		return "🔴 blocker"
	case analysis.SeverityMajor:
		return "🟡 major"
	case analysis.SeverityAdvisory:
		return "🔵 advisory"
	default:
		return "⚪ info"
	}
}

// heatBarWidth is the number of cells in a rendered heat bar.
const heatBarWidth = 5

func heatBar(score float64) string {
	filled := int(score * heatBarWidth)
	bar := strings.Repeat("█", filled) + strings.Repeat("░", heatBarWidth-filled)
	return fmt.Sprintf("`%s`", bar)
}

// mermaidID converts an arbitrary string into a safe Mermaid node identifier.
func mermaidID(s string) string {
	var out strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			out.WriteRune(r)
		} else {
			out.WriteRune('_')
		}
	}
	return out.String()
}

// shortenPath trims leading path components to keep table rows readable.
func shortenPath(p string) string {
	const maxLen = 60
	if len(p) <= maxLen {
		return p
	}
	return "…" + p[len(p)-maxLen+1:]
}

// mdEscape escapes pipe characters so they don't break Markdown tables.
func mdEscape(s string) string {
	return strings.ReplaceAll(s, "|", "\\|")
}

// ratingBadge returns a short Markdown badge for the A-E rating.
func ratingBadge(r string) string {
	switch r {
	case "A":
		return "![A](https://img.shields.io/badge/reliability-A-brightgreen)"
	case "B":
		return "![B](https://img.shields.io/badge/reliability-B-green)"
	case "C":
		return "![C](https://img.shields.io/badge/reliability-C-yellow)"
	case "D":
		return "![D](https://img.shields.io/badge/reliability-D-orange)"
	case "E":
		return "![E](https://img.shields.io/badge/reliability-E-red)"
	default:
		return ""
	}
}

// writeDeltaLine emits a trend line comparing the current run to the baseline.
func writeDeltaLine(b *strings.Builder, r Report) {
	d := r.Summary.Delta
	if d == nil {
		return
	}
	if d.IsNew {
		fmt.Fprintf(b, "> 📊 **Baseline saved** — future runs will track trends against these %d findings.\n\n", r.Summary.TotalFindings)
		return
	}
	sign := func(n int) string {
		if n > 0 {
			return fmt.Sprintf("+%d", n)
		}
		return fmt.Sprintf("%d", n)
	}
	icon := "📈"
	if d.Total <= 0 {
		icon = "📉"
	}
	fmt.Fprintf(b, "> %s **vs baseline** — blockers %s · majors %s · total %s\n\n",
		icon, sign(d.Blockers), sign(d.Majors), sign(d.Total))
}
