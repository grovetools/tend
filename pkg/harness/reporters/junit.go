package reporters

import (
	"encoding/xml"
	"fmt"
	"io"
	"runtime"
	"time"

	"github.com/grovetools/tend/pkg/harness"
)

// JUnitTestSuites represents the root element
type JUnitTestSuites struct {
	XMLName    xml.Name         `xml:"testsuites"`
	Name       string           `xml:"name,attr"`
	Tests      int              `xml:"tests,attr"`
	Failures   int              `xml:"failures,attr"`
	Errors     int              `xml:"errors,attr"`
	Time       float64          `xml:"time,attr"`
	TestSuites []JUnitTestSuite `xml:"testsuite"`
}

// JUnitTestSuite represents a test suite
type JUnitTestSuite struct {
	Name       string          `xml:"name,attr"`
	Tests      int             `xml:"tests,attr"`
	Failures   int             `xml:"failures,attr"`
	Errors     int             `xml:"errors,attr"`
	Time       float64         `xml:"time,attr"`
	Timestamp  string          `xml:"timestamp,attr"`
	Properties []JUnitProperty `xml:"properties>property,omitempty"`
	TestCases  []JUnitTestCase `xml:"testcase"`
}

// JUnitTestCase represents a test case
type JUnitTestCase struct {
	Name      string        `xml:"name,attr"`
	ClassName string        `xml:"classname,attr"`
	Time      float64       `xml:"time,attr"`
	Failure   *JUnitFailure `xml:"failure,omitempty"`
	Error     *JUnitError   `xml:"error,omitempty"`
	Skipped   *JUnitSkipped `xml:"skipped,omitempty"`
	SystemOut string        `xml:"system-out,omitempty"`
	SystemErr string        `xml:"system-err,omitempty"`
}

// JUnitProperty represents a property
type JUnitProperty struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

// JUnitFailure represents a test failure
type JUnitFailure struct {
	Type    string `xml:"type,attr"`
	Message string `xml:"message,attr"`
	Text    string `xml:",chardata"`
}

// JUnitError represents a test error
type JUnitError struct {
	Type    string `xml:"type,attr"`
	Message string `xml:"message,attr"`
	Text    string `xml:",chardata"`
}

// JUnitSkipped represents a skipped test
type JUnitSkipped struct {
	Message string `xml:"message,attr,omitempty"`
}

// JUnitReporter generates JUnit XML reports
type JUnitReporter struct {
	suiteName string
	timestamp time.Time
}

// NewJUnitReporter creates a new JUnit reporter
func NewJUnitReporter(suiteName string) *JUnitReporter {
	return &JUnitReporter{
		suiteName: suiteName,
		timestamp: time.Now(),
	}
}

// WriteReport writes JUnit XML report for test results
func (r *JUnitReporter) WriteReport(w io.Writer, results []*harness.Result) error {
	suites := &JUnitTestSuites{
		Name: r.suiteName,
	}

	// Group results by some criteria (for now, one suite)
	suite := JUnitTestSuite{
		Name:      r.suiteName,
		Timestamp: r.timestamp.Format(time.RFC3339),
		Properties: []JUnitProperty{
			{Name: "go.version", Value: runtime.Version()},
			{Name: "os.name", Value: runtime.GOOS},
			{Name: "os.arch", Value: runtime.GOARCH},
		},
	}

	totalTime := 0.0

	for _, result := range results {
		testCase := JUnitTestCase{
			Name:      result.ScenarioName,
			ClassName: "tend.scenarios",
			Time:      result.Duration.Seconds(),
		}

		if !result.Success {
			suite.Failures++

			var message string
			var details string

			if result.Error != nil {
				message = result.Error.Error()
				details = fmt.Sprintf("Failed at step: %s\nError: %v",
					result.FailedStep, result.Error)
			} else {
				message = "Test failed"
				details = fmt.Sprintf("Failed at step: %s", result.FailedStep)
			}

			testCase.Failure = &JUnitFailure{
				Type:    "AssertionError",
				Message: message,
				Text:    details,
			}
		}

		suite.TestCases = append(suite.TestCases, testCase)
		suite.Tests++
		totalTime += result.Duration.Seconds()
	}

	suite.Time = totalTime
	suites.TestSuites = append(suites.TestSuites, suite)
	suites.Tests = suite.Tests
	suites.Failures = suite.Failures
	suites.Time = totalTime

	// Write XML
	encoder := xml.NewEncoder(w)
	encoder.Indent("", "  ")

	if _, err := w.Write([]byte(xml.Header)); err != nil {
		return err
	}

	return encoder.Encode(suites)
}
