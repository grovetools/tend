// File: tests/e2e/scenarios_sandboxing.go
package main

import (
	"fmt"

	"github.com/mattsolo1/grove-tend/pkg/assert"
	"github.com/mattsolo1/grove-tend/pkg/fs"
	"github.com/mattsolo1/grove-tend/pkg/harness"
)

// EnvironmentSandboxingScenario tests the automatic environment sandboxing via ctx.Command()
func EnvironmentSandboxingScenario() *harness.Scenario {
	return &harness.Scenario{
		Name:        "environment-sandboxing",
		Description: "Tests that ctx.Command() automatically sets HOME and XDG_* environment variables.",
		Tags:        []string{"harness", "sandboxing", "context"},
		Steps: []harness.Step{
			harness.SetupMocks(
				harness.Mock{CommandName: "print-env"},
			),
			harness.NewStep("Create a basic grove.yml", func(ctx *harness.Context) error {
				return fs.WriteBasicGroveConfig(ctx.RootDir)
			}),
			harness.NewStep("Run mock command and verify sandboxed environment", func(ctx *harness.Context) error {
				cmd := ctx.Command("print-env")
				result := cmd.Run()
				ctx.ShowCommandOutput(cmd.String(), result.Stdout, result.Stderr)
				if result.Error != nil {
					return result.Error
				}

				output := result.Stdout

				// Assert that HOME is set to the sandboxed directory
				expectedHome := fmt.Sprintf("HOME=%s", ctx.HomeDir())
				if err := assert.Contains(output, expectedHome, "HOME variable was not correctly sandboxed"); err != nil {
					return err
				}

				// Assert XDG_CONFIG_HOME
				expectedConfig := fmt.Sprintf("XDG_CONFIG_HOME=%s", ctx.ConfigDir())
				if err := assert.Contains(output, expectedConfig, "XDG_CONFIG_HOME was not correctly sandboxed"); err != nil {
					return err
				}

				// Assert XDG_DATA_HOME
				expectedData := fmt.Sprintf("XDG_DATA_HOME=%s", ctx.DataDir())
				if err := assert.Contains(output, expectedData, "XDG_DATA_HOME was not correctly sandboxed"); err != nil {
					return err
				}

				// Assert XDG_CACHE_HOME
				expectedCache := fmt.Sprintf("XDG_CACHE_HOME=%s", ctx.CacheDir())
				if err := assert.Contains(output, expectedCache, "XDG_CACHE_HOME was not correctly sandboxed"); err != nil {
					return err
				}

				return nil
			}),
		},
	}
}
