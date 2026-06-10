package demo

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestFilterOverrideConfig(t *testing.T) {
	full := map[string]interface{}{
		"anthropic": map[string]interface{}{
			"api_key": "FAKE-abc123",
			"model":   "some-model",
			"other":   "should-not-copy",
		},
		"gemini": map[string]interface{}{
			"api_key_command": "echo FAKE",
		},
		"agent": map[string]interface{}{
			"args": "--verbose",
		},
		// Must never leak into the demo
		"groves":            map[string]interface{}{"real": map[string]interface{}{"path": "/home/user/code"}},
		"explicit_projects": []interface{}{"/home/user/code/secret"},
		"notebooks":         map[string]interface{}{"definitions": map[string]interface{}{}},
	}

	filtered, copied := filterOverrideConfig(full)

	wantKeys := []string{"anthropic.api_key", "gemini.api_key_command", "agent.args"}
	if !reflect.DeepEqual(copied, wantKeys) {
		t.Errorf("copied keys = %v, want %v", copied, wantKeys)
	}

	anthropic, ok := filtered["anthropic"].(map[string]interface{})
	if !ok {
		t.Fatalf("anthropic section missing from filtered config")
	}
	if anthropic["api_key"] != "FAKE-abc123" {
		t.Errorf("anthropic.api_key not copied")
	}
	if _, ok := anthropic["model"]; ok {
		t.Errorf("anthropic.model copied; provider sections must be narrowed to credential keys")
	}
	if _, ok := anthropic["other"]; ok {
		t.Errorf("anthropic.other copied; provider sections must be narrowed to credential keys")
	}
	for _, forbidden := range []string{"groves", "explicit_projects", "notebooks"} {
		if _, ok := filtered[forbidden]; ok {
			t.Errorf("%s leaked into filtered config", forbidden)
		}
	}
}

func TestFilterOverrideConfigEmpty(t *testing.T) {
	filtered, copied := filterOverrideConfig(map[string]interface{}{
		"groves": map[string]interface{}{},
	})
	if len(filtered) != 0 {
		t.Errorf("expected empty filtered config, got %v", filtered)
	}
	if len(copied) != 0 {
		t.Errorf("expected no copied keys, got %v", copied)
	}
}

// TestCopyUserOverride exercises the real copy path against a scratch HOME
// containing only FAKE values.
func TestCopyUserOverride(t *testing.T) {
	fakeHome := t.TempDir()
	t.Setenv("HOME", fakeHome)

	srcDir := filepath.Join(fakeHome, ".config", "grove")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(srcDir, "grove.override.yml")
	overrideYAML := `gemini:
  api_key: abc123
  model: fake-model
groves:
  real:
    path: /home/user/code
`
	if err := os.WriteFile(src, []byte(overrideYAML), 0o600); err != nil {
		t.Fatal(err)
	}

	demoDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(demoDir, "config", "grove"), 0o755); err != nil {
		t.Fatal(err)
	}

	g := &Generator{rootDir: demoDir, demoName: "test", copyCredentials: true}
	cred, err := g.copyUserOverride()
	if err != nil {
		t.Fatalf("copyUserOverride: %v", err)
	}
	if cred == nil {
		t.Fatal("expected a CredentialCopy record, got nil")
	}
	if cred.SourcePath != src {
		t.Errorf("SourcePath = %q, want %q", cred.SourcePath, src)
	}
	if want := []string{"gemini.api_key"}; !reflect.DeepEqual(cred.Keys, want) {
		t.Errorf("Keys = %v, want %v", cred.Keys, want)
	}

	dest := filepath.Join(demoDir, "config", "grove", "grove.override.yml")
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("reading demo override: %v", err)
	}
	var out map[string]interface{}
	if err := yaml.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	gemini, ok := out["gemini"].(map[string]interface{})
	if !ok || gemini["api_key"] != "abc123" {
		t.Errorf("demo override missing gemini.api_key: %s", data)
	}
	if _, ok := gemini["model"]; ok {
		t.Errorf("gemini.model copied; should be narrowed out")
	}
	if _, ok := out["groves"]; ok {
		t.Errorf("groves leaked into demo override: %s", data)
	}

	info, err := os.Stat(dest)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("demo override perms = %o, want 0600", perm)
	}
}

// TestCopyUserOverrideNoSource verifies nothing is written or reported when
// the user has no override file.
func TestCopyUserOverrideNoSource(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	demoDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(demoDir, "config", "grove"), 0o755); err != nil {
		t.Fatal(err)
	}

	g := &Generator{rootDir: demoDir, demoName: "test", copyCredentials: true}
	cred, err := g.copyUserOverride()
	if err != nil {
		t.Fatalf("copyUserOverride: %v", err)
	}
	if cred != nil {
		t.Errorf("expected nil CredentialCopy, got %+v", cred)
	}
	if _, err := os.Stat(filepath.Join(demoDir, "config", "grove", "grove.override.yml")); !os.IsNotExist(err) {
		t.Errorf("demo override should not exist when there is no source")
	}
}

// TestWithoutCredentialsOption verifies the option flips the generator flag
// that Generate uses to skip copyUserOverride entirely.
func TestWithoutCredentialsOption(t *testing.T) {
	g, err := NewGenerator(t.TempDir(), "homelab", WithoutCredentials())
	if err != nil {
		t.Fatal(err)
	}
	if g.copyCredentials {
		t.Error("WithoutCredentials() did not disable credential copying")
	}

	g2, err := NewGenerator(t.TempDir(), "homelab")
	if err != nil {
		t.Fatal(err)
	}
	if !g2.copyCredentials {
		t.Error("credential copying should default to enabled")
	}
}
