package steps

import (
	"fmt"
	"time"
	
	"github.com/grovepm/grove-tend/internal/harness"
	"github.com/grovepm/grove-tend/pkg/retry"
	"github.com/grovepm/grove-tend/pkg/verify"
	"github.com/grovepm/grove-tend/pkg/wait"
)

// WaitFor creates a step that waits for a condition
func WaitFor(name string, condition wait.Condition, timeout time.Duration) harness.Step {
	return harness.NewStep(name, func(ctx *harness.Context) error {
		opts := wait.DefaultOptions()
		opts.Timeout = timeout
		return wait.For(condition, opts)
	})
}

// WaitForService creates a step that waits for a service to be running
func WaitForService(serviceName string) harness.Step {
	return harness.NewStep(
		fmt.Sprintf("Wait for service %s", serviceName),
		func(ctx *harness.Context) error {
			workDir := ctx.GetString("workdir")
			if workDir == "" {
				workDir = ctx.RootDir
			}
			
			return verify.ServiceRunning(ctx.GroveBinary, workDir, serviceName)
		},
	)
}

// RetryOnFailure creates a step that retries on failure
func RetryOnFailure(name string, fn harness.StepFunc, attempts int) harness.Step {
	return harness.NewStep(name, func(ctx *harness.Context) error {
		opts := retry.DefaultOptions()
		opts.MaxAttempts = attempts
		
		return retry.Do(func() error {
			return fn(ctx)
		}, opts)
	})
}

// AssertCondition creates a step that asserts a condition
func AssertCondition(name string, condition func(*harness.Context) error) harness.Step {
	return harness.NewStep(name, condition)
}

// AssertEventually creates a step that waits for an assertion to pass
func AssertEventually(name string, assertion func() error, timeout time.Duration) harness.Step {
	return harness.NewStep(name, func(ctx *harness.Context) error {
		return wait.For(func() (bool, error) {
			err := assertion()
			return err == nil, nil
		}, wait.Options{
			Timeout:      timeout,
			PollInterval: 500 * time.Millisecond,
		})
	})
}

// VerifyEndpoint creates a step that verifies an HTTP endpoint
func VerifyEndpoint(url string, expectedStatus int) harness.Step {
	return harness.NewStep(
		fmt.Sprintf("Verify %s returns %d", url, expectedStatus),
		func(ctx *harness.Context) error {
			return wait.ForHTTP(url, expectedStatus, 30*time.Second)
		},
	)
}

// VerifyNoErrors creates a step that checks for errors in logs
func VerifyNoErrors(containerName string, errorPatterns ...string) harness.Step {
	return harness.NewStep(
		fmt.Sprintf("Verify no errors in %s logs", containerName),
		func(ctx *harness.Context) error {
			if len(errorPatterns) == 0 {
				errorPatterns = []string{"ERROR", "FATAL", "panic"}
			}
			return verify.NoErrors(containerName, errorPatterns)
		},
	)
}