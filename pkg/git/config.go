package git

import "fmt"

// SetConfig sets a git configuration value
func (g *Git) SetConfig(key, value string) error {
	_, err := g.exec("config", key, value)
	return err
}

// GetConfig gets a git configuration value
func (g *Git) GetConfig(key string) (string, error) {
	return g.exec("config", "--get", key)
}

// SetUser sets the user name and email for the repository
func (g *Git) SetUser(name, email string) error {
	if err := g.SetConfig("user.name", name); err != nil {
		return fmt.Errorf("setting user.name: %w", err)
	}

	if err := g.SetConfig("user.email", email); err != nil {
		return fmt.Errorf("setting user.email: %w", err)
	}

	return nil
}

// SetupTestRepo configures a repo with test defaults
func SetupTestRepo(dir string) (*Git, error) {
	// Initialize the repository
	if err := Init(dir); err != nil {
		return nil, fmt.Errorf("initializing repo: %w", err)
	}

	g := New(dir)

	// Set test user
	if err := g.SetUser("Test User", "test@example.com"); err != nil {
		return nil, fmt.Errorf("setting test user: %w", err)
	}

	// Disable GPG signing for tests
	if err := g.SetConfig("commit.gpgsign", "false"); err != nil {
		return nil, fmt.Errorf("disabling gpg signing: %w", err)
	}

	return g, nil
}
