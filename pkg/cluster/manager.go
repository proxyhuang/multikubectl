package cluster

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// KubeConfig represents the structure of a kubeconfig file
type KubeConfig struct {
	APIVersion     string          `yaml:"apiVersion"`
	Kind           string          `yaml:"kind"`
	CurrentContext string          `yaml:"current-context"`
	Clusters       []ClusterEntry  `yaml:"clusters"`
	Contexts       []ContextEntry  `yaml:"contexts"`
	Users          []UserEntry     `yaml:"users"`
}

type ClusterEntry struct {
	Name    string  `yaml:"name"`
	Cluster Cluster `yaml:"cluster"`
}

type Cluster struct {
	Server                   string `yaml:"server"`
	CertificateAuthorityData string `yaml:"certificate-authority-data,omitempty"`
	CertificateAuthority     string `yaml:"certificate-authority,omitempty"`
	InsecureSkipTLSVerify    bool   `yaml:"insecure-skip-tls-verify,omitempty"`
}

type ContextEntry struct {
	Name    string  `yaml:"name"`
	Context Context `yaml:"context"`
}

type Context struct {
	Cluster   string `yaml:"cluster"`
	User      string `yaml:"user"`
	Namespace string `yaml:"namespace,omitempty"`
}

type UserEntry struct {
	Name string `yaml:"name"`
	User User   `yaml:"user"`
}

type User struct {
	ClientCertificateData string `yaml:"client-certificate-data,omitempty"`
	ClientKeyData         string `yaml:"client-key-data,omitempty"`
	ClientCertificate     string `yaml:"client-certificate,omitempty"`
	ClientKey             string `yaml:"client-key,omitempty"`
	Token                 string `yaml:"token,omitempty"`
}

// Manager manages multiple kubernetes clusters
type Manager struct {
	kubeConfigPath string
	config         *KubeConfig
}

// NewManager creates a new cluster manager
func NewManager(kubeConfigPath string) (*Manager, error) {
	if kubeConfigPath == "" {
		kubeConfigPath = getDefaultKubeConfigPath()
	}

	m := &Manager{
		kubeConfigPath: kubeConfigPath,
	}

	if err := m.loadConfig(); err != nil {
		return nil, err
	}

	return m, nil
}

func getDefaultKubeConfigPath() string {
	if envPath := os.Getenv("KUBECONFIG"); envPath != "" {
		return envPath
	}
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".kube", "config")
}

func (m *Manager) loadConfig() error {
	data, err := os.ReadFile(m.kubeConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read kubeconfig: %w", err)
	}

	var config KubeConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse kubeconfig: %w", err)
	}

	m.config = &config
	return nil
}

// GetContexts returns all available context names
func (m *Manager) GetContexts() []string {
	var contexts []string
	for _, ctx := range m.config.Contexts {
		contexts = append(contexts, ctx.Name)
	}
	return contexts
}

// GetCurrentContext returns the current context name
func (m *Manager) GetCurrentContext() string {
	return m.config.CurrentContext
}

// GetKubeConfigPath returns the kubeconfig file path
func (m *Manager) GetKubeConfigPath() string {
	return m.kubeConfigPath
}

// FilterContexts filters contexts based on the provided list
// If contexts is empty, returns all contexts
func (m *Manager) FilterContexts(contexts []string) []string {
	if len(contexts) == 0 {
		return m.GetContexts()
	}

	availableContexts := make(map[string]bool)
	for _, ctx := range m.config.Contexts {
		availableContexts[ctx.Name] = true
	}

	var filtered []string
	for _, ctx := range contexts {
		if availableContexts[ctx] {
			filtered = append(filtered, ctx)
		}
	}
	return filtered
}
