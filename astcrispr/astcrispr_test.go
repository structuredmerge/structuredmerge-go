package astcrispr

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestBoundaryReportFixture(t *testing.T) {
	fixturePath := filepath.Join(
		"..",
		"..",
		"fixtures",
		"diagnostics",
		"slice-916-ast-crispr-package-boundary",
		"ast-crispr-package-boundary.json",
	)
	source, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var fixture map[string]any
	if err := json.Unmarshal(source, &fixture); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	if !reflect.DeepEqual(BoundaryReport(), fixture["boundary"]) {
		t.Fatalf("unexpected boundary report: %+v", BoundaryReport())
	}
	if AstMergeContractAnchor() != "StructuredEditCrisprExampleParityReport" {
		t.Fatalf("unexpected contract anchor: %s", AstMergeContractAnchor())
	}
}

func TestLimitHelpersFixture(t *testing.T) {
	fixturePath := filepath.Join(
		"..",
		"..",
		"fixtures",
		"diagnostics",
		"slice-917-ast-crispr-limit-helpers",
		"ast-crispr-limit-helpers.json",
	)
	source, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var fixture map[string]any
	if err := json.Unmarshal(source, &fixture); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		limit, err := NewLimit(testCase["spec"])
		if err != nil {
			t.Fatalf("limit %s: %v", testCase["name"], err)
		}
		if limit.Describe() != testCase["expected_description"].(string) {
			t.Fatalf("unexpected description for %s: %s", testCase["name"], limit.Describe())
		}
		for _, rawExpectation := range testCase["expectations"].([]any) {
			expectation := rawExpectation.(map[string]any)
			count := int(expectation["count"].(float64))
			if limit.Allows(count) != expectation["allowed"].(bool) {
				t.Fatalf("unexpected limit result for %s count %d", testCase["name"], count)
			}
		}
	}

	for _, rawCase := range fixture["invalid_cases"].([]any) {
		testCase := rawCase.(map[string]any)
		_, err := NewLimit(testCase["spec"])
		if err == nil {
			t.Fatalf("expected error for %s", testCase["name"])
		}
		crisprErr, ok := err.(Error)
		if !ok {
			t.Fatalf("expected astcrispr error for %s: %T", testCase["name"], err)
		}
		if crisprErr.Code != testCase["expected_error"].(string) {
			t.Fatalf("unexpected error code for %s: %s", testCase["name"], crisprErr.Code)
		}
	}
}
