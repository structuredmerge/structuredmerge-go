package astmerge

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const portableBenchmarkFixtureDigest = "1015727c1a9c4f132dfb8bb51118a594f6ad170459e2af9c73cdb586ec02ce22"

func portableBenchmarkFixturePath(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "..", "fixtures", "diagnostics", "slice-1022-portable-benchmark-contract", "contract.json"))
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func TestPortableBenchmarkContractCanonicalSummary(t *testing.T) {
	consumer, err := LoadPortableBenchmarkContract(portableBenchmarkFixturePath(t))
	if err != nil {
		t.Fatalf("LoadPortableBenchmarkContract() error = %v", err)
	}

	expected := PortableBenchmarkCanonicalSummary{
		Schema: "structuredmerge.benchmark/v1",
		Counts: PortableBenchmarkSummaryCounts{
			Adapters:                          2,
			Cases:                             5,
			Results:                           7,
			SelectedCases:                     5,
			ScoreEligibleResults:              5,
			QualityDenominatorExcludedResults: 2,
			EligibleFalseAutoMerges:           1,
		},
		CaseIDsByOperation: map[string][]string{
			"diff":           {"case.diff.json.object-update.v1"},
			"merge3":         {"case.merge3.json.independent-fields.v1", "case.merge3.json.region-conflict.v1"},
			"history_replay": {"case.history.ruby.release-events-dotenv-provider.v1"},
			"metamorphic":    {"case.metamorphic.json.reorder-format.v1"},
		},
		CaseIDsByPartition: map[string][]string{
			"sentinel":    {"case.diff.json.object-update.v1"},
			"gold":        {"case.merge3.json.independent-fields.v1", "case.merge3.json.region-conflict.v1"},
			"metamorphic": {"case.metamorphic.json.reorder-format.v1"},
			"history":     {"case.history.ruby.release-events-dotenv-provider.v1"},
			"holdout":     {},
		},
		ScoreEligibleResultIDs: []string{
			"result.base.clean.false-conflict",
			"result.base.conflict.true",
			"result.candidate.clean.correct",
			"result.candidate.conflict.false-auto-merge",
			"result.candidate.diff.error",
		},
		QualityDenominatorExcludedResultIDs: []string{
			"result.candidate.history.excluded-ambiguous",
			"result.candidate.metamorphic.unsupported",
		},
		FalseAutoMerges: []PortableBenchmarkFalseAutoMergeSummary{
			{ID: "result.candidate.conflict.false-auto-merge", Severity: "critical"},
		},
		SafetyGate: PortableBenchmarkSafetyGateSummary{
			Status:                      "fail",
			EligibleFalseAutoMergeCount: 1,
			NonCompensable:              true,
		},
		SelectionReasonCategories: []string{
			"budget_exceed",
			"changed_paths",
			"direct_cases",
			"inferred_capabilities",
			"neighbor_samples",
			"sentinels",
		},
		ContractDigest: portableBenchmarkFixtureDigest,
	}

	summary, err := consumer.CanonicalSummary()
	if err != nil {
		t.Fatalf("CanonicalSummary() error = %v", err)
	}
	if !reflect.DeepEqual(summary, expected) {
		actualJSON, _ := json.MarshalIndent(summary, "", "  ")
		expectedJSON, _ := json.MarshalIndent(expected, "", "  ")
		t.Fatalf("CanonicalSummary() mismatch\nactual:\n%s\nexpected:\n%s", actualJSON, expectedJSON)
	}
	if consumer.ContractDigest != portableBenchmarkFixtureDigest {
		t.Fatalf("ContractDigest = %q, want %q", consumer.ContractDigest, portableBenchmarkFixtureDigest)
	}
}

func TestPortableBenchmarkContractAdversarialValidation(t *testing.T) {
	tests := []struct {
		name     string
		category PortableBenchmarkValidationCategory
		message  string
		mutate   func(map[string]any)
	}{
		{
			name:     "bad digest",
			category: PortableBenchmarkInvalidDigest,
			message:  "SHA-256 does not match exact bytes",
			mutate: func(contract map[string]any) {
				cases := contract["cases"].([]any)
				cases[0].(map[string]any)["inputs"].(map[string]any)["before"].(map[string]any)["sha256"] = strings.Repeat("0", 64)
			},
		},
		{
			name:     "duplicate case ID",
			category: PortableBenchmarkInvalidReference,
			message:  "duplicate case ID",
			mutate: func(contract map[string]any) {
				cases := contract["cases"].([]any)
				cases[1].(map[string]any)["id"] = cases[0].(map[string]any)["id"]
			},
		},
		{
			name:     "dangling result case ref",
			category: PortableBenchmarkInvalidReference,
			message:  "case_id is dangling",
			mutate: func(contract map[string]any) {
				contract["case_results"].([]any)[0].(map[string]any)["case_id"] = "case.missing"
			},
		},
		{
			name:     "eligible excluded result",
			category: PortableBenchmarkInvalidEligibility,
			message:  "score eligibility conflicts",
			mutate: func(contract map[string]any) {
				for _, value := range contract["case_results"].([]any) {
					result := value.(map[string]any)
					if result["outcome"] == "excluded_ambiguous" {
						result["score_eligible"] = true
					}
				}
			},
		},
		{
			name:     "compensable false auto-merge",
			category: PortableBenchmarkInvalidSafety,
			message:  "cannot be compensable",
			mutate: func(contract map[string]any) {
				for _, value := range contract["case_results"].([]any) {
					result := value.(map[string]any)
					if result["outcome"] == "false_auto_merge" {
						result["dimensions"].(map[string]any)["safety"].(map[string]any)["compensable"] = true
					}
				}
			},
		},
		{
			name:     "LLM micro hard gate",
			category: PortableBenchmarkProhibitedLLMGate,
			message:  "LLM oracle cannot participate in micro hard gates",
			mutate: func(contract map[string]any) {
				contract["run_manifest"].(map[string]any)["profile"] = "micro"
				contract["cases"].([]any)[0].(map[string]any)["oracle"].(map[string]any)["class"] = "llm"
			},
		},
	}

	source, err := os.ReadFile(portableBenchmarkFixturePath(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var contract map[string]any
			if err := json.Unmarshal(source, &contract); err != nil {
				t.Fatal(err)
			}
			test.mutate(contract)
			mutated, err := json.Marshal(contract)
			if err != nil {
				t.Fatal(err)
			}

			_, err = ParsePortableBenchmarkContract(mutated)
			var validationError *PortableBenchmarkValidationError
			if !errors.As(err, &validationError) {
				t.Fatalf("error = %v, want *PortableBenchmarkValidationError", err)
			}
			if validationError.Category != test.category {
				t.Errorf("category = %q, want %q", validationError.Category, test.category)
			}
			if !strings.Contains(validationError.Message, test.message) {
				t.Errorf("message = %q, want substring %q", validationError.Message, test.message)
			}
		})
	}
}
