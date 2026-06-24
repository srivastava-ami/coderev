package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/srivastava-ami/coderev/internal/plugin"
)

var cmdPlugin = &cobra.Command{
	Use:   "plugin",
	Short: "Manage coderev plugins",
	Long: `Install, list, and manage coderev analysis plugins.
Plugins extend coderev with additional analysis capabilities.`,
}

var cmdPluginInstall = &cobra.Command{
	Use:   "install <manifest-path>",
	Short: "Install a plugin from its manifest file",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		return runPluginInstall(args)
	},
}

var cmdPluginList = &cobra.Command{
	Use:   "list",
	Short: "List installed plugins",
	RunE: func(_ *cobra.Command, _ []string) error {
		return runPluginList()
	},
}

func init() {
	cmdPlugin.AddCommand(cmdPluginInstall, cmdPluginList)
}

func runPluginInstall(args []string) error {
	manifestPath := args[0]
	m, err := plugin.LoadManifest(manifestPath)
	if err != nil {
		return err
	}
	pluginDir, err := ensurePluginDir()
	if err != nil {
		return err
	}
	dest := filepath.Join(pluginDir, m.Name+"-plugin.toml")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return err
	}
	if err := os.WriteFile(dest, data, 0644); err != nil {
		return err
	}
	fmt.Printf("installed plugin: %s (%s)\n", m.Name, m.Version)
	return nil
}

func runPluginList() error {
	pluginDir, err := getPluginDir()
	if err != nil {
		return err
	}
	plugins, err := plugin.DiscoverPlugins(pluginDir)
	if err != nil {
		return err
	}
	if len(plugins) == 0 {
		fmt.Println("no plugins installed")
		return nil
	}
	fmt.Printf("%-20s %-10s %-40s\n", "NAME", "VERSION", "DESCRIPTION")
	for _, p := range plugins {
		fmt.Printf("%-20s %-10s %-40s\n", p.Manifest.Name, p.Manifest.Version, p.Manifest.Description)
	}
	return nil
}

func ensurePluginDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".config", "coderev", "plugins")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

func getPluginDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "coderev", "plugins"), nil
}
