package pigeontomlmerge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/tomlmerge"
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

func TestSharedFixtureTOMLProviderFeatureProfile(t *testing.T) {
	familyFixture := readFixture(t, "diagnostics", "slice-90-toml-family-feature-profile", "toml-feature-profile.json")
	fixture := readFixture(t, "diagnostics", "slice-269-toml-provider-feature-profiles", "go-toml-provider-feature-profiles.json")
	familyProfile := tomlmerge.TOMLFeatureProfileInfo()
	if actual := jsonReady(t, map[string]any{
		"family":             familyProfile.Family,
		"supported_dialects": familyProfile.SupportedDialects,
		"supported_policies": familyProfile.SupportedPolicies,
	}); !reflect.DeepEqual(actual, familyFixture["feature_profile"]) {
		t.Fatalf("unexpected family profile: %+v", actual)
	}
	if len(AvailableTOMLBackends()) != 1 || AvailableTOMLBackends()[0] != BackendPigeon {
		t.Fatalf("unexpected backends: %+v", AvailableTOMLBackends())
	}
	expected := fixture["providers"].(map[string]any)["pigeon"].(map[string]any)["feature_profile"]
	profile := TOMLBackendFeatureProfileInfo()
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
	if backend := treehaver.BackendReferenceByID(BackendPigeon); backend == nil || backend.ID != BackendPigeon || backend.Family != "peg" {
		t.Fatalf("unexpected registered backend: %+v", backend)
	}
}

func TestSharedFixtureTOMLProviderPlanContext(t *testing.T) {
	fixture := readFixture(t, "diagnostics", "slice-270-toml-provider-plan-contexts", "go-toml-provider-plan-contexts.json")
	if context := TOMLPlanContext(); context.FeatureProfile == nil || context.FeatureProfile.Backend != fixture["providers"].(map[string]any)["pigeon"].(map[string]any)["feature_profile"].(map[string]any)["backend"].(string) {
		t.Fatalf("unexpected provider plan context: %+v", context)
	}
}

func TestSharedFixtureTOMLProviderParseAndMerge(t *testing.T) {
	valid := readFixture(t, "toml", "slice-91-parse", "valid-document.json")
	validResult := ParseTOML(valid["source"].(string), tomlmerge.DialectTOML)
	if !validResult.OK || validResult.Analysis == nil {
		t.Fatalf("expected parse success: %+v", validResult)
	}
	structureFixture := readFixture(t, "toml", "slice-92-structure", "table-and-array.json")
	structureResult := ParseTOML(structureFixture["source"].(string), tomlmerge.DialectTOML)
	if !structureResult.OK || structureResult.Analysis == nil {
		t.Fatalf("expected structure parse success: %+v", structureResult)
	}
	owners := make([]map[string]any, 0, len(structureResult.Analysis.Owners))
	for _, owner := range structureResult.Analysis.Owners {
		entry := map[string]any{
			"path":       owner.Path,
			"owner_kind": owner.OwnerKind,
		}
		if owner.MatchKey != "" {
			entry["match_key"] = owner.MatchKey
		}
		owners = append(owners, entry)
	}
	if actual := jsonReady(t, owners); !reflect.DeepEqual(actual, structureFixture["expected"].(map[string]any)["owners"]) {
		t.Fatalf("unexpected structure owners: %+v", actual)
	}

	matchingFixture := readFixture(t, "toml", "slice-93-matching", "path-equality.json")
	template := ParseTOML(matchingFixture["template"].(string), tomlmerge.DialectTOML)
	destination := ParseTOML(matchingFixture["destination"].(string), tomlmerge.DialectTOML)
	if !template.OK || template.Analysis == nil || !destination.OK || destination.Analysis == nil {
		t.Fatalf("expected matching parse success")
	}
	result := MatchTOMLOwners(*template.Analysis, *destination.Analysis)
	if len(result.Matched) != len(matchingFixture["expected"].(map[string]any)["matched"].([]any)) {
		t.Fatalf("unexpected matches: %+v", result.Matched)
	}

	mergeFixture := readFixture(t, "toml", "slice-94-merge", "table-merge.json")
	mergeResult := MergeTOML(mergeFixture["template"].(string), mergeFixture["destination"].(string), tomlmerge.DialectTOML)
	if !mergeResult.OK || mergeResult.Output == nil {
		t.Fatalf("expected merge success: %+v", mergeResult)
	}
	if *mergeResult.Output != mergeFixture["expected"].(map[string]any)["output"].(string) {
		t.Fatalf("unexpected merge output:\n%s", *mergeResult.Output)
	}
}

