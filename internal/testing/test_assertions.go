package testutil

import (
	"strings"
	"testing"
)

// Assert checks a condition and logs a human-readable behavior description
func Assert(t *testing.T, condition bool, behavior string) {
	t.Helper()
	if condition {
		t.Logf("✓ %s", behavior)
	} else {
		t.Errorf("✗ %s", behavior)
	}
}

// AssertEqual checks equality and logs what behavior was verified
func AssertEqual(t *testing.T, expected, actual interface{}, behavior string) {
	t.Helper()
	if expected == actual {
		t.Logf("✓ %s", behavior)
	} else {
		t.Errorf("✗ %s\n  Expected: %v\n  Got: %v", behavior, expected, actual)
	}
}

// AssertNoError checks for nil error and logs success behavior
func AssertNoError(t *testing.T, err error, behavior string) {
	t.Helper()
	if err == nil {
		t.Logf("✓ %s", behavior)
	} else {
		t.Errorf("✗ %s: %v", behavior, err)
	}
}

// AssertError checks that an error occurred and logs the expected failure behavior
func AssertError(t *testing.T, err error, behavior string) {
	t.Helper()
	if err != nil {
		t.Logf("✓ %s", behavior)
	} else {
		t.Errorf("✗ %s (expected error but got none)", behavior)
	}
}

// AssertContains checks if a string contains a substring
func AssertContains(t *testing.T, haystack, needle string, behavior string) {
	t.Helper()
	if strings.Contains(haystack, needle) {
		t.Logf("✓ %s", behavior)
	} else {
		t.Errorf("✗ %s\n  Looking for: %q\n  In: %q", behavior, needle, haystack)
	}
}

// AssertErrorCode checks specific error codes for JSON-RPC
func AssertErrorCode(t *testing.T, code int, expectedCode int, behavior string) {
	t.Helper()
	if code == expectedCode {
		t.Logf("✓ %s", behavior)
	} else {
		t.Errorf("✗ %s\n  Expected code: %d\n  Got code: %d", behavior, expectedCode, code)
	}
}

// TestScenario describes a test case with expected behavior
type TestScenario struct {
	Name     string
	Behavior string
	Test     func(t *testing.T)
}

// RunScenarios executes a set of test scenarios with descriptive output
func RunScenarios(t *testing.T, scenarios []TestScenario) {
	for _, scenario := range scenarios {
		t.Run(scenario.Name, func(t *testing.T) {
			t.Logf("SCENARIO: %s", scenario.Behavior)
			scenario.Test(t)
		})
	}
}

// Output formatting helpers
func Section(t *testing.T, name string) {
	t.Logf("\n=== %s ===", name)
}

func Given(t *testing.T, context string) {
	t.Logf("GIVEN: %s", context)
}

func When(t *testing.T, action string) {
	t.Logf("WHEN: %s", action)
}

func Then(t *testing.T, expectation string) {
	t.Logf("THEN: %s", expectation)
}

// Summary prints a test summary
func Summary(t *testing.T, tested string) {
	t.Logf("\n📋 TESTED: %s", tested)
}
