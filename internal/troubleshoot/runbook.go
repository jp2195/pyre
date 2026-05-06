package troubleshoot

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"go.yaml.in/yaml/v4"
)

// defaultStepTimeout is the per-step timeout used when Step.Timeout is unset.
const defaultStepTimeout = 30 * time.Second

//go:embed runbooks/*.yaml
var embeddedRunbooks embed.FS

// StepType represents the type of step.
type StepType string

const (
	StepTypeAPI StepType = "api"
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
}

// Step represents a single troubleshooting step.
type Step struct {
	ID          string        `yaml:"id"`
	Name        string        `yaml:"name"`
	Description string        `yaml:"description"`
	Type        StepType      `yaml:"type"`
	APICall     string        `yaml:"api_call"` // For API steps
	Patterns    []Pattern     `yaml:"patterns"`
	Required    bool          `yaml:"required"`          // Stop on failure?
	Timeout     time.Duration `yaml:"timeout,omitempty"` // Per-step timeout; zero means defaultStepTimeout.
}

// effectiveTimeout returns Step.Timeout if positive, else defaultStepTimeout.
func (s *Step) effectiveTimeout() time.Duration {
	if s.Timeout > 0 {
		return s.Timeout
	}
	return defaultStepTimeout
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

		if err := runbook.Validate(); err != nil {
			return fmt.Errorf("invalid runbook %s: %w", path, err)
		}

		r.Register(&runbook)
		return nil
	})
}

// Validate checks that the runbook is structurally valid.
func (rb *Runbook) Validate() error {
	if rb.ID == "" {
		return fmt.Errorf("runbook ID must not be empty")
	}
	if rb.Name == "" {
		return fmt.Errorf("runbook %q: name must not be empty", rb.ID)
	}
	if len(rb.Steps) == 0 {
		return fmt.Errorf("runbook %q: must have at least one step", rb.ID)
	}
	for i, step := range rb.Steps {
		if err := step.Validate(); err != nil {
			return fmt.Errorf("runbook %q: step %d: %w", rb.ID, i, err)
		}
	}
	return nil
}

// validStepTypes enumerates the accepted Step.Type values.
var validStepTypes = map[StepType]struct{}{
	StepTypeAPI: {},
}

// validAPICalls lists the api_call values supported by Engine.executeAPIStep.
// Keep in lockstep with the switch in engine.go.
var validAPICalls = map[string]struct{}{
	"system_info":      {},
	"system_resources": {},
	"ha_status":        {},
	"session_info":     {},
}

// Validate checks that the step is structurally valid.
func (s *Step) Validate() error {
	if s.ID == "" {
		return fmt.Errorf("step ID must not be empty")
	}
	if _, ok := validStepTypes[s.Type]; !ok {
		return fmt.Errorf("step %q: unknown type %q", s.ID, s.Type)
	}
	if s.Type == StepTypeAPI {
		if s.APICall == "" {
			return fmt.Errorf("step %q: type=api requires non-empty api_call", s.ID)
		}
		if _, ok := validAPICalls[s.APICall]; !ok {
			return fmt.Errorf("step %q: unknown api_call %q", s.ID, s.APICall)
		}
	}
	for i, p := range s.Patterns {
		if _, err := regexp.Compile(p.Regex); err != nil {
			return fmt.Errorf("step %q: pattern %d (%q): invalid regex: %w", s.ID, i, p.ID, err)
		}
	}
	return nil
}

// LoadFromFile loads a runbook from a YAML file on the local filesystem.
func (r *Registry) LoadFromFile(path string) error {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return fmt.Errorf("failed to read runbook file: %w", err)
	}

	var runbook Runbook
	if err := yaml.Unmarshal(data, &runbook); err != nil {
		return fmt.Errorf("failed to parse runbook: %w", err)
	}

	if err := runbook.Validate(); err != nil {
		return fmt.Errorf("invalid runbook: %w", err)
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

// List returns all registered runbooks sorted by Name. Map iteration order
// is randomized, so we sort here to give callers a stable, deterministic
// ordering without forcing each one to re-sort.
func (r *Registry) List() []*Runbook {
	r.mu.RLock()
	defer r.mu.RUnlock()

	runbooks := make([]*Runbook, 0, len(r.runbooks))
	for _, rb := range r.runbooks {
		runbooks = append(runbooks, rb)
	}
	slices.SortFunc(runbooks, func(a, b *Runbook) int {
		return strings.Compare(a.Name, b.Name)
	})
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
