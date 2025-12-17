package tui

// StartConfig holds configuration for launching a TUI session.
type StartConfig struct {
	Env []string
	Cwd string
}

// StartOption is a function that configures a StartConfig.
type StartOption func(*StartConfig)

// WithEnv sets environment variables for the TUI session.
// Variables should be in KEY=VALUE format.
func WithEnv(env ...string) StartOption {
	return func(c *StartConfig) {
		c.Env = append(c.Env, env...)
	}
}

// WithCwd sets the working directory for the TUI session.
func WithCwd(cwd string) StartOption {
	return func(c *StartConfig) {
		c.Cwd = cwd
	}
}
