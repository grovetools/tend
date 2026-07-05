package demo

// DemoSpec defines the interface for demo specifications.
// Each demo type (homelab, bake-off, etc.) implements this interface
// to define how it generates its content.
//
// Invariant: a DemoSpec must NEVER emit a [tui] section in any config file it
// generates (the demo's grove.toml or any ecosystem/project grove.toml). The
// user's real [tui] choices are synced into the demo's grove.override.yml by
// SyncUserTUIConfig; because that override is consumed as the global override
// layer while ecosystem/project configs merge on top of it, any [tui] a spec
// wrote would beat the synced values. Enforced by tests (see generator_test.go).
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
