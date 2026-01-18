package executor

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"sync"
	"time"
)

// Result represents the result of a kubectl command execution
type Result struct {
	Context  string
	Output   string
	Error    error
	ExitCode int
}

// Executor executes kubectl commands across multiple clusters
type Executor struct {
	kubeConfigPath string
	timeout        time.Duration
}

// NewExecutor creates a new kubectl executor
func NewExecutor(kubeConfigPath string, timeout time.Duration) *Executor {
	return &Executor{
		kubeConfigPath: kubeConfigPath,
		timeout:        timeout,
	}
}

// Execute runs a kubectl command against multiple contexts in parallel
func (e *Executor) Execute(contexts []string, args []string) []Result {
	var wg sync.WaitGroup
	results := make([]Result, len(contexts))

	for i, ctx := range contexts {
		wg.Add(1)
		go func(index int, context string) {
			defer wg.Done()
			results[index] = e.executeOne(context, args)
		}(i, ctx)
	}

	wg.Wait()
	return results
}

func (e *Executor) executeOne(contextName string, args []string) Result {
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	// Build kubectl command with context
	cmdArgs := []string{"--context", contextName}
	if e.kubeConfigPath != "" {
		cmdArgs = append([]string{"--kubeconfig", e.kubeConfigPath}, cmdArgs...)
	}
	cmdArgs = append(cmdArgs, args...)

	cmd := exec.CommandContext(ctx, "kubectl", cmdArgs...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := Result{
		Context: contextName,
		Output:  stdout.String(),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			result.Error = fmt.Errorf("%s", stderr.String())
		} else {
			result.Error = err
		}
	}

	return result
}
