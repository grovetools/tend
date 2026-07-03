package demo

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/grovetools/core/config"
	"github.com/grovetools/core/pkg/workspace"
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

// TestBuildCmdEnvStripsGroveScope verifies delegated generation commands do
// not inherit the caller's pinned scope: treemux sets GROVE_SCOPE
// process-wide, and a leaked scope would point the demo's grove/nb/flow
// calls at the host's scoped daemon instead of the isolated GROVE_HOME.
func TestBuildCmdEnvStripsGroveScope(t *testing.T) {
	t.Setenv("GROVE_SCOPE", "/home/user/real-ecosystem")
	root := t.TempDir()
	g := &homelabGenerator{rootDir: root, tmuxSocket: SocketName("homelab")}

	env := g.buildCmdEnv()
	var sawHome bool
	for _, kv := range env {
		if strings.HasPrefix(kv, "GROVE_SCOPE=") {
			t.Errorf("GROVE_SCOPE leaked into delegated command env: %s", kv)
		}
		if kv == "GROVE_HOME="+root {
			sawHome = true
		}
	}
	if !sawHome {
		t.Error("demo overlay GROVE_HOME missing from delegated command env")
	}
}

// TestWithoutMuxSkipsMuxPreflight verifies that WithoutMux drops the mux
// binary from the preflight requirements (the embedding caller is the
// terminal) while the delegated CLIs are still required.
func TestWithoutMuxSkipsMuxPreflight(t *testing.T) {
	// A PATH/bin dir with grove/nb/flow stubs but no tmux/tuimux.
	binDir := t.TempDir()
	for _, tool := range []string{"grove", "nb", "flow"} {
		if err := os.WriteFile(filepath.Join(binDir, tool), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", binDir)
	t.Setenv("GROVE_BIN", binDir)

	withMux, err := NewGenerator(t.TempDir(), "homelab")
	if err != nil {
		t.Fatal(err)
	}
	if err := withMux.Preflight(); err == nil {
		t.Error("Preflight without WithoutMux should require a mux binary")
	}

	withoutMux, err := NewGenerator(t.TempDir(), "homelab", WithoutMux())
	if err != nil {
		t.Fatal(err)
	}
	if err := withoutMux.Preflight(); err != nil {
		t.Errorf("Preflight with WithoutMux should not require a mux binary: %v", err)
	}
	if !withoutMux.withoutMux {
		t.Error("WithoutMux() did not set the generator flag")
	}
}

// TestWithProgressReportsSteps verifies the option wires the callback and
// reportStep invokes it; a nil callback is a safe no-op.
func TestWithProgressReportsSteps(t *testing.T) {
	var steps []string
	g, err := NewGenerator(t.TempDir(), "homelab", WithProgress(func(s string) { steps = append(steps, s) }))
	if err != nil {
		t.Fatal(err)
	}
	g.reportStep("one")
	g.reportStep("two")
	if len(steps) != 2 || steps[0] != "one" || steps[1] != "two" {
		t.Errorf("reportStep did not reach the WithProgress callback: %v", steps)
	}

	plain, err := NewGenerator(t.TempDir(), "homelab")
	if err != nil {
		t.Fatal(err)
	}
	plain.reportStep("no-op") // must not panic with no callback
}

// assertNoVersionKey asserts the emitted file carries no legacy top-level
// version key. Checked on the raw bytes because the loader's SetDefaults
// backfills Version="1.0" on the parsed struct either way.
func assertNoVersionKey(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "version") {
			t.Errorf("emitted config still carries legacy version key: %q", line)
		}
	}
}

// demoTestEcosystems returns a representative ecosystem set for config tests.
func demoTestEcosystems(root string) []EcosystemMeta {
	return []EcosystemMeta{
		{Name: "homelab", Path: filepath.Join(root, "ecosystems", "homelab"), RepoCount: 8, Description: "Main homelab ecosystem"},
		{Name: "contrib", Path: filepath.Join(root, "ecosystems", "contrib"), RepoCount: 3, Description: "Community contributions"},
		{Name: "infra", Path: filepath.Join(root, "ecosystems", "infra"), RepoCount: 2, Description: "Infrastructure and deployment"},
	}
}

// assertDemoGlobalConfig loads the emitted global config through core's real
// loader (parse + schema validation) and asserts the modern schema: one
// [groves.<eco>] source and one [notebooks.definitions.<eco>] with root_dir
// per ecosystem, and NO legacy version/workspaces.paths remnants.
func assertDemoGlobalConfig(t *testing.T, root, path string) *config.Config {
	t.Helper()

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("core config loader rejected emitted demo config: %v", err)
	}

	assertNoVersionKey(t, path)

	for _, eco := range demoTestEcosystems(root) {
		grove, ok := cfg.Groves[eco.Name]
		if !ok {
			t.Errorf("groves.%s missing from emitted config", eco.Name)
			continue
		}
		if grove.Path != eco.Path {
			t.Errorf("groves.%s.path = %q, want %q", eco.Name, grove.Path, eco.Path)
		}
		if grove.Notebook != eco.Name {
			t.Errorf("groves.%s.notebook = %q, want %q", eco.Name, grove.Notebook, eco.Name)
		}
		if grove.Enabled == nil || !*grove.Enabled {
			t.Errorf("groves.%s.enabled should be true", eco.Name)
		}
		if grove.Description != eco.Description {
			t.Errorf("groves.%s.description = %q, want %q", eco.Name, grove.Description, eco.Description)
		}

		if cfg.Notebooks == nil || cfg.Notebooks.Definitions == nil {
			t.Fatal("notebooks.definitions missing from emitted config")
		}
		nb, ok := cfg.Notebooks.Definitions[eco.Name]
		if !ok || nb == nil {
			t.Errorf("notebooks.definitions.%s missing from emitted config", eco.Name)
			continue
		}
		wantRoot := filepath.Join(root, "notebooks", eco.Name)
		if nb.RootDir != wantRoot {
			t.Errorf("notebooks.definitions.%s.root_dir = %q, want %q", eco.Name, nb.RootDir, wantRoot)
		}
	}

	return cfg
}

