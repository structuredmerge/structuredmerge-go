package goparsermerge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/gomerge"
	"github.com/structuredmerge/structuredmerge-go/treehaver"
)

func jsonReady(t *testing.T, value any) any {
	t.Helper()
	source, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal value: %v", err)
	}

	var normalized any
	if err := json.Unmarshal(source, &normalized); err != nil {
		t.Fatalf("decode value: %v", err)
	}

	return normalized
}

func readFixture(t *testing.T, parts ...string) map[string]any {
	t.Helper()
	pathParts := append([]string{"..", "..", "fixtures"}, parts...)
	source, err := os.ReadFile(filepath.Join(pathParts...))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var fixture map[string]any
	if err := json.Unmarshal(source, &fixture); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	return fixture
}

func TestSharedFixtureGoProviderFeatureProfile(t *testing.T) {
	familyFixture := readFixture(t, "diagnostics", "slice-109-go-family-feature-profile", "go-feature-profile.json")
	fixture := readFixture(t, "diagnostics", "slice-281-go-provider-feature-profiles", "go-go-provider-feature-profiles.json")
	familyProfile := gomerge.GoFeatureProfileInfo()
	if actual := jsonReady(t, map[string]any{
		"family":             familyProfile.Family,
		"supported_dialects": familyProfile.SupportedDialects,
		"supported_policies": familyProfile.SupportedPolicies,
	}); !reflect.DeepEqual(actual, familyFixture["feature_profile"]) {
		t.Fatalf("unexpected family profile: %+v", actual)
	}
	if len(AvailableGoBackends()) != 1 || AvailableGoBackends()[0] != BackendGoParser {
		t.Fatalf("unexpected backends: %+v", AvailableGoBackends())
	}
	expected := fixture["providers"].(map[string]any)["go_parser"].(map[string]any)["feature_profile"]
	profile := GoBackendFeatureProfileInfo()
	actual := jsonReady(t, map[string]any{
		"family":             profile.Family,
		"supported_dialects": profile.SupportedDialects,
		"supported_policies": profile.SupportedPolicies,
		"backend":            profile.Backend,
		"backend_ref": map[string]any{
			"id":     profile.BackendRef.ID,
			"family": profile.BackendRef.Family,
		},
	})
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("unexpected provider profile: %+v", profile)
	}
	if backend := treehaver.BackendReferenceByID(BackendGoParser); backend == nil || backend.ID != BackendGoParser || backend.Family != "native" {
		t.Fatalf("unexpected registered backend: %+v", backend)
	}
}

func TestSharedFixtureGoProviderPlanContext(t *testing.T) {
	fixture := readFixture(t, "diagnostics", "slice-282-go-provider-plan-contexts", "go-go-provider-plan-contexts.json")
	if context := GoPlanContext(); context.FeatureProfile == nil || context.FeatureProfile.Backend != fixture["providers"].(map[string]any)["go_parser"].(map[string]any)["feature_profile"].(map[string]any)["backend"].(string) {
		t.Fatalf("unexpected provider plan context: %+v", context)
	}
}

func TestSharedFixtureGoProviderParseAndMerge(t *testing.T) {
	parityFixture := readFixture(t, "go", "slice-114-native", "module-parity.json")
	parseResult := ParseGo(parityFixture["source"].(string), gomerge.DialectGo)
	if !parseResult.OK || parseResult.Analysis == nil {
		t.Fatalf("expected parse success: %+v", parseResult)
	}
	owners := make([]map[string]any, 0, len(parseResult.Analysis.Owners))
	for _, owner := range parseResult.Analysis.Owners {
		owners = append(owners, map[string]any{
			"path":       owner.Path,
			"owner_kind": owner.OwnerKind,
			"match_key":  owner.MatchKey,
		})
	}
	if actual := jsonReady(t, owners); !reflect.DeepEqual(actual, parityFixture["expected"].(map[string]any)["owners"]) {
		t.Fatalf("unexpected owners: %+v", actual)
	}

	mergeResult := MergeGo(parityFixture["template"].(string), parityFixture["destination"].(string), gomerge.DialectGo)
	if !mergeResult.OK || mergeResult.Output == nil {
		t.Fatalf("expected merge success: %+v", mergeResult)
	}
	if *mergeResult.Output != parityFixture["expected"].(map[string]any)["output"].(string) {
		t.Fatalf("unexpected merge output:\n%s", *mergeResult.Output)
	}
}

