package assert

import (
	"fmt"
	"reflect"
	"strings"
)

// Error represents an assertion error
type Error struct {
	Message  string
	Expected interface{}
	Actual   interface{}
}

func (e *Error) Error() string {
	if e.Expected != nil && e.Actual != nil {
		return fmt.Sprintf("%s\nExpected: %v\nActual: %v", e.Message, e.Expected, e.Actual)
	}
	return e.Message
}

// Equal asserts that two values are equal
func Equal(expected, actual interface{}, message ...string) error {
	if !reflect.DeepEqual(expected, actual) {
		msg := "values are not equal"
		if len(message) > 0 {
			msg = strings.Join(message, " ")
		}
		return &Error{
			Message:  msg,
			Expected: expected,
			Actual:   actual,
		}
	}
	return nil
}

// NotEqual asserts that two values are not equal
func NotEqual(expected, actual interface{}, message ...string) error {
	if reflect.DeepEqual(expected, actual) {
		msg := "values are equal"
		if len(message) > 0 {
			msg = strings.Join(message, " ")
		}
		return &Error{
			Message:  msg,
			Expected: fmt.Sprintf("not %v", expected),
			Actual:   actual,
		}
	}
	return nil
}

// Contains asserts that a string contains a substring
func Contains(s, substr string, message ...string) error {
	if !strings.Contains(s, substr) {
		msg := "string does not contain substring"
		if len(message) > 0 {
			msg = strings.Join(message, " ")
		}
		return &Error{
			Message:  msg,
			Expected: fmt.Sprintf("contains '%s'", substr),
			Actual:   s,
		}
	}
	return nil
}

// NotContains asserts that a string does not contain a substring
func NotContains(s, substr string, message ...string) error {
	if strings.Contains(s, substr) {
		msg := "string contains substring"
		if len(message) > 0 {
			msg = strings.Join(message, " ")
		}
		return &Error{
			Message:  msg,
			Expected: fmt.Sprintf("does not contain '%s'", substr),
			Actual:   s,
		}
	}
	return nil
}

// True asserts that a value is true
func True(value bool, message ...string) error {
	if !value {
		msg := "expected true, got false"
		if len(message) > 0 {
			msg = strings.Join(message, " ")
		}
		return &Error{Message: msg}
	}
	return nil
}

// False asserts that a value is false
func False(value bool, message ...string) error {
	if value {
		msg := "expected false, got true"
		if len(message) > 0 {
			msg = strings.Join(message, " ")
		}
		return &Error{Message: msg}
	}
	return nil
}

// Nil asserts that a value is nil
func Nil(value interface{}, message ...string) error {
	if value != nil && !reflect.ValueOf(value).IsNil() {
		msg := "expected nil"
		if len(message) > 0 {
			msg = strings.Join(message, " ")
		}
		return &Error{
			Message: msg,
			Actual:  value,
		}
	}
	return nil
}

// NotNil asserts that a value is not nil
func NotNil(value interface{}, message ...string) error {
	if value == nil || reflect.ValueOf(value).IsNil() {
		msg := "expected non-nil value"
		if len(message) > 0 {
			msg = strings.Join(message, " ")
		}
		return &Error{Message: msg}
	}
	return nil
}

// Empty asserts that a string, slice, or map is empty
func Empty(value interface{}, message ...string) error {
	v := reflect.ValueOf(value)
	
	switch v.Kind() {
	case reflect.String, reflect.Slice, reflect.Map:
		if v.Len() != 0 {
			msg := "expected empty value"
			if len(message) > 0 {
				msg = strings.Join(message, " ")
			}
			return &Error{
				Message: msg,
				Actual:  fmt.Sprintf("length %d", v.Len()),
			}
		}
	default:
		return fmt.Errorf("Empty() only supports string, slice, and map types")
	}
	
	return nil
}