package baseline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

const baselineFile = ".coderev/baseline.json"

// Snapshot records finding counts at a point in time.
type Snapshot struct {
	CreatedAt string         `json:"created_at"`
	Counts    map[string]int `json:"counts"` // severity → count
}

// Delta describes the change between the current run and a saved baseline.
type Delta struct {
	Blockers int // positive = more blockers than baseline, negative = improvement
	Majors   int
	Total    int
	IsNew    bool // true when no baseline existed yet
}

// Save writes the current run as the new baseline.
func Save(target string, findings []analysis.Finding) error {
	dir := filepath.Join(target, ".coderev")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("baseline: creating .coderev dir: %w", err)
	}
	s := Snapshot{
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		Counts:    countBySeverity(findings),
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(target, baselineFile)
	return os.WriteFile(path, data, 0o644)
}

// Load reads the baseline file from target. Returns nil, nil if no baseline exists.
func Load(target string) (*Snapshot, error) {
	data, err := os.ReadFile(filepath.Join(target, baselineFile))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("baseline: reading baseline: %w", err)
	}
	var s Snapshot
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("baseline: parsing baseline: %w", err)
	}
	return &s, nil
}

// Compute returns the delta between a current finding list and a saved baseline.
// If base is nil, Delta.IsNew is true and counts reflect the current run.
func Compute(base *Snapshot, current []analysis.Finding) Delta {
	now := countBySeverity(current)
	if base == nil {
		return Delta{
			Blockers: now["blocker"],
			Majors:   now["major"],
			Total:    len(current),
			IsNew:    true,
		}
	}
	return Delta{
		Blockers: now["blocker"] - base.Counts["blocker"],
		Majors:   now["major"] - base.Counts["major"],
		Total:    len(current) - totalFrom(base.Counts),
	}
}

func countBySeverity(findings []analysis.Finding) map[string]int {
	m := make(map[string]int)
	for _, f := range findings {
		m[string(f.Severity)]++
	}
	return m
}

func totalFrom(counts map[string]int) int {
	t := 0
	for _, v := range counts {
		t += v
	}
	return t
}
