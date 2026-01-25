package troubleshoot

import (
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

//go:embed runbooks/*.yaml
var embeddedRunbooks embed.FS

// StepType represents the type of step (API or SSH).
type StepType string

const (
	StepTypeAPI StepType = "api"
	StepTypeSSH StepType = "ssh"
)

// Severity represents the severity level of an issue.
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// Runbook represents a troubleshooting runbook with steps.
type Runbook struct {
	ID          string   `yaml:"id"`
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Category    string   `yaml:"category"`
	Tags        []string `yaml:"tags"`
	Steps       []Step   `yaml:"steps"`
	RequiresSSH bool     `yaml:"requires_ssh"`
}

// Step represents a single troubleshooting step.
type Step struct {
	ID          string    `yaml:"id"`
	Name        string    `yaml:"name"`
	Description string    `yaml:"description"`
	Type        StepType  `yaml:"type"`
	Command     string    `yaml:"command"`  // For SSH steps
	APICall     string    `yaml:"api_call"` // For API steps
	Patterns    []Pattern `yaml:"patterns"`
	Required    bool      `yaml:"required"` // Stop on failure?
}

// Pattern represents a regex pattern to match in output.
type Pattern struct {
	ID          string   `yaml:"id"`
	Name        string   `yaml:"name"`
	Regex       string   `yaml:"regex"`
	Severity    Severity `yaml:"severity"`
	Message     string   `yaml:"message"`
	KBArticles  []string `yaml:"kb_articles"`
	Remediation string   `yaml:"remediation"`
}

// Registry holds all loaded runbooks.
type Registry struct {
	mu       sync.RWMutex
	runbooks map[string]*Runbook
}

// NewRegistry creates a new runbook registry.
func NewRegistry() *Registry {
	return &Registry{
		runbooks: make(map[string]*Runbook),
	}
}

// LoadEmbedded loads all embedded runbooks.
func (r *Registry) LoadEmbedded() error {
	return fs.WalkDir(embeddedRunbooks, "runbooks", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || filepath.Ext(path) != ".yaml" {
			return nil
		}

		data, err := embeddedRunbooks.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read embedded runbook %s: %w", path, err)
		}

		var runbook Runbook
		if err := yaml.Unmarshal(data, &runbook); err != nil {
			return fmt.Errorf("failed to parse runbook %s: %w", path, err)
		}

		r.Register(&runbook)
		return nil
	})
}

// LoadFromFile loads a runbook from a YAML file.
func (r *Registry) LoadFromFile(path string) error {
	data, err := fs.ReadFile(embeddedRunbooks, path)
	if err != nil {
		return fmt.Errorf("failed to read runbook file: %w", err)
	}

	var runbook Runbook
	if err := yaml.Unmarshal(data, &runbook); err != nil {
		return fmt.Errorf("failed to parse runbook: %w", err)
	}

	r.Register(&runbook)
	return nil
}

// Register adds a runbook to the registry.
func (r *Registry) Register(runbook *Runbook) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.runbooks[runbook.ID] = runbook
}

// Get retrieves a runbook by ID.
func (r *Registry) Get(id string) (*Runbook, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	runbook, ok := r.runbooks[id]
	return runbook, ok
}

// List returns all registered runbooks.
func (r *Registry) List() []*Runbook {
	r.mu.RLock()
	defer r.mu.RUnlock()

	runbooks := make([]*Runbook, 0, len(r.runbooks))
	for _, rb := range r.runbooks {
		runbooks = append(runbooks, rb)
	}
	return runbooks
}

// ListByCategory returns runbooks filtered by category.
func (r *Registry) ListByCategory(category string) []*Runbook {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var runbooks []*Runbook
	for _, rb := range r.runbooks {
		if rb.Category == category {
			runbooks = append(runbooks, rb)
		}
	}
	return runbooks
}

// Categories returns all unique categories.
func (r *Registry) Categories() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	categorySet := make(map[string]struct{})
	for _, rb := range r.runbooks {
		if rb.Category != "" {
			categorySet[rb.Category] = struct{}{}
		}
	}

	categories := make([]string, 0, len(categorySet))
	for cat := range categorySet {
		categories = append(categories, cat)
	}
	return categories
}

// Count returns the number of registered runbooks.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.runbooks)
}