func TestSharedFixtureTOMLProviderNamedSuitePlans(t *testing.T) {
	fixture := readFixture(t, "diagnostics", "slice-271-toml-provider-named-suite-plans", "go-toml-provider-named-suite-plans.json")
	source, err := json.Marshal(fixture["manifest"])
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}

	var manifest astmerge.ConformanceManifest
	if err := json.Unmarshal(source, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}

	plans := astmerge.PlanNamedConformanceSuites(manifest, map[string]astmerge.ConformanceFamilyPlanContext{
		"toml": TOMLPlanContext(),
	})

	projected := make([]map[string]any, 0, len(plans))
	for _, entry := range plans {
		planEntries := make([]map[string]any, 0, len(entry.Plan.Entries))
		for _, planEntry := range entry.Plan.Entries {
			run := map[string]any{
				"ref":          map[string]any{"family": planEntry.Run.Ref.Family, "role": planEntry.Run.Ref.Role, "case": planEntry.Run.Ref.Case},
				"requirements": map[string]any{},
			}
			run["family_profile"] = map[string]any{
				"family":             planEntry.Run.FamilyProfile.Family,
				"supported_dialects": planEntry.Run.FamilyProfile.SupportedDialects,
				"supported_policies": planEntry.Run.FamilyProfile.SupportedPolicies,
			}
			run["feature_profile"] = map[string]any{
				"backend":            planEntry.Run.FeatureProfile.Backend,
				"supports_dialects":  planEntry.Run.FeatureProfile.SupportsDialects,
				"supported_policies": planEntry.Run.FeatureProfile.SupportedPolicies,
			}
			planEntries = append(planEntries, map[string]any{
				"ref":  map[string]any{"family": planEntry.Ref.Family, "role": planEntry.Ref.Role, "case": planEntry.Ref.Case},
				"path": planEntry.Path,
				"run":  run,
			})
		}

		projected = append(projected, map[string]any{
			"suite": entry.Suite,
			"plan": map[string]any{
				"family":        entry.Plan.Family,
				"entries":       planEntries,
				"missing_roles": entry.Plan.MissingRoles,
			},
		})
	}

	if actual := jsonReady(t, projected); !reflect.DeepEqual(actual, fixture["expected_entries"]) {
		t.Fatalf("unexpected plans: %+v", actual)
	}
}

func TestSharedFixtureTOMLProviderManifestReport(t *testing.T) {
	fixture := readFixture(t, "diagnostics", "slice-272-toml-provider-manifest-report", "go-toml-provider-manifest-report.json")
	source, err := json.Marshal(fixture["manifest"])
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}

	var manifest astmerge.ConformanceManifest
	if err := json.Unmarshal(source, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}

	executionsSource, err := json.Marshal(fixture["executions"])
	if err != nil {
		t.Fatalf("marshal executions: %v", err)
	}

	var executions map[string]astmerge.ConformanceCaseExecution
	if err := json.Unmarshal(executionsSource, &executions); err != nil {
		t.Fatalf("decode executions: %v", err)
	}

	entries := astmerge.ReportPlannedNamedConformanceSuites(
		astmerge.PlanNamedConformanceSuites(manifest, map[string]astmerge.ConformanceFamilyPlanContext{
			"toml": TOMLPlanContext(),
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
	projected := map[string]any{
		"entries": make([]map[string]any, 0, len(report.Entries)),
		"summary": map[string]any{
			"total": report.Summary.Total, "passed": report.Summary.Passed,
			"failed": report.Summary.Failed, "skipped": report.Summary.Skipped,
		},
	}
	for _, entry := range report.Entries {
		results := make([]map[string]any, 0, len(entry.Report.Results))
		for _, result := range entry.Report.Results {
			results = append(results, map[string]any{
				"ref": map[string]any{
					"family": result.Ref.Family,
					"role":   result.Ref.Role,
					"case":   result.Ref.Case,
				},
				"outcome":  result.Outcome,
				"messages": result.Messages,
			})
		}
		projected["entries"] = append(projected["entries"].([]map[string]any), map[string]any{
			"suite": entry.Suite,
			"report": map[string]any{
				"results": results,
				"summary": map[string]any{
					"total": entry.Report.Summary.Total, "passed": entry.Report.Summary.Passed,
					"failed": entry.Report.Summary.Failed, "skipped": entry.Report.Summary.Skipped,
				},
			},
		})
	}

	if actual := jsonReady(t, projected); !reflect.DeepEqual(actual, fixture["expected_report"]) {
		t.Fatalf("unexpected report: %+v", actual)
	}
}
