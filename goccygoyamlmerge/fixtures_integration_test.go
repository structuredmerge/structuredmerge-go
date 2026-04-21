package goccygoyamlmerge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/treehaver"
	"github.com/structuredmerge/structuredmerge-go/yamlmerge"
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

func TestSharedFixtureYAMLProviderFeatureProfile(t *testing.T) {
	fixture := readFixture(t, "diagnostics", "slice-277-yaml-provider-feature-profiles", "go-yaml-provider-feature-profiles.json")
	if len(AvailableYAMLBackends()) != 1 || AvailableYAMLBackends()[0] != BackendGoccyGoYAML {
		t.Fatalf("unexpected backends: %+v", AvailableYAMLBackends())
	}
	expected := fixture["providers"].(map[string]any)["goccy_go_yaml"].(map[string]any)["feature_profile"]
	profile := YAMLBackendFeatureProfileInfo()
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
	if backend := treehaver.BackendReferenceByID(BackendGoccyGoYAML); backend == nil || backend.ID != BackendGoccyGoYAML || backend.Family != "native" {
		t.Fatalf("unexpected registered backend: %+v", backend)
	}
}

func TestSharedFixtureYAMLProviderPlanContext(t *testing.T) {
	fixture := readFixture(t, "diagnostics", "slice-278-yaml-provider-plan-contexts", "go-yaml-provider-plan-contexts.json")
	if context := YAMLPlanContext(); context.FeatureProfile == nil || context.FeatureProfile.Backend != fixture["providers"].(map[string]any)["goccy_go_yaml"].(map[string]any)["feature_profile"].(map[string]any)["backend"].(string) {
		t.Fatalf("unexpected provider plan context: %+v", context)
	}
}

func TestSharedFixtureYAMLProviderParseAndMerge(t *testing.T) {
	valid := readFixture(t, "yaml", "slice-96-parse", "valid-document.json")
	validResult := ParseYAML(valid["source"].(string), yamlmerge.DialectYAML)
	if !validResult.OK || validResult.Analysis == nil {
		t.Fatalf("expected parse success: %+v", validResult)
	}

	matchingFixture := readFixture(t, "yaml", "slice-98-matching", "path-equality.json")
	template := ParseYAML(matchingFixture["template"].(string), yamlmerge.DialectYAML)
	destination := ParseYAML(matchingFixture["destination"].(string), yamlmerge.DialectYAML)
	if !template.OK || template.Analysis == nil || !destination.OK || destination.Analysis == nil {
		t.Fatalf("expected matching parse success")
	}
	result := MatchYAMLOwners(*template.Analysis, *destination.Analysis)
	if len(result.Matched) != len(matchingFixture["expected"].(map[string]any)["matched"].([]any)) {
		t.Fatalf("unexpected matches: %+v", result.Matched)
	}

	mergeFixture := readFixture(t, "yaml", "slice-99-merge", "mapping-merge.json")
	mergeResult := MergeYAML(mergeFixture["template"].(string), mergeFixture["destination"].(string), yamlmerge.DialectYAML)
	if !mergeResult.OK || mergeResult.Output == nil {
		t.Fatalf("expected merge success: %+v", mergeResult)
	}
	if *mergeResult.Output != mergeFixture["expected"].(map[string]any)["output"].(string) {
		t.Fatalf("unexpected merge output:\n%s", *mergeResult.Output)
	}
}

func TestSharedFixtureYAMLProviderNamedSuitePlans(t *testing.T) {
	fixture := readFixture(t, "diagnostics", "slice-279-yaml-provider-named-suite-plans", "go-yaml-provider-named-suite-plans.json")
	source, err := json.Marshal(fixture["manifest"])
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}

	var manifest astmerge.ConformanceManifest
	if err := json.Unmarshal(source, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}

	expectedRaw := fixture["expected_entries"].(map[string]any)["goccy_go_yaml"].([]any)
	expected := jsonReady(t, expectedRaw)

	plans := astmerge.PlanNamedConformanceSuites(manifest, map[string]astmerge.ConformanceFamilyPlanContext{
		"yaml": YAMLPlanContext(),
	})
	projected := jsonReady(t, plans)

	if !reflect.DeepEqual(projected, expected) {
		t.Fatalf("unexpected plans: %+v", projected)
	}
}

func TestSharedFixtureYAMLProviderManifestReport(t *testing.T) {
	fixture := readFixture(t, "diagnostics", "slice-280-yaml-provider-manifest-report", "go-yaml-provider-manifest-report.json")
	source, err := json.Marshal(fixture["manifest"])
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}

	var manifest astmerge.ConformanceManifest
	if err := json.Unmarshal(source, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}

	executionsSource, err := json.Marshal(fixture["executions"].(map[string]any)["goccy_go_yaml"])
	if err != nil {
		t.Fatalf("marshal executions: %v", err)
	}

	var executions map[string]astmerge.ConformanceCaseExecution
	if err := json.Unmarshal(executionsSource, &executions); err != nil {
		t.Fatalf("decode executions: %v", err)
	}

	entries := astmerge.ReportPlannedNamedConformanceSuites(
		astmerge.PlanNamedConformanceSuites(manifest, map[string]astmerge.ConformanceFamilyPlanContext{
			"yaml": YAMLPlanContext(),
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
	expected := jsonReady(t, fixture["expected_reports"].(map[string]any)["goccy_go_yaml"])
	actual := jsonReady(t, report)

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("unexpected report: %+v", actual)
	}
}
