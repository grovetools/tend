package verify

import (
	"fmt"
	"strings"
)

// VerificationError holds multiple assertion failures.
type VerificationError struct {
	Errors []error
}

// Error formats all collected assertion failures into a single string.
func (e *VerificationError) Error() string {
	if e == nil || len(e.Errors) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("%d assertion(s) failed:", len(e.Errors)))
	for i, err := range e.Errors {
		// Indent each error for readability
		errorStr := strings.ReplaceAll(err.Error(), "\n", "\n    ")
		b.WriteString(fmt.Sprintf("\n  %d. %s", i+1, errorStr))
	}
	return b.String()
}
