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

func TestMatchProfileHelpersFixture(t *testing.T) {
	fixturePath := filepath.Join(
		"..",
		"..",
		"fixtures",
		"diagnostics",
		"slice-918-ast-crispr-match-profile-helpers",
		"ast-crispr-match-profile-helpers.json",
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
		rawProfile := testCase["profile"].(map[string]any)
		profile := NewMatchProfile(
			rawProfile["start_boundary"].(string),
			rawProfile["end_boundary"].(string),
			rawProfile["payload_kind"].(string),
		)
		if !reflect.DeepEqual(profile.Report(), testCase["expected"]) {
			t.Fatalf("unexpected match profile report for %s: %+v", testCase["name"], profile.Report())
		}
	}
}

func TestSelectionProfileHelpersFixture(t *testing.T) {
	fixturePath := filepath.Join(
		"..",
		"..",
		"fixtures",
		"diagnostics",
		"slice-919-ast-crispr-selection-profile-helpers",
		"ast-crispr-selection-profile-helpers.json",
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
		rawProfile := testCase["profile"].(map[string]any)
		commentRegion, _ := rawProfile["comment_region"].(string)
		profile := NewSelectionProfile(
			rawProfile["owner_scope"].(string),
			rawProfile["owner_selector"].(string),
			rawProfile["selector_kind"].(string),
			rawProfile["selection_intent"].(string),
			commentRegion,
			rawProfile["include_trailing_gap"].(bool),
		)
		if !reflect.DeepEqual(profile.Report(), testCase["expected"]) {
			t.Fatalf("unexpected selection profile report for %s: %+v", testCase["name"], profile.Report())
		}
	}
}

func TestDestinationProfileHelpersFixture(t *testing.T) {
	fixturePath := filepath.Join(
		"..",
		"..",
		"fixtures",
		"diagnostics",
		"slice-920-ast-crispr-destination-profile-helpers",
		"ast-crispr-destination-profile-helpers.json",
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
		rawProfile := testCase["profile"].(map[string]any)
		profile := NewDestinationProfile(
			rawProfile["resolution_kind"].(string),
			rawProfile["resolution_source"].(string),
			rawProfile["anchor_boundary"].(string),
			rawProfile["used_if_missing"].(bool),
		)
		if !reflect.DeepEqual(profile.Report(), testCase["expected"]) {
			t.Fatalf("unexpected destination profile report for %s: %+v", testCase["name"], profile.Report())
		}
	}
}

func TestOperationProfileHelpersFixture(t *testing.T) {
	fixturePath := filepath.Join(
		"..",
		"..",
		"fixtures",
		"diagnostics",
		"slice-921-ast-crispr-operation-profile-helpers",
		"ast-crispr-operation-profile-helpers.json",
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
		rawProfile := testCase["profile"].(map[string]any)
		profile := NewOperationProfile(
			rawProfile["operation_kind"].(string),
			rawProfile["source_requirement"].(string),
			rawProfile["destination_requirement"].(string),
			rawProfile["replacement_source"].(string),
			rawProfile["captures_source_text"].(bool),
			rawProfile["supports_if_missing"].(bool),
		)
		if !reflect.DeepEqual(profile.Report(), testCase["expected"]) {
			t.Fatalf("unexpected operation profile report for %s: %+v", testCase["name"], profile.Report())
		}
	}
}

func TestOperationHelpersFixture(t *testing.T) {
	fixturePath := filepath.Join(
		"..",
		"..",
		"fixtures",
		"diagnostics",
		"slice-922-ast-crispr-operation-helpers",
		"ast-crispr-operation-helpers.json",
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
		var profile OperationProfile
		switch testCase["helper"].(string) {
		case "replace":
			profile = ReplaceOperation()
		case "delete":
			profile = DeleteOperation()
		case "insert":
			profile = InsertOperation()
		case "move":
			profile = MoveOperation()
		default:
			t.Fatalf("unknown helper %s", testCase["helper"])
		}
		if !reflect.DeepEqual(profile.Report(), testCase["expected_operation_profile"]) {
			t.Fatalf("unexpected operation helper report for %s: %+v", testCase["name"], profile.Report())
		}
	}
}
