package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/multikubectl/pkg/cluster"
	"github.com/multikubectl/pkg/config"
	"github.com/multikubectl/pkg/executor"
	"github.com/multikubectl/pkg/output"
	"github.com/spf13/cobra"
)

var (
	kubeConfig      string
	contexts        []string
	allContexts     bool
	timeout         time.Duration
	nonTableCommands = []string{"logs", "describe", "explain", "edit", "exec", "attach", "port-forward", "proxy", "cp"}
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "multikubectl [kubectl args...]",
	Short: "Multi-cluster kubectl - run kubectl commands across multiple clusters",
	Long: `multikubectl is a multi-cluster version of kubectl that allows you to
run kubectl commands across multiple Kubernetes clusters simultaneously.

It adds a CLUSTER column to the output showing which cluster each resource belongs to.

Examples:
  # Get pods from all clusters
  multikubectl get pods

  # Get pods from specific clusters
  multikubectl --contexts=cluster1,cluster2 get pods -n kube-system

  # Get all deployments from all contexts
  multikubectl --all-contexts get deployments

  # Use a specific kubeconfig file
  multikubectl --kubeconfig=/path/to/config get nodes`,
	DisableFlagParsing: false,
	Run:                runMultiKubectl,
}

func init() {
	rootCmd.Flags().StringVar(&kubeConfig, "kubeconfig", "", "Path to the kubeconfig file")
	rootCmd.Flags().StringSliceVar(&contexts, "contexts", nil, "Comma-separated list of contexts to use (overrides config)")
	rootCmd.Flags().BoolVar(&allContexts, "all-contexts", false, "Use all available contexts (ignores config)")
	rootCmd.Flags().DurationVar(&timeout, "timeout", 30*time.Second, "Timeout for kubectl commands")

	// Allow unknown flags to pass through to kubectl
	rootCmd.FParseErrWhitelist.UnknownFlags = true

	// Add config subcommand
	rootCmd.AddCommand(configCmd)
}

func Execute() {
	args := os.Args[1:]

	// Check if first arg is a subcommand (like "config") or help
	if len(args) > 0 {
		switch args[0] {
		case "config", "help", "completion", "--help", "-h":
			// Let cobra handle subcommands normally
			if err := rootCmd.Execute(); err != nil {
				os.Exit(1)
			}
			return
		}
	}

	// For kubectl passthrough commands, handle manually
	if len(args) == 0 {
		rootCmd.Help()
		return
	}

	// Separate our flags from kubectl flags
	ourArgs, kubectlArgs := separateArgs(args)

	// Parse our flags manually
	if err := rootCmd.ParseFlags(ourArgs); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	// Call runMultiKubectl directly with kubectl args
	runMultiKubectl(rootCmd, kubectlArgs)
}

// separateArgs separates multikubectl-specific flags from kubectl flags
func separateArgs(args []string) (ourArgs []string, kubectlArgs []string) {
	ourFlags := map[string]bool{
		"--kubeconfig":   true,
		"--contexts":     true,
		"--all-contexts": true,
		"--timeout":      true,
	}

	i := 0
	for i < len(args) {
		arg := args[i]

		// Check if it's one of our flags
		isOurFlag := false
		for flag := range ourFlags {
			if arg == flag || strings.HasPrefix(arg, flag+"=") {
				isOurFlag = true
				break
			}
		}

		if isOurFlag {
			ourArgs = append(ourArgs, arg)
			// If it doesn't contain '=', the next arg is the value
			if !strings.Contains(arg, "=") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				ourArgs = append(ourArgs, args[i])
			}
		} else {
			kubectlArgs = append(kubectlArgs, arg)
		}
		i++
	}

	return ourArgs, kubectlArgs
}

func runMultiKubectl(cmd *cobra.Command, args []string) {
	if len(args) == 0 {
		cmd.Help()
		return
	}

	// Initialize cluster manager
	mgr, err := cluster.NewManager(kubeConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading kubeconfig: %v\n", err)
		os.Exit(1)
	}

	// Determine which contexts to use
	// Priority: 1. --contexts flag  2. --all-contexts flag  3. ~/.multikube/config  4. all contexts
	var targetContexts []string

	if len(contexts) > 0 {
		// Command line --contexts takes highest priority
		targetContexts = mgr.FilterContexts(contexts)
	} else if allContexts {
		// --all-contexts flag ignores config file
		targetContexts = mgr.GetContexts()
	} else if config.Exists() {
		// Use ~/.multikube/config if it exists
		cfg, err := config.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading multikube config: %v\n", err)
			os.Exit(1)
		}
		if len(cfg.Contexts) > 0 {
			targetContexts = mgr.FilterContexts(cfg.Contexts)
		} else {
			targetContexts = mgr.GetContexts()
		}
	} else {
		// Default: use all contexts
		targetContexts = mgr.GetContexts()
	}

	if len(targetContexts) == 0 {
		fmt.Fprintln(os.Stderr, "No valid contexts found")
		os.Exit(1)
	}

	// Create executor
	exec := executor.NewExecutor(mgr.GetKubeConfigPath(), timeout)

	// Execute kubectl command across all contexts
	results := exec.Execute(targetContexts, args)

	// Merge and print results
	merger := output.NewMerger()

	// Check if this is a non-table command
	isNonTableCmd := false
	for _, nonTableCmd := range nonTableCommands {
		if len(args) > 0 && args[0] == nonTableCmd {
			isNonTableCmd = true
			break
		}
	}

	var mergedOutput string
	if isNonTableCmd {
		mergedOutput = merger.MergeNonTableOutput(results)
	} else {
		mergedOutput = merger.MergeResults(results, true)
	}

	fmt.Print(mergedOutput)

	// Check for any errors and set exit code
	for _, r := range results {
		if r.Error != nil {
			os.Exit(1)
		}
	}
}
