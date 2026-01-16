package app

import (
	"context"

	"github.com/grovetools/core/cli"
	"github.com/grovetools/tend/internal/cmd"
	"github.com/grovetools/tend/pkg/harness"
	"github.com/spf13/cobra"
)

// New creates the root cobra command for a tend application, configured with the provided scenarios.
func New(scenarios []*harness.Scenario) *cobra.Command {
	return cmd.NewRootCmd(scenarios)
}

// Execute creates the app and executes it.
func Execute(ctx context.Context, scenarios []*harness.Scenario) error {
	app := New(scenarios)
	return cli.ExecuteContext(ctx, app)
}