package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/multikubectl/pkg/cluster"
	"github.com/multikubectl/pkg/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage multikubectl configuration",
	Long: `Manage which clusters multikubectl operates on.

Configuration is stored in ~/.multikube/config. When this file exists,
multikubectl will only operate on the configured contexts by default.`,
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available contexts and show which are configured",
	Run:   runConfigList,
}

var configAddCmd = &cobra.Command{
	Use:   "add <context> [context...]",
	Short: "Add context(s) to the configuration",
	Args:  cobra.MinimumNArgs(1),
	Run:   runConfigAdd,
}

var configRemoveCmd = &cobra.Command{
	Use:   "remove <context> [context...]",
	Short: "Remove context(s) from the configuration",
	Args:  cobra.MinimumNArgs(1),
	Run:   runConfigRemove,
}

var configUseCmd = &cobra.Command{
	Use:   "use <context1,context2,...>",
	Short: "Set the contexts to use (replaces existing configuration)",
	Args:  cobra.ExactArgs(1),
	Run:   runConfigUse,
}

var configClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear the configuration (use all contexts from kubeconfig)",
	Run:   runConfigClear,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Run:   runConfigShow,
}

var configSelectCmd = &cobra.Command{
	Use:   "select",
	Short: "Interactively select contexts to use",
	Long: `Interactively select which contexts to use with a multi-select interface.

Use arrow keys to navigate, space to select/deselect, and enter to confirm.
Previously configured contexts will be pre-selected.`,
	Run: runConfigSelect,
}

func init() {
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configAddCmd)
	configCmd.AddCommand(configRemoveCmd)
	configCmd.AddCommand(configUseCmd)
	configCmd.AddCommand(configClearCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSelectCmd)
}

func runConfigList(cmd *cobra.Command, args []string) {
	// Load kubeconfig to get all available contexts
	mgr, err := cluster.NewManager("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading kubeconfig: %v\n", err)
		os.Exit(1)
	}

	// Load multikube config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading multikube config: %v\n", err)
		os.Exit(1)
	}

	allContexts := mgr.GetContexts()
	currentContext := mgr.GetCurrentContext()
	hasConfig := config.Exists() && len(cfg.Contexts) > 0

	fmt.Println("Available contexts from kubeconfig:")
	fmt.Println()

	for _, ctx := range allContexts {
		marker := "  "
		if hasConfig && cfg.HasContext(ctx) {
			marker = "* "
		}
		current := ""
		if ctx == currentContext {
			current = " (current)"
		}
		fmt.Printf("%s%s%s\n", marker, ctx, current)
	}

	fmt.Println()
	if hasConfig {
		fmt.Printf("Configured contexts (* marked): %d/%d\n", len(cfg.Contexts), len(allContexts))
		fmt.Printf("Config file: %s\n", config.GetConfigPath())
	} else {
		fmt.Println("No multikube config found. Using all contexts.")
		fmt.Printf("Run 'multikubectl config use <contexts>' to configure specific contexts.\n")
	}
}

func runConfigAdd(cmd *cobra.Command, args []string) {
	// Load kubeconfig to validate contexts
	mgr, err := cluster.NewManager("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading kubeconfig: %v\n", err)
		os.Exit(1)
	}

	availableContexts := make(map[string]bool)
	for _, ctx := range mgr.GetContexts() {
		availableContexts[ctx] = true
	}

	// Load existing config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	added := 0
	for _, ctx := range args {
		if !availableContexts[ctx] {
			fmt.Fprintf(os.Stderr, "Warning: context '%s' not found in kubeconfig, skipping\n", ctx)
			continue
		}
		if cfg.AddContext(ctx) {
			fmt.Printf("Added context: %s\n", ctx)
			added++
		} else {
			fmt.Printf("Context already configured: %s\n", ctx)
		}
	}

	if added > 0 {
		if err := config.Save(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\nConfiguration saved to %s\n", config.GetConfigPath())
	}
}

