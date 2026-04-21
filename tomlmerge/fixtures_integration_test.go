package tomlmerge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/treehaver"
)

func readTOMLFixture(t *testing.T, parts ...string) map[string]any {
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

func TestSharedFixtureTOMLFeatureProfile(t *testing.T) {
	fixture := readTOMLFixture(t, "diagnostics", "slice-90-toml-family-feature-profile", "toml-feature-profile.json")
	profile := TOMLFeatureProfileInfo()

	expected := fixture["feature_profile"].(map[string]any)
	if profile.Family != expected["family"].(string) {
		t.Fatalf("unexpected family: %+v", profile)
	}
	if len(profile.SupportedDialects) != 1 || string(profile.SupportedDialects[0]) != "toml" {
		t.Fatalf("unexpected dialects: %+v", profile.SupportedDialects)
	}
	if len(profile.SupportedPolicies) != 1 || profile.SupportedPolicies[0].Name != "destination_wins_array" {
		t.Fatalf("unexpected policies: %+v", profile.SupportedPolicies)
	}

	treeSitterProfile := TOMLBackendFeatureProfileInfo(BackendTreeSitter)
	if treeSitterProfile.Backend != "kreuzberg-language-pack" || treeSitterProfile.BackendRef == nil || treeSitterProfile.BackendRef.Family != "tree-sitter" {
		t.Fatalf("unexpected tree-sitter backend profile: %+v", treeSitterProfile)
	}
}

func TestSharedFixtureTOMLBackendFeatureProfiles(t *testing.T) {
	fixture := readTOMLFixture(t, "diagnostics", "slice-135-toml-family-backend-feature-profiles", "go-toml-backend-feature-profiles.json")

	treeSitterProfile := TOMLBackendFeatureProfileInfo(BackendTreeSitter)
	if treeSitterProfile.Backend != fixture["tree_sitter"].(map[string]any)["backend"].(string) {
		t.Fatalf("unexpected tree-sitter backend feature profile: %+v", treeSitterProfile)
	}
	if backend := treehaver.BackendReferenceByID(string(BackendTreeSitter)); backend == nil || backend.ID != string(BackendTreeSitter) || backend.Family != "tree-sitter" {
		t.Fatalf("unexpected registered backend: %+v", backend)
	}
}

func TestSharedFixtureTOMLPlanContexts(t *testing.T) {
	fixture := readTOMLFixture(t, "diagnostics", "slice-136-toml-family-plan-contexts", "go-toml-plan-contexts.json")

	context := TOMLPlanContext(BackendTreeSitter)
	if context.FamilyProfile.Family != fixture["tree_sitter"].(map[string]any)["family_profile"].(map[string]any)["family"].(string) {
		t.Fatalf("unexpected plan context: %+v", context)
	}
	if context.FeatureProfile == nil || context.FeatureProfile.Backend != fixture["tree_sitter"].(map[string]any)["feature_profile"].(map[string]any)["backend"].(string) {
		t.Fatalf("unexpected feature profile: %+v", context.FeatureProfile)
	}
}

func TestSharedFixtureTOMLManifest(t *testing.T) {
	fixture := readTOMLFixture(t, "conformance", "slice-137-toml-family-manifest", "toml-family-manifest.json")
	source, err := json.Marshal(fixture)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}

	var manifest astmerge.ConformanceManifest
	if err := json.Unmarshal(source, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}

	if path := astmerge.ConformanceFamilyFeatureProfilePath(manifest, "toml"); path == nil || filepath.Join(path...) != filepath.Join("diagnostics", "slice-90-toml-family-feature-profile", "toml-feature-profile.json") {
		t.Fatalf("unexpected family feature profile path: %v", path)
	}
	if path := astmerge.ConformanceFixturePath(manifest, "toml", "analysis"); path == nil || filepath.Join(path...) != filepath.Join("toml", "slice-92-structure", "table-and-array.json") {
		t.Fatalf("unexpected analysis fixture path: %v", path)
	}
	if path := astmerge.ConformanceFixturePath(manifest, "toml", "matching"); path == nil || filepath.Join(path...) != filepath.Join("toml", "slice-93-matching", "path-equality.json") {
		t.Fatalf("unexpected matching fixture path: %v", path)
	}
	if path := astmerge.ConformanceFixturePath(manifest, "toml", "merge"); path == nil || filepath.Join(path...) != filepath.Join("toml", "slice-94-merge", "table-merge.json") {
		t.Fatalf("unexpected merge fixture path: %v", path)
	}
}

