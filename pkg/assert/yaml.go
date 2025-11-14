package assert

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// YAMLField asserts that a YAML file contains a field with the expected value.
// It parses the YAML file and checks if the specified field equals the expected value.
func YAMLField(path, field, expectedValue string, message ...string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		msg := fmt.Sprintf("reading YAML file '%s'", path)
		if len(message) > 0 {
			msg = message[0]
		}
		return &Error{
			Message: msg,
			Actual:  fmt.Sprintf("error: %v", err),
		}
	}

	var data map[string]interface{}
	if err := yaml.Unmarshal(content, &data); err != nil {
		msg := fmt.Sprintf("parsing YAML file '%s'", path)
		if len(message) > 0 {
			msg = message[0]
		}
		return &Error{
			Message: msg,
			Actual:  fmt.Sprintf("parse error: %v", err),
		}
	}

	actualValue, ok := data[field]
	if !ok {
		msg := fmt.Sprintf("field '%s' not found in YAML file '%s'", field, path)
		if len(message) > 0 {
			msg = message[0]
		}
		return &Error{
			Message:  msg,
			Expected: fmt.Sprintf("field '%s' exists", field),
			Actual:   "field not found",
		}
	}

	actualStr := fmt.Sprintf("%v", actualValue)
	if actualStr != expectedValue {
		msg := fmt.Sprintf("field '%s' in YAML file '%s' has wrong value", field, path)
		if len(message) > 0 {
			msg = message[0]
		}
		return &Error{
			Message:  msg,
			Expected: expectedValue,
			Actual:   actualStr,
		}
	}

	return nil
}

// YAMLFieldExists asserts that a YAML file contains a field (regardless of value).
func YAMLFieldExists(path, field string, message ...string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		msg := fmt.Sprintf("reading YAML file '%s'", path)
		if len(message) > 0 {
			msg = message[0]
		}
		return &Error{
			Message: msg,
			Actual:  fmt.Sprintf("error: %v", err),
		}
	}

	var data map[string]interface{}
	if err := yaml.Unmarshal(content, &data); err != nil {
		msg := fmt.Sprintf("parsing YAML file '%s'", path)
		if len(message) > 0 {
			msg = message[0]
		}
		return &Error{
			Message: msg,
			Actual:  fmt.Sprintf("parse error: %v", err),
		}
	}

	if _, ok := data[field]; !ok {
		msg := fmt.Sprintf("field '%s' not found in YAML file '%s'", field, path)
		if len(message) > 0 {
			msg = message[0]
		}
		return &Error{
			Message:  msg,
			Expected: fmt.Sprintf("field '%s' exists", field),
			Actual:   "field not found",
		}
	}

	return nil
}

// YAMLContains asserts that a YAML file's field contains a substring.
func YAMLContains(path, field, substr string, message ...string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		msg := fmt.Sprintf("reading YAML file '%s'", path)
		if len(message) > 0 {
			msg = message[0]
		}
		return &Error{
			Message: msg,
			Actual:  fmt.Sprintf("error: %v", err),
		}
	}

	var data map[string]interface{}
	if err := yaml.Unmarshal(content, &data); err != nil {
		msg := fmt.Sprintf("parsing YAML file '%s'", path)
		if len(message) > 0 {
			msg = message[0]
		}
		return &Error{
			Message: msg,
			Actual:  fmt.Sprintf("parse error: %v", err),
		}
	}

	actualValue, ok := data[field]
	if !ok {
		msg := fmt.Sprintf("field '%s' not found in YAML file '%s'", field, path)
		if len(message) > 0 {
			msg = message[0]
		}
		return &Error{
			Message:  msg,
			Expected: fmt.Sprintf("field '%s' exists", field),
			Actual:   "field not found",
		}
	}

	actualStr := fmt.Sprintf("%v", actualValue)
	return Contains(actualStr, substr, message...)
}
