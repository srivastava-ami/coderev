package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/srivastava-ami/coderev/internal/config"
	"github.com/srivastava-ami/coderev/internal/llm"
)

var cmdAsk = &cobra.Command{
	Use:   "ask <prompt...>",
	Short: "Send a prompt to the configured LLM provider",
	Long: `Load the LLM configuration from tool_config.toml and send the
prompt to the configured provider (cli or api). The provider must
be enabled first via: coderev config llm --enable --provider ...`,
	Args: cobra.MinimumNArgs(1),
	RunE: runAsk,
}

func runAsk(_ *cobra.Command, args []string) error {
	cfgPath := resolveLLMConfigPath()
	if cfgPath == "" {
		cfgPath = "tool_config.toml"
	}
	tc, err := config.LoadToolConfig(cfgPath)
	if err != nil {
		return fmt.Errorf("loading tool config: %w", err)
	}
	if !tc.LLM.Enabled {
		return fmt.Errorf("LLM enrichment is not enabled. Run `coderev config llm --enable --provider ...` first")
	}
	provider, err := llm.New(tc.LLM)
	if err != nil {
		return fmt.Errorf("creating LLM provider: %w", err)
	}
	prompt := strings.Join(args, " ")
	result, _, err := provider.Complete(context.Background(), prompt)
	if err != nil {
		return fmt.Errorf("LLM request failed: %w", err)
	}
	fmt.Print(result)
	return nil
}