func TestCanonicalManifestIncludesTOMLPaths(t *testing.T) {
	fixture := readTOMLFixture(t, "conformance", "slice-24-manifest", "family-feature-profiles.json")
	source, err := json.Marshal(fixture)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}

	var manifest astmerge.ConformanceManifest
	if err := json.Unmarshal(source, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}

	if path := astmerge.ConformanceFamilyFeatureProfilePath(manifest, "toml"); path == nil || filepath.Join(path...) != filepath.Join("diagnostics", "slice-90-toml-family-feature-profile", "toml-feature-profile.json") {
		t.Fatalf("unexpected canonical family feature profile path: %v", path)
	}
	if path := astmerge.ConformanceFixturePath(manifest, "toml", "analysis"); path == nil || filepath.Join(path...) != filepath.Join("toml", "slice-92-structure", "table-and-array.json") {
		t.Fatalf("unexpected canonical analysis fixture path: %v", path)
	}
}

func TestSharedFixtureTOMLParse(t *testing.T) {
	valid := readTOMLFixture(t, "toml", "slice-91-parse", "valid-document.json")
	validResult := ParseTOML(valid["source"].(string), DialectTOML)
	if !validResult.OK || validResult.Analysis == nil || string(validResult.Analysis.RootKind) != "table" {
		t.Fatalf("unexpected valid parse result: %+v", validResult)
	}
	if len(validResult.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %+v", validResult.Diagnostics)
	}

	invalid := readTOMLFixture(t, "toml", "slice-91-parse", "invalid-document.json")
	invalidResult := ParseTOML(invalid["source"].(string), DialectTOML)
	if invalidResult.OK {
		t.Fatalf("expected invalid parse failure: %+v", invalidResult)
	}
	if len(invalidResult.Diagnostics) != 1 || string(invalidResult.Diagnostics[0].Category) != "parse_error" {
		t.Fatalf("unexpected invalid diagnostics: %+v", invalidResult.Diagnostics)
	}
}

func TestSharedFixtureTOMLStructure(t *testing.T) {
	fixture := readTOMLFixture(t, "toml", "slice-92-structure", "table-and-array.json")
	result := ParseTOML(fixture["source"].(string), DialectTOML)
	if !result.OK || result.Analysis == nil {
		t.Fatalf("expected parse success: %+v", result)
	}

	expectedOwners := fixture["expected"].(map[string]any)["owners"].([]any)
	if len(result.Analysis.Owners) != len(expectedOwners) {
		t.Fatalf("unexpected owners length: %+v", result.Analysis.Owners)
	}
	for index, item := range expectedOwners {
		expected := item.(map[string]any)
		owner := result.Analysis.Owners[index]
		if owner.Path != expected["path"].(string) || string(owner.OwnerKind) != expected["owner_kind"].(string) {
			t.Fatalf("unexpected owner at %d: %+v", index, owner)
		}
		if expectedMatchKey, ok := expected["match_key"]; ok && owner.MatchKey != expectedMatchKey.(string) {
			t.Fatalf("unexpected match_key at %d: %+v", index, owner)
		}
	}
}

