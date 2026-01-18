package output

import (
	"fmt"
	"strings"

	"github.com/multikubectl/pkg/executor"
)

// Merger merges output from multiple clusters
type Merger struct {
	clusterColumnWidth int
}

// NewMerger creates a new output merger
func NewMerger() *Merger {
	return &Merger{
		clusterColumnWidth: 0,
	}
}

// MergeResults merges results from multiple clusters into a single output
func (m *Merger) MergeResults(results []executor.Result, showHeaders bool) string {
	if len(results) == 0 {
		return ""
	}

	// Calculate the max cluster name length for alignment
	m.clusterColumnWidth = 7 // minimum width for "CLUSTER"
	for _, r := range results {
		if len(r.Context) > m.clusterColumnWidth {
			m.clusterColumnWidth = len(r.Context)
		}
	}

	var output strings.Builder
	headerPrinted := false

	for _, result := range results {
		if result.Error != nil {
			output.WriteString(fmt.Sprintf("# Error from cluster %s: %v\n", result.Context, result.Error))
			continue
		}

		if result.Output == "" {
			continue
		}

		lines := strings.Split(strings.TrimSuffix(result.Output, "\n"), "\n")
		if len(lines) == 0 {
			continue
		}

		// Process each line
		for i, line := range lines {
			if line == "" {
				continue
			}

			// Check if this looks like a header line (first line with common header patterns)
			isHeader := i == 0 && m.isHeaderLine(line)

			if isHeader {
				if !headerPrinted && showHeaders {
					// Print header with CLUSTER column
					output.WriteString(m.formatLine("CLUSTER", line))
					output.WriteString("\n")
					headerPrinted = true
				}
				// Skip header lines after the first one
				continue
			}

			// Regular data line - add cluster name
			output.WriteString(m.formatLine(result.Context, line))
			output.WriteString("\n")
		}
	}

	return output.String()
}

// isHeaderLine checks if a line looks like a table header
func (m *Merger) isHeaderLine(line string) bool {
	// Common kubectl header patterns
	headerKeywords := []string{
		"NAME", "NAMESPACE", "STATUS", "READY", "AGE", "RESTARTS",
		"CLUSTER-IP", "EXTERNAL-IP", "PORT", "NODE", "NOMINATED",
		"READINESS", "REASON", "MESSAGE", "TYPE", "DATA", "CAPACITY",
		"ACCESS", "STORAGECLASS", "VOLUMEATTRIBUTESCLASS", "PROVISIONER",
		"RECLAIMPOLICY", "VOLUMEBINDINGMODE", "ALLOWVOLUMEEXPANSION",
		"COMPLETIONS", "DURATION", "SCHEDULE", "SUSPEND", "ACTIVE",
		"LAST", "DESIRED", "CURRENT", "UP-TO-DATE", "AVAILABLE",
		"REFERENCE", "TARGETS", "MINPODS", "MAXPODS", "REPLICAS",
	}

	upperLine := strings.ToUpper(line)
	matchCount := 0
	for _, keyword := range headerKeywords {
		if strings.Contains(upperLine, keyword) {
			matchCount++
		}
	}

	// If line contains multiple header keywords, it's likely a header
	return matchCount >= 2
}

// formatLine formats a line with the cluster column
func (m *Merger) formatLine(cluster, line string) string {
	format := fmt.Sprintf("%%-%ds   %%s", m.clusterColumnWidth)
	return fmt.Sprintf(format, cluster, line)
}

// MergeNonTableOutput merges non-table output (like logs, describe, etc.)
func (m *Merger) MergeNonTableOutput(results []executor.Result) string {
	var output strings.Builder

	for _, result := range results {
		if result.Error != nil {
			output.WriteString(fmt.Sprintf("=== Cluster: %s (Error: %v) ===\n", result.Context, result.Error))
			continue
		}

		output.WriteString(fmt.Sprintf("=== Cluster: %s ===\n", result.Context))
		output.WriteString(result.Output)
		if !strings.HasSuffix(result.Output, "\n") {
			output.WriteString("\n")
		}
		output.WriteString("\n")
	}

	return output.String()
}
