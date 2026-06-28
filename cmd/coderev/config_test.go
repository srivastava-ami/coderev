package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateLLMConfig_existingSection(t *testing.T) {
	input := `[scan]
batch_size = 1000

# LLM enrichment
[llm]
enabled      = false
provider     = "cli"
cli_command  = "claude -p {prompt}"

[github]
base_url = "https://api.github.com"

[sarif]
schema_url = "https://sarif.example.com"
`

	dir := t.TempDir()
	path := filepath.Join(dir, "tool_config.toml")
	if err := os.WriteFile(path, []byte(input), 0644); err != nil {
		t.Fatal(err)
	}

	updates := map[string]string{
		"enabled":  "true",
		"provider": "api",
		"model":    "claude-3-opus-20240229",
	}
	if err := updateLLMConfig(path, updates); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)

	if !strings.Contains(got, `enabled = true`) {
		t.Errorf("expected enabled = true, got:\n%s", got)
	}
	if !strings.Contains(got, `provider = "api"`) {
		t.Errorf("expected provider = \"api\", got:\n%s", got)
	}
	if !strings.Contains(got, `model = "claude-3-opus-20240229"`) {
		t.Errorf("expected model = \"claude-3-opus-20240229\", got:\n%s", got)
	}
	if !strings.Contains(got, `cli_command  = "claude -p {prompt}"`) {
		t.Errorf("expected cli_command to remain unchanged, got:\n%s", got)
	}
	if !strings.Contains(got, `[github]`) || !strings.Contains(got, `base_url = "https://api.github.com"`) {
		t.Errorf("expected github section to be preserved, got:\n%s", got)
	}
	if !strings.Contains(got, `[sarif]`) || !strings.Contains(got, `schema_url`) {
		t.Errorf("expected sarif section to be preserved, got:\n%s", got)
	}
	if !strings.Contains(got, `[scan]`) || !strings.Contains(got, `batch_size = 1000`) {
		t.Errorf("expected scan section to be preserved, got:\n%s", got)
	}
}

func TestUpdateLLMConfig_missingSection(t *testing.T) {
	input := `[scan]
batch_size = 1000

[github]
base_url = "https://api.github.com"
`

	dir := t.TempDir()
	path := filepath.Join(dir, "tool_config.toml")
	if err := os.WriteFile(path, []byte(input), 0644); err != nil {
		t.Fatal(err)
	}

	updates := map[string]string{
		"enabled":     "true",
		"provider":    "api",
		"model":       "claude-3-opus-20240229",
		"api_key_env": "MY_API_KEY",
	}
	if err := updateLLMConfig(path, updates); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)

	if !strings.Contains(got, "[llm]") {
		t.Errorf("expected [llm] section to be appended, got:\n%s", got)
	}
	if !strings.Contains(got, `api_key_env = "MY_API_KEY"`) {
		t.Errorf("expected api_key_env to be set, got:\n%s", got)
	}
	// Verify other sections still present
	if !strings.Contains(got, `[scan]`) || !strings.Contains(got, `[github]`) {
		t.Errorf("expected existing sections preserved, got:\n%s", got)
	}
}

func TestUpdateLLMConfig_newFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tool_config.toml")

	updates := map[string]string{
		"enabled":  "true",
		"provider": "cli",
	}
	if err := updateLLMConfig(path, updates); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)

	if !strings.Contains(got, "[llm]") {
		t.Errorf("expected [llm] section, got:\n%s", got)
	}
	if !strings.Contains(got, `enabled = true`) {
		t.Errorf("expected enabled = true, got:\n%s", got)
	}
	if !strings.Contains(got, `provider = "cli"`) {
		t.Errorf("expected provider = \"cli\", got:\n%s", got)
	}
}

func TestUpdateLLMConfig_byteIdentityForOtherSections(t *testing.T) {
	input := `[scan]
batch_size = 1000

[llm]
enabled      = false
provider     = "cli"

[github]
base_url = "https://api.github.com"
`

	dir := t.TempDir()
	path := filepath.Join(dir, "tool_config.toml")
	if err := os.WriteFile(path, []byte(input), 0644); err != nil {
		t.Fatal(err)
	}

	updates := map[string]string{"provider": "api", "model": "X"}
	if err := updateLLMConfig(path, updates); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)

	// Verify [scan] section byte-identical
	if !strings.Contains(got, "[scan]\nbatch_size = 1000") {
		t.Errorf("expected [scan] section exactly preserved, got:\n%s", got)
	}
	// Verify [github] section byte-identical
	if !strings.Contains(got, "[github]\nbase_url = \"https://api.github.com\"") {
		t.Errorf("expected [github] section exactly preserved, got:\n%s", got)
	}
	// Verify provider was updated
	if !strings.Contains(got, `provider = "api"`) {
		t.Errorf("expected provider updated, got:\n%s", got)
	}
	// Verify enabled line preserved
	if !strings.Contains(got, `enabled      = false`) {
		t.Errorf("expected enabled line preserved, got:\n%s", got)
	}
}
