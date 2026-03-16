// File: tests/e2e/scenarios_sandboxing.go
package main

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/grovetools/tend/pkg/fs"
	"github.com/grovetools/tend/pkg/harness"
	"github.com/grovetools/tend/pkg/verify"
)

// EnvironmentSandboxingScenario tests the automatic environment sandboxing via ctx.Command()
func EnvironmentSandboxingScenario() *harness.Scenario {
	return harness.NewScenario(
		"environment-sandboxing",
		"Tests that ctx.Command() automatically sets HOME, XDG_* and XDG_RUNTIME_DIR environment variables.",
		[]string{"harness", "sandboxing", "context"},
		[]harness.Step{
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

				// Verify all environment variables are correctly sandboxed
				expectedHome := fmt.Sprintf("HOME=%s", ctx.HomeDir())
				expectedConfig := fmt.Sprintf("XDG_CONFIG_HOME=%s", ctx.ConfigDir())
				expectedData := fmt.Sprintf("XDG_DATA_HOME=%s", ctx.DataDir())
				expectedCache := fmt.Sprintf("XDG_CACHE_HOME=%s", ctx.CacheDir())
				expectedRuntime := fmt.Sprintf("XDG_RUNTIME_DIR=%s", ctx.RuntimeDir())

				return ctx.Verify(func(v *verify.Collector) {
					v.Contains("HOME variable is sandboxed", output, expectedHome)
					v.Contains("XDG_CONFIG_HOME is sandboxed", output, expectedConfig)
					v.Contains("XDG_DATA_HOME is sandboxed", output, expectedData)
					v.Contains("XDG_CACHE_HOME is sandboxed", output, expectedCache)
					v.Contains("XDG_RUNTIME_DIR is sandboxed", output, expectedRuntime)

					// Verify XDG_RUNTIME_DIR uses short path on non-Windows systems
					if runtime.GOOS != "windows" {
						v.True("XDG_RUNTIME_DIR starts with /tmp/", strings.HasPrefix(ctx.RuntimeDir(), "/tmp/"))
					}
				})
			}),
		},
	)
}
