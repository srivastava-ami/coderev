package analysis

import "context"

// RunRequest carries everything an adapter needs to run its analysis.
type RunRequest struct {
	Target    string
	Files     []FileInfo
	RuleIDs   []string
	ExtraArgs []string
}

// ToolAdapter is the single swap boundary between the runner and any analysis
// tool. To replace a tool: implement this interface, register it, update
// tool_config.toml — no other code changes required.
type ToolAdapter interface {
	Name()         string
	IsAvailable()  bool
	Capabilities() []string
	Run(ctx context.Context, req RunRequest) ([]Finding, error)
}