func TestSharedFixtureGoProviderNamedSuitePlans(t *testing.T) {
	fixture := readFixture(t, "diagnostics", "slice-283-go-provider-named-suite-plans", "go-go-provider-named-suite-plans.json")
	source, err := json.Marshal(fixture["manifest"])
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}

	var manifest astmerge.ConformanceManifest
	if err := json.Unmarshal(source, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}

	expectedRaw := fixture["expected_entries"].(map[string]any)["go_parser"].([]any)
	expected := jsonReady(t, expectedRaw)

	plans := astmerge.PlanNamedConformanceSuites(manifest, map[string]astmerge.ConformanceFamilyPlanContext{
		"go": GoPlanContext(),
	})
	projected := jsonReady(t, plans)

	if !reflect.DeepEqual(projected, expected) {
		t.Fatalf("unexpected plans: %+v", projected)
	}
}

func TestSharedFixtureGoProviderManifestReport(t *testing.T) {
	fixture := readFixture(t, "diagnostics", "slice-284-go-provider-manifest-report", "go-go-provider-manifest-report.json")
	source, err := json.Marshal(fixture["manifest"])
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}

	var manifest astmerge.ConformanceManifest
	if err := json.Unmarshal(source, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}

	executionsSource, err := json.Marshal(fixture["executions"].(map[string]any)["go_parser"])
	if err != nil {
		t.Fatalf("marshal executions: %v", err)
	}

	var executions map[string]astmerge.ConformanceCaseExecution
	if err := json.Unmarshal(executionsSource, &executions); err != nil {
		t.Fatalf("decode executions: %v", err)
	}

	entries := astmerge.ReportPlannedNamedConformanceSuites(
		astmerge.PlanNamedConformanceSuites(manifest, map[string]astmerge.ConformanceFamilyPlanContext{
			"go": GoPlanContext(),
		}),
		func(run astmerge.ConformanceCaseRun) astmerge.ConformanceCaseExecution {
			key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
			if result, ok := executions[key]; ok {
				return result
			}
			return astmerge.ConformanceCaseExecution{Outcome: "failed", Messages: []string{"missing execution"}}
		},
	)

	report := astmerge.ReportNamedConformanceSuiteEnvelope(entries)
	expected := jsonReady(t, fixture["expected_reports"].(map[string]any)["go_parser"])
	actual := jsonReady(t, report)

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("unexpected report: %+v", actual)
	}
}

func TestSharedFixtureGoProviderRejectsUnsupportedBackendOverrides(t *testing.T) {
	expectedDiagnostics := []map[string]any{{
		"severity": "error",
		"category": "unsupported_feature",
		"message":  "Unsupported Go backend kreuzberg-language-pack.",
	}}

	parseResult := ParseGo("func Greet() string { return \"hi\" }\n", gomerge.DialectGo, "kreuzberg-language-pack")
	if parseResult.OK {
		t.Fatalf("expected parse failure: %+v", parseResult)
	}
	if actual := jsonReady(t, parseResult.Diagnostics); !reflect.DeepEqual(actual, jsonReady(t, expectedDiagnostics)) {
		t.Fatalf("unexpected parse diagnostics: %+v", actual)
	}

	mergeResult := MergeGo("func A() {}\n", "func B() {}\n", gomerge.DialectGo, "kreuzberg-language-pack")
	if mergeResult.OK {
		t.Fatalf("expected merge failure: %+v", mergeResult)
	}
	if actual := jsonReady(t, mergeResult.Diagnostics); !reflect.DeepEqual(actual, jsonReady(t, expectedDiagnostics)) {
		t.Fatalf("unexpected merge diagnostics: %+v", actual)
	}
}

