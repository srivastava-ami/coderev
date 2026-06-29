package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/srivastava-ami/coderev/internal/config"
)

var (
	flagLLMEnable   bool
	flagLLMDisable  bool
	flagLLMProvider string
	flagLLMCommand  string
	flagLLMBaseURL  string
	flagLLMKeyEnv   string
	flagLLMModel    string
)

var cmdConfig = &cobra.Command{
	Use:   "config",
	Short: "Manage coderev configuration",
}

var cmdConfigLLM = &cobra.Command{
	Use:   "llm",
	Short: "Configure LLM enrichment settings",
	Long: `Read and update the [llm] section of tool_config.toml.

With no flags, print the current [llm] settings (the api_key_env
name is shown; the key value itself is never read or printed).

With flags, update the specified keys in the [llm] section while
preserving the rest of the file verbatim. An [llm] block is
appended if the file has none.`,
	RunE: runConfigLLM,
}

func init() {
	cmdConfig.AddCommand(cmdConfigLLM)

	cmdConfigLLM.Flags().BoolVar(&flagLLMEnable, "enable", false, "enable LLM enrichment")
	cmdConfigLLM.Flags().BoolVar(&flagLLMDisable, "disable", false, "disable LLM enrichment")
	cmdConfigLLM.Flags().StringVar(&flagLLMProvider, "provider", "", `provider: "cli", "api", or "oauth"`)
	cmdConfigLLM.Flags().StringVar(&flagLLMCommand, "command", "", `CLI command template with {prompt}`)
	cmdConfigLLM.Flags().StringVar(&flagLLMBaseURL, "api-base-url", "", "API base URL for api provider")
	cmdConfigLLM.Flags().StringVar(&flagLLMKeyEnv, "api-key-env", "", "name of env var holding the API key")
	cmdConfigLLM.Flags().StringVar(&flagLLMModel, "model", "", "model identifier for api provider")
}

func runConfigLLM(cmd *cobra.Command, _ []string) error {
	cfgPath := resolveLLMConfigPath()
	if cfgPath == "" {
		cfgPath = "tool_config.toml"
	}

	hasFlags := false
	for _, name := range []string{"enable", "disable", "provider", "command", "api-base-url", "api-key-env", "model"} {
		if cmd.Flags().Changed(name) {
			hasFlags = true
			break
		}
	}

	if !hasFlags {
		return printLLMConfig(cfgPath)
	}

	if cmd.Flags().Changed("enable") && cmd.Flags().Changed("disable") {
		return fmt.Errorf("--enable and --disable are mutually exclusive")
	}

	updates := buildLLMUpdates(cmd)
	return updateLLMConfig(cfgPath, updates)
}

func resolveLLMConfigPath() string {
	if flagConfig != "" {
		return flagConfig
	}
	p, _ := config.DiscoverToolConfig(".")
	if p == "" {
		p = filepath.Join(".coderev", "tool_config.toml")
	}
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	return p
}

func buildLLMUpdates(cmd *cobra.Command) map[string]string {
	updates := make(map[string]string)
	if cmd.Flags().Changed("enable") {
		updates["enabled"] = "true"
	} else if cmd.Flags().Changed("disable") {
		updates["enabled"] = "false"
	}
	if cmd.Flags().Changed("provider") {
		updates["provider"] = flagLLMProvider
	}
	if cmd.Flags().Changed("command") {
		updates["cli_command"] = flagLLMCommand
	}
	if cmd.Flags().Changed("api-base-url") {
		updates["api_base_url"] = flagLLMBaseURL
	}
	if cmd.Flags().Changed("api-key-env") {
		updates["api_key_env"] = flagLLMKeyEnv
	}
	if cmd.Flags().Changed("model") {
		updates["model"] = flagLLMModel
	}
	return updates
}

func updateLLMConfig(path string, updates map[string]string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("reading config: %w", err)
		}
		data = []byte{}
	}
	lines := strings.Split(string(data), "\n")
	if llmIdx := findSectionStart(lines, "[llm]"); llmIdx >= 0 {
		lines = updateExistingSection(lines, llmIdx, updates)
	} else {
		lines = appendNewSection(lines, updates)
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
}

// updateExistingSection rewrites matching keys inside the [llm] block and inserts
// any keys not already present, preserving every other line verbatim.
func updateExistingSection(lines []string, llmIdx int, updates map[string]string) []string {
	endIdx := findSectionEnd(lines, llmIdx+1)
	found := make(map[string]bool, len(updates))
	for i := llmIdx + 1; i < endIdx; i++ {
		key, _, ok := parseAssignment(lines[i])
		if !ok {
			continue
		}
		if newVal, exists := updates[key]; exists {
			lines[i] = key + " = " + formatTOMLValue(key, newVal)
			found[key] = true
		}
	}
	missing := missingKeys(updates, found)
	if len(missing) == 0 {
		return lines
	}
	result := make([]string, 0, len(lines)+len(missing))
	result = append(result, lines[:endIdx]...)
	for _, k := range missing {
		result = append(result, k+" = "+formatTOMLValue(k, updates[k]))
	}
	return append(result, lines[endIdx:]...)
}

// appendNewSection adds a fresh [llm] block with the keys in sorted order.
func appendNewSection(lines []string, updates map[string]string) []string {
	if len(lines) > 0 && lines[len(lines)-1] != "" {
		lines = append(lines, "")
	}
	lines = append(lines, "[llm]")
	for _, k := range sortedKeys(updates) {
		lines = append(lines, k+" = "+formatTOMLValue(k, updates[k]))
	}
	return lines
}

func missingKeys(updates map[string]string, found map[string]bool) []string {
	var missing []string
	for k := range updates {
		if !found[k] {
			missing = append(missing, k)
		}
	}
	sort.Strings(missing)
	return missing
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func printLLMConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	llmIdx := findSectionStart(lines, "[llm]")
	if llmIdx < 0 {
		fmt.Println("no [llm] section in " + path)
		return nil
	}

	endIdx := findSectionEnd(lines, llmIdx+1)
	fmt.Printf("LLM enrichment settings (%s):\n", path)
	for _, line := range lines[llmIdx : endIdx+1] {
		fmt.Println(line)
	}
	return nil
}

func findSectionStart(lines []string, header string) int {
	for i, line := range lines {
		if strings.TrimSpace(line) == header {
			return i
		}
	}
	return -1
}

func findSectionEnd(lines []string, start int) int {
	for i := start; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			return i
		}
	}
	return len(lines)
}

func parseAssignment(line string) (key, value string, ok bool) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "[") {
		return "", "", false
	}
	eqIdx := strings.Index(trimmed, "=")
	if eqIdx < 0 {
		return "", "", false
	}
	key = strings.TrimSpace(trimmed[:eqIdx])
	value = strings.TrimSpace(trimmed[eqIdx+1:])
	if key == "" {
		return "", "", false
	}
	return key, value, true
}

func formatTOMLValue(key, val string) string {
	if key == "enabled" {
		return val
	}
	return `"` + strings.ReplaceAll(val, `"`, `\"`) + `"`
}
