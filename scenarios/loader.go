package scenarios

import (
	"fmt"
	"sort"

	"github.com/grovepm/grove-tend/internal/harness"
	"github.com/grovepm/grove-tend/scenarios/agent"
)

// Loader handles loading and managing test scenarios
type Loader struct {
	scenarioDir string
	scenarios   map[string]*harness.Scenario
}

// NewLoader creates a new scenario loader
func NewLoader(scenarioDir string) *Loader {
	return &Loader{
		scenarioDir: scenarioDir,
		scenarios:   make(map[string]*harness.Scenario),
	}
}

// LoadAll loads all available scenarios
func (l *Loader) LoadAll() ([]*harness.Scenario, error) {
	// Register built-in scenarios
	l.registerBuiltinScenarios()
	
	// TODO: In the future, we can also load scenarios from files
	// if we decide to support external scenario definitions
	
	// Convert map to sorted slice
	var scenarios []*harness.Scenario
	var names []string
	
	for name := range l.scenarios {
		names = append(names, name)
	}
	sort.Strings(names)
	
	for _, name := range names {
		scenarios = append(scenarios, l.scenarios[name])
	}
	
	return scenarios, nil
}

// LoadByName loads a specific scenario by name
func (l *Loader) LoadByName(name string) (*harness.Scenario, error) {
	l.registerBuiltinScenarios()
	
	scenario, exists := l.scenarios[name]
	if !exists {
		return nil, fmt.Errorf("scenario '%s' not found", name)
	}
	
	return scenario, nil
}

// GetAvailableNames returns the names of all available scenarios
func (l *Loader) GetAvailableNames() []string {
	l.registerBuiltinScenarios()
	
	var names []string
	for name := range l.scenarios {
		names = append(names, name)
	}
	sort.Strings(names)
	
	return names
}

// registerBuiltinScenarios registers all built-in scenarios
func (l *Loader) registerBuiltinScenarios() {
	// Only register once
	if len(l.scenarios) > 0 {
		return
	}
	
	// Register example scenarios
	l.scenarios["example-basic"] = ExampleBasicScenario()
	l.scenarios["example-git"] = ExampleGitScenario()
	l.scenarios["example-command"] = ExampleCommandScenario()
	l.scenarios["example-grove-version"] = ExampleGroveVersionScenario()
	
	// Register agent scenarios
	l.scenarios["agent-isolation"] = agent.AgentIsolationScenario
}