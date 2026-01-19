package demo

import (
	"fmt"
	"sort"
)

// Registry manages available demo specifications.
type Registry struct {
	specs map[string]DemoSpec
}

// defaultRegistry is the global registry of demo specifications.
var defaultRegistry = &Registry{
	specs: make(map[string]DemoSpec),
}

// Register adds a demo specification to the global registry.
// This is typically called in init() functions of demo spec implementations.
func Register(spec DemoSpec) {
	defaultRegistry.specs[spec.Name()] = spec
}

// Get retrieves a demo specification by name.
// Returns an error if the demo name is not found.
func Get(name string) (DemoSpec, error) {
	spec, ok := defaultRegistry.specs[name]
	if !ok {
		return nil, fmt.Errorf("unknown demo: %s (available: %v)", name, List())
	}
	return spec, nil
}

// List returns a sorted list of all registered demo names.
func List() []string {
	names := make([]string, 0, len(defaultRegistry.specs))
	for name := range defaultRegistry.specs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
