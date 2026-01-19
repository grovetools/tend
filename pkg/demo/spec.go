package demo

// DemoSpec defines the interface for demo specifications.
// Each demo type (homelab, bake-off, etc.) implements this interface
// to define how it generates its content.
type DemoSpec interface {
	// Name returns the unique identifier for this demo type.
	Name() string

	// Description returns a human-readable description of the demo.
	Description() string

	// Generate creates the demo content in the specified root directory.
	// Returns metadata about the created ecosystems and configuration.
	Generate(rootDir string) (*DemoContent, error)
}

// DemoContent contains the generated content from a demo specification.
// This is returned by DemoSpec.Generate() and used by the Generator
// to create the final demo environment.
type DemoContent struct {
	// Ecosystems contains metadata about each ecosystem created.
	Ecosystems []EcosystemMeta

	// NotebookDirs lists the notebook directories created (optional).
	NotebookDirs []string

	// TmuxNeeded indicates whether this demo requires a tmux session.
	TmuxNeeded bool

	// CustomData allows specs to pass additional data to the generator (optional).
	CustomData map[string]interface{}
}
