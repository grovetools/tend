// File: tests/e2e/scenarios_assertions.go
package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/mattsolo1/grove-tend/pkg/assert"
	"github.com/mattsolo1/grove-tend/pkg/harness"
	"github.com/mattsolo1/grove-tend/pkg/verify"
)

// HardAssertionPassScenario tests that ctx.Check passes when assertions succeed
func HardAssertionPassScenario() *harness.Scenario {
	return harness.NewScenario(
		"hard-assertion-pass",
		"Tests that ctx.Check passes when all assertions succeed",
		[]string{"assertions", "smoke"},
		[]harness.Step{
			harness.NewStep("Perform successful hard assertions", func(ctx *harness.Context) error {
				if err := ctx.Check("first check is true", assert.True(true)); err != nil {
					return err
				}
				if err := ctx.Check("second check is equal", assert.Equal(1, 1)); err != nil {
					return err
				}
				if err := ctx.Check("third check contains substring", assert.Contains("hello world", "world")); err != nil {
					return err
				}
				return nil
			}),
		},
	)
}

// HardAssertionFailScenario tests that ctx.Check properly wraps errors with descriptions
func HardAssertionFailScenario() *harness.Scenario {
	return harness.NewScenario(
		"hard-assertion-fail",
		"Tests that ctx.Check properly wraps errors with descriptions when assertions fail",
		[]string{"assertions"},
		[]harness.Step{
			harness.NewStep("Verify error wrapping behavior", func(ctx *harness.Context) error {
				// First, do a successful check
				if err := ctx.Check("this check should pass", assert.True(true)); err != nil {
					return fmt.Errorf("a successful check unexpectedly returned an error: %w", err)
				}

				// Now test that a failing check returns a properly wrapped error
				err := ctx.Check("this check should fail", assert.True(false, "expected true but got false"))
				if err == nil {
					return errors.New("expected ctx.Check to return an error for a failing assertion, but got nil")
				}

				// Verify the error contains the description
				if !strings.Contains(err.Error(), "this check should fail") {
					return fmt.Errorf("error should contain the check description, got: %s", err.Error())
				}

				// Verify subsequent code can run (we're testing error wrapping, not actual failure)
				ctx.Set("error_validated", true)

				// This step intentionally validates error behavior, so we return nil to pass
				return nil
			}),
		},
	)
}

// SoftAssertionPassScenario tests that ctx.Verify passes when all assertions succeed
func SoftAssertionPassScenario() *harness.Scenario {
	return harness.NewScenario(
		"soft-assertion-pass",
		"Tests that ctx.Verify passes when all assertions succeed",
		[]string{"assertions"},
		[]harness.Step{
			harness.NewStep("Perform multiple successful soft assertions", func(ctx *harness.Context) error {
				return ctx.Verify(func(v *verify.Collector) {
					v.True("first check is true", true)
					v.Equal("second check is equal", "hello", "hello")
					v.Contains("third check contains substring", "hello world", "world")
					v.NotContains("fourth check does not contain substring", "hello world", "goodbye")
				})
			}),
		},
	)
}

// SoftAssertionFailScenario tests that ctx.Verify collects and reports multiple failures
func SoftAssertionFailScenario() *harness.Scenario {
	return harness.NewScenario(
		"soft-assertion-fail",
		"Tests that ctx.Verify collects and reports multiple failures correctly",
		[]string{"assertions"},
		[]harness.Step{
			harness.NewStep("Perform a mix of passing and failing soft assertions", func(ctx *harness.Context) error {
				err := ctx.Verify(func(v *verify.Collector) {
					v.True("this soft check should pass", true)
					v.Equal("this soft check should fail", "a", "b")
					v.Equal("this is another passing check", 1, 1)
					v.Contains("this is another failing check", "hello", "world")
				})

				// The step should fail, so an error is expected
				if err == nil {
					return errors.New("expected ctx.Verify to return an error, but it was nil")
				}

				// Check that the error is the correct type for aggregated failures
				verifyErr, ok := err.(*verify.VerificationError)
				if !ok {
					return fmt.Errorf("expected error of type *verify.VerificationError, but got %T", err)
				}

				// Check that we have exactly 2 failures
				if len(verifyErr.Errors) != 2 {
					return fmt.Errorf("expected 2 failures, but got %d", len(verifyErr.Errors))
				}

				// Check the content of the aggregated error message
				errMsg := err.Error()
				if err := assert.Contains(errMsg, "2 assertion(s) failed:"); err != nil {
					return fmt.Errorf("error message should report 2 failures: %w", err)
				}
				if err := assert.Contains(errMsg, "this soft check should fail"); err != nil {
					return fmt.Errorf("error message should contain first failure description: %w", err)
				}
				if err := assert.Contains(errMsg, "this is another failing check"); err != nil {
					return fmt.Errorf("error message should contain second failure description: %w", err)
				}

				// This step validates the error behavior, so we return nil to indicate success
				return nil
			}),
		},
	)
}

