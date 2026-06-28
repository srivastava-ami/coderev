// Package graphanalyze provides a ToolAdapter that analyses code-graph metrics
// (fan-in, fan-out, centrality) produced by internal/graph and emits ADVISORY
// findings for excessive coupling and change-risk hotspots.
package graphanalyze

import (
	"context"
	"fmt"

	"github.com/srivastava-ami/coderev/internal/analysis"
	"github.com/srivastava-ami/coderev/internal/graph"
)

type Adapter struct {
	cfg analysis.GraphAnalyzeConfig
}

func New(cfg analysis.GraphAnalyzeConfig) *Adapter {
	return &Adapter{cfg: cfg}
}

func (a *Adapter) Name() string { return "graphanalyze" }

func (a *Adapter) IsAvailable() bool { return true }

func (a *Adapter) Capabilities() []string {
	return []string{"architecture.coupling", "architecture.hotspot"}
}

func (a *Adapter) Run(ctx context.Context, req analysis.RunRequest) ([]analysis.Finding, error) {
	g, err := graph.Build(req.Target)
	if err != nil || g == nil {
		return nil, nil
	}
	graph.ComputeMetrics(g)

	var findings []analysis.Finding
	ruleSet := make(map[string]bool, len(a.cfg.Rules))
	for _, r := range a.cfg.Rules {
		ruleSet[r] = true
	}

	for _, n := range g.Nodes {
		fanOut := g.FanOut[n.ID]
		fanIn := g.FanIn[n.ID]

		if ruleSet["architecture.coupling"] && fanOut > a.cfg.FanOutMax {
			findings = append(findings, analysis.Finding{
				Rule:        "architecture.coupling",
				Pillar:      "architecture",
				Severity:    analysis.SeverityAdvisory,
				File:        n.SourceFile,
				Message:     fmt.Sprintf("%s: fan-out %d exceeds max %d (high coupling)", n.Label, fanOut, a.cfg.FanOutMax),
				Remediation: "Refactor to reduce outgoing dependencies: extract interfaces, split the module, or invert dependencies.",
				Source:      "graphanalyze",
			})
		}

		if ruleSet["architecture.hotspot"] && fanIn > a.cfg.FanInMax && fanOut > a.cfg.FanOutMax {
			findings = append(findings, analysis.Finding{
				Rule:        "architecture.hotspot",
				Pillar:      "architecture",
				Severity:    analysis.SeverityAdvisory,
				File:        n.SourceFile,
				Message:     fmt.Sprintf("%s: hotspot (fan-in %d, fan-out %d) — change-risky", n.Label, fanIn, fanOut),
				Remediation: "Reduce both fan-in and fan-out: split responsibilities, introduce intermediate abstractions, and avoid god-object patterns.",
				Source:      "graphanalyze",
			})
		}
	}

	return findings, nil
}
