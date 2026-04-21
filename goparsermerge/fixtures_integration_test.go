package goparsermerge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/gomerge"
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
	fixture := readFixture(t, "diagnostics", "slice-281-go-provider-feature-profiles", "go-go-provider-feature-profiles.json")
	if len(AvailableGoBackends()) != 1 || AvailableGoBackends()[0] != BackendGoParser {
		t.Fatalf("unexpected backends: %+v", AvailableGoBackends())
	}
	if profile := GoBackendFeatureProfileInfo(); profile.Backend != fixture["providers"].(map[string]any)["go_parser"].(map[string]any)["feature_profile"].(map[string]any)["backend"].(string) {
		t.Fatalf("unexpected provider profile: %+v", profile)
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
	if len(parseResult.Analysis.Owners) != len(parityFixture["expected"].(map[string]any)["owners"].([]any)) {
		t.Fatalf("unexpected owners: %+v", parseResult.Analysis.Owners)
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