// MixedAssertionsScenario tests using both hard and soft assertions in the same step
func MixedAssertionsScenario() *harness.Scenario {
	return harness.NewScenario(
		"mixed-assertions",
		"Tests using both ctx.Check and ctx.Verify in the same step",
		[]string{"assertions"},
		[]harness.Step{
			harness.NewStep("Use hard assertion for critical check", func(ctx *harness.Context) error {
				// Critical precondition check
				if err := ctx.Check("critical precondition is met", assert.True(true)); err != nil {
					return err
				}

				// Now verify multiple related properties
				return ctx.Verify(func(v *verify.Collector) {
					v.Equal("property A matches", 1, 1)
					v.Equal("property B matches", "test", "test")
					v.True("property C is true", true)
				})
			}),
		},
	)
}

// AssertionDescriptionsScenario tests that assertion descriptions are meaningful
func AssertionDescriptionsScenario() *harness.Scenario {
	return harness.NewScenario(
		"assertion-descriptions",
		"Tests that assertion descriptions provide clear context in errors",
		[]string{"assertions"},
		[]harness.Step{
			harness.NewStep("Validate error messages contain descriptions", func(ctx *harness.Context) error {
				// Test that the description is included in the error
				err := ctx.Check("custom description for this check", assert.Equal(1, 2))
				if err == nil {
					return errors.New("expected check to fail but it passed")
				}

				// Verify the description is in the error message
				if !strings.Contains(err.Error(), "custom description for this check") {
					return fmt.Errorf("error message should contain the description, got: %s", err.Error())
				}

				// This step validates error messages, so return nil
				return nil
			}),
		},
	)
}

// MultipleVerifyBlocksScenario tests multiple ctx.Verify blocks in sequence
func MultipleVerifyBlocksScenario() *harness.Scenario {
	return harness.NewScenario(
		"multiple-verify-blocks",
		"Tests using multiple ctx.Verify blocks in sequence",
		[]string{"assertions"},
		[]harness.Step{
			harness.NewStep("Execute multiple verify blocks", func(ctx *harness.Context) error {
				// First verification block
				if err := ctx.Verify(func(v *verify.Collector) {
					v.True("first block check 1", true)
					v.Equal("first block check 2", 1, 1)
				}); err != nil {
					return fmt.Errorf("first verification block failed: %w", err)
				}

				// Second verification block
				if err := ctx.Verify(func(v *verify.Collector) {
					v.Contains("second block check 1", "hello", "ello")
					v.NotContains("second block check 2", "goodbye", "hello")
				}); err != nil {
					return fmt.Errorf("second verification block failed: %w", err)
				}

				return nil
			}),
		},
	)
}

// VerifyWithNilValuesScenario tests soft assertions with various value types
func VerifyWithNilValuesScenario() *harness.Scenario {
	return harness.NewScenario(
		"verify-with-nil-values",
		"Tests soft assertions with various value types including strings and numbers",
		[]string{"assertions"},
		[]harness.Step{
			harness.NewStep("Verify assertions with different value types", func(ctx *harness.Context) error {
				value1 := "test"
				value2 := "test"

				return ctx.Verify(func(v *verify.Collector) {
					v.Equal("string values are equal", value1, value2)
					v.NotEqual("different numbers are not equal", 1, 2)
					v.Equal("same numbers are equal", 42, 42)
				})
			}),
		},
	)
}