func TestSharedFixtureTOMLMatching(t *testing.T) {
	fixture := readTOMLFixture(t, "toml", "slice-93-matching", "path-equality.json")
	template := ParseTOML(fixture["template"].(string), DialectTOML)
	destination := ParseTOML(fixture["destination"].(string), DialectTOML)
	result := MatchTOMLOwners(*template.Analysis, *destination.Analysis)

	expected := fixture["expected"].(map[string]any)
	if len(result.Matched) != len(expected["matched"].([]any)) {
		t.Fatalf("unexpected matched owners: %+v", result.Matched)
	}
	if len(result.UnmatchedTemplate) != len(expected["unmatched_template"].([]any)) {
		t.Fatalf("unexpected unmatched template: %+v", result.UnmatchedTemplate)
	}
	if len(result.UnmatchedDestination) != len(expected["unmatched_destination"].([]any)) {
		t.Fatalf("unexpected unmatched destination: %+v", result.UnmatchedDestination)
	}
}

func TestSharedFixtureTOMLMerge(t *testing.T) {
	mergeFixture := readTOMLFixture(t, "toml", "slice-94-merge", "table-merge.json")
	mergeResult := MergeTOML(mergeFixture["template"].(string), mergeFixture["destination"].(string), DialectTOML)
	if !mergeResult.OK || mergeResult.Output == nil {
		t.Fatalf("expected merge success: %+v", mergeResult)
	}
	if *mergeResult.Output != mergeFixture["expected"].(map[string]any)["output"].(string) {
		t.Fatalf("unexpected merge output:\n%s", *mergeResult.Output)
	}

	invalidTemplate := readTOMLFixture(t, "toml", "slice-94-merge", "invalid-template.json")
	invalidTemplateResult := MergeTOML(invalidTemplate["template"].(string), invalidTemplate["destination"].(string), DialectTOML)
	if invalidTemplateResult.OK || len(invalidTemplateResult.Diagnostics) != 1 || string(invalidTemplateResult.Diagnostics[0].Category) != "parse_error" {
		t.Fatalf("unexpected invalid template result: %+v", invalidTemplateResult)
	}

	invalidDestination := readTOMLFixture(t, "toml", "slice-94-merge", "invalid-destination.json")
	invalidDestinationResult := MergeTOML(invalidDestination["template"].(string), invalidDestination["destination"].(string), DialectTOML)
	if invalidDestinationResult.OK || len(invalidDestinationResult.Diagnostics) != 1 || string(invalidDestinationResult.Diagnostics[0].Category) != "destination_parse_error" {
		t.Fatalf("unexpected invalid destination result: %+v", invalidDestinationResult)
	}
}

func TestSharedFixtureTOMLNamedSuitePlans(t *testing.T) {
	fixture := readTOMLFixture(t, "diagnostics", "slice-139-toml-family-named-suite-plans", "go-toml-named-suite-plans.json")
	source, err := json.Marshal(fixture["manifest"])
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}

	var manifest astmerge.ConformanceManifest
	if err := json.Unmarshal(source, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}

	plans := astmerge.PlanNamedConformanceSuites(manifest, map[string]astmerge.ConformanceFamilyPlanContext{
		"toml": TOMLPlanContext(BackendTreeSitter),
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

func TestSharedFixtureTOMLManifestReport(t *testing.T) {
	fixture := readTOMLFixture(t, "diagnostics", "slice-140-toml-family-manifest-report", "go-toml-manifest-report.json")
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
			"toml": TOMLPlanContext(BackendTreeSitter),
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
		"report": map[string]any{
			"entries": make([]map[string]any, 0, len(report.Entries)),
			"summary": map[string]any{
				"total": report.Summary.Total, "passed": report.Summary.Passed,
				"failed": report.Summary.Failed, "skipped": report.Summary.Skipped,
			},
		},
		"diagnostics": []any{},
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
		projected["report"].(map[string]any)["entries"] = append(projected["report"].(map[string]any)["entries"].([]map[string]any), map[string]any{
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
