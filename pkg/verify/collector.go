package verify

import (
	"github.com/grovetools/tend/pkg/assert"
)

// AssertionLogger is an interface for logging assertion results.
type AssertionLogger interface {
	AddAssertion(description string, err error)
}

// Collector gathers assertion failures without stopping the test step.
// It logs every assertion (pass or fail) to the harness context for rich reporting.
type Collector struct {
	ctx    AssertionLogger
	errors []error
}

// New creates a new Collector linked to a context that can log assertions.
func New(ctx AssertionLogger) *Collector {
	return &Collector{ctx: ctx}
}

// Check returns a VerificationError if any assertions failed, otherwise nil.
func (c *Collector) Check() error {
	if len(c.errors) == 0 {
		return nil
	}
	return &VerificationError{Errors: c.errors}
}

// Contains asserts that a string contains a substring, collecting failures.
func (c *Collector) Contains(description, s, substr string) {
	err := assert.Contains(s, substr, description)
	if err != nil {
		c.errors = append(c.errors, err)
	}
	c.ctx.AddAssertion(description, err)
}

// NotContains asserts that a string does not contain a substring, collecting failures.
func (c *Collector) NotContains(description, s, substr string) {
	err := assert.NotContains(s, substr, description)
	if err != nil {
		c.errors = append(c.errors, err)
	}
	c.ctx.AddAssertion(description, err)
}

// Equal asserts that two values are equal, collecting failures.
func (c *Collector) Equal(description string, expected, actual interface{}) {
	err := assert.Equal(expected, actual, description)
	if err != nil {
		c.errors = append(c.errors, err)
	}
	c.ctx.AddAssertion(description, err)
}

// NotEqual asserts that two values are not equal, collecting failures.
func (c *Collector) NotEqual(description string, expected, actual interface{}) {
	err := assert.NotEqual(expected, actual, description)
	if err != nil {
		c.errors = append(c.errors, err)
	}
	c.ctx.AddAssertion(description, err)
}

// True asserts that a value is true, collecting failures.
func (c *Collector) True(description string, value bool) {
	err := assert.True(value, description)
	if err != nil {
		c.errors = append(c.errors, err)
	}
	c.ctx.AddAssertion(description, err)
}
