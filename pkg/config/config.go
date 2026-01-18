package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	DefaultConfigDir  = ".multikube"
	DefaultConfigFile = "config"
)

// MultiKubeConfig represents the multikubectl configuration
type MultiKubeConfig struct {
	// Contexts is the list of contexts to use
	Contexts []string `yaml:"contexts,omitempty"`
	// KubeConfig is the path to the kubeconfig file (optional)
	KubeConfig string `yaml:"kubeconfig,omitempty"`
}

// GetConfigPath returns the path to the multikube config file
func GetConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, DefaultConfigDir, DefaultConfigFile)
}

// GetConfigDir returns the path to the multikube config directory
func GetConfigDir() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, DefaultConfigDir)
}

// Exists checks if the multikube config file exists
func Exists() bool {
	_, err := os.Stat(GetConfigPath())
	return err == nil
}

// Load loads the multikube configuration from file
func Load() (*MultiKubeConfig, error) {
	configPath := GetConfigPath()

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &MultiKubeConfig{}, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var config MultiKubeConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

// Save saves the multikube configuration to file
func Save(config *MultiKubeConfig) error {
	configDir := GetConfigDir()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	configPath := GetConfigPath()
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// AddContext adds a context to the configuration
func (c *MultiKubeConfig) AddContext(context string) bool {
	for _, ctx := range c.Contexts {
		if ctx == context {
			return false // already exists
		}
	}
	c.Contexts = append(c.Contexts, context)
	return true
}

// RemoveContext removes a context from the configuration
func (c *MultiKubeConfig) RemoveContext(context string) bool {
	for i, ctx := range c.Contexts {
		if ctx == context {
			c.Contexts = append(c.Contexts[:i], c.Contexts[i+1:]...)
			return true
		}
	}
	return false
}

// HasContext checks if a context exists in the configuration
func (c *MultiKubeConfig) HasContext(context string) bool {
	for _, ctx := range c.Contexts {
		if ctx == context {
			return true
		}
	}
	return false
}

// Clear removes all contexts from the configuration
func (c *MultiKubeConfig) Clear() {
	c.Contexts = nil
}

// SetContexts sets the contexts to use
func (c *MultiKubeConfig) SetContexts(contexts []string) {
	c.Contexts = contexts
}