func runConfigRemove(cmd *cobra.Command, args []string) {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	removed := 0
	for _, ctx := range args {
		if cfg.RemoveContext(ctx) {
			fmt.Printf("Removed context: %s\n", ctx)
			removed++
		} else {
			fmt.Printf("Context not found in config: %s\n", ctx)
		}
	}

	if removed > 0 {
		if err := config.Save(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\nConfiguration saved to %s\n", config.GetConfigPath())
	}
}

func runConfigUse(cmd *cobra.Command, args []string) {
	// Load kubeconfig to validate contexts
	mgr, err := cluster.NewManager("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading kubeconfig: %v\n", err)
		os.Exit(1)
	}

	availableContexts := make(map[string]bool)
	for _, ctx := range mgr.GetContexts() {
		availableContexts[ctx] = true
	}

	// Parse comma-separated contexts
	contextsStr := args[0]
	contextsList := strings.Split(contextsStr, ",")

	var validContexts []string
	for _, ctx := range contextsList {
		ctx = strings.TrimSpace(ctx)
		if ctx == "" {
			continue
		}
		if !availableContexts[ctx] {
			fmt.Fprintf(os.Stderr, "Warning: context '%s' not found in kubeconfig, skipping\n", ctx)
			continue
		}
		validContexts = append(validContexts, ctx)
	}

	if len(validContexts) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no valid contexts specified")
		os.Exit(1)
	}

	cfg := &config.MultiKubeConfig{}
	cfg.SetContexts(validContexts)

	if err := config.Save(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Configured contexts:\n")
	for _, ctx := range validContexts {
		fmt.Printf("  - %s\n", ctx)
	}
	fmt.Printf("\nConfiguration saved to %s\n", config.GetConfigPath())
}

func runConfigClear(cmd *cobra.Command, args []string) {
	configPath := config.GetConfigPath()

	if !config.Exists() {
		fmt.Println("No configuration file exists.")
		return
	}

	if err := os.Remove(configPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error removing config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Configuration cleared. Removed %s\n", configPath)
	fmt.Println("multikubectl will now use all contexts from kubeconfig.")
}

func runConfigShow(cmd *cobra.Command, args []string) {
	if !config.Exists() {
		fmt.Println("No multikube configuration file exists.")
		fmt.Println("Using all contexts from kubeconfig.")
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Config file: %s\n", config.GetConfigPath())
	fmt.Println()

	if len(cfg.Contexts) == 0 {
		fmt.Println("No contexts configured.")
	} else {
		fmt.Println("Configured contexts:")
		for _, ctx := range cfg.Contexts {
			fmt.Printf("  - %s\n", ctx)
		}
	}

	if cfg.KubeConfig != "" {
		fmt.Printf("\nKubeconfig: %s\n", cfg.KubeConfig)
	}
}

func runConfigSelect(cmd *cobra.Command, args []string) {
	// Load kubeconfig to get all available contexts
	mgr, err := cluster.NewManager("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading kubeconfig: %v\n", err)
		os.Exit(1)
	}

	allContexts := mgr.GetContexts()
	if len(allContexts) == 0 {
		fmt.Fprintln(os.Stderr, "No contexts found in kubeconfig")
		os.Exit(1)
	}

	// Load existing config to pre-select configured contexts
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Determine which contexts should be pre-selected
	var defaultSelected []string
	if config.Exists() && len(cfg.Contexts) > 0 {
		for _, ctx := range cfg.Contexts {
			for _, available := range allContexts {
				if ctx == available {
					defaultSelected = append(defaultSelected, ctx)
					break
				}
			}
		}
	}

	// Create multi-select prompt
	var selectedContexts []string
	prompt := &survey.MultiSelect{
		Message:  "Select contexts to use (space to select, enter to confirm):",
		Options:  allContexts,
		Default:  defaultSelected,
		PageSize: 15,
	}

	err = survey.AskOne(prompt, &selectedContexts, survey.WithKeepFilter(true))
	if err != nil {
		if err.Error() == "interrupt" {
			fmt.Println("\nCancelled.")
			return
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(selectedContexts) == 0 {
		// User selected nothing - ask if they want to clear config
		var clearConfig bool
		confirmPrompt := &survey.Confirm{
			Message: "No contexts selected. Clear configuration and use all contexts?",
			Default: false,
		}
		survey.AskOne(confirmPrompt, &clearConfig)

		if clearConfig {
			if config.Exists() {
				if err := os.Remove(config.GetConfigPath()); err != nil {
					fmt.Fprintf(os.Stderr, "Error removing config: %v\n", err)
					os.Exit(1)
				}
				fmt.Println("Configuration cleared. Using all contexts.")
			} else {
				fmt.Println("No configuration to clear. Using all contexts.")
			}
		} else {
			fmt.Println("No changes made.")
		}
		return
	}

	// Save selected contexts
	newConfig := &config.MultiKubeConfig{}
	newConfig.SetContexts(selectedContexts)

	if err := config.Save(newConfig); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nConfigured %d context(s):\n", len(selectedContexts))
	for _, ctx := range selectedContexts {
		fmt.Printf("  - %s\n", ctx)
	}
	fmt.Printf("\nConfiguration saved to %s\n", config.GetConfigPath())
}