func TestGoParserEditProjectionExecutionFixture(t *testing.T) {
	fixture := readFixture(t, "diagnostics", "slice-931-go-parser-edit-projection-execution", "edit-projection-execution.json")

	request := editProjectionExecutionRequestFromFixture(fixture["request"])
	result := ApplyEditProjection(request)
	expected := editProjectionExecutionResultFromFixture(fixture["expected_result"])
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("unexpected go/parser edit projection result:\n%#v", result)
	}
}

func TestGoParserInsertChildEditProjectionFixture(t *testing.T) {
	fixture := readFixture(t, "diagnostics", "slice-933-go-parser-insert-child-edit-projection", "insert-child-edit-projection.json")

	request := editProjectionExecutionRequestFromFixture(fixture["request"])
	result := ApplyEditProjection(request)
	expected := editProjectionExecutionResultFromFixture(fixture["expected_result"])
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("unexpected go/parser insert-child edit projection result:\n%#v", result)
	}
}

func TestGoParserDeleteNodeEditProjectionFixture(t *testing.T) {
	fixture := readFixture(t, "diagnostics", "slice-934-go-parser-delete-node-edit-projection", "delete-node-edit-projection.json")

	request := editProjectionExecutionRequestFromFixture(fixture["request"])
	result := ApplyEditProjection(request)
	expected := editProjectionExecutionResultFromFixture(fixture["expected_result"])
	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("unexpected go/parser delete-node edit projection result:\n%#v", result)
	}
}

func editProjectionExecutionRequestFromFixture(value any) treehaver.EditProjectionExecutionRequest {
	fixture := value.(map[string]any)
	operations := []treehaver.EditProjectionOperationRequest{}
	for _, rawOperation := range fixture["operations"].([]any) {
		operationFixture := rawOperation.(map[string]any)
		operations = append(operations, treehaver.EditProjectionOperationRequest{
			Operation:         operationFixture["operation"].(string),
			TargetNodeID:      operationFixture["target_node_id"].(string),
			TargetNodePath:    operationFixture["target_node_path"].(string),
			ReplacementSource: operationFixture["replacement_source"].(string),
		})
	}
	backendRefFixture := fixture["backend_ref"].(map[string]any)
	return treehaver.EditProjectionExecutionRequest{
		ProviderID: fixture["provider_id"].(string),
		BackendRef: treehaver.BackendReference{
			ID:     backendRefFixture["id"].(string),
			Family: backendRefFixture["family"].(string),
		},
		Language:   fixture["language"].(string),
		Source:     fixture["source"].(string),
		Operations: operations,
	}
}

func editProjectionExecutionResultFromFixture(value any) treehaver.EditProjectionExecutionResult {
	fixture := value.(map[string]any)
	applied := []treehaver.AppliedEditProjectionOperation{}
	for _, rawOperation := range fixture["applied_operations"].([]any) {
		operationFixture := rawOperation.(map[string]any)
		applied = append(applied, treehaver.AppliedEditProjectionOperation{
			Operation:        operationFixture["operation"].(string),
			TargetNodeID:     operationFixture["target_node_id"].(string),
			CorrelationKey:   operationFixture["correlation_key"].(string),
			CorrelationValue: operationFixture["correlation_value"].(string),
		})
	}
	diagnostics := []treehaver.ProviderDiagnostic{}
	for _, rawDiagnostic := range fixture["diagnostics"].([]any) {
		diagnosticFixture := rawDiagnostic.(map[string]any)
		diagnostics = append(diagnostics, treehaver.ProviderDiagnostic{
			Severity: diagnosticFixture["severity"].(string),
			Category: diagnosticFixture["category"].(string),
			Code:     diagnosticFixture["code"].(string),
			Message:  diagnosticFixture["message"].(string),
			Path:     diagnosticFixture["path"].(string),
			Blocking: diagnosticFixture["blocking"].(bool),
		})
	}
	return treehaver.EditProjectionExecutionResult{
		OK:                fixture["ok"].(bool),
		Status:            fixture["status"].(string),
		Source:            fixture["source"].(string),
		AppliedOperations: applied,
		Diagnostics:       diagnostics,
	}
}