// TestCreateGlobalConfigModernTOML verifies the generator's overlay config is
// modern grove.toml that core parses fully, and — the bug-fix property — that
// the default notebook path templates resolve to exactly where the demo seeds
// its notes (<root_dir>/workspaces/<eco>/<category>). The legacy YAML emission
// relied on a notebooks...workspaces.<eco>.paths block that modern config
// parsing silently dropped, leaving the seeded notes invisible to nb.
func TestCreateGlobalConfigModernTOML(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "config", "grove"), 0o755); err != nil {
		t.Fatal(err)
	}

	g := &Generator{rootDir: root, demoName: "homelab"}
	content := &DemoContent{Ecosystems: demoTestEcosystems(root)}
	if err := g.createGlobalConfig(content); err != nil {
		t.Fatalf("createGlobalConfig: %v", err)
	}

	if base := filepath.Base(g.configPath()); base != "grove.toml" {
		t.Errorf("config filename = %q, want grove.toml", base)
	}

	cfg := assertDemoGlobalConfig(t, root, g.configPath())
	if cfg.Name != "grove-demo-homelab" {
		t.Errorf("config name = %q, want grove-demo-homelab", cfg.Name)
	}

	// Bug-fix property: notes seeded at notebooks/<eco>/workspaces/<eco>/<cat>
	// must be exactly what the locator resolves for the ecosystem node using
	// only root_dir + default templates.
	locator := workspace.NewNotebookLocator(cfg)
	node := &workspace.WorkspaceNode{
		Name:         "homelab",
		Path:         filepath.Join(root, "ecosystems", "homelab"),
		Kind:         workspace.KindEcosystemRoot,
		NotebookName: "homelab",
	}
	for _, category := range []string{"inbox", "concepts", "learn"} {
		got, err := locator.GetNotesDir(node, category)
		if err != nil {
			t.Fatalf("GetNotesDir(%s): %v", category, err)
		}
		want := filepath.Join(root, "notebooks", "homelab", "workspaces", "homelab", category)
		if got != want {
			t.Errorf("notes dir for %s = %q, want seeded location %q", category, got, want)
		}
	}
	plans, err := locator.GetPlansDir(node)
	if err != nil {
		t.Fatalf("GetPlansDir: %v", err)
	}
	if want := filepath.Join(root, "notebooks", "homelab", "workspaces", "homelab", "plans"); plans != want {
		t.Errorf("plans dir = %q, want seeded location %q", plans, want)
	}
}

// TestHomelabCreateConfigModernTOML verifies the mid-generation config written
// before notebook seeding (which delegated nb/flow calls resolve against) has
// the same modern shape.
func TestHomelabCreateConfigModernTOML(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "config", "grove"), 0o755); err != nil {
		t.Fatal(err)
	}

	g := &homelabGenerator{rootDir: root, tmuxSocket: SocketName("homelab")}
	if err := g.createConfig(demoTestEcosystems(root)); err != nil {
		t.Fatalf("createConfig: %v", err)
	}

	if base := filepath.Base(g.configPath()); base != "grove.toml" {
		t.Errorf("config filename = %q, want grove.toml", base)
	}
	assertDemoGlobalConfig(t, root, g.configPath())
}

// TestCreateEmptyConfigModernTOML verifies the pre-generation placeholder
// config is valid modern TOML that core's loader accepts.
func TestCreateEmptyConfigModernTOML(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "config", "grove"), 0o755); err != nil {
		t.Fatal(err)
	}

	g := &Generator{rootDir: root, demoName: "homelab"}
	if err := g.createEmptyConfig(); err != nil {
		t.Fatalf("createEmptyConfig: %v", err)
	}

	cfg, err := config.Load(g.configPath())
	if err != nil {
		t.Fatalf("core config loader rejected empty demo config: %v", err)
	}
	if len(cfg.Groves) != 0 {
		t.Errorf("empty config should have no groves, got %v", cfg.Groves)
	}
}

// TestWriteEcosystemConfigModernTOML verifies per-ecosystem configs are
// grove.toml with name + workspaces and no legacy version field.
func TestWriteEcosystemConfigModernTOML(t *testing.T) {
	ecoDir := t.TempDir()
	g := &homelabGenerator{rootDir: t.TempDir(), tmuxSocket: SocketName("homelab")}

	workspaces := []string{"dashboard", "sentinel", "vault"}
	if err := g.writeEcosystemConfig(ecoDir, "homelab", workspaces); err != nil {
		t.Fatalf("writeEcosystemConfig: %v", err)
	}

	if _, err := os.Stat(filepath.Join(ecoDir, "grove.yml")); !os.IsNotExist(err) {
		t.Error("legacy grove.yml should no longer be written")
	}

	path := filepath.Join(ecoDir, "grove.toml")
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("core config loader rejected ecosystem config: %v", err)
	}
	if cfg.Name != "homelab" {
		t.Errorf("ecosystem name = %q, want homelab", cfg.Name)
	}
	if !reflect.DeepEqual(cfg.Workspaces, workspaces) {
		t.Errorf("workspaces = %v, want %v", cfg.Workspaces, workspaces)
	}
	assertNoVersionKey(t, path)

	// Discovery must recognize the ecosystem via its grove.toml.
	if found := config.FindEcosystemConfig(ecoDir); found != path {
		t.Errorf("FindEcosystemConfig(%q) = %q, want %q", ecoDir, found, path)
	}
}
