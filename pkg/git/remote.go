package git

import (
	"strings"
)

// Remote represents a git remote
type Remote struct {
	Name string
	URL  string
}

// AddRemote adds a new remote
func (g *Git) AddRemote(name, url string) error {
	_, err := g.exec("remote", "add", name, url)
	return err
}

// RemoveRemote removes a remote
func (g *Git) RemoveRemote(name string) error {
	_, err := g.exec("remote", "remove", name)
	return err
}

// ListRemotes returns all configured remotes
func (g *Git) ListRemotes() ([]Remote, error) {
	output, err := g.exec("remote", "-v")
	if err != nil {
		return nil, err
	}

	remotes := make(map[string]string)
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) >= 2 {
			remotes[parts[0]] = parts[1]
		}
	}

	var result []Remote
	for name, url := range remotes {
		result = append(result, Remote{Name: name, URL: url})
	}

	return result, nil
}

// Push pushes changes to a remote
func (g *Git) Push(remote, branch string) error {
	_, err := g.exec("push", remote, branch)
	return err
}

// Pull pulls changes from a remote
func (g *Git) Pull(remote, branch string) error {
	_, err := g.exec("pull", remote, branch)
	return err
}
