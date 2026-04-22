package astmerge

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
)

func readDiagnosticFixtureFromPath(t *testing.T, path string) map[string]any {
	t.Helper()

	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var fixture map[string]any
	if err := json.Unmarshal(source, &fixture); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	return fixture
}

func decodeFixtureValue[T any](t *testing.T, raw any) T {
	t.Helper()

	data, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("marshal fixture value: %v", err)
	}

	var decoded T
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal fixture value: %v", err)
	}

	return decoded
}

func decodeFixtureValueUntyped[T any](raw any) T {
	data, err := json.Marshal(raw)
	if err != nil {
		panic(err)
	}

	var decoded T
	if err := json.Unmarshal(data, &decoded); err != nil {
		panic(err)
	}

	return decoded
}

func fixtureJSONEqual(t *testing.T, actual any, expected any) bool {
	t.Helper()

	actualJSON, err := json.Marshal(actual)
	if err != nil {
		t.Fatalf("marshal actual fixture value: %v", err)
	}

	expectedJSON, err := json.Marshal(expected)
	if err != nil {
		t.Fatalf("marshal expected fixture value: %v", err)
	}

	return bytes.Equal(actualJSON, expectedJSON)
}

func readManifest(t *testing.T) ConformanceManifest {
	t.Helper()

	path := filepath.Join("..", "..", "fixtures", "conformance", "slice-24-manifest", "family-feature-profiles.json")
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}

	var manifest ConformanceManifest
	if err := json.Unmarshal(source, &manifest); err != nil {
		t.Fatalf("parse manifest: %v", err)
	}

	return manifest
}

func diagnosticsFixturePath(t *testing.T, role string) string {
	t.Helper()

	manifest := readManifest(t)
	path := ConformanceFixturePath(manifest, "diagnostics", role)
	if path == nil {
		t.Fatalf("missing diagnostics fixture entry for %s", role)
	}

	return filepath.Join(append([]string{"..", "..", "fixtures"}, path...)...)
}

func TestSharedFixtureDiagnosticVocabulary(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "diagnostic_vocabulary"))

	severities := []DiagnosticSeverity{
		SeverityInfo,
		SeverityWarning,
		SeverityError,
	}
	categories := []DiagnosticCategory{
		CategoryParseError,
		CategoryDestinationParseError,
		CategoryUnsupportedFeature,
		CategoryFallbackApplied,
		CategoryAmbiguity,
		CategoryAssumedDefault,
		CategoryConfigurationError,
		CategoryReplayRejected,
	}

	expectedSeverities := fixture["severities"].([]any)
	if len(severities) != len(expectedSeverities) {
		t.Fatalf("unexpected severities: %+v", severities)
	}
	for index, severity := range severities {
		if string(severity) != expectedSeverities[index].(string) {
			t.Fatalf("unexpected severity at %d: %s", index, severity)
		}
	}

	expectedCategories := fixture["categories"].([]any)
	if len(categories) != len(expectedCategories) {
		t.Fatalf("unexpected categories: %+v", categories)
	}
	for index, category := range categories {
		if string(category) != expectedCategories[index].(string) {
			t.Fatalf("unexpected category at %d: %s", index, category)
		}
	}
}

func TestSharedFixturePolicyVocabulary(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "policy_vocabulary"))

	surfaces := []PolicySurface{
		PolicySurfaceFallback,
		PolicySurfaceArray,
	}
	policies := []PolicyReference{
		{
			Surface: PolicySurfaceFallback,
			Name:    "trailing_comma_destination_fallback",
		},
		{
			Surface: PolicySurfaceArray,
			Name:    "destination_wins_array",
		},
	}

	expectedSurfaces := fixture["surfaces"].([]any)
	if len(surfaces) != len(expectedSurfaces) {
		t.Fatalf("unexpected surfaces: %+v", surfaces)
	}
	for index, surface := range surfaces {
		if string(surface) != expectedSurfaces[index].(string) {
			t.Fatalf("unexpected surface at %d: %s", index, surface)
		}
	}

	expectedPolicies := fixture["policies"].([]any)
	if len(policies) != len(expectedPolicies) {
		t.Fatalf("unexpected policies: %+v", policies)
	}
	for index, policy := range policies {
		expected := expectedPolicies[index].(map[string]any)
		if string(policy.Surface) != expected["surface"].(string) || policy.Name != expected["name"].(string) {
			t.Fatalf("unexpected policy at %d: %+v", index, policy)
		}
	}
}

func TestSharedFixturePolicyReporting(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "policy_reporting"))

	policies := []PolicyReference{
		{
			Surface: PolicySurfaceArray,
			Name:    "destination_wins_array",
		},
		{
			Surface: PolicySurfaceFallback,
			Name:    "trailing_comma_destination_fallback",
		},
	}

	expectedPolicies := fixture["merge_policies"].([]any)
	if len(policies) != len(expectedPolicies) {
		t.Fatalf("unexpected policies: %+v", policies)
	}
	for index, policy := range policies {
		expected := expectedPolicies[index].(map[string]any)
		if string(policy.Surface) != expected["surface"].(string) || policy.Name != expected["name"].(string) {
			t.Fatalf("unexpected policy at %d: %+v", index, policy)
		}
	}
}

func TestSharedFixtureFamilyFeatureProfile(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "shared_family_feature_profile"))

	profile := FamilyFeatureProfile{
		Family:            "example",
		SupportedDialects: []string{"alpha", "beta"},
		SupportedPolicies: []PolicyReference{
			{
				Surface: PolicySurfaceArray,
				Name:    "destination_wins_array",
			},
		},
	}

	expected := fixture["feature_profile"].(map[string]any)
	if profile.Family != expected["family"].(string) {
		t.Fatalf("unexpected family: %+v", profile)
	}
	expectedDialects := expected["supported_dialects"].([]any)
	if len(profile.SupportedDialects) != len(expectedDialects) {
		t.Fatalf("unexpected supported dialects: %+v", profile.SupportedDialects)
	}
	for index, dialect := range profile.SupportedDialects {
		if dialect != expectedDialects[index].(string) {
			t.Fatalf("unexpected dialect at %d: %s", index, dialect)
		}
	}
	assertExpectedPolicies(t, profile.SupportedPolicies, expected["supported_policies"].([]any))
}

func TestTemplateSourcePathMappingFixture(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "template_source_path_mapping"))

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		actual := NormalizeTemplateSourcePath(testCase["template_source_path"].(string))
		if actual != testCase["expected_destination_path"].(string) {
			t.Fatalf("expected %q, got %q", testCase["expected_destination_path"], actual)
		}
	}
}

func TestTemplateTargetClassificationFixture(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "template_target_classification"))

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		actual := ClassifyTemplateTargetPath(testCase["destination_path"].(string))
		expected := decodeFixtureValue[TemplateTargetClassification](t, testCase["expected"])
		if !reflect.DeepEqual(actual, expected) {
			t.Fatalf("expected classification for %q to match fixture", testCase["destination_path"])
		}
	}
}

func TestTemplateDestinationMappingFixture(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "template_destination_mapping"))

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		context := decodeFixtureValueUntyped[TemplateDestinationContext](testCase["context"])
		actual := ResolveTemplateDestinationPath(testCase["logical_destination_path"].(string), &context)
		expectedRaw := testCase["expected_destination_path"]
		if expectedRaw == nil {
			if actual != nil {
				t.Fatalf("expected nil destination path for %q", testCase["logical_destination_path"])
			}
			continue
		}

		if actual == nil || *actual != expectedRaw.(string) {
			t.Fatalf("expected %q, got %#v", expectedRaw.(string), actual)
		}
	}
}

func TestTemplateStrategySelectionFixture(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "template_strategy_selection"))

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		overrides := decodeFixtureValueUntyped[[]TemplateStrategyOverride](testCase["overrides"])
		actual := SelectTemplateStrategy(
			testCase["destination_path"].(string),
			TemplateStrategy(testCase["default_strategy"].(string)),
			overrides,
		)
		if string(actual) != testCase["expected_strategy"].(string) {
			t.Fatalf("expected %q, got %q", testCase["expected_strategy"], actual)
		}
	}
}

func TestTemplateEntryPlanFixture(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "template_entry_plan"))

	context := decodeFixtureValueUntyped[TemplateDestinationContext](fixture["context"])
	overrides := decodeFixtureValueUntyped[[]TemplateStrategyOverride](fixture["overrides"])
	templateSourcePaths := decodeFixtureValueUntyped[[]string](fixture["template_source_paths"])
	actual := PlanTemplateEntries(
		templateSourcePaths,
		&context,
		TemplateStrategy(fixture["default_strategy"].(string)),
		overrides,
	)
	expected := decodeFixtureValue[[]TemplatePlanEntry](t, fixture["expected_entries"])
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("expected template entry plan to match fixture")
	}
}

func TestSharedFixtureConformanceRunnerShape(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "runner_shape"))

	caseRef := ConformanceCaseRef{
		Family: "json",
		Role:   "tree_sitter_adapter",
		Case:   "valid_strict_json",
	}
	result := ConformanceCaseResult{
		Ref:      caseRef,
		Outcome:  ConformancePassed,
		Messages: []string{},
	}

	expectedRef := fixture["case_ref"].(map[string]any)
	if caseRef.Family != expectedRef["family"].(string) ||
		caseRef.Role != expectedRef["role"].(string) ||
		caseRef.Case != expectedRef["case"].(string) {
		t.Fatalf("unexpected case ref: %+v", caseRef)
	}

	expectedResult := fixture["result"].(map[string]any)
	expectedResultRef := expectedResult["ref"].(map[string]any)
	if result.Ref.Family != expectedResultRef["family"].(string) ||
		result.Ref.Role != expectedResultRef["role"].(string) ||
		result.Ref.Case != expectedResultRef["case"].(string) ||
		string(result.Outcome) != expectedResult["outcome"].(string) {
		t.Fatalf("unexpected runner result: %+v", result)
	}
	if len(result.Messages) != len(expectedResult["messages"].([]any)) {
		t.Fatalf("unexpected runner messages: %+v", result.Messages)
	}
}

func TestSharedFixtureNormalizedManifestContract(t *testing.T) {
	manifest := readManifest(t)

	jsonProfilePath := ConformanceFamilyFeatureProfilePath(manifest, "json")
	if filepath.Join(jsonProfilePath...) != filepath.Join("diagnostics", "slice-21-family-feature-profile", "json-feature-profile.json") {
		t.Fatalf("unexpected json family profile path: %v", jsonProfilePath)
	}

	textAnalysisPath := ConformanceFixturePath(manifest, "text", "analysis")
	if filepath.Join(textAnalysisPath...) != filepath.Join("text", "slice-03-analysis", "whitespace-and-blocks.json") {
		t.Fatalf("unexpected text analysis path: %v", textAnalysisPath)
	}

	runnerShapePath := ConformanceFixturePath(manifest, "diagnostics", "runner_shape")
	if filepath.Join(runnerShapePath...) != filepath.Join("diagnostics", "slice-28-conformance-runner", "runner-shape.json") {
		t.Fatalf("unexpected runner shape path: %v", runnerShapePath)
	}
}

func TestSharedFixtureConformanceSuiteSummary(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "runner_summary"))

	rawResults := fixture["results"].([]any)
	results := make([]ConformanceCaseResult, 0, len(rawResults))
	for _, item := range rawResults {
		entry := item.(map[string]any)
		ref := entry["ref"].(map[string]any)
		messages := entry["messages"].([]any)
		normalizedMessages := make([]string, 0, len(messages))
		for _, message := range messages {
			normalizedMessages = append(normalizedMessages, message.(string))
		}

		results = append(results, ConformanceCaseResult{
			Ref: ConformanceCaseRef{
				Family: ref["family"].(string),
				Role:   ref["role"].(string),
				Case:   ref["case"].(string),
			},
			Outcome:  ConformanceOutcome(entry["outcome"].(string)),
			Messages: normalizedMessages,
		})
	}

	expected := fixture["summary"].(map[string]any)
	summary := SummarizeConformanceResults(results)
	if summary.Total != int(expected["total"].(float64)) ||
		summary.Passed != int(expected["passed"].(float64)) ||
		summary.Failed != int(expected["failed"].(float64)) ||
		summary.Skipped != int(expected["skipped"].(float64)) {
		t.Fatalf("unexpected summary: %+v", summary)
	}
}

func TestSharedFixtureCapabilityAwareSelection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "capability_selection"))

	for _, item := range fixture["cases"].([]any) {
		testCase := item.(map[string]any)
		rawRef := testCase["ref"].(map[string]any)
		ref := ConformanceCaseRef{
			Family: rawRef["family"].(string),
			Role:   rawRef["role"].(string),
			Case:   rawRef["case"].(string),
		}

		rawRequirements := testCase["requirements"].(map[string]any)
		requirements := ConformanceCaseRequirements{}
		if dialect, ok := rawRequirements["dialect"]; ok {
			requirements.Dialect = dialect.(string)
		}
		if rawPolicies, ok := rawRequirements["policies"]; ok {
			requirements.Policies = make([]PolicyReference, 0, len(rawPolicies.([]any)))
			for _, item := range rawPolicies.([]any) {
				policy := item.(map[string]any)
				requirements.Policies = append(requirements.Policies, PolicyReference{
					Surface: PolicySurface(policy["surface"].(string)),
					Name:    policy["name"].(string),
				})
			}
		}

		rawFamilyProfile := testCase["family_profile"].(map[string]any)
		familyProfile := FamilyFeatureProfile{
			Family:            rawFamilyProfile["family"].(string),
			SupportedDialects: make([]string, 0, len(rawFamilyProfile["supported_dialects"].([]any))),
			SupportedPolicies: make([]PolicyReference, 0, len(rawFamilyProfile["supported_policies"].([]any))),
		}
		for _, dialect := range rawFamilyProfile["supported_dialects"].([]any) {
			familyProfile.SupportedDialects = append(familyProfile.SupportedDialects, dialect.(string))
		}
		for _, item := range rawFamilyProfile["supported_policies"].([]any) {
			policy := item.(map[string]any)
			familyProfile.SupportedPolicies = append(familyProfile.SupportedPolicies, PolicyReference{
				Surface: PolicySurface(policy["surface"].(string)),
				Name:    policy["name"].(string),
			})
		}

		rawFeatureProfile := testCase["feature_profile"].(map[string]any)
		featureProfile := &ConformanceFeatureProfileView{
			Backend:           rawFeatureProfile["backend"].(string),
			SupportsDialects:  rawFeatureProfile["supports_dialects"].(bool),
			SupportedPolicies: []PolicyReference{},
		}
		for _, item := range rawFeatureProfile["supported_policies"].([]any) {
			policy := item.(map[string]any)
			featureProfile.SupportedPolicies = append(featureProfile.SupportedPolicies, PolicyReference{
				Surface: PolicySurface(policy["surface"].(string)),
				Name:    policy["name"].(string),
			})
		}

		selection := SelectConformanceCase(ref, requirements, familyProfile, featureProfile)
		expected := testCase["expected"].(map[string]any)
		if string(selection.Status) != expected["status"].(string) {
			t.Fatalf("unexpected selection status for %+v: %+v", ref, selection)
		}
		expectedMessages := expected["messages"].([]any)
		if len(selection.Messages) != len(expectedMessages) {
			t.Fatalf("unexpected selection messages for %+v: %+v", ref, selection.Messages)
		}
		for index, message := range expectedMessages {
			if selection.Messages[index] != message.(string) {
				t.Fatalf("unexpected selection message at %d for %+v: %+v", index, ref, selection.Messages)
			}
		}
	}
}

func TestSharedFixtureBackendAwareSelection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "backend_selection"))

	for _, item := range fixture["cases"].([]any) {
		testCase := item.(map[string]any)
		rawRef := testCase["ref"].(map[string]any)
		ref := ConformanceCaseRef{
			Family: rawRef["family"].(string),
			Role:   rawRef["role"].(string),
			Case:   rawRef["case"].(string),
		}

		requirements := parseConformanceCaseRequirements(testCase["requirements"].(map[string]any))
		familyProfile := parseFamilyFeatureProfile(testCase["family_profile"].(map[string]any))
		featureProfile := parseFeatureProfilePointer(testCase["feature_profile"])

		selection := SelectConformanceCase(ref, requirements, familyProfile, featureProfile)
		expected := testCase["expected"].(map[string]any)
		if string(selection.Status) != expected["status"].(string) {
			t.Fatalf("unexpected selection status: %+v", selection)
		}
		expectedMessages := make([]string, 0, len(expected["messages"].([]any)))
		for _, message := range expected["messages"].([]any) {
			expectedMessages = append(expectedMessages, message.(string))
		}
		if !reflect.DeepEqual(selection.Messages, expectedMessages) {
			t.Fatalf("unexpected selection messages: %+v", selection.Messages)
		}
	}
}

func TestSharedFixtureConformanceCaseRunner(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "case_runner"))

	for _, item := range fixture["cases"].([]any) {
		testCase := item.(map[string]any)
		run := parseConformanceCaseRun(t, testCase["run"].(map[string]any))
		execution := parseConformanceCaseExecution(testCase["execution"].(map[string]any))
		expected := parseConformanceCaseResult(testCase["expected"].(map[string]any))

		result := RunConformanceCase(run, func(ConformanceCaseRun) ConformanceCaseExecution {
			return execution
		})
		if !reflect.DeepEqual(result, expected) {
			t.Fatalf("unexpected case runner result: %+v", result)
		}
	}
}

func TestSharedFixtureConformanceSuiteRunner(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "suite_runner"))
	runsRaw := fixture["cases"].([]any)
	runs := make([]ConformanceCaseRun, 0, len(runsRaw))
	for _, item := range runsRaw {
		runs = append(runs, parseConformanceCaseRun(t, item.(map[string]any)))
	}

	executionsRaw := fixture["executions"].(map[string]any)
	expectedRaw := fixture["expected_results"].([]any)
	expected := make([]ConformanceCaseResult, 0, len(expectedRaw))
	for _, item := range expectedRaw {
		expected = append(expected, parseConformanceCaseResult(item.(map[string]any)))
	}

	results := RunConformanceSuite(runs, func(run ConformanceCaseRun) ConformanceCaseExecution {
		key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
		if raw, ok := executionsRaw[key]; ok {
			return parseConformanceCaseExecution(raw.(map[string]any))
		}

		return ConformanceCaseExecution{
			Outcome:  ConformanceFailed,
			Messages: []string{"missing execution"},
		}
	})

	if len(results) != len(expected) {
		t.Fatalf("unexpected suite runner results: %+v", results)
	}
	for index := range results {
		if !reflect.DeepEqual(results[index], expected[index]) {
			t.Fatalf("unexpected suite runner result at %d: %+v", index, results[index])
		}
	}
}

func TestSharedFixtureConformanceSuiteReport(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "suite_report"))
	rawResults := fixture["results"].([]any)
	results := make([]ConformanceCaseResult, 0, len(rawResults))
	for _, item := range rawResults {
		results = append(results, parseConformanceCaseResult(item.(map[string]any)))
	}

	rawReport := fixture["report"].(map[string]any)
	report := ReportConformanceSuite(results)
	expectedSummary := rawReport["summary"].(map[string]any)
	if report.Summary.Total != int(expectedSummary["total"].(float64)) ||
		report.Summary.Passed != int(expectedSummary["passed"].(float64)) ||
		report.Summary.Failed != int(expectedSummary["failed"].(float64)) ||
		report.Summary.Skipped != int(expectedSummary["skipped"].(float64)) {
		t.Fatalf("unexpected suite report summary: %+v", report.Summary)
	}

	expectedResults := rawReport["results"].([]any)
	if len(report.Results) != len(expectedResults) {
		t.Fatalf("unexpected suite report results: %+v", report.Results)
	}
	for index, item := range expectedResults {
		expected := parseConformanceCaseResult(item.(map[string]any))
		if !reflect.DeepEqual(report.Results[index], expected) {
			t.Fatalf("unexpected suite report result at %d: %+v", index, report.Results[index])
		}
	}
}

func TestSharedFixtureConformanceSuitePlan(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "suite_plan"))
	manifest := readManifest(t)

	rolesRaw := fixture["roles"].([]any)
	roles := make([]string, 0, len(rolesRaw))
	for _, item := range rolesRaw {
		roles = append(roles, item.(string))
	}

	familyProfile := parseFamilyFeatureProfile(fixture["family_profile"].(map[string]any))
	featureProfile := parseFeatureProfilePointer(fixture["feature_profile"])
	expected := parseConformanceSuitePlan(t, fixture["expected"].(map[string]any))

	plan := PlanConformanceSuite(
		manifest,
		fixture["family"].(string),
		roles,
		familyProfile,
		featureProfile,
	)

	if !reflect.DeepEqual(plan, expected) {
		t.Fatalf("unexpected suite plan: %+v", plan)
	}
}

func TestSharedFixturePlannedConformanceSuiteRunner(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "planned_suite_runner"))
	plan := parseConformanceSuitePlan(t, fixture["plan"].(map[string]any))
	executionsRaw := fixture["executions"].(map[string]any)
	expectedRaw := fixture["expected_results"].([]any)
	expected := make([]ConformanceCaseResult, 0, len(expectedRaw))
	for _, item := range expectedRaw {
		expected = append(expected, parseConformanceCaseResult(item.(map[string]any)))
	}

	results := RunPlannedConformanceSuite(plan, func(run ConformanceCaseRun) ConformanceCaseExecution {
		key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
		if raw, ok := executionsRaw[key]; ok {
			return parseConformanceCaseExecution(raw.(map[string]any))
		}

		return ConformanceCaseExecution{
			Outcome:  ConformanceFailed,
			Messages: []string{"missing execution"},
		}
	})

	if !reflect.DeepEqual(results, expected) {
		t.Fatalf("unexpected planned suite runner results: %+v", results)
	}
}

func TestSharedFixturePlannedConformanceSuiteReport(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "planned_suite_report"))
	plan := parseConformanceSuitePlan(t, fixture["plan"].(map[string]any))
	executionsRaw := fixture["executions"].(map[string]any)
	expected := fixture["expected_report"].(map[string]any)

	report := ReportPlannedConformanceSuite(plan, func(run ConformanceCaseRun) ConformanceCaseExecution {
		key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
		if raw, ok := executionsRaw[key]; ok {
			return parseConformanceCaseExecution(raw.(map[string]any))
		}

		return ConformanceCaseExecution{
			Outcome:  ConformanceFailed,
			Messages: []string{"missing execution"},
		}
	})

	expectedSummary := expected["summary"].(map[string]any)
	if report.Summary.Total != int(expectedSummary["total"].(float64)) ||
		report.Summary.Passed != int(expectedSummary["passed"].(float64)) ||
		report.Summary.Failed != int(expectedSummary["failed"].(float64)) ||
		report.Summary.Skipped != int(expectedSummary["skipped"].(float64)) {
		t.Fatalf("unexpected planned suite report summary: %+v", report.Summary)
	}

	expectedResults := expected["results"].([]any)
	if len(report.Results) != len(expectedResults) {
		t.Fatalf("unexpected planned suite report results: %+v", report.Results)
	}
	for index, item := range expectedResults {
		expectedResult := parseConformanceCaseResult(item.(map[string]any))
		if !reflect.DeepEqual(report.Results[index], expectedResult) {
			t.Fatalf("unexpected planned suite report result at %d: %+v", index, report.Results[index])
		}
	}
}

func TestSharedFixtureManifestCaseRequirements(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "manifest_requirements"))
	manifest := readManifest(t)
	rolesRaw := fixture["roles"].([]any)
	roles := make([]string, 0, len(rolesRaw))
	for _, item := range rolesRaw {
		roles = append(roles, item.(string))
	}

	plan := PlanConformanceSuite(
		manifest,
		fixture["family"].(string),
		roles,
		parseFamilyFeatureProfile(fixture["family_profile"].(map[string]any)),
		nil,
	)

	expectedRaw := fixture["expected_requirements"].(map[string]any)
	actual := make(map[string]ConformanceCaseRequirements, len(plan.Entries))
	for _, entry := range plan.Entries {
		actual[entry.Ref.Role] = entry.Run.Requirements
	}

	if len(actual) != len(expectedRaw) {
		t.Fatalf("unexpected manifest requirements entries: %+v", actual)
	}
	for role, raw := range expectedRaw {
		expected := parseConformanceCaseRequirements(raw.(map[string]any))
		if !reflect.DeepEqual(actual[role], expected) {
			t.Fatalf("unexpected requirements for %s: %+v", role, actual[role])
		}
	}
}

func TestSharedFixtureManifestBackendRequirements(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "manifest_backend_requirements"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(fixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	rolesRaw := fixture["roles"].([]any)
	roles := make([]string, 0, len(rolesRaw))
	for _, item := range rolesRaw {
		roles = append(roles, item.(string))
	}

	plan := PlanConformanceSuite(
		manifest,
		fixture["family"].(string),
		roles,
		parseFamilyFeatureProfile(fixture["family_profile"].(map[string]any)),
		parseFeatureProfilePointer(fixture["feature_profile"]),
	)

	expected := parseConformanceSuitePlan(t, fixture["expected"].(map[string]any))
	if !reflect.DeepEqual(plan, expected) {
		t.Fatalf("unexpected manifest backend requirements plan: %+v", plan)
	}
}

func TestSharedFixtureManifestBackendReport(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "manifest_backend_report"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(fixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	rolesRaw := fixture["roles"].([]any)
	roles := make([]string, 0, len(rolesRaw))
	for _, item := range rolesRaw {
		roles = append(roles, item.(string))
	}

	plan := PlanConformanceSuite(
		manifest,
		fixture["family"].(string),
		roles,
		parseFamilyFeatureProfile(fixture["family_profile"].(map[string]any)),
		parseFeatureProfilePointer(fixture["feature_profile"]),
	)

	report := ReportPlannedConformanceSuite(
		plan,
		func(ConformanceCaseRun) ConformanceCaseExecution {
			return ConformanceCaseExecution{
				Outcome:  ConformanceFailed,
				Messages: []string{"unexpected execution"},
			}
		},
	)

	expected := parseConformanceSuiteReport(fixture["expected_report"].(map[string]any))
	if !reflect.DeepEqual(report, expected) {
		t.Fatalf("unexpected manifest backend report: %+v", report)
	}
}

func TestSharedFixtureConformanceSuiteDefinitions(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "suite_definitions"))
	manifest := readManifest(t)
	selector := parseConformanceSuiteSelector(fixture["suite_selector"].(map[string]any))
	expected := parseConformanceSuiteDefinition(fixture["expected"].(map[string]any))

	definition := ConformanceSuiteDefinitionForSelector(manifest, selector)
	if definition == nil || !reflect.DeepEqual(*definition, expected) {
		t.Fatalf("unexpected suite definition: %+v", definition)
	}

	planned := PlanNamedConformanceSuite(
		manifest,
		selector,
		FamilyFeatureProfile{
			Family:            "json",
			SupportedDialects: []string{"json", "jsonc"},
			SupportedPolicies: []PolicyReference{
				{Surface: PolicySurfaceArray, Name: "destination_wins_array"},
				{Surface: PolicySurfaceFallback, Name: "trailing_comma_destination_fallback"},
			},
		},
		nil,
	)
	explicit := PlanConformanceSuite(
		manifest,
		expected.Subject.Grammar,
		expected.Roles,
		FamilyFeatureProfile{
			Family:            "json",
			SupportedDialects: []string{"json", "jsonc"},
			SupportedPolicies: []PolicyReference{
				{Surface: PolicySurfaceArray, Name: "destination_wins_array"},
				{Surface: PolicySurfaceFallback, Name: "trailing_comma_destination_fallback"},
			},
		},
		nil,
	)
	if planned == nil || !reflect.DeepEqual(*planned, explicit) {
		t.Fatalf("unexpected planned suite from definition: %+v", planned)
	}
}

func TestSharedFixtureNamedConformanceSuiteReport(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "named_suite_report"))
	manifest := readManifest(t)
	selector := parseConformanceSuiteSelector(fixture["suite_selector"].(map[string]any))
	executionsRaw := fixture["executions"].(map[string]any)
	expected := fixture["expected_report"].(map[string]any)

	report := ReportNamedConformanceSuite(
		manifest,
		selector,
		parseFamilyFeatureProfile(fixture["family_profile"].(map[string]any)),
		func(run ConformanceCaseRun) ConformanceCaseExecution {
			key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
			if raw, ok := executionsRaw[key]; ok {
				return parseConformanceCaseExecution(raw.(map[string]any))
			}

			return ConformanceCaseExecution{
				Outcome:  ConformanceFailed,
				Messages: []string{"missing execution"},
			}
		},
		&ConformanceFeatureProfileView{
			Backend:           "kreuzberg-language-pack",
			SupportsDialects:  false,
			SupportedPolicies: []PolicyReference{{Surface: PolicySurfaceArray, Name: "destination_wins_array"}},
		},
	)

	if report == nil {
		t.Fatalf("expected named suite report")
	}

	expectedSummary := expected["summary"].(map[string]any)
	if report.Summary.Total != int(expectedSummary["total"].(float64)) ||
		report.Summary.Passed != int(expectedSummary["passed"].(float64)) ||
		report.Summary.Failed != int(expectedSummary["failed"].(float64)) ||
		report.Summary.Skipped != int(expectedSummary["skipped"].(float64)) {
		t.Fatalf("unexpected named suite report summary: %+v", report.Summary)
	}

	expectedResults := expected["results"].([]any)
	if len(report.Results) != len(expectedResults) {
		t.Fatalf("unexpected named suite report results: %+v", report.Results)
	}
	for index, item := range expectedResults {
		expectedResult := parseConformanceCaseResult(item.(map[string]any))
		if !reflect.DeepEqual(report.Results[index], expectedResult) {
			t.Fatalf("unexpected named suite report result at %d: %+v", index, report.Results[index])
		}
	}
}

func TestSharedFixtureNamedConformanceSuiteRunner(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "named_suite_runner"))
	manifest := readManifest(t)
	selector := parseConformanceSuiteSelector(fixture["suite_selector"].(map[string]any))
	executionsRaw := fixture["executions"].(map[string]any)
	expectedResults := fixture["expected_results"].([]any)

	results := RunNamedConformanceSuite(
		manifest,
		selector,
		parseFamilyFeatureProfile(fixture["family_profile"].(map[string]any)),
		func(run ConformanceCaseRun) ConformanceCaseExecution {
			key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
			if raw, ok := executionsRaw[key]; ok {
				return parseConformanceCaseExecution(raw.(map[string]any))
			}

			return ConformanceCaseExecution{
				Outcome:  ConformanceFailed,
				Messages: []string{"missing execution"},
			}
		},
		&ConformanceFeatureProfileView{
			Backend:           "kreuzberg-language-pack",
			SupportsDialects:  false,
			SupportedPolicies: []PolicyReference{{Surface: PolicySurfaceArray, Name: "destination_wins_array"}},
		},
	)

	if len(results) != len(expectedResults) {
		t.Fatalf("unexpected named suite runner results: %+v", results)
	}
	for index, item := range expectedResults {
		expectedResult := parseConformanceCaseResult(item.(map[string]any))
		if !reflect.DeepEqual(results[index], expectedResult) {
			t.Fatalf("unexpected named suite runner result at %d: %+v", index, results[index])
		}
	}
}

func TestSharedFixtureConformanceSuiteNames(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "suite_names"))
	manifest := readManifest(t)

	expectedRaw := fixture["suite_selectors"].([]any)
	expected := make([]ConformanceSuiteSelector, 0, len(expectedRaw))
	for _, selector := range expectedRaw {
		expected = append(expected, parseConformanceSuiteSelector(selector.(map[string]any)))
	}

	if selectors := ConformanceSuiteSelectors(manifest); !reflect.DeepEqual(selectors, expected) {
		t.Fatalf("unexpected suite selectors: %+v", selectors)
	}
}

func TestSlice125SourceFamilySuiteDefinitions(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-125-source-family-suite-definitions", "source-suite-definitions.json"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(fixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}

	expectedRaw := fixture["suite_selectors"].([]any)
	expectedSelectors := make([]ConformanceSuiteSelector, 0, len(expectedRaw))
	for _, selector := range expectedRaw {
		expectedSelectors = append(expectedSelectors, parseConformanceSuiteSelector(selector.(map[string]any)))
	}
	if selectors := ConformanceSuiteSelectors(manifest); !reflect.DeepEqual(selectors, expectedSelectors) {
		t.Fatalf("unexpected source suite selectors: %+v", selectors)
	}

	definitionsRaw := fixture["suite_definitions"].([]any)
	for index, raw := range definitionsRaw {
		expected := parseConformanceSuiteDefinition(raw.(map[string]any))
		actual := ConformanceSuiteDefinitionForSelector(manifest, expectedSelectors[index])
		if actual == nil || !reflect.DeepEqual(*actual, expected) {
			t.Fatalf("unexpected source suite definition at %d: %+v", index, actual)
		}
	}
}

func TestSharedFixtureNamedConformanceSuiteEntry(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "named_suite_entry"))
	manifest := readManifest(t)
	selector := parseConformanceSuiteSelector(fixture["suite_selector"].(map[string]any))
	executionsRaw := fixture["executions"].(map[string]any)
	expectedRaw := fixture["expected_entry"].(map[string]any)

	entry := ReportNamedConformanceSuiteEntry(
		manifest,
		selector,
		parseFamilyFeatureProfile(fixture["family_profile"].(map[string]any)),
		func(run ConformanceCaseRun) ConformanceCaseExecution {
			key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
			if raw, ok := executionsRaw[key]; ok {
				return parseConformanceCaseExecution(raw.(map[string]any))
			}

			return ConformanceCaseExecution{
				Outcome:  ConformanceFailed,
				Messages: []string{"missing execution"},
			}
		},
		&ConformanceFeatureProfileView{
			Backend:           "kreuzberg-language-pack",
			SupportsDialects:  false,
			SupportedPolicies: []PolicyReference{{Surface: PolicySurfaceArray, Name: "destination_wins_array"}},
		},
	)

	if entry == nil {
		t.Fatalf("expected named suite entry")
	}
	if !reflect.DeepEqual(entry.Suite, parseConformanceSuiteDefinition(expectedRaw["suite"].(map[string]any))) {
		t.Fatalf("unexpected named suite entry suite: %+v", entry)
	}

	expectedReport := expectedRaw["report"].(map[string]any)
	expectedSummary := expectedReport["summary"].(map[string]any)
	if entry.Report.Summary.Total != int(expectedSummary["total"].(float64)) ||
		entry.Report.Summary.Passed != int(expectedSummary["passed"].(float64)) ||
		entry.Report.Summary.Failed != int(expectedSummary["failed"].(float64)) ||
		entry.Report.Summary.Skipped != int(expectedSummary["skipped"].(float64)) {
		t.Fatalf("unexpected named suite entry summary: %+v", entry.Report.Summary)
	}

	expectedResults := expectedReport["results"].([]any)
	if len(entry.Report.Results) != len(expectedResults) {
		t.Fatalf("unexpected named suite entry results: %+v", entry.Report.Results)
	}
	for index, item := range expectedResults {
		expectedResult := parseConformanceCaseResult(item.(map[string]any))
		if !reflect.DeepEqual(entry.Report.Results[index], expectedResult) {
			t.Fatalf("unexpected named suite entry result at %d: %+v", index, entry.Report.Results[index])
		}
	}
}

func TestSharedFixtureNamedConformanceSuitePlanEntry(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "named_suite_plan_entry"))
	manifest := readManifest(t)
	selector := parseConformanceSuiteSelector(fixture["suite_selector"].(map[string]any))

	context := parseConformanceFamilyPlanContext(fixture["context"].(map[string]any))
	entry := PlanNamedConformanceSuiteEntry(manifest, selector, context)
	if entry == nil {
		t.Fatalf("expected named suite plan entry")
	}

	expected := parseNamedConformanceSuitePlan(t, fixture["expected_entry"].(map[string]any))
	if !fixtureJSONEqual(t, *entry, expected) {
		t.Fatalf("unexpected named suite plan entry: %+v", entry)
	}
}

func TestSharedFixtureConformanceFamilyPlanContext(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "family_plan_context"))
	context := parseConformanceFamilyPlanContext(fixture["context"].(map[string]any))

	expected := ConformanceFamilyPlanContext{
		FamilyProfile: FamilyFeatureProfile{
			Family:            "json",
			SupportedDialects: []string{"json", "jsonc"},
			SupportedPolicies: []PolicyReference{
				{Surface: PolicySurfaceArray, Name: "destination_wins_array"},
				{Surface: PolicySurfaceFallback, Name: "trailing_comma_destination_fallback"},
			},
		},
		FeatureProfile: &ConformanceFeatureProfileView{
			Backend:           "kreuzberg-language-pack",
			SupportsDialects:  false,
			SupportedPolicies: []PolicyReference{{Surface: PolicySurfaceArray, Name: "destination_wins_array"}},
		},
	}

	if !reflect.DeepEqual(context, expected) {
		t.Fatalf("unexpected family plan context: %+v", context)
	}
}

func TestSharedFixtureNamedConformanceSuitePlans(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "named_suite_plans"))
	manifest := readManifest(t)
	contextsRaw := fixture["contexts"].(map[string]any)
	contexts := make(map[string]ConformanceFamilyPlanContext, len(contextsRaw))
	for family, raw := range contextsRaw {
		contexts[family] = parseConformanceFamilyPlanContext(raw.(map[string]any))
	}

	expectedRaw := fixture["expected_entries"].([]any)
	expected := make([]NamedConformanceSuitePlan, 0, len(expectedRaw))
	for _, raw := range expectedRaw {
		expected = append(expected, parseNamedConformanceSuitePlan(t, raw.(map[string]any)))
	}

	if plans := PlanNamedConformanceSuites(manifest, contexts); !fixtureJSONEqual(t, plans, expected) {
		t.Fatalf("unexpected named suite plans: %+v", plans)
	}
}

func TestSlice126SourceFamilyNamedSuitePlans(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-126-source-family-named-suite-plans", "source-named-suite-plans.json"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(fixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}

	contextsRaw := fixture["contexts"].(map[string]any)
	contexts := make(map[string]ConformanceFamilyPlanContext, len(contextsRaw))
	for family, raw := range contextsRaw {
		contexts[family] = parseConformanceFamilyPlanContext(raw.(map[string]any))
	}

	expectedRaw := fixture["expected_entries"].([]any)
	expected := make([]NamedConformanceSuitePlan, 0, len(expectedRaw))
	for _, raw := range expectedRaw {
		expected = append(expected, parseNamedConformanceSuitePlan(t, raw.(map[string]any)))
	}

	if plans := PlanNamedConformanceSuites(manifest, contexts); !fixtureJSONEqual(t, plans, expected) {
		t.Fatalf("unexpected source named suite plans: %+v", plans)
	}
}

func TestSlice127SourceFamilyNativeSuitePlans(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-127-source-family-native-suite-plans", "source-native-named-suite-plans.json"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(fixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}

	contextsRaw := fixture["contexts"].(map[string]any)
	contexts := make(map[string]ConformanceFamilyPlanContext, len(contextsRaw))
	for family, raw := range contextsRaw {
		contexts[family] = parseConformanceFamilyPlanContext(raw.(map[string]any))
	}

	expectedRaw := fixture["expected_entries"].([]any)
	expected := make([]NamedConformanceSuitePlan, 0, len(expectedRaw))
	for _, raw := range expectedRaw {
		expected = append(expected, parseNamedConformanceSuitePlan(t, raw.(map[string]any)))
	}

	if plans := PlanNamedConformanceSuites(manifest, contexts); !fixtureJSONEqual(t, plans, expected) {
		t.Fatalf("unexpected source native named suite plans: %+v", plans)
	}
}

func TestSlice138TOMLFamilySuiteDefinitions(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-138-toml-family-suite-definitions", "toml-suite-definitions.json"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(fixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}

	expectedSelectors := []ConformanceSuiteSelector{{Kind: "portable", Subject: ConformanceSuiteSubject{Grammar: "toml"}}}
	if selectors := ConformanceSuiteSelectors(manifest); !reflect.DeepEqual(selectors, expectedSelectors) {
		t.Fatalf("unexpected TOML suite selectors: %+v", selectors)
	}
	expectedDefinition := ConformanceSuiteDefinition{Kind: "portable", Subject: ConformanceSuiteSubject{Grammar: "toml"}, Roles: []string{"analysis", "matching", "merge"}}
	if definition := ConformanceSuiteDefinitionForSelector(manifest, expectedSelectors[0]); !reflect.DeepEqual(definition, &expectedDefinition) {
		t.Fatalf("unexpected TOML suite definition: %+v", definition)
	}
}

func TestSlice139TOMLFamilyNamedSuitePlans(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-139-toml-family-named-suite-plans", "go-toml-named-suite-plans.json"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(fixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}

	contextsRaw := fixture["contexts"].(map[string]any)
	contexts := make(map[string]ConformanceFamilyPlanContext, len(contextsRaw))
	for family, raw := range contextsRaw {
		contexts[family] = parseConformanceFamilyPlanContext(raw.(map[string]any))
	}

	expectedRaw := fixture["expected_entries"].([]any)
	expected := make([]NamedConformanceSuitePlan, 0, len(expectedRaw))
	for _, raw := range expectedRaw {
		expected = append(expected, parseNamedConformanceSuitePlan(t, raw.(map[string]any)))
	}

	if plans := PlanNamedConformanceSuites(manifest, contexts); !fixtureJSONEqual(t, plans, expected) {
		t.Fatalf("unexpected TOML named suite plans: %+v", plans)
	}
}

func TestSlice200MarkdownFamilySuiteDefinitions(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-200-markdown-family-suite-definitions", "markdown-suite-definitions.json"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(fixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}

	expectedSelectors := []ConformanceSuiteSelector{{Kind: "portable", Subject: ConformanceSuiteSubject{Grammar: "markdown"}}}
	if selectors := ConformanceSuiteSelectors(manifest); !reflect.DeepEqual(selectors, expectedSelectors) {
		t.Fatalf("unexpected Markdown suite selectors: %+v", selectors)
	}
	expectedDefinition := ConformanceSuiteDefinition{Kind: "portable", Subject: ConformanceSuiteSubject{Grammar: "markdown"}, Roles: []string{"analysis", "matching", "merge"}}
	if definition := ConformanceSuiteDefinitionForSelector(manifest, expectedSelectors[0]); !reflect.DeepEqual(definition, &expectedDefinition) {
		t.Fatalf("unexpected Markdown suite definition: %+v", definition)
	}
}

func TestSlice201MarkdownFamilyNamedSuitePlans(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-201-markdown-family-named-suite-plans", "go-markdown-named-suite-plans.json"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(fixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}

	contextsRaw := fixture["contexts"].(map[string]any)
	contexts := make(map[string]ConformanceFamilyPlanContext, len(contextsRaw))
	for family, raw := range contextsRaw {
		contexts[family] = parseConformanceFamilyPlanContext(raw.(map[string]any))
	}

	expectedRaw := fixture["expected_entries"].([]any)
	expected := make([]NamedConformanceSuitePlan, 0, len(expectedRaw))
	for _, raw := range expectedRaw {
		expected = append(expected, parseNamedConformanceSuitePlan(t, raw.(map[string]any)))
	}

	if plans := PlanNamedConformanceSuites(manifest, contexts); !fixtureJSONEqual(t, plans, expected) {
		t.Fatalf("unexpected Markdown named suite plans: %+v", plans)
	}
}

func TestSlice202MarkdownFamilyManifestReport(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-202-markdown-family-manifest-report", "go-markdown-manifest-report.json"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(fixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	options := parseConformanceManifestPlanningOptions(fixture["options"].(map[string]any))
	expected := parseConformanceManifestReport(fixture["expected_report"].(map[string]any))
	executionsRaw := fixture["executions"].(map[string]any)
	executions := make(map[string]ConformanceCaseExecution, len(executionsRaw))
	for key, raw := range executionsRaw {
		executions[key] = parseConformanceCaseExecution(raw.(map[string]any))
	}

	report := ReportConformanceManifest(manifest, options, func(run ConformanceCaseRun) ConformanceCaseExecution {
		key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
		if execution, ok := executions[key]; ok {
			return execution
		}
		return ConformanceCaseExecution{Outcome: ConformanceFailed, Messages: []string{"missing execution"}}
	})

	if !reflect.DeepEqual(report, expected) {
		t.Fatalf("unexpected Markdown manifest report: %+v", report)
	}
}

func TestSlice246MarkdownNestedSuiteDefinitions(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-246-markdown-nested-suite-definitions", "markdown-nested-suite-definitions.json"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(fixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	expectedSelectors := []ConformanceSuiteSelector{{Kind: "portable", Subject: ConformanceSuiteSubject{Grammar: "markdown", Variant: "nested"}}}
	if selectors := ConformanceSuiteSelectors(manifest); !reflect.DeepEqual(selectors, expectedSelectors) {
		t.Fatalf("unexpected Markdown nested suite selectors: %+v", selectors)
	}
	expectedDefinition := ConformanceSuiteDefinition{Kind: "portable", Subject: ConformanceSuiteSubject{Grammar: "markdown", Variant: "nested"}, Roles: []string{"analysis", "matching", "embedded_families", "discovered_surfaces", "delegated_child_operations", "delegated_child_review_transport", "delegated_child_review_state", "delegated_child_apply_plan"}}
	if definition := ConformanceSuiteDefinitionForSelector(manifest, expectedSelectors[0]); !reflect.DeepEqual(definition, &expectedDefinition) {
		t.Fatalf("unexpected Markdown nested suite definition: %+v", definition)
	}
}

func TestSlice247MarkdownNestedNamedSuitePlans(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-247-markdown-nested-named-suite-plans", "markdown-nested-named-suite-plans.json"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(fixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	contextsRaw := fixture["contexts"].(map[string]any)
	contexts := make(map[string]ConformanceFamilyPlanContext, len(contextsRaw))
	for family, raw := range contextsRaw {
		contexts[family] = parseConformanceFamilyPlanContext(raw.(map[string]any))
	}
	expectedRaw := fixture["expected_entries"].([]any)
	expected := make([]NamedConformanceSuitePlan, 0, len(expectedRaw))
	for _, raw := range expectedRaw {
		expected = append(expected, parseNamedConformanceSuitePlan(t, raw.(map[string]any)))
	}
	if plans := PlanNamedConformanceSuites(manifest, contexts); !fixtureJSONEqual(t, plans, expected) {
		t.Fatalf("unexpected Markdown nested named suite plans: %+v", plans)
	}
}

func TestSlice248MarkdownNestedManifestReport(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-248-markdown-nested-manifest-report", "markdown-nested-manifest-report.json"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(fixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	options := parseConformanceManifestPlanningOptions(fixture["options"].(map[string]any))
	expected := parseConformanceManifestReport(fixture["expected_report"].(map[string]any))
	executionsRaw := fixture["executions"].(map[string]any)
	executions := make(map[string]ConformanceCaseExecution, len(executionsRaw))
	for key, raw := range executionsRaw {
		executions[key] = parseConformanceCaseExecution(raw.(map[string]any))
	}
	report := ReportConformanceManifest(manifest, options, func(run ConformanceCaseRun) ConformanceCaseExecution {
		key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
		if execution, ok := executions[key]; ok {
			return execution
		}
		return ConformanceCaseExecution{Outcome: ConformanceFailed, Messages: []string{"missing execution"}}
	})
	if !reflect.DeepEqual(report, expected) {
		t.Fatalf("unexpected Markdown nested manifest report: %+v", report)
	}
}

func TestSlice249RubyNestedSuiteDefinitions(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-249-ruby-nested-suite-definitions", "ruby-nested-suite-definitions.json"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(fixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	expectedSelectors := []ConformanceSuiteSelector{{Kind: "portable", Subject: ConformanceSuiteSubject{Grammar: "ruby", Variant: "nested"}}}
	if selectors := ConformanceSuiteSelectors(manifest); !reflect.DeepEqual(selectors, expectedSelectors) {
		t.Fatalf("unexpected Ruby nested suite selectors: %+v", selectors)
	}
	expectedDefinition := ConformanceSuiteDefinition{Kind: "portable", Subject: ConformanceSuiteSubject{Grammar: "ruby", Variant: "nested"}, Roles: []string{"analysis", "matching", "discovered_surfaces", "delegated_child_operations", "delegated_child_review_transport", "delegated_child_review_state", "delegated_child_apply_plan"}}
	if definition := ConformanceSuiteDefinitionForSelector(manifest, expectedSelectors[0]); !reflect.DeepEqual(definition, &expectedDefinition) {
		t.Fatalf("unexpected Ruby nested suite definition: %+v", definition)
	}
}

func TestSlice250RubyNestedNamedSuitePlans(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-250-ruby-nested-named-suite-plans", "ruby-nested-named-suite-plans.json"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(fixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	contextsRaw := fixture["contexts"].(map[string]any)
	contexts := make(map[string]ConformanceFamilyPlanContext, len(contextsRaw))
	for family, raw := range contextsRaw {
		contexts[family] = parseConformanceFamilyPlanContext(raw.(map[string]any))
	}
	expectedRaw := fixture["expected_entries"].([]any)
	expected := make([]NamedConformanceSuitePlan, 0, len(expectedRaw))
	for _, raw := range expectedRaw {
		expected = append(expected, parseNamedConformanceSuitePlan(t, raw.(map[string]any)))
	}
	if plans := PlanNamedConformanceSuites(manifest, contexts); !fixtureJSONEqual(t, plans, expected) {
		t.Fatalf("unexpected Ruby nested named suite plans: %+v", plans)
	}
}

func TestSlice251RubyNestedManifestReport(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-251-ruby-nested-manifest-report", "ruby-nested-manifest-report.json"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(fixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	options := parseConformanceManifestPlanningOptions(fixture["options"].(map[string]any))
	expected := parseConformanceManifestReport(fixture["expected_report"].(map[string]any))
	executionsRaw := fixture["executions"].(map[string]any)
	executions := make(map[string]ConformanceCaseExecution, len(executionsRaw))
	for key, raw := range executionsRaw {
		executions[key] = parseConformanceCaseExecution(raw.(map[string]any))
	}
	report := ReportConformanceManifest(manifest, options, func(run ConformanceCaseRun) ConformanceCaseExecution {
		key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
		if execution, ok := executions[key]; ok {
			return execution
		}
		return ConformanceCaseExecution{Outcome: ConformanceFailed, Messages: []string{"missing execution"}}
	})
	if !reflect.DeepEqual(report, expected) {
		t.Fatalf("unexpected Ruby nested manifest report: %+v", report)
	}
}

func TestSlice140TOMLFamilyManifestReport(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-140-toml-family-manifest-report", "go-toml-manifest-report.json"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(fixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	options := parseConformanceManifestPlanningOptions(fixture["options"].(map[string]any))
	expected := parseConformanceManifestReport(fixture["expected_report"].(map[string]any))
	executionsRaw := fixture["executions"].(map[string]any)
	executions := make(map[string]ConformanceCaseExecution, len(executionsRaw))
	for key, raw := range executionsRaw {
		executions[key] = parseConformanceCaseExecution(raw.(map[string]any))
	}

	report := ReportConformanceManifest(manifest, options, func(run ConformanceCaseRun) ConformanceCaseExecution {
		key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
		if execution, ok := executions[key]; ok {
			return execution
		}
		return ConformanceCaseExecution{Outcome: ConformanceFailed, Messages: []string{"missing execution"}}
	})

	if !reflect.DeepEqual(report, expected) {
		t.Fatalf("unexpected TOML manifest report: %+v", report)
	}
}

func TestSlice144YAMLFamilySuiteDefinitions(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-144-yaml-family-suite-definitions", "yaml-suite-definitions.json"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(fixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}

	expectedSelectors := []ConformanceSuiteSelector{{Kind: "portable", Subject: ConformanceSuiteSubject{Grammar: "yaml"}}}
	if selectors := ConformanceSuiteSelectors(manifest); !reflect.DeepEqual(selectors, expectedSelectors) {
		t.Fatalf("unexpected YAML suite selectors: %+v", selectors)
	}
	expectedDefinition := ConformanceSuiteDefinition{Kind: "portable", Subject: ConformanceSuiteSubject{Grammar: "yaml"}, Roles: []string{"analysis", "matching", "merge"}}
	if definition := ConformanceSuiteDefinitionForSelector(manifest, expectedSelectors[0]); !reflect.DeepEqual(definition, &expectedDefinition) {
		t.Fatalf("unexpected YAML suite definition: %+v", definition)
	}
}

func TestSlice145YAMLFamilyNamedSuitePlans(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-145-yaml-family-named-suite-plans", "go-yaml-named-suite-plans.json"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(fixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}

	contextsRaw := fixture["contexts"].(map[string]any)
	contexts := make(map[string]ConformanceFamilyPlanContext, len(contextsRaw))
	for family, raw := range contextsRaw {
		contexts[family] = parseConformanceFamilyPlanContext(raw.(map[string]any))
	}

	expectedRaw := fixture["expected_entries"].([]any)
	expected := make([]NamedConformanceSuitePlan, 0, len(expectedRaw))
	for _, raw := range expectedRaw {
		expected = append(expected, parseNamedConformanceSuitePlan(t, raw.(map[string]any)))
	}

	if plans := PlanNamedConformanceSuites(manifest, contexts); !fixtureJSONEqual(t, plans, expected) {
		t.Fatalf("unexpected YAML named suite plans: %+v", plans)
	}
}

func TestSlice146YAMLFamilyManifestReport(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-146-yaml-family-manifest-report", "go-yaml-manifest-report.json"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(fixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	options := parseConformanceManifestPlanningOptions(fixture["options"].(map[string]any))
	expected := parseConformanceManifestReport(fixture["expected_report"].(map[string]any))
	executionsRaw := fixture["executions"].(map[string]any)
	executions := make(map[string]ConformanceCaseExecution, len(executionsRaw))
	for key, raw := range executionsRaw {
		executions[key] = parseConformanceCaseExecution(raw.(map[string]any))
	}

	report := ReportConformanceManifest(manifest, options, func(run ConformanceCaseRun) ConformanceCaseExecution {
		key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
		if execution, ok := executions[key]; ok {
			return execution
		}
		return ConformanceCaseExecution{Outcome: ConformanceFailed, Messages: []string{"missing execution"}}
	})

	if !reflect.DeepEqual(report, expected) {
		t.Fatalf("unexpected YAML manifest report: %+v", report)
	}
}

func TestSlice173YAMLFamilyBackendNamedSuitePlans(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-173-yaml-family-backend-named-suite-plans", "go-yaml-backend-named-suite-plans.json"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(fixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}

	contextsRaw := fixture["contexts"].(map[string]any)
	contexts := make(map[string]ConformanceFamilyPlanContext, len(contextsRaw))
	for family, raw := range contextsRaw {
		contexts[family] = parseConformanceFamilyPlanContext(raw.(map[string]any))
	}

	expectedRaw := fixture["expected_entries"].([]any)
	expected := make([]NamedConformanceSuitePlan, 0, len(expectedRaw))
	for _, raw := range expectedRaw {
		expected = append(expected, parseNamedConformanceSuitePlan(t, raw.(map[string]any)))
	}

	if plans := PlanNamedConformanceSuites(manifest, contexts); !fixtureJSONEqual(t, plans, expected) {
		t.Fatalf("unexpected backend YAML named suite plans: %+v", plans)
	}
}

func TestSlice174YAMLFamilyBackendManifestReport(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-174-yaml-family-backend-manifest-report", "go-yaml-backend-manifest-report.json"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(fixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	options := parseConformanceManifestPlanningOptions(fixture["options"].(map[string]any))
	expected := parseConformanceManifestReport(fixture["expected_report"].(map[string]any))
	executionsRaw := fixture["executions"].(map[string]any)
	executions := make(map[string]ConformanceCaseExecution, len(executionsRaw))
	for key, raw := range executionsRaw {
		executions[key] = parseConformanceCaseExecution(raw.(map[string]any))
	}

	report := ReportConformanceManifest(manifest, options, func(run ConformanceCaseRun) ConformanceCaseExecution {
		key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
		if execution, ok := executions[key]; ok {
			return execution
		}
		return ConformanceCaseExecution{Outcome: ConformanceFailed, Messages: []string{"missing execution"}}
	})

	if !reflect.DeepEqual(report, expected) {
		t.Fatalf("unexpected backend YAML manifest report: %+v", report)
	}
}

func TestSlice185YAMLFamilyPolyglotBackendNamedSuitePlans(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-185-yaml-family-polyglot-backend-named-suite-plans", "go-yaml-polyglot-named-suite-plans.json"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(fixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}

	contextsRaw := fixture["contexts"].(map[string]any)
	contexts := make(map[string]ConformanceFamilyPlanContext, len(contextsRaw))
	for family, raw := range contextsRaw {
		contexts[family] = parseConformanceFamilyPlanContext(raw.(map[string]any))
	}

	expectedRaw := fixture["expected_entries"].([]any)
	expected := make([]NamedConformanceSuitePlan, 0, len(expectedRaw))
	for _, raw := range expectedRaw {
		expected = append(expected, parseNamedConformanceSuitePlan(t, raw.(map[string]any)))
	}

	if plans := PlanNamedConformanceSuites(manifest, contexts); !fixtureJSONEqual(t, plans, expected) {
		t.Fatalf("unexpected polyglot YAML named suite plans: %+v", plans)
	}
}

func TestSlice186YAMLFamilyPolyglotBackendManifestReport(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-186-yaml-family-polyglot-backend-manifest-report", "go-yaml-polyglot-manifest-report.json"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(fixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	options := parseConformanceManifestPlanningOptions(fixture["options"].(map[string]any))
	expected := parseConformanceManifestReport(fixture["expected_report"].(map[string]any))
	executionsRaw := fixture["executions"].(map[string]any)
	executions := make(map[string]ConformanceCaseExecution, len(executionsRaw))
	for key, raw := range executionsRaw {
		executions[key] = parseConformanceCaseExecution(raw.(map[string]any))
	}

	report := ReportConformanceManifest(manifest, options, func(run ConformanceCaseRun) ConformanceCaseExecution {
		key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
		if execution, ok := executions[key]; ok {
			return execution
		}
		return ConformanceCaseExecution{Outcome: ConformanceFailed, Messages: []string{"missing execution"}}
	})

	if !reflect.DeepEqual(report, expected) {
		t.Fatalf("unexpected polyglot YAML manifest report: %+v", report)
	}
}

func TestSlice148ConfigFamilyAggregateManifest(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-148-config-family-aggregate-manifest", "config-family-aggregate.json"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(fixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}

	expectedSelectors := []ConformanceSuiteSelector{
		{Kind: "portable", Subject: ConformanceSuiteSubject{Grammar: "json"}},
		{Kind: "portable", Subject: ConformanceSuiteSubject{Grammar: "text"}},
		{Kind: "portable", Subject: ConformanceSuiteSubject{Grammar: "toml"}},
		{Kind: "portable", Subject: ConformanceSuiteSubject{Grammar: "yaml"}},
	}
	if selectors := ConformanceSuiteSelectors(manifest); !reflect.DeepEqual(selectors, expectedSelectors) {
		t.Fatalf("unexpected aggregate suite selectors: %+v", selectors)
	}
}

func TestSlice149ConfigFamilyAggregateSuitePlans(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-149-config-family-aggregate-suite-plans", "config-family-aggregate-suite-plans.json"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(fixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}

	contextsRaw := fixture["contexts"].(map[string]any)
	contexts := make(map[string]ConformanceFamilyPlanContext, len(contextsRaw))
	for family, raw := range contextsRaw {
		contexts[family] = parseConformanceFamilyPlanContext(raw.(map[string]any))
	}

	expectedRaw := fixture["expected_entries"].([]any)
	expected := make([]NamedConformanceSuitePlan, 0, len(expectedRaw))
	for _, raw := range expectedRaw {
		expected = append(expected, parseNamedConformanceSuitePlan(t, raw.(map[string]any)))
	}

	if plans := PlanNamedConformanceSuites(manifest, contexts); !fixtureJSONEqual(t, plans, expected) {
		t.Fatalf("unexpected aggregate named suite plans: %+v", plans)
	}
}

func TestSlice150ConfigFamilyAggregateManifestReport(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-150-config-family-aggregate-manifest-report", "config-family-aggregate-manifest-report.json"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(fixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	options := parseConformanceManifestPlanningOptions(fixture["options"].(map[string]any))
	expected := parseConformanceManifestReport(fixture["expected_report"].(map[string]any))
	executionsRaw := fixture["executions"].(map[string]any)
	executions := make(map[string]ConformanceCaseExecution, len(executionsRaw))
	for key, raw := range executionsRaw {
		executions[key] = parseConformanceCaseExecution(raw.(map[string]any))
	}

	report := ReportConformanceManifest(manifest, options, func(run ConformanceCaseRun) ConformanceCaseExecution {
		key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
		if execution, ok := executions[key]; ok {
			return execution
		}
		return ConformanceCaseExecution{Outcome: ConformanceFailed, Messages: []string{"missing execution"}}
	})

	if !reflect.DeepEqual(report, expected) {
		t.Fatalf("unexpected aggregate manifest report: %+v", report)
	}
}

func TestAggregateConfigFamilyReviewStateFixtures(t *testing.T) {
	for _, fixtureName := range []string{
		filepath.Join("slice-151-config-family-aggregate-review-state", "config-family-aggregate-review-state.json"),
		filepath.Join("slice-152-config-family-aggregate-reviewed-default", "config-family-aggregate-reviewed-default.json"),
		filepath.Join("slice-153-config-family-aggregate-replay-application", "config-family-aggregate-replay-application.json"),
	} {
		fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", fixtureName))
		var manifest ConformanceManifest
		if raw, err := json.Marshal(fixture["manifest"]); err != nil {
			t.Fatalf("marshal manifest: %v", err)
		} else if err := json.Unmarshal(raw, &manifest); err != nil {
			t.Fatalf("unmarshal manifest: %v", err)
		}
		options := parseConformanceManifestReviewOptions(fixture["options"].(map[string]any))
		expected := parseConformanceManifestReviewState(fixture["expected_state"].(map[string]any))
		executionsRaw := fixture["executions"].(map[string]any)
		executions := make(map[string]ConformanceCaseExecution, len(executionsRaw))
		for key, raw := range executionsRaw {
			executions[key] = parseConformanceCaseExecution(raw.(map[string]any))
		}

		state := ReviewConformanceManifest(manifest, options, func(run ConformanceCaseRun) ConformanceCaseExecution {
			key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
			if execution, ok := executions[key]; ok {
				return execution
			}
			return ConformanceCaseExecution{Outcome: ConformanceFailed, Messages: []string{"missing execution"}}
		})

		if !reflect.DeepEqual(state, expected) {
			t.Fatalf("unexpected aggregate review state for %s: %+v", fixtureName, state)
		}
	}
}

func TestCanonicalStableSuitePlanningAndReviewFixtures(t *testing.T) {
	plansFixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-155-canonical-stable-suite-plans", "canonical-stable-suite-plans.json"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(plansFixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	contextsRaw := plansFixture["contexts"].(map[string]any)
	contexts := make(map[string]ConformanceFamilyPlanContext, len(contextsRaw))
	for family, raw := range contextsRaw {
		contexts[family] = parseConformanceFamilyPlanContext(raw.(map[string]any))
	}
	expectedPlansRaw := plansFixture["expected_entries"].([]any)
	expectedPlans := make([]NamedConformanceSuitePlan, 0, len(expectedPlansRaw))
	for _, raw := range expectedPlansRaw {
		expectedPlans = append(expectedPlans, parseNamedConformanceSuitePlan(t, raw.(map[string]any)))
	}
	if plans := PlanNamedConformanceSuites(manifest, contexts); !fixtureJSONEqual(t, plans, expectedPlans) {
		t.Fatalf("unexpected canonical stable suite plans: %+v", plans)
	}

	reportFixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-156-canonical-stable-suite-report", "canonical-stable-suite-report.json"))
	reportOptions := parseConformanceManifestPlanningOptions(reportFixture["options"].(map[string]any))
	expectedReport := parseConformanceManifestReport(reportFixture["expected_report"].(map[string]any))
	reportExecutionsRaw := reportFixture["executions"].(map[string]any)
	reportExecutions := make(map[string]ConformanceCaseExecution, len(reportExecutionsRaw))
	for key, raw := range reportExecutionsRaw {
		reportExecutions[key] = parseConformanceCaseExecution(raw.(map[string]any))
	}
	report := ReportConformanceManifest(manifest, reportOptions, func(run ConformanceCaseRun) ConformanceCaseExecution {
		key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
		if execution, ok := reportExecutions[key]; ok {
			return execution
		}
		return ConformanceCaseExecution{Outcome: ConformanceFailed, Messages: []string{"missing execution"}}
	})
	if !reflect.DeepEqual(report, expectedReport) {
		t.Fatalf("unexpected canonical stable suite report: %+v", report)
	}

	reviewFixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-157-canonical-stable-suite-review-state", "canonical-stable-suite-review-state.json"))
	reviewOptions := parseConformanceManifestReviewOptions(reviewFixture["options"].(map[string]any))
	expectedState := parseConformanceManifestReviewState(reviewFixture["expected_state"].(map[string]any))
	reviewExecutionsRaw := reviewFixture["executions"].(map[string]any)
	reviewExecutions := make(map[string]ConformanceCaseExecution, len(reviewExecutionsRaw))
	for key, raw := range reviewExecutionsRaw {
		reviewExecutions[key] = parseConformanceCaseExecution(raw.(map[string]any))
	}
	state := ReviewConformanceManifest(manifest, reviewOptions, func(run ConformanceCaseRun) ConformanceCaseExecution {
		key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
		if execution, ok := reviewExecutions[key]; ok {
			return execution
		}
		return ConformanceCaseExecution{Outcome: ConformanceFailed, Messages: []string{"missing execution"}}
	})
	if !reflect.DeepEqual(state, expectedState) {
		t.Fatalf("unexpected canonical stable suite review state: %+v", state)
	}
}

func TestCanonicalStableSuiteBackendFixtures(t *testing.T) {
	plansFixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-175-canonical-stable-suite-backend-plans", "go-canonical-stable-suite-backend-plans.json"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(plansFixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	contextsRaw := plansFixture["contexts"].(map[string]any)
	contexts := make(map[string]ConformanceFamilyPlanContext, len(contextsRaw))
	for family, raw := range contextsRaw {
		contexts[family] = parseConformanceFamilyPlanContext(raw.(map[string]any))
	}
	expectedPlansRaw := plansFixture["expected_entries"].([]any)
	expectedPlans := make([]NamedConformanceSuitePlan, 0, len(expectedPlansRaw))
	for _, raw := range expectedPlansRaw {
		expectedPlans = append(expectedPlans, parseNamedConformanceSuitePlan(t, raw.(map[string]any)))
	}
	if plans := PlanNamedConformanceSuites(manifest, contexts); !fixtureJSONEqual(t, plans, expectedPlans) {
		t.Fatalf("unexpected canonical stable suite backend plans: %+v", plans)
	}

	reportFixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-176-canonical-stable-suite-backend-report", "go-canonical-stable-suite-backend-report.json"))
	reportOptions := parseConformanceManifestPlanningOptions(reportFixture["options"].(map[string]any))
	expectedReport := parseConformanceManifestReport(reportFixture["expected_report"].(map[string]any))
	reportExecutionsRaw := reportFixture["executions"].(map[string]any)
	reportExecutions := make(map[string]ConformanceCaseExecution, len(reportExecutionsRaw))
	for key, raw := range reportExecutionsRaw {
		reportExecutions[key] = parseConformanceCaseExecution(raw.(map[string]any))
	}
	report := ReportConformanceManifest(manifest, reportOptions, func(run ConformanceCaseRun) ConformanceCaseExecution {
		key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
		if execution, ok := reportExecutions[key]; ok {
			return execution
		}
		return ConformanceCaseExecution{Outcome: ConformanceFailed, Messages: []string{"missing execution"}}
	})
	if !reflect.DeepEqual(report, expectedReport) {
		t.Fatalf("unexpected canonical stable suite backend report: %+v", report)
	}

	reviewFixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-177-canonical-stable-suite-backend-review-state", "go-canonical-stable-suite-backend-review-state.json"))
	reviewOptions := parseConformanceManifestReviewOptions(reviewFixture["options"].(map[string]any))
	expectedState := parseConformanceManifestReviewState(reviewFixture["expected_state"].(map[string]any))
	reviewExecutionsRaw := reviewFixture["executions"].(map[string]any)
	reviewExecutions := make(map[string]ConformanceCaseExecution, len(reviewExecutionsRaw))
	for key, raw := range reviewExecutionsRaw {
		reviewExecutions[key] = parseConformanceCaseExecution(raw.(map[string]any))
	}
	state := ReviewConformanceManifest(manifest, reviewOptions, func(run ConformanceCaseRun) ConformanceCaseExecution {
		key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
		if execution, ok := reviewExecutions[key]; ok {
			return execution
		}
		return ConformanceCaseExecution{Outcome: ConformanceFailed, Messages: []string{"missing execution"}}
	})
	if !reflect.DeepEqual(state, expectedState) {
		t.Fatalf("unexpected canonical stable suite backend review state: %+v", state)
	}
}

func TestCanonicalWidenedSuiteBackendFixtures(t *testing.T) {
	for _, fixtureSet := range []struct {
		plansPath   string
		reportPath  string
		reviewPaths []string
	}{
		{
			plansPath:  filepath.Join("..", "..", "fixtures", "diagnostics", "slice-178-canonical-widened-suite-backend-plans", "go-canonical-widened-suite-backend-plans.json"),
			reportPath: filepath.Join("..", "..", "fixtures", "diagnostics", "slice-179-canonical-widened-suite-backend-report", "go-canonical-widened-suite-backend-report.json"),
			reviewPaths: []string{
				filepath.Join("..", "..", "fixtures", "diagnostics", "slice-180-canonical-widened-suite-backend-review-state", "go-canonical-widened-suite-backend-review-state.json"),
				filepath.Join("..", "..", "fixtures", "diagnostics", "slice-181-canonical-widened-suite-backend-reviewed-default", "go-canonical-widened-suite-backend-reviewed-default.json"),
				filepath.Join("..", "..", "fixtures", "diagnostics", "slice-182-canonical-widened-suite-backend-replay-application", "go-canonical-widened-suite-backend-replay-application.json"),
			},
		},
		{
			plansPath:  filepath.Join("..", "..", "fixtures", "diagnostics", "slice-187-canonical-widened-suite-polyglot-backend-plans", "go-canonical-widened-suite-polyglot-backend-plans.json"),
			reportPath: filepath.Join("..", "..", "fixtures", "diagnostics", "slice-188-canonical-widened-suite-polyglot-backend-report", "go-canonical-widened-suite-polyglot-backend-report.json"),
			reviewPaths: []string{
				filepath.Join("..", "..", "fixtures", "diagnostics", "slice-189-canonical-widened-suite-polyglot-backend-review-state", "go-canonical-widened-suite-polyglot-backend-review-state.json"),
				filepath.Join("..", "..", "fixtures", "diagnostics", "slice-190-canonical-widened-suite-polyglot-backend-reviewed-default", "go-canonical-widened-suite-polyglot-backend-reviewed-default.json"),
				filepath.Join("..", "..", "fixtures", "diagnostics", "slice-191-canonical-widened-suite-polyglot-backend-replay-application", "go-canonical-widened-suite-polyglot-backend-replay-application.json"),
			},
		},
	} {
		plansFixture := readDiagnosticFixtureFromPath(t, fixtureSet.plansPath)
		var manifest ConformanceManifest
		if raw, err := json.Marshal(plansFixture["manifest"]); err != nil {
			t.Fatalf("marshal manifest: %v", err)
		} else if err := json.Unmarshal(raw, &manifest); err != nil {
			t.Fatalf("unmarshal manifest: %v", err)
		}
		contextsRaw := plansFixture["contexts"].(map[string]any)
		contexts := make(map[string]ConformanceFamilyPlanContext, len(contextsRaw))
		for family, raw := range contextsRaw {
			contexts[family] = parseConformanceFamilyPlanContext(raw.(map[string]any))
		}
		expectedPlansRaw := plansFixture["expected_entries"].([]any)
		expectedPlans := make([]NamedConformanceSuitePlan, 0, len(expectedPlansRaw))
		for _, raw := range expectedPlansRaw {
			expectedPlans = append(expectedPlans, parseNamedConformanceSuitePlan(t, raw.(map[string]any)))
		}
		if plans := PlanNamedConformanceSuites(manifest, contexts); !fixtureJSONEqual(t, plans, expectedPlans) {
			t.Fatalf("unexpected canonical widened suite backend plans: %+v", plans)
		}

		reportFixture := readDiagnosticFixtureFromPath(t, fixtureSet.reportPath)
		reportOptions := parseConformanceManifestPlanningOptions(reportFixture["options"].(map[string]any))
		expectedReport := parseConformanceManifestReport(reportFixture["expected_report"].(map[string]any))
		reportExecutionsRaw := reportFixture["executions"].(map[string]any)
		reportExecutions := make(map[string]ConformanceCaseExecution, len(reportExecutionsRaw))
		for key, raw := range reportExecutionsRaw {
			reportExecutions[key] = parseConformanceCaseExecution(raw.(map[string]any))
		}
		report := ReportConformanceManifest(manifest, reportOptions, func(run ConformanceCaseRun) ConformanceCaseExecution {
			key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
			if execution, ok := reportExecutions[key]; ok {
				return execution
			}
			return ConformanceCaseExecution{Outcome: ConformanceFailed, Messages: []string{"missing execution"}}
		})
		if !reflect.DeepEqual(report, expectedReport) {
			t.Fatalf("unexpected canonical widened suite backend report: %+v", report)
		}

		for _, fixturePath := range fixtureSet.reviewPaths {
			fixture := readDiagnosticFixtureFromPath(t, fixturePath)
			reviewOptions := parseConformanceManifestReviewOptions(fixture["options"].(map[string]any))
			expectedState := parseConformanceManifestReviewState(fixture["expected_state"].(map[string]any))
			reviewExecutionsRaw := fixture["executions"].(map[string]any)
			reviewExecutions := make(map[string]ConformanceCaseExecution, len(reviewExecutionsRaw))
			for key, raw := range reviewExecutionsRaw {
				reviewExecutions[key] = parseConformanceCaseExecution(raw.(map[string]any))
			}
			state := ReviewConformanceManifest(manifest, reviewOptions, func(run ConformanceCaseRun) ConformanceCaseExecution {
				key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
				if execution, ok := reviewExecutions[key]; ok {
					return execution
				}
				return ConformanceCaseExecution{Outcome: ConformanceFailed, Messages: []string{"missing execution"}}
			})
			if !reflect.DeepEqual(state, expectedState) {
				t.Fatalf("unexpected canonical widened suite backend review state for %s: %+v", fixturePath, state)
			}
		}
	}
}

func TestSharedFixtureNamedConformanceSuiteResults(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "named_suite_results"))
	manifest := readManifest(t)
	selector := parseConformanceSuiteSelector(fixture["suite_selector"].(map[string]any))
	executionsRaw := fixture["executions"].(map[string]any)

	entry := RunNamedConformanceSuiteEntry(
		manifest,
		selector,
		parseFamilyFeatureProfile(fixture["family_profile"].(map[string]any)),
		func(run ConformanceCaseRun) ConformanceCaseExecution {
			key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
			if raw, ok := executionsRaw[key]; ok {
				return parseConformanceCaseExecution(raw.(map[string]any))
			}

			return ConformanceCaseExecution{
				Outcome:  ConformanceFailed,
				Messages: []string{"missing execution"},
			}
		},
		&ConformanceFeatureProfileView{
			Backend:           "kreuzberg-language-pack",
			SupportsDialects:  false,
			SupportedPolicies: []PolicyReference{{Surface: PolicySurfaceArray, Name: "destination_wins_array"}},
		},
	)

	if entry == nil {
		t.Fatalf("expected named suite results entry")
	}

	expected := parseNamedConformanceSuiteResults(fixture["expected_entry"].(map[string]any))
	if !fixtureJSONEqual(t, *entry, expected) {
		t.Fatalf("unexpected named suite results entry: %+v", entry)
	}
}

func TestSharedFixturePlannedNamedConformanceSuiteRunner(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "named_suite_runner_entries"))
	manifest := readManifest(t)
	contextsRaw := fixture["contexts"].(map[string]any)
	contexts := make(map[string]ConformanceFamilyPlanContext, len(contextsRaw))
	for family, raw := range contextsRaw {
		contexts[family] = parseConformanceFamilyPlanContext(raw.(map[string]any))
	}
	executionsRaw := fixture["executions"].(map[string]any)

	expectedRaw := fixture["expected_entries"].([]any)
	expected := make([]NamedConformanceSuiteResults, 0, len(expectedRaw))
	for _, raw := range expectedRaw {
		expected = append(expected, parseNamedConformanceSuiteResults(raw.(map[string]any)))
	}

	entries := RunPlannedNamedConformanceSuites(
		PlanNamedConformanceSuites(manifest, contexts),
		func(run ConformanceCaseRun) ConformanceCaseExecution {
			key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
			if raw, ok := executionsRaw[key]; ok {
				return parseConformanceCaseExecution(raw.(map[string]any))
			}

			return ConformanceCaseExecution{
				Outcome:  ConformanceFailed,
				Messages: []string{"missing execution"},
			}
		},
	)

	if !reflect.DeepEqual(entries, expected) {
		t.Fatalf("unexpected planned named suite runner entries: %+v", entries)
	}
}

func TestSlice128SourceFamilyManifestReport(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-128-source-family-manifest-report", "source-manifest-report.json"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(fixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}

	options := parseConformanceManifestPlanningOptions(fixture["options"].(map[string]any))
	expected := parseConformanceManifestReport(fixture["expected_report"].(map[string]any))
	executionsRaw := fixture["executions"].(map[string]any)

	report := ReportConformanceManifest(
		manifest,
		options,
		func(run ConformanceCaseRun) ConformanceCaseExecution {
			key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
			if raw, ok := executionsRaw[key]; ok {
				return parseConformanceCaseExecution(raw.(map[string]any))
			}
			return ConformanceCaseExecution{
				Outcome:  ConformanceFailed,
				Messages: []string{"missing execution"},
			}
		},
	)

	if !reflect.DeepEqual(report, expected) {
		t.Fatalf("unexpected source manifest report: %+v", report)
	}
}

func TestSlice129SourceFamilyBackendRestrictedPlans(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-129-source-family-backend-restricted-plans", "source-backend-restricted-plans.json"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(fixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}

	contextsRaw := fixture["contexts"].(map[string]any)
	contexts := make(map[string]ConformanceFamilyPlanContext, len(contextsRaw))
	for family, raw := range contextsRaw {
		contexts[family] = parseConformanceFamilyPlanContext(raw.(map[string]any))
	}

	expectedRaw := fixture["expected_entries"].([]any)
	expected := make([]NamedConformanceSuitePlan, 0, len(expectedRaw))
	for _, raw := range expectedRaw {
		expected = append(expected, parseNamedConformanceSuitePlan(t, raw.(map[string]any)))
	}

	if plans := PlanNamedConformanceSuites(manifest, contexts); !fixtureJSONEqual(t, plans, expected) {
		t.Fatalf("unexpected backend-restricted source plans: %+v", plans)
	}
}

func TestSlice130SourceFamilyBackendRestrictedReport(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-130-source-family-backend-restricted-report", "source-backend-restricted-report.json"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(fixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}

	options := parseConformanceManifestPlanningOptions(fixture["options"].(map[string]any))
	expected := parseConformanceManifestReport(fixture["expected_report"].(map[string]any))
	executionsRaw := fixture["executions"].(map[string]any)

	report := ReportConformanceManifest(
		manifest,
		options,
		func(run ConformanceCaseRun) ConformanceCaseExecution {
			key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
			if raw, ok := executionsRaw[key]; ok {
				return parseConformanceCaseExecution(raw.(map[string]any))
			}
			return ConformanceCaseExecution{
				Outcome:  ConformanceFailed,
				Messages: []string{"missing execution"},
			}
		},
	)

	if !reflect.DeepEqual(report, expected) {
		t.Fatalf("unexpected backend-restricted source report: %+v", report)
	}
}

func TestSlice131CanonicalManifestSourceFamilyPaths(t *testing.T) {
	manifest := readManifest(t)

	if path := ConformanceFamilyFeatureProfilePath(manifest, "typescript"); path == nil || filepath.Join(path...) != filepath.Join("diagnostics", "slice-101-typescript-family-feature-profile", "typescript-feature-profile.json") {
		t.Fatalf("unexpected canonical typescript family profile path: %+v", path)
	}
	if path := ConformanceFamilyFeatureProfilePath(manifest, "rust"); path == nil || filepath.Join(path...) != filepath.Join("diagnostics", "slice-105-rust-family-feature-profile", "rust-feature-profile.json") {
		t.Fatalf("unexpected canonical rust family profile path: %+v", path)
	}
	if path := ConformanceFamilyFeatureProfilePath(manifest, "go"); path == nil || filepath.Join(path...) != filepath.Join("diagnostics", "slice-109-go-family-feature-profile", "go-feature-profile.json") {
		t.Fatalf("unexpected canonical go family profile path: %+v", path)
	}
	if path := ConformanceFixturePath(manifest, "typescript", "analysis"); path == nil || filepath.Join(path...) != filepath.Join("typescript", "slice-102-analysis", "module-owners.json") {
		t.Fatalf("unexpected canonical typescript analysis path: %+v", path)
	}
	if path := ConformanceFixturePath(manifest, "rust", "matching"); path == nil || filepath.Join(path...) != filepath.Join("rust", "slice-107-matching", "path-equality.json") {
		t.Fatalf("unexpected canonical rust matching path: %+v", path)
	}
	if path := ConformanceFixturePath(manifest, "go", "merge"); path == nil || filepath.Join(path...) != filepath.Join("go", "slice-112-merge", "module-merge.json") {
		t.Fatalf("unexpected canonical go merge path: %+v", path)
	}
}

func TestSourceFamilyReviewStateFixtures(t *testing.T) {
	for _, relative := range []string{
		filepath.Join("..", "..", "fixtures", "diagnostics", "slice-158-source-family-review-state", "source-family-review-state.json"),
		filepath.Join("..", "..", "fixtures", "diagnostics", "slice-159-source-family-reviewed-default", "source-family-reviewed-default.json"),
		filepath.Join("..", "..", "fixtures", "diagnostics", "slice-160-source-family-replay-application", "source-family-replay-application.json"),
	} {
		fixture := readDiagnosticFixtureFromPath(t, relative)
		var manifest ConformanceManifest
		if raw, err := json.Marshal(fixture["manifest"]); err != nil {
			t.Fatalf("marshal manifest: %v", err)
		} else if err := json.Unmarshal(raw, &manifest); err != nil {
			t.Fatalf("unmarshal manifest: %v", err)
		}

		options := parseConformanceManifestReviewOptions(fixture["options"].(map[string]any))
		expected := parseConformanceManifestReviewState(fixture["expected_state"].(map[string]any))
		executionsRaw := fixture["executions"].(map[string]any)

		state := ReviewConformanceManifest(
			manifest,
			options,
			func(run ConformanceCaseRun) ConformanceCaseExecution {
				key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
				if raw, ok := executionsRaw[key]; ok {
					return parseConformanceCaseExecution(raw.(map[string]any))
				}
				return ConformanceCaseExecution{
					Outcome:  ConformanceFailed,
					Messages: []string{"missing execution"},
				}
			},
		)

		if !reflect.DeepEqual(state, expected) {
			t.Fatalf("unexpected source-family review state: %+v", state)
		}
	}
}

func TestCanonicalWidenedSuiteFixtures(t *testing.T) {
	plansFixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-162-canonical-widened-suite-plans", "canonical-widened-suite-plans.json"))
	var plansManifest ConformanceManifest
	if raw, err := json.Marshal(plansFixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &plansManifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	planContextsRaw := plansFixture["contexts"].(map[string]any)
	planContexts := make(map[string]ConformanceFamilyPlanContext, len(planContextsRaw))
	for family, raw := range planContextsRaw {
		planContexts[family] = parseConformanceFamilyPlanContext(raw.(map[string]any))
	}
	expectedEntriesRaw := plansFixture["expected_entries"].([]any)
	expectedEntries := make([]NamedConformanceSuitePlan, 0, len(expectedEntriesRaw))
	for _, raw := range expectedEntriesRaw {
		expectedEntries = append(expectedEntries, parseNamedConformanceSuitePlan(t, raw.(map[string]any)))
	}
	if plans := PlanNamedConformanceSuites(plansManifest, planContexts); !fixtureJSONEqual(t, plans, expectedEntries) {
		t.Fatalf("unexpected canonical widened suite plans: %+v", plans)
	}

	reportFixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-163-canonical-widened-suite-report", "canonical-widened-suite-report.json"))
	var reportManifest ConformanceManifest
	if raw, err := json.Marshal(reportFixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &reportManifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	reportOptions := parseConformanceManifestPlanningOptions(reportFixture["options"].(map[string]any))
	expectedReport := parseConformanceManifestReport(reportFixture["expected_report"].(map[string]any))
	reportExecutionsRaw := reportFixture["executions"].(map[string]any)
	report := ReportConformanceManifest(
		reportManifest,
		reportOptions,
		func(run ConformanceCaseRun) ConformanceCaseExecution {
			key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
			if raw, ok := reportExecutionsRaw[key]; ok {
				return parseConformanceCaseExecution(raw.(map[string]any))
			}
			return ConformanceCaseExecution{Outcome: ConformanceFailed, Messages: []string{"missing execution"}}
		},
	)
	if !reflect.DeepEqual(report, expectedReport) {
		t.Fatalf("unexpected canonical widened suite report: %+v", report)
	}

	for _, relative := range []string{
		filepath.Join("..", "..", "fixtures", "diagnostics", "slice-164-canonical-widened-suite-review-state", "canonical-widened-suite-review-state.json"),
		filepath.Join("..", "..", "fixtures", "diagnostics", "slice-165-canonical-widened-suite-reviewed-default", "canonical-widened-suite-reviewed-default.json"),
		filepath.Join("..", "..", "fixtures", "diagnostics", "slice-166-canonical-widened-suite-replay-application", "canonical-widened-suite-replay-application.json"),
	} {
		fixture := readDiagnosticFixtureFromPath(t, relative)
		var manifest ConformanceManifest
		if raw, err := json.Marshal(fixture["manifest"]); err != nil {
			t.Fatalf("marshal manifest: %v", err)
		} else if err := json.Unmarshal(raw, &manifest); err != nil {
			t.Fatalf("unmarshal manifest: %v", err)
		}
		options := parseConformanceManifestReviewOptions(fixture["options"].(map[string]any))
		expected := parseConformanceManifestReviewState(fixture["expected_state"].(map[string]any))
		executionsRaw := fixture["executions"].(map[string]any)
		state := ReviewConformanceManifest(
			manifest,
			options,
			func(run ConformanceCaseRun) ConformanceCaseExecution {
				key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
				if raw, ok := executionsRaw[key]; ok {
					return parseConformanceCaseExecution(raw.(map[string]any))
				}
				return ConformanceCaseExecution{Outcome: ConformanceFailed, Messages: []string{"missing execution"}}
			},
		)
		if !reflect.DeepEqual(state, expected) {
			t.Fatalf("unexpected canonical widened suite review state: %+v", state)
		}
	}
}

func TestBackendSensitiveAggregateFixtures(t *testing.T) {
	plansFixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-167-backend-sensitive-aggregate-suite-plans", "backend-sensitive-aggregate-suite-plans.json"))
	var plansManifest ConformanceManifest
	if raw, err := json.Marshal(plansFixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &plansManifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	planContextsRaw := plansFixture["contexts"].(map[string]any)
	planContexts := make(map[string]ConformanceFamilyPlanContext, len(planContextsRaw))
	for family, raw := range planContextsRaw {
		planContexts[family] = parseConformanceFamilyPlanContext(raw.(map[string]any))
	}
	expectedEntriesRaw := plansFixture["expected_entries"].([]any)
	expectedEntries := make([]NamedConformanceSuitePlan, 0, len(expectedEntriesRaw))
	for _, raw := range expectedEntriesRaw {
		expectedEntries = append(expectedEntries, parseNamedConformanceSuitePlan(t, raw.(map[string]any)))
	}
	if plans := PlanNamedConformanceSuites(plansManifest, planContexts); !fixtureJSONEqual(t, plans, expectedEntries) {
		t.Fatalf("unexpected backend-sensitive aggregate plans: %+v", plans)
	}

	for _, relative := range []string{
		filepath.Join("..", "..", "fixtures", "diagnostics", "slice-168-backend-sensitive-aggregate-tree-sitter-report", "backend-sensitive-aggregate-tree-sitter-report.json"),
		filepath.Join("..", "..", "fixtures", "diagnostics", "slice-169-backend-sensitive-aggregate-native-report", "backend-sensitive-aggregate-native-report.json"),
	} {
		fixture := readDiagnosticFixtureFromPath(t, relative)
		var manifest ConformanceManifest
		if raw, err := json.Marshal(fixture["manifest"]); err != nil {
			t.Fatalf("marshal manifest: %v", err)
		} else if err := json.Unmarshal(raw, &manifest); err != nil {
			t.Fatalf("unmarshal manifest: %v", err)
		}
		options := parseConformanceManifestPlanningOptions(fixture["options"].(map[string]any))
		expected := parseConformanceManifestReport(fixture["expected_report"].(map[string]any))
		executionsRaw := fixture["executions"].(map[string]any)
		report := ReportConformanceManifest(
			manifest,
			options,
			func(run ConformanceCaseRun) ConformanceCaseExecution {
				key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
				if raw, ok := executionsRaw[key]; ok {
					return parseConformanceCaseExecution(raw.(map[string]any))
				}
				return ConformanceCaseExecution{Outcome: ConformanceFailed, Messages: []string{"missing execution"}}
			},
		)
		if !reflect.DeepEqual(report, expected) {
			t.Fatalf("unexpected backend-sensitive aggregate report: %+v", report)
		}
	}

	for _, relative := range []string{
		filepath.Join("..", "..", "fixtures", "diagnostics", "slice-192-backend-sensitive-aggregate-tree-sitter-review-state", "backend-sensitive-aggregate-tree-sitter-review-state.json"),
		filepath.Join("..", "..", "fixtures", "diagnostics", "slice-193-backend-sensitive-aggregate-native-review-state", "backend-sensitive-aggregate-native-review-state.json"),
	} {
		fixture := readDiagnosticFixtureFromPath(t, relative)
		var manifest ConformanceManifest
		if raw, err := json.Marshal(fixture["manifest"]); err != nil {
			t.Fatalf("marshal manifest: %v", err)
		} else if err := json.Unmarshal(raw, &manifest); err != nil {
			t.Fatalf("unmarshal manifest: %v", err)
		}
		options := parseConformanceManifestReviewOptions(fixture["options"].(map[string]any))
		expected := parseConformanceManifestReviewState(fixture["expected_state"].(map[string]any))
		executionsRaw := fixture["executions"].(map[string]any)
		state := ReviewConformanceManifest(
			manifest,
			options,
			func(run ConformanceCaseRun) ConformanceCaseExecution {
				key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
				if raw, ok := executionsRaw[key]; ok {
					return parseConformanceCaseExecution(raw.(map[string]any))
				}
				return ConformanceCaseExecution{Outcome: ConformanceFailed, Messages: []string{"missing execution"}}
			},
		)
		if !reflect.DeepEqual(state, expected) {
			t.Fatalf("unexpected backend-sensitive aggregate review state: %+v", state)
		}
	}
}

func TestSharedFixturePlannedNamedConformanceSuiteReports(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "named_suite_report_entries"))
	manifest := readManifest(t)
	contextsRaw := fixture["contexts"].(map[string]any)
	contexts := make(map[string]ConformanceFamilyPlanContext, len(contextsRaw))
	for family, raw := range contextsRaw {
		contexts[family] = parseConformanceFamilyPlanContext(raw.(map[string]any))
	}
	executionsRaw := fixture["executions"].(map[string]any)

	expectedRaw := fixture["expected_entries"].([]any)
	expected := make([]NamedConformanceSuiteReport, 0, len(expectedRaw))
	for _, raw := range expectedRaw {
		expected = append(expected, parseNamedConformanceSuiteReport(raw.(map[string]any)))
	}

	entries := ReportPlannedNamedConformanceSuites(
		PlanNamedConformanceSuites(manifest, contexts),
		func(run ConformanceCaseRun) ConformanceCaseExecution {
			key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
			if raw, ok := executionsRaw[key]; ok {
				return parseConformanceCaseExecution(raw.(map[string]any))
			}

			return ConformanceCaseExecution{
				Outcome:  ConformanceFailed,
				Messages: []string{"missing execution"},
			}
		},
	)

	if !reflect.DeepEqual(entries, expected) {
		t.Fatalf("unexpected planned named suite report entries: %+v", entries)
	}
}

func TestSharedFixtureNamedConformanceSuiteSummary(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "named_suite_summary"))
	entriesRaw := fixture["entries"].([]any)
	entries := make([]NamedConformanceSuiteReport, 0, len(entriesRaw))
	for _, raw := range entriesRaw {
		entries = append(entries, parseNamedConformanceSuiteReport(raw.(map[string]any)))
	}

	expectedRaw := fixture["expected_summary"].(map[string]any)
	expected := ConformanceSuiteSummary{
		Total:   int(expectedRaw["total"].(float64)),
		Passed:  int(expectedRaw["passed"].(float64)),
		Failed:  int(expectedRaw["failed"].(float64)),
		Skipped: int(expectedRaw["skipped"].(float64)),
	}

	if summary := SummarizeNamedConformanceSuiteReports(entries); !reflect.DeepEqual(summary, expected) {
		t.Fatalf("unexpected named suite summary: %+v", summary)
	}
}

func TestSharedFixtureNamedConformanceSuiteReportEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "named_suite_report_envelope"))
	entriesRaw := fixture["entries"].([]any)
	entries := make([]NamedConformanceSuiteReport, 0, len(entriesRaw))
	for _, raw := range entriesRaw {
		entries = append(entries, parseNamedConformanceSuiteReport(raw.(map[string]any)))
	}

	expected := parseNamedConformanceSuiteReportEnvelope(fixture["expected_report"].(map[string]any))
	if report := ReportNamedConformanceSuiteEnvelope(entries); !reflect.DeepEqual(report, expected) {
		t.Fatalf("unexpected named suite report envelope: %+v", report)
	}
}

func TestSharedFixtureNamedConformanceSuiteReportManifest(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "named_suite_report_manifest"))
	manifest := readManifest(t)
	contextsRaw := fixture["contexts"].(map[string]any)
	contexts := make(map[string]ConformanceFamilyPlanContext, len(contextsRaw))
	for family, raw := range contextsRaw {
		contexts[family] = parseConformanceFamilyPlanContext(raw.(map[string]any))
	}
	executionsRaw := fixture["executions"].(map[string]any)
	expected := parseNamedConformanceSuiteReportEnvelope(fixture["expected_report"].(map[string]any))

	report := ReportNamedConformanceSuiteManifest(
		manifest,
		contexts,
		func(run ConformanceCaseRun) ConformanceCaseExecution {
			key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
			if raw, ok := executionsRaw[key]; ok {
				return parseConformanceCaseExecution(raw.(map[string]any))
			}

			return ConformanceCaseExecution{
				Outcome:  ConformanceFailed,
				Messages: []string{"missing execution"},
			}
		},
	)

	if !reflect.DeepEqual(report, expected) {
		t.Fatalf("unexpected named suite report manifest: %+v", report)
	}
}

func TestSharedFixtureDefaultFamilyContext(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "default_family_context"))
	family := fixture["family"].(string)
	familyProfile := parseFamilyFeatureProfile(fixture["family_profile"].(map[string]any))
	expectedContext := parseConformanceFamilyPlanContext(fixture["expected_context"].(map[string]any))
	expectedDiagnostic := parseDiagnostic(fixture["expected_diagnostic"].(map[string]any))

	if context := DefaultConformanceFamilyContext(familyProfile); !reflect.DeepEqual(context, expectedContext) {
		t.Fatalf("unexpected default family context: %+v", context)
	}

	context, diagnostics := ResolveConformanceFamilyContext(family, ConformanceManifestPlanningOptions{
		FamilyProfiles: map[string]FamilyFeatureProfile{
			family: familyProfile,
		},
	})
	if context == nil || !reflect.DeepEqual(*context, expectedContext) {
		t.Fatalf("unexpected resolved family context: %+v", context)
	}
	if !reflect.DeepEqual(diagnostics, []Diagnostic{expectedDiagnostic}) {
		t.Fatalf("unexpected default family diagnostics: %+v", diagnostics)
	}
}

func TestSharedFixtureExplicitFamilyContextMode(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "explicit_family_context_mode"))
	options := parseConformanceManifestPlanningOptions(fixture["options"].(map[string]any))
	expectedDiagnostic := parseDiagnostic(fixture["expected_diagnostic"].(map[string]any))

	_, diagnostics := ResolveConformanceFamilyContext("text", options)
	if !reflect.DeepEqual(diagnostics, []Diagnostic{expectedDiagnostic}) {
		t.Fatalf("unexpected explicit family diagnostics: %+v", diagnostics)
	}
}

func TestSharedFixtureMissingSuiteRoles(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "missing_suite_roles"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(fixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	options := parseConformanceManifestPlanningOptions(fixture["options"].(map[string]any))
	expectedDiagnostic := parseDiagnostic(fixture["expected_diagnostic"].(map[string]any))

	planned := PlanNamedConformanceSuitesWithDiagnostics(manifest, options)
	if len(planned.Entries) != 0 {
		t.Fatalf("expected no planned entries: %+v", planned.Entries)
	}
	if !slices.Contains(planned.Diagnostics, expectedDiagnostic) {
		t.Fatalf("missing expected missing-role diagnostic: %+v", planned.Diagnostics)
	}
}

func TestSharedFixtureConformanceManifestReport(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "conformance_manifest_report"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(fixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	options := parseConformanceManifestPlanningOptions(fixture["options"].(map[string]any))
	executionsRaw := fixture["executions"].(map[string]any)
	expected := parseConformanceManifestReport(fixture["expected_report"].(map[string]any))

	report := ReportConformanceManifest(
		manifest,
		options,
		func(run ConformanceCaseRun) ConformanceCaseExecution {
			key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
			if raw, ok := executionsRaw[key]; ok {
				return parseConformanceCaseExecution(raw.(map[string]any))
			}

			return ConformanceCaseExecution{
				Outcome:  ConformanceFailed,
				Messages: []string{"missing execution"},
			}
		},
	)

	if !reflect.DeepEqual(report, expected) {
		t.Fatalf("unexpected conformance manifest report: %+v", report)
	}
}

func TestSharedFixtureReviewHostHints(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "review_host_hints"))
	options := parseConformanceManifestReviewOptions(fixture["options"].(map[string]any))
	expected := parseReviewHostHints(fixture["expected_hints"].(map[string]any))

	if hints := ConformanceReviewHostHints(options); !reflect.DeepEqual(hints, expected) {
		t.Fatalf("unexpected review host hints: %+v", hints)
	}
}

func TestSharedFixtureFamilyContextReviewRequest(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "family_context_review_request"))
	family := fixture["family"].(string)
	options := parseConformanceManifestReviewOptions(fixture["options"].(map[string]any))
	expectedDiagnostic := parseDiagnostic(fixture["expected_diagnostic"].(map[string]any))
	expectedRequest := parseReviewRequest(fixture["expected_request"].(map[string]any))

	_, diagnostics, requests, _ := ReviewConformanceFamilyContext(family, options)
	if ReviewRequestIDForFamilyContext(family) != expectedRequest.ID {
		t.Fatalf("unexpected request id: %s", ReviewRequestIDForFamilyContext(family))
	}
	if !reflect.DeepEqual(diagnostics, []Diagnostic{expectedDiagnostic}) {
		t.Fatalf("unexpected family-context diagnostics: %+v", diagnostics)
	}
	if !reflect.DeepEqual(requests, []ReviewRequest{expectedRequest}) {
		t.Fatalf("unexpected family-context requests: %+v", requests)
	}
}

func TestSharedFixtureFamilyContextReviewProposal(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "family_context_review_proposal"))
	family := fixture["family"].(string)
	options := parseConformanceManifestReviewOptions(fixture["options"].(map[string]any))
	expectedRequest := parseReviewRequest(fixture["expected_request"].(map[string]any))

	_, _, requests, _ := ReviewConformanceFamilyContext(family, options)
	if !reflect.DeepEqual(requests, []ReviewRequest{expectedRequest}) {
		t.Fatalf("unexpected family-context proposal requests: %+v", requests)
	}
}

func TestSharedFixtureFamilyContextExplicitReviewDecision(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "family_context_explicit_review_decision"))
	family := fixture["family"].(string)
	options := parseConformanceManifestReviewOptions(fixture["options"].(map[string]any))
	expectedContext := parseConformanceFamilyPlanContext(fixture["expected_context"].(map[string]any))
	expectedApplied := make([]ReviewDecision, 0, len(fixture["expected_applied_decisions"].([]any)))
	for _, item := range fixture["expected_applied_decisions"].([]any) {
		expectedApplied = append(expectedApplied, parseReviewDecision(item.(map[string]any)))
	}

	context, diagnostics, requests, applied := ReviewConformanceFamilyContext(family, options)
	if context == nil || !reflect.DeepEqual(*context, expectedContext) {
		t.Fatalf("unexpected explicit review context: %+v", context)
	}
	if len(diagnostics) != 0 || len(requests) != 0 {
		t.Fatalf("unexpected explicit review diagnostics/requests: %+v %+v", diagnostics, requests)
	}
	if !reflect.DeepEqual(applied, expectedApplied) {
		t.Fatalf("unexpected explicit review applied decisions: %+v", applied)
	}
}

func TestSharedFixtureExplicitReviewDecisionPayloadValidation(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "explicit_review_decision_missing_context"))
	family := fixture["family"].(string)
	options := parseConformanceManifestReviewOptions(fixture["options"].(map[string]any))
	expectedDiagnostic := parseDiagnostic(fixture["expected_diagnostic"].(map[string]any))
	expectedRequest := parseReviewRequest(fixture["expected_request"].(map[string]any))

	context, diagnostics, requests, applied := ReviewConformanceFamilyContext(family, options)
	if context != nil || len(applied) != 0 {
		t.Fatalf("unexpected explicit payload validation application: %+v %+v", context, applied)
	}
	if !reflect.DeepEqual(diagnostics, []Diagnostic{expectedDiagnostic}) {
		t.Fatalf("unexpected explicit payload diagnostics: %+v", diagnostics)
	}
	if !reflect.DeepEqual(requests, []ReviewRequest{expectedRequest}) {
		t.Fatalf("unexpected explicit payload requests: %+v", requests)
	}
}

func TestSharedFixtureExplicitReviewDecisionFamilyValidation(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "explicit_review_decision_family_mismatch"))
	family := fixture["family"].(string)
	options := parseConformanceManifestReviewOptions(fixture["options"].(map[string]any))
	expectedDiagnostic := parseDiagnostic(fixture["expected_diagnostic"].(map[string]any))
	expectedRequest := parseReviewRequest(fixture["expected_request"].(map[string]any))

	context, diagnostics, requests, applied := ReviewConformanceFamilyContext(family, options)
	if context != nil || len(applied) != 0 {
		t.Fatalf("unexpected explicit family validation application: %+v %+v", context, applied)
	}
	if !reflect.DeepEqual(diagnostics, []Diagnostic{expectedDiagnostic}) {
		t.Fatalf("unexpected explicit family diagnostics: %+v", diagnostics)
	}
	if !reflect.DeepEqual(requests, []ReviewRequest{expectedRequest}) {
		t.Fatalf("unexpected explicit family requests: %+v", requests)
	}
}

func TestSharedFixtureConformanceManifestReviewState(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "conformance_manifest_review_state"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(fixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	options := parseConformanceManifestReviewOptions(fixture["options"].(map[string]any))
	executionsRaw := fixture["executions"].(map[string]any)
	expected := parseConformanceManifestReviewState(fixture["expected_state"].(map[string]any))
	expectedReplayContext := parseReviewReplayContext(fixture["expected_state"].(map[string]any)["replay_context"].(map[string]any))

	state := ReviewConformanceManifest(
		manifest,
		options,
		func(run ConformanceCaseRun) ConformanceCaseExecution {
			key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
			if raw, ok := executionsRaw[key]; ok {
				return parseConformanceCaseExecution(raw.(map[string]any))
			}

			return ConformanceCaseExecution{
				Outcome:  ConformanceFailed,
				Messages: []string{"missing execution"},
			}
		},
	)

	if !reflect.DeepEqual(state, expected) {
		t.Fatalf("unexpected conformance manifest review state: %+v", state)
	}
	if replayContext := ConformanceManifestReplayContext(manifest, options); !reflect.DeepEqual(replayContext, expectedReplayContext) {
		t.Fatalf("unexpected replay context: %+v", replayContext)
	}
}

func TestSharedFixtureReviewedDefaultContext(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "reviewed_default_context"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(fixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	options := parseConformanceManifestReviewOptions(fixture["options"].(map[string]any))
	executionsRaw := fixture["executions"].(map[string]any)
	expected := parseConformanceManifestReviewState(fixture["expected_state"].(map[string]any))

	state := ReviewConformanceManifest(
		manifest,
		options,
		func(run ConformanceCaseRun) ConformanceCaseExecution {
			key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
			if raw, ok := executionsRaw[key]; ok {
				return parseConformanceCaseExecution(raw.(map[string]any))
			}

			return ConformanceCaseExecution{
				Outcome:  ConformanceFailed,
				Messages: []string{"missing execution"},
			}
		},
	)

	if !reflect.DeepEqual(state, expected) {
		t.Fatalf("unexpected reviewed default-context state: %+v", state)
	}
}

func TestSharedFixtureReviewReplayCompatibility(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "review_replay_compatibility"))
	current := parseReviewReplayContext(fixture["current_context"].(map[string]any))
	compatible := parseReviewReplayContext(fixture["compatible_context"].(map[string]any))
	incompatible := parseReviewReplayContext(fixture["incompatible_context"].(map[string]any))

	if !ReviewReplayContextCompatible(current, &compatible) {
		t.Fatalf("expected compatible replay context")
	}
	if ReviewReplayContextCompatible(current, &incompatible) {
		t.Fatalf("expected incompatible replay context")
	}
	if ReviewReplayContextCompatible(current, nil) {
		t.Fatalf("expected missing replay context to be incompatible")
	}
}

func TestSharedFixtureReviewReplayRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "review_replay_rejection"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(fixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	options := parseConformanceManifestReviewOptions(fixture["options"].(map[string]any))
	executionsRaw := fixture["executions"].(map[string]any)
	expected := parseConformanceManifestReviewState(fixture["expected_state"].(map[string]any))

	state := ReviewConformanceManifest(
		manifest,
		options,
		func(run ConformanceCaseRun) ConformanceCaseExecution {
			key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
			if raw, ok := executionsRaw[key]; ok {
				return parseConformanceCaseExecution(raw.(map[string]any))
			}

			return ConformanceCaseExecution{
				Outcome:  ConformanceFailed,
				Messages: []string{"missing execution"},
			}
		},
	)

	if !reflect.DeepEqual(state, expected) {
		t.Fatalf("unexpected review replay rejection state: %+v", state)
	}
}

func TestSharedFixtureReviewRequestIDs(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "review_request_ids"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(fixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	options := parseConformanceManifestReviewOptions(fixture["options"].(map[string]any))
	expectedRaw := fixture["expected_request_ids"].([]any)
	expected := make([]string, 0, len(expectedRaw))
	for _, item := range expectedRaw {
		expected = append(expected, item.(string))
	}

	if requestIDs := ConformanceManifestReviewRequestIDs(manifest, options); !reflect.DeepEqual(requestIDs, expected) {
		t.Fatalf("unexpected review request ids: %+v", requestIDs)
	}
}

func TestSharedFixtureStaleReviewDecision(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "stale_review_decision"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(fixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	options := parseConformanceManifestReviewOptions(fixture["options"].(map[string]any))
	executionsRaw := fixture["executions"].(map[string]any)
	expected := parseConformanceManifestReviewState(fixture["expected_state"].(map[string]any))

	state := ReviewConformanceManifest(
		manifest,
		options,
		func(run ConformanceCaseRun) ConformanceCaseExecution {
			key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
			if raw, ok := executionsRaw[key]; ok {
				return parseConformanceCaseExecution(raw.(map[string]any))
			}

			return ConformanceCaseExecution{
				Outcome:  ConformanceFailed,
				Messages: []string{"missing execution"},
			}
		},
	)

	if !reflect.DeepEqual(state, expected) {
		t.Fatalf("unexpected stale review decision state: %+v", state)
	}
}

func TestSharedFixtureReviewReplayBundle(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "review_replay_bundle"))
	bundle := parseReviewReplayBundle(fixture["replay_bundle"].(map[string]any))

	replayContext, decisions, reviewedNestedExecutions := ReviewReplayBundleInputs(ConformanceManifestReviewOptions{
		ReviewReplayBundle: &bundle,
	})

	if replayContext == nil || !reflect.DeepEqual(*replayContext, bundle.ReplayContext) {
		t.Fatalf("unexpected replay bundle context: %+v", replayContext)
	}
	if !reflect.DeepEqual(decisions, bundle.Decisions) {
		t.Fatalf("unexpected replay bundle decisions: %+v", decisions)
	}
	if !reflect.DeepEqual(reviewedNestedExecutions, bundle.ReviewedNestedExecutions) {
		t.Fatalf("unexpected replay bundle reviewed nested executions: %+v", reviewedNestedExecutions)
	}
}

func TestSharedFixtureReviewReplayBundleReviewedNestedExecutions(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "review_replay_bundle_reviewed_nested_executions"))
	bundle := parseReviewReplayBundle(fixture["replay_bundle"].(map[string]any))

	replayContext, decisions, reviewedNestedExecutions := ReviewReplayBundleInputs(ConformanceManifestReviewOptions{
		ReviewReplayBundle: &bundle,
	})

	if replayContext == nil || !reflect.DeepEqual(*replayContext, bundle.ReplayContext) {
		t.Fatalf("unexpected replay bundle context: %+v", replayContext)
	}
	if !reflect.DeepEqual(decisions, bundle.Decisions) {
		t.Fatalf("unexpected replay bundle decisions: %+v", decisions)
	}
	if !reflect.DeepEqual(reviewedNestedExecutions, bundle.ReviewedNestedExecutions) {
		t.Fatalf("unexpected reviewed nested executions: %+v", reviewedNestedExecutions)
	}
}

func TestSharedFixtureReviewReplayBundleApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "review_replay_bundle_application"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(fixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	options := parseConformanceManifestReviewOptions(fixture["options"].(map[string]any))
	executionsRaw := fixture["executions"].(map[string]any)
	expected := parseConformanceManifestReviewState(fixture["expected_state"].(map[string]any))

	state := ReviewConformanceManifest(
		manifest,
		options,
		func(run ConformanceCaseRun) ConformanceCaseExecution {
			key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
			if raw, ok := executionsRaw[key]; ok {
				return parseConformanceCaseExecution(raw.(map[string]any))
			}

			return ConformanceCaseExecution{
				Outcome:  ConformanceFailed,
				Messages: []string{"missing execution"},
			}
		},
	)

	if !reflect.DeepEqual(state, expected) {
		t.Fatalf("unexpected review replay bundle application state: %+v", state)
	}
}

func TestSharedFixtureExplicitReviewReplayBundleApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "explicit_review_replay_bundle_application"))
	var manifest ConformanceManifest
	if raw, err := json.Marshal(fixture["manifest"]); err != nil {
		t.Fatalf("marshal manifest: %v", err)
	} else if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	options := parseConformanceManifestReviewOptions(fixture["options"].(map[string]any))
	expected := parseConformanceManifestReviewState(fixture["expected_state"].(map[string]any))
	executionsRaw := fixture["executions"].(map[string]any)

	state := ReviewConformanceManifest(
		manifest,
		options,
		func(run ConformanceCaseRun) ConformanceCaseExecution {
			key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
			if raw, ok := executionsRaw[key]; ok {
				return parseConformanceCaseExecution(raw.(map[string]any))
			}

			return ConformanceCaseExecution{
				Outcome:  ConformanceFailed,
				Messages: []string{"missing execution"},
			}
		},
	)

	if !reflect.DeepEqual(state, expected) {
		t.Fatalf("unexpected explicit replay bundle state: %+v", state)
	}
}

func TestSharedFixtureSurfaceOwnership(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "surface_ownership"))
	var surface DiscoveredSurface
	if raw, err := json.Marshal(fixture["surface"]); err != nil {
		t.Fatalf("marshal surface: %v", err)
	} else if err := json.Unmarshal(raw, &surface); err != nil {
		t.Fatalf("unmarshal surface: %v", err)
	}

	roundtrip, err := json.Marshal(surface)
	if err != nil {
		t.Fatalf("marshal roundtrip surface: %v", err)
	}
	var decoded DiscoveredSurface
	if err := json.Unmarshal(roundtrip, &decoded); err != nil {
		t.Fatalf("unmarshal roundtrip surface: %v", err)
	}

	if !reflect.DeepEqual(decoded, surface) {
		t.Fatalf("unexpected surface roundtrip: %+v", decoded)
	}
}

func TestSharedFixtureDelegatedChildOperation(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "delegated_child_operation"))
	var operation DelegatedChildOperation
	if raw, err := json.Marshal(fixture["operation"]); err != nil {
		t.Fatalf("marshal operation: %v", err)
	} else if err := json.Unmarshal(raw, &operation); err != nil {
		t.Fatalf("unmarshal operation: %v", err)
	}

	roundtrip, err := json.Marshal(operation)
	if err != nil {
		t.Fatalf("marshal roundtrip operation: %v", err)
	}
	var decoded DelegatedChildOperation
	if err := json.Unmarshal(roundtrip, &decoded); err != nil {
		t.Fatalf("unmarshal roundtrip operation: %v", err)
	}

	if !reflect.DeepEqual(decoded, operation) {
		t.Fatalf("unexpected delegated child operation roundtrip: %+v", decoded)
	}
}

func TestSharedFixtureProjectedChildReviewCases(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "projected_child_review_cases"))
	var cases []ProjectedChildReviewCase
	if raw, err := json.Marshal(fixture["cases"]); err != nil {
		t.Fatalf("marshal projected child review cases: %v", err)
	} else if err := json.Unmarshal(raw, &cases); err != nil {
		t.Fatalf("unmarshal projected child review cases: %v", err)
	}

	roundtrip, err := json.Marshal(cases)
	if err != nil {
		t.Fatalf("marshal roundtrip projected child review cases: %v", err)
	}
	var decoded []ProjectedChildReviewCase
	if err := json.Unmarshal(roundtrip, &decoded); err != nil {
		t.Fatalf("unmarshal roundtrip projected child review cases: %v", err)
	}

	if !reflect.DeepEqual(decoded, cases) {
		t.Fatalf("unexpected projected child review cases roundtrip: %+v", decoded)
	}
}

func TestSharedFixtureProjectedChildReviewGroups(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "projected_child_review_groups"))
	source, err := json.Marshal(fixture["cases"])
	if err != nil {
		t.Fatalf("marshal projected child review cases: %v", err)
	}
	var cases []ProjectedChildReviewCase
	if err := json.Unmarshal(source, &cases); err != nil {
		t.Fatalf("unmarshal projected child review cases: %v", err)
	}
	grouped := GroupProjectedChildReviewCases(cases)
	encoded, err := json.Marshal(grouped)
	if err != nil {
		t.Fatalf("marshal projected child review groups: %v", err)
	}
	var decoded any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal projected child review groups: %v", err)
	}
	if !reflect.DeepEqual(decoded, fixture["expected_groups"]) {
		t.Fatalf("unexpected projected child review groups: %+v", decoded)
	}
}

func TestSharedFixtureProjectedChildReviewGroupProgress(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "projected_child_review_group_progress"))
	groupSource, err := json.Marshal(fixture["groups"])
	if err != nil {
		t.Fatalf("marshal projected child review groups: %v", err)
	}
	var groups []ProjectedChildReviewGroup
	if err := json.Unmarshal(groupSource, &groups); err != nil {
		t.Fatalf("unmarshal projected child review groups: %v", err)
	}
	resolvedSource, err := json.Marshal(fixture["resolved_case_ids"])
	if err != nil {
		t.Fatalf("marshal resolved case ids: %v", err)
	}
	var resolvedCaseIDs []string
	if err := json.Unmarshal(resolvedSource, &resolvedCaseIDs); err != nil {
		t.Fatalf("unmarshal resolved case ids: %v", err)
	}
	encoded, err := json.Marshal(SummarizeProjectedChildReviewGroupProgress(groups, resolvedCaseIDs))
	if err != nil {
		t.Fatalf("marshal projected child review group progress: %v", err)
	}
	var decoded any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal projected child review group progress: %v", err)
	}
	if !reflect.DeepEqual(decoded, fixture["expected_progress"]) {
		t.Fatalf("unexpected projected child review group progress: %+v", decoded)
	}
}

func TestSharedFixtureProjectedChildReviewGroupsReadyForApply(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "projected_child_review_groups_ready_for_apply"))
	groupSource, err := json.Marshal(fixture["groups"])
	if err != nil {
		t.Fatalf("marshal projected child review groups: %v", err)
	}
	var groups []ProjectedChildReviewGroup
	if err := json.Unmarshal(groupSource, &groups); err != nil {
		t.Fatalf("unmarshal projected child review groups: %v", err)
	}
	resolvedSource, err := json.Marshal(fixture["resolved_case_ids"])
	if err != nil {
		t.Fatalf("marshal resolved case ids: %v", err)
	}
	var resolvedCaseIDs []string
	if err := json.Unmarshal(resolvedSource, &resolvedCaseIDs); err != nil {
		t.Fatalf("unmarshal resolved case ids: %v", err)
	}
	encoded, err := json.Marshal(SelectProjectedChildReviewGroupsReadyForApply(groups, resolvedCaseIDs))
	if err != nil {
		t.Fatalf("marshal ready projected child review groups: %v", err)
	}
	var decoded any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal ready projected child review groups: %v", err)
	}
	if !reflect.DeepEqual(decoded, fixture["expected_ready_groups"]) {
		t.Fatalf("unexpected ready projected child review groups: %+v", decoded)
	}
}

func TestSharedFixtureDelegatedChildGroupReviewRequest(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "delegated_child_group_review_request"))
	group := parseProjectedChildReviewGroup(fixture["group"].(map[string]any))
	expectedRequest := parseReviewRequest(fixture["expected_request"].(map[string]any))

	if actualID := ReviewRequestIDForProjectedChildGroup(group); actualID != expectedRequest.ID {
		t.Fatalf("unexpected delegated child review request id: %s", actualID)
	}
	if actual := ProjectedChildGroupReviewRequest(group, fixture["family"].(string)); !reflect.DeepEqual(actual, expectedRequest) {
		t.Fatalf("unexpected delegated child review request: %+v", actual)
	}
}

func TestSharedFixtureDelegatedChildGroupsAcceptedForApply(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "delegated_child_groups_accepted_for_apply"))
	groupSource, err := json.Marshal(fixture["groups"])
	if err != nil {
		t.Fatalf("marshal projected child review groups: %v", err)
	}
	var groups []ProjectedChildReviewGroup
	if err := json.Unmarshal(groupSource, &groups); err != nil {
		t.Fatalf("unmarshal projected child review groups: %v", err)
	}
	decisions := make([]ReviewDecision, 0, len(fixture["decisions"].([]any)))
	for _, item := range fixture["decisions"].([]any) {
		decisions = append(decisions, parseReviewDecision(item.(map[string]any)))
	}
	encoded, err := json.Marshal(SelectProjectedChildReviewGroupsAcceptedForApply(groups, fixture["family"].(string), decisions))
	if err != nil {
		t.Fatalf("marshal accepted projected child review groups: %v", err)
	}
	var decoded any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal accepted projected child review groups: %v", err)
	}
	if !reflect.DeepEqual(decoded, fixture["expected_accepted_groups"]) {
		t.Fatalf("unexpected accepted projected child review groups: %+v", decoded)
	}
}

func TestSharedFixtureDelegatedChildGroupReviewState(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "delegated_child_group_review_state"))
	groupSource, err := json.Marshal(fixture["groups"])
	if err != nil {
		t.Fatalf("marshal projected child review groups: %v", err)
	}
	var groups []ProjectedChildReviewGroup
	if err := json.Unmarshal(groupSource, &groups); err != nil {
		t.Fatalf("unmarshal projected child review groups: %v", err)
	}
	decisions := make([]ReviewDecision, 0, len(fixture["decisions"].([]any)))
	for _, item := range fixture["decisions"].([]any) {
		decisions = append(decisions, parseReviewDecision(item.(map[string]any)))
	}
	encoded, err := json.Marshal(ReviewProjectedChildGroups(groups, fixture["family"].(string), decisions))
	if err != nil {
		t.Fatalf("marshal delegated child review state: %v", err)
	}
	var decoded any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal delegated child review state: %v", err)
	}
	if !reflect.DeepEqual(decoded, fixture["expected_state"]) {
		t.Fatalf("unexpected delegated child review state: %+v", decoded)
	}
}

func TestSharedFixtureDelegatedChildApplyPlan(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "delegated_child_apply_plan"))
	stateSource, err := json.Marshal(fixture["review_state"])
	if err != nil {
		t.Fatalf("marshal delegated child review state: %v", err)
	}
	var state DelegatedChildGroupReviewState
	if err := json.Unmarshal(stateSource, &state); err != nil {
		t.Fatalf("unmarshal delegated child review state: %v", err)
	}
	encoded, err := json.Marshal(DelegatedChildApplyPlanForState(state, fixture["family"].(string)))
	if err != nil {
		t.Fatalf("marshal delegated child apply plan: %v", err)
	}
	var decoded any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal delegated child apply plan: %v", err)
	}
	if !reflect.DeepEqual(decoded, fixture["expected_plan"]) {
		t.Fatalf("unexpected delegated child apply plan: %+v", decoded)
	}
}

func TestSharedFixtureDelegatedChildNestedOutputResolution(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "delegated_child_nested_output_resolution"))
	operations := decodeFixtureValue[[]DelegatedChildOperation](t, fixture["operations"])
	nestedOutputs := decodeFixtureValue[[]DelegatedChildSurfaceOutput](t, fixture["nested_outputs"])
	options := DelegatedChildOutputResolutionOptions{
		DefaultFamily:   fixture["default_family"].(string),
		RequestIDPrefix: fixture["request_id_prefix"].(string),
	}

	encoded, err := json.Marshal(ResolveDelegatedChildOutputs(operations, nestedOutputs, options))
	if err != nil {
		t.Fatalf("marshal delegated child nested output resolution: %v", err)
	}
	var decoded any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal delegated child nested output resolution: %v", err)
	}
	if !reflect.DeepEqual(decoded, fixture["expected"]) {
		t.Fatalf("unexpected delegated child nested output resolution: %+v", decoded)
	}
}

func TestSharedFixtureDelegatedChildNestedOutputRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "delegated_child_nested_output_rejection"))
	operations := decodeFixtureValue[[]DelegatedChildOperation](t, fixture["operations"])
	nestedOutputs := decodeFixtureValue[[]DelegatedChildSurfaceOutput](t, fixture["nested_outputs"])
	options := DelegatedChildOutputResolutionOptions{
		DefaultFamily:   fixture["default_family"].(string),
		RequestIDPrefix: fixture["request_id_prefix"].(string),
	}

	encoded, err := json.Marshal(ResolveDelegatedChildOutputs(operations, nestedOutputs, options))
	if err != nil {
		t.Fatalf("marshal delegated child nested output rejection: %v", err)
	}
	var decoded any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal delegated child nested output rejection: %v", err)
	}
	if !reflect.DeepEqual(decoded, fixture["expected"]) {
		t.Fatalf("unexpected delegated child nested output rejection: %+v", decoded)
	}
}

func TestSharedFixtureReviewStateJSONRoundtrip(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "review_state_json_roundtrip"))
	state := parseConformanceManifestReviewState(fixture["state"].(map[string]any))

	raw, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("marshal review state: %v", err)
	}
	var roundTripped ConformanceManifestReviewState
	if err := json.Unmarshal(raw, &roundTripped); err != nil {
		t.Fatalf("unmarshal review state: %v", err)
	}

	if !reflect.DeepEqual(roundTripped, state) {
		t.Fatalf("unexpected review state roundtrip: %+v", roundTripped)
	}
}

func TestSharedFixtureReviewReplayBundleJSONRoundtrip(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "review_replay_bundle_json_roundtrip"))
	bundle := parseReviewReplayBundle(fixture["replay_bundle"].(map[string]any))

	raw, err := json.Marshal(bundle)
	if err != nil {
		t.Fatalf("marshal replay bundle: %v", err)
	}
	var roundTripped ReviewReplayBundle
	if err := json.Unmarshal(raw, &roundTripped); err != nil {
		t.Fatalf("unmarshal replay bundle: %v", err)
	}

	if !reflect.DeepEqual(roundTripped, bundle) {
		t.Fatalf("unexpected replay bundle roundtrip: %+v", roundTripped)
	}
}

func TestSharedFixtureReviewStateTransportEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "review_state_envelope"))
	state := parseConformanceManifestReviewState(fixture["state"].(map[string]any))
	expected := parseConformanceManifestReviewStateEnvelope(fixture["expected_envelope"].(map[string]any))

	envelope := ConformanceManifestReviewStateEnvelopeFor(state)
	if !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected review state envelope: %+v", envelope)
	}
	imported, importErr := ImportConformanceManifestReviewStateEnvelope(expected)
	if importErr != nil {
		t.Fatalf("unexpected review state import error: %+v", importErr)
	}
	if imported == nil || !reflect.DeepEqual(*imported, state) {
		t.Fatalf("unexpected imported review state: %+v", imported)
	}
}

func TestSharedFixtureReviewReplayBundleTransportEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "review_replay_bundle_envelope"))
	bundle := parseReviewReplayBundle(fixture["replay_bundle"].(map[string]any))
	expected := parseReviewReplayBundleEnvelope(fixture["expected_envelope"].(map[string]any))

	envelope := ReviewReplayBundleEnvelopeFor(bundle)
	if !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected replay bundle envelope: %+v", envelope)
	}
	imported, importErr := ImportReviewReplayBundleEnvelope(expected)
	if importErr != nil {
		t.Fatalf("unexpected replay bundle import error: %+v", importErr)
	}
	if imported == nil || !reflect.DeepEqual(*imported, bundle) {
		t.Fatalf("unexpected imported replay bundle: %+v", imported)
	}
}

func TestSharedFixtureReviewStateTransportRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "review_state_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		rejectionCase := rawCase.(map[string]any)
		envelope := parseConformanceManifestReviewStateEnvelope(rejectionCase["envelope"].(map[string]any))
		expected := parseReviewTransportImportError(rejectionCase["expected_error"].(map[string]any))

		imported, importErr := ImportConformanceManifestReviewStateEnvelope(envelope)
		if imported != nil || !reflect.DeepEqual(importErr, &expected) {
			t.Fatalf("unexpected review state rejection: imported=%+v error=%+v", imported, importErr)
		}
	}
}

func TestSharedFixtureReviewReplayBundleTransportRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "review_replay_bundle_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		rejectionCase := rawCase.(map[string]any)
		envelope := parseReviewReplayBundleEnvelope(rejectionCase["envelope"].(map[string]any))
		expected := parseReviewTransportImportError(rejectionCase["expected_error"].(map[string]any))

		imported, importErr := ImportReviewReplayBundleEnvelope(envelope)
		if imported != nil || !reflect.DeepEqual(importErr, &expected) {
			t.Fatalf("unexpected replay bundle rejection: imported=%+v error=%+v", imported, importErr)
		}
	}
}

func TestSharedFixtureReviewReplayBundleEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "review_replay_bundle_envelope_application"))
	manifest := decodeFixtureValue[ConformanceManifest](t, fixture["manifest"])
	options := parseConformanceManifestReviewOptions(fixture["options"].(map[string]any))
	envelope := parseReviewReplayBundleEnvelope(fixture["review_replay_bundle_envelope"].(map[string]any))
	expected := parseConformanceManifestReviewState(fixture["expected_state"].(map[string]any))
	executionsRaw := fixture["executions"].(map[string]any)

	state := ReviewConformanceManifestWithReplayBundleEnvelope(manifest, options, envelope, func(run ConformanceCaseRun) ConformanceCaseExecution {
		key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
		if raw, ok := executionsRaw[key]; ok {
			return parseConformanceCaseExecution(raw.(map[string]any))
		}

		return ConformanceCaseExecution{
			Outcome:  ConformanceFailed,
			Messages: []string{"missing execution"},
		}
	})

	if !reflect.DeepEqual(state, expected) {
		t.Fatalf("unexpected replay bundle envelope application state: %+v", state)
	}
}

func TestSharedFixtureExplicitReviewReplayBundleEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "explicit_review_replay_bundle_envelope_application"))
	manifest := decodeFixtureValue[ConformanceManifest](t, fixture["manifest"])
	options := parseConformanceManifestReviewOptions(fixture["options"].(map[string]any))
	envelope := parseReviewReplayBundleEnvelope(fixture["review_replay_bundle_envelope"].(map[string]any))
	expected := parseConformanceManifestReviewState(fixture["expected_state"].(map[string]any))
	executionsRaw := fixture["executions"].(map[string]any)

	state := ReviewConformanceManifestWithReplayBundleEnvelope(manifest, options, envelope, func(run ConformanceCaseRun) ConformanceCaseExecution {
		key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
		if raw, ok := executionsRaw[key]; ok {
			return parseConformanceCaseExecution(raw.(map[string]any))
		}

		return ConformanceCaseExecution{
			Outcome:  ConformanceFailed,
			Messages: []string{"missing execution"},
		}
	})

	if !reflect.DeepEqual(state, expected) {
		t.Fatalf("unexpected explicit replay bundle envelope application state: %+v", state)
	}
}

func TestSharedFixtureReviewReplayBundleEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "review_replay_bundle_envelope_review_rejection"))
	manifest := decodeFixtureValue[ConformanceManifest](t, fixture["manifest"])
	options := parseConformanceManifestReviewOptions(fixture["options"].(map[string]any))
	executionsRaw := fixture["executions"].(map[string]any)

	for _, rawCase := range fixture["cases"].([]any) {
		fixtureCase := rawCase.(map[string]any)
		envelope := parseReviewReplayBundleEnvelope(fixtureCase["review_replay_bundle_envelope"].(map[string]any))
		expected := parseConformanceManifestReviewState(fixtureCase["expected_state"].(map[string]any))

		state := ReviewConformanceManifestWithReplayBundleEnvelope(manifest, options, envelope, func(run ConformanceCaseRun) ConformanceCaseExecution {
			key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
			if raw, ok := executionsRaw[key]; ok {
				return parseConformanceCaseExecution(raw.(map[string]any))
			}

			return ConformanceCaseExecution{
				Outcome:  ConformanceFailed,
				Messages: []string{"missing execution"},
			}
		})

		if !reflect.DeepEqual(state, expected) {
			t.Fatalf("unexpected replay bundle envelope rejection state: %+v", state)
		}
	}
}

func TestSharedFixtureReviewedNestedExecutionJSONRoundtrip(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "reviewed_nested_execution_json_roundtrip"))
	execution := parseReviewedNestedExecution(fixture["execution"].(map[string]any))

	raw, err := json.Marshal(execution)
	if err != nil {
		t.Fatalf("marshal reviewed nested execution: %v", err)
	}
	var roundTripped ReviewedNestedExecution
	if err := json.Unmarshal(raw, &roundTripped); err != nil {
		t.Fatalf("unmarshal reviewed nested execution: %v", err)
	}

	if !reflect.DeepEqual(roundTripped, execution) {
		t.Fatalf("unexpected reviewed nested execution roundtrip: %+v", roundTripped)
	}
}

func TestSharedFixtureReviewedNestedExecutionTransportEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "reviewed_nested_execution_envelope"))
	execution := parseReviewedNestedExecution(fixture["execution"].(map[string]any))
	expected := parseReviewedNestedExecutionEnvelope(fixture["expected_envelope"].(map[string]any))

	envelope := ReviewedNestedExecutionEnvelopeFor(execution)
	if !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected reviewed nested execution envelope: %+v", envelope)
	}
	imported, importErr := ImportReviewedNestedExecutionEnvelope(expected)
	if importErr != nil {
		t.Fatalf("unexpected reviewed nested execution import error: %+v", importErr)
	}
	if imported == nil || !reflect.DeepEqual(*imported, execution) {
		t.Fatalf("unexpected imported reviewed nested execution: %+v", imported)
	}
}

func TestSharedFixtureReviewedNestedExecutionTransportRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "reviewed_nested_execution_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		rejectionCase := rawCase.(map[string]any)
		envelope := parseReviewedNestedExecutionEnvelope(rejectionCase["envelope"].(map[string]any))
		expected := parseReviewTransportImportError(rejectionCase["expected_error"].(map[string]any))

		imported, importErr := ImportReviewedNestedExecutionEnvelope(envelope)
		if imported != nil || !reflect.DeepEqual(importErr, &expected) {
			t.Fatalf("unexpected reviewed nested execution rejection: imported=%+v error=%+v", imported, importErr)
		}
	}
}

func TestSharedFixtureReviewedNestedExecutionPayload(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "reviewed_nested_execution_payload"))
	reviewState := parseDelegatedChildGroupReviewState(fixture["review_state"].(map[string]any))
	appliedChildren := parseReviewedNestedExecution(fixture["expected_execution"].(map[string]any)).AppliedChildren
	expected := parseReviewedNestedExecution(fixture["expected_execution"].(map[string]any))

	execution := ReviewedNestedExecutionFor(
		fixture["family"].(string),
		reviewState,
		appliedChildren,
	)
	if !reflect.DeepEqual(execution, expected) {
		t.Fatalf("unexpected reviewed nested execution payload: %+v", execution)
	}
}

func TestSharedFixtureReviewStateReviewedNestedExecutions(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "review_state_reviewed_nested_executions"))
	manifest := decodeFixtureValue[ConformanceManifest](t, fixture["manifest"])
	options := parseConformanceManifestReviewOptions(fixture["options"].(map[string]any))
	expected := parseConformanceManifestReviewState(fixture["expected_state"].(map[string]any))
	executionsRaw := fixture["executions"].(map[string]any)

	state := ReviewConformanceManifest(manifest, options, func(run ConformanceCaseRun) ConformanceCaseExecution {
		key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
		if raw, ok := executionsRaw[key]; ok {
			return parseConformanceCaseExecution(raw.(map[string]any))
		}

		return ConformanceCaseExecution{
			Outcome:  ConformanceFailed,
			Messages: []string{"missing execution"},
		}
	})

	if !reflect.DeepEqual(state, expected) {
		t.Fatalf("unexpected reviewed nested execution review state: %+v", state)
	}
}

func TestSharedFixtureReviewReplayBundleReviewedNestedExecutionApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "review_replay_bundle_reviewed_nested_execution_application"))
	bundle := parseReviewReplayBundle(fixture["replay_bundle"].(map[string]any))
	expected := fixture["expected_results"].([]any)

	results := ExecuteReviewReplayBundleReviewedNestedExecutions(bundle, reviewedNestedExecutionCallbacksForFixture(t, expected))
	assertReviewedNestedExecutionResults(t, results, expected)
}

func TestSharedFixtureReviewStateReviewedNestedExecutionApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "review_state_reviewed_nested_execution_application"))
	state := parseConformanceManifestReviewState(fixture["review_state"].(map[string]any))
	expected := fixture["expected_results"].([]any)

	results := ExecuteReviewStateReviewedNestedExecutions(state, reviewedNestedExecutionCallbacksForFixture(t, expected))
	assertReviewedNestedExecutionResults(t, results, expected)
}

func TestSharedFixtureReviewReplayBundleEnvelopeReviewedNestedExecutionApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "review_replay_bundle_envelope_reviewed_nested_execution_application"))
	envelope := parseReviewReplayBundleEnvelope(fixture["replay_bundle_envelope"].(map[string]any))
	expected := fixture["expected_application"].(map[string]any)
	expectedResults := expected["results"].([]any)

	application := ExecuteReviewReplayBundleEnvelopeReviewedNestedExecutions(envelope, reviewedNestedExecutionCallbacksForFixture(t, expectedResults))
	if !reflect.DeepEqual(application.Diagnostics, []Diagnostic{}) {
		t.Fatalf("unexpected replay bundle envelope application diagnostics: %+v", application.Diagnostics)
	}
	assertReviewedNestedExecutionResults(t, application.Results, expectedResults)
}

func TestSharedFixtureReviewStateEnvelopeReviewedNestedExecutionApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "review_state_envelope_reviewed_nested_execution_application"))
	envelope := parseConformanceManifestReviewStateEnvelope(fixture["review_state_envelope"].(map[string]any))
	expected := fixture["expected_application"].(map[string]any)
	expectedResults := expected["results"].([]any)

	application := ExecuteReviewStateEnvelopeReviewedNestedExecutions(envelope, reviewedNestedExecutionCallbacksForFixture(t, expectedResults))
	if !reflect.DeepEqual(application.Diagnostics, []Diagnostic{}) {
		t.Fatalf("unexpected review state envelope application diagnostics: %+v", application.Diagnostics)
	}
	assertReviewedNestedExecutionResults(t, application.Results, expectedResults)
}

func TestSharedFixtureReviewReplayBundleEnvelopeReviewedNestedExecutionRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "review_replay_bundle_envelope_reviewed_nested_execution_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		fixtureCase := rawCase.(map[string]any)
		envelope := parseReviewReplayBundleEnvelope(fixtureCase["replay_bundle_envelope"].(map[string]any))
		expected := fixtureCase["expected_application"].(map[string]any)
		expectedDiagnostics := make([]Diagnostic, 0, len(expected["diagnostics"].([]any)))
		for _, rawDiagnostic := range expected["diagnostics"].([]any) {
			expectedDiagnostics = append(expectedDiagnostics, parseDiagnostic(rawDiagnostic.(map[string]any)))
		}

		application := ExecuteReviewReplayBundleEnvelopeReviewedNestedExecutions[string](envelope, func(ReviewedNestedExecution, int) NestedMergeExecutionCallbacks[string] {
			t.Fatal("callbacks should not run for rejected replay bundle envelopes")
			return NestedMergeExecutionCallbacks[string]{}
		})

		if !reflect.DeepEqual(application.Diagnostics, expectedDiagnostics) || len(application.Results) != 0 {
			t.Fatalf("unexpected replay bundle envelope rejection application: %+v", application)
		}
	}
}

func TestSharedFixtureReviewStateEnvelopeReviewedNestedExecutionRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "review_state_envelope_reviewed_nested_execution_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		fixtureCase := rawCase.(map[string]any)
		envelope := parseConformanceManifestReviewStateEnvelope(fixtureCase["review_state_envelope"].(map[string]any))
		expected := fixtureCase["expected_application"].(map[string]any)
		expectedDiagnostics := make([]Diagnostic, 0, len(expected["diagnostics"].([]any)))
		for _, rawDiagnostic := range expected["diagnostics"].([]any) {
			expectedDiagnostics = append(expectedDiagnostics, parseDiagnostic(rawDiagnostic.(map[string]any)))
		}

		application := ExecuteReviewStateEnvelopeReviewedNestedExecutions[string](envelope, func(ReviewedNestedExecution, int) NestedMergeExecutionCallbacks[string] {
			t.Fatal("callbacks should not run for rejected review state envelopes")
			return NestedMergeExecutionCallbacks[string]{}
		})

		if !reflect.DeepEqual(application.Diagnostics, expectedDiagnostics) || len(application.Results) != 0 {
			t.Fatalf("unexpected review state envelope rejection application: %+v", application)
		}
	}
}

func TestSharedFixtureReviewReplayBundleEnvelopeReviewedNestedManifestApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "review_replay_bundle_envelope_reviewed_nested_manifest_application"))
	manifest := decodeFixtureValue[ConformanceManifest](t, fixture["manifest"])
	options := parseConformanceManifestReviewOptions(fixture["options"].(map[string]any))
	envelope := parseReviewReplayBundleEnvelope(fixture["review_replay_bundle_envelope"].(map[string]any))
	expectedState := parseConformanceManifestReviewState(fixture["expected_state"].(map[string]any))
	expectedApplication := fixture["expected_application"].(map[string]any)
	expectedResults := expectedApplication["results"].([]any)

	application := ReviewAndExecuteConformanceManifestWithReplayBundleEnvelope(
		manifest,
		options,
		envelope,
		func(run ConformanceCaseRun) ConformanceCaseExecution {
			key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
			if raw, ok := fixture["executions"].(map[string]any)[key]; ok {
				return parseConformanceCaseExecution(raw.(map[string]any))
			}

			return ConformanceCaseExecution{
				Outcome:  ConformanceFailed,
				Messages: []string{"missing execution"},
			}
		},
		reviewedNestedExecutionCallbacksForFixture(t, expectedResults),
	)

	if !reflect.DeepEqual(application.State, expectedState) {
		t.Fatalf("unexpected replay bundle envelope reviewed nested application state: %+v", application.State)
	}
	assertReviewedNestedExecutionResults(t, application.Results, expectedResults)
}

func TestSharedFixtureReviewReplayBundleEnvelopeReviewedNestedManifestRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "review_replay_bundle_envelope_reviewed_nested_manifest_rejection"))
	manifest := decodeFixtureValue[ConformanceManifest](t, fixture["manifest"])
	options := parseConformanceManifestReviewOptions(fixture["options"].(map[string]any))
	executionsRaw := fixture["executions"].(map[string]any)

	for _, rawCase := range fixture["cases"].([]any) {
		fixtureCase := rawCase.(map[string]any)
		envelope := parseReviewReplayBundleEnvelope(fixtureCase["review_replay_bundle_envelope"].(map[string]any))
		expectedState := parseConformanceManifestReviewState(fixtureCase["expected_state"].(map[string]any))

		application := ReviewAndExecuteConformanceManifestWithReplayBundleEnvelope[string](
			manifest,
			options,
			envelope,
			func(run ConformanceCaseRun) ConformanceCaseExecution {
				key := run.Ref.Family + ":" + run.Ref.Role + ":" + run.Ref.Case
				if raw, ok := executionsRaw[key]; ok {
					return parseConformanceCaseExecution(raw.(map[string]any))
				}

				return ConformanceCaseExecution{
					Outcome:  ConformanceFailed,
					Messages: []string{"missing execution"},
				}
			},
			func(ReviewedNestedExecution, int) NestedMergeExecutionCallbacks[string] {
				t.Fatal("callbacks should not run for rejected replay bundle envelopes")
				return NestedMergeExecutionCallbacks[string]{}
			},
		)

		if !reflect.DeepEqual(application.State, expectedState) || len(application.Results) != 0 {
			t.Fatalf("unexpected replay bundle envelope reviewed nested rejection application: %+v", application)
		}
	}
}

func assertExpectedPolicies(t *testing.T, policies []PolicyReference, expected []any) {
	t.Helper()

	if len(policies) != len(expected) {
		t.Fatalf("unexpected policies: %+v", policies)
	}
	for index, policy := range policies {
		expectedPolicy := expected[index].(map[string]any)
		if string(policy.Surface) != expectedPolicy["surface"].(string) || policy.Name != expectedPolicy["name"].(string) {
			t.Fatalf("unexpected policy at %d: %+v", index, policy)
		}
	}
}

func assertReviewedNestedExecutionResults(t *testing.T, results []ReviewedNestedExecutionResult[string], expected []any) {
	t.Helper()

	if len(results) != len(expected) {
		t.Fatalf("unexpected reviewed nested execution results: %+v", results)
	}

	for index, run := range results {
		expectedRun := expected[index].(map[string]any)
		if run.Execution.Family != expectedRun["execution_family"].(string) {
			t.Fatalf("unexpected execution family at %d: %+v", index, run)
		}

		expectedResult := expectedRun["result"].(map[string]any)
		if run.Result.OK != expectedResult["ok"].(bool) {
			t.Fatalf("unexpected result ok at %d: %+v", index, run)
		}

		expectedOutput, hasOutput := expectedResult["output"].(string)
		switch {
		case hasOutput && (run.Result.Output == nil || *run.Result.Output != expectedOutput):
			t.Fatalf("unexpected result output at %d: %+v", index, run)
		case !hasOutput && run.Result.Output != nil:
			t.Fatalf("unexpected result output at %d: %+v", index, run)
		}

		if len(run.Result.Diagnostics) != len(expectedResult["diagnostics"].([]any)) {
			t.Fatalf("unexpected result diagnostics at %d: %+v", index, run)
		}
		if len(run.Result.Policies) != len(expectedResult["policies"].([]any)) {
			t.Fatalf("unexpected result policies at %d: %+v", index, run)
		}
	}
}

func reviewedNestedExecutionCallbacksForFixture(t *testing.T, expected []any) func(ReviewedNestedExecution, int) NestedMergeExecutionCallbacks[string] {
	t.Helper()

	return func(execution ReviewedNestedExecution, index int) NestedMergeExecutionCallbacks[string] {
		return NestedMergeExecutionCallbacks[string]{
			MergeParent: func() MergeResult[string] {
				output := execution.Family + "-merged-parent"
				return MergeResult[string]{OK: true, Diagnostics: []Diagnostic{}, Output: &output, Policies: []PolicyReference{}}
			},
			DiscoverOperations: func(string) NestedMergeDiscoveryResult {
				operations := make([]DelegatedChildOperation, 0, len(execution.ReviewState.AcceptedGroups))
				for _, group := range execution.ReviewState.AcceptedGroups {
					switch execution.Family {
					case "markdown":
						operations = append(operations, DelegatedChildOperation{
							OperationID:       group.ChildOperationID,
							ParentOperationID: group.ParentOperationID,
							RequestedStrategy: "delegate_child_surface",
							LanguageChain:     []string{"markdown", "typescript"},
							Surface: DiscoveredSurface{
								SurfaceKind:            "fenced_code_block",
								EffectiveLanguage:      "typescript",
								Address:                group.DelegatedRuntimeSurfacePath,
								Owner:                  SurfaceOwnerRef{Kind: SurfaceOwnerOwnedRegion, Address: "/code_fence/0"},
								ReconstructionStrategy: "portable_write",
								Metadata:               map[string]any{"family": "typescript"},
							},
						})
					default:
						operations = append(operations, DelegatedChildOperation{
							OperationID:       group.ChildOperationID,
							ParentOperationID: group.ParentOperationID,
							RequestedStrategy: "delegate_child_surface",
							LanguageChain:     []string{"ruby", "ruby"},
							Surface: DiscoveredSurface{
								SurfaceKind:            "yard_example",
								EffectiveLanguage:      "ruby",
								Address:                group.DelegatedRuntimeSurfacePath,
								Owner:                  SurfaceOwnerRef{Kind: SurfaceOwnerOwnedRegion, Address: "/yard_example/1"},
								ReconstructionStrategy: "portable_write",
								Metadata:               map[string]any{"family": "ruby"},
							},
						})
					}
				}
				return NestedMergeDiscoveryResult{OK: true, Diagnostics: []Diagnostic{}, Operations: operations}
			},
			ApplyResolvedOutputs: func(_ string, _ []DelegatedChildOperation, _ DelegatedChildApplyPlan, appliedChildren []AppliedDelegatedChildOutput) MergeResult[string] {
				if !reflect.DeepEqual(appliedChildren, execution.AppliedChildren) {
					t.Fatalf("unexpected applied children: %+v", appliedChildren)
				}
				expectedResult := expected[index].(map[string]any)["result"].(map[string]any)
				output := expectedResult["output"].(string)
				return MergeResult[string]{OK: true, Diagnostics: []Diagnostic{}, Output: &output, Policies: []PolicyReference{}}
			},
		}
	}
}

func parseConformanceCaseRun(t *testing.T, raw map[string]any) ConformanceCaseRun {
	t.Helper()

	rawRef := raw["ref"].(map[string]any)
	run := ConformanceCaseRun{
		Ref: ConformanceCaseRef{
			Family: rawRef["family"].(string),
			Role:   rawRef["role"].(string),
			Case:   rawRef["case"].(string),
		},
		Requirements:  parseConformanceCaseRequirements(raw["requirements"].(map[string]any)),
		FamilyProfile: parseFamilyFeatureProfile(raw["family_profile"].(map[string]any)),
	}
	if rawFeatureProfile, ok := raw["feature_profile"]; ok {
		featureProfile := rawFeatureProfile.(map[string]any)
		run.FeatureProfile = &ConformanceFeatureProfileView{
			Backend:           featureProfile["backend"].(string),
			SupportsDialects:  featureProfile["supports_dialects"].(bool),
			SupportedPolicies: parsePolicyReferences(featureProfile["supported_policies"].([]any)),
		}
	}

	return run
}

func parseConformanceSuiteDefinition(raw map[string]any) ConformanceSuiteDefinition {
	return decodeFixtureValueUntyped[ConformanceSuiteDefinition](raw)
}

func parseConformanceSuiteSubject(raw map[string]any) ConformanceSuiteSubject {
	subject := ConformanceSuiteSubject{
		Grammar: raw["grammar"].(string),
	}
	if variant, ok := raw["variant"]; ok {
		subject.Variant = variant.(string)
	}
	return subject
}

func parseConformanceSuiteSelector(raw map[string]any) ConformanceSuiteSelector {
	return ConformanceSuiteSelector{
		Kind:    raw["kind"].(string),
		Subject: parseConformanceSuiteSubject(raw["subject"].(map[string]any)),
	}
}

func parseConformanceFamilyPlanContext(raw map[string]any) ConformanceFamilyPlanContext {
	context := ConformanceFamilyPlanContext{
		FamilyProfile: parseFamilyFeatureProfile(raw["family_profile"].(map[string]any)),
	}

	if rawFeatureProfile, ok := raw["feature_profile"]; ok {
		featureProfile := rawFeatureProfile.(map[string]any)
		context.FeatureProfile = &ConformanceFeatureProfileView{
			Backend:           featureProfile["backend"].(string),
			SupportsDialects:  featureProfile["supports_dialects"].(bool),
			SupportedPolicies: make([]PolicyReference, 0, len(featureProfile["supported_policies"].([]any))),
		}
		for _, item := range featureProfile["supported_policies"].([]any) {
			policy := item.(map[string]any)
			context.FeatureProfile.SupportedPolicies = append(context.FeatureProfile.SupportedPolicies, PolicyReference{
				Surface: PolicySurface(policy["surface"].(string)),
				Name:    policy["name"].(string),
			})
		}
	}

	return context
}

func parseNamedConformanceSuitePlan(t *testing.T, raw map[string]any) NamedConformanceSuitePlan {
	return decodeFixtureValue[NamedConformanceSuitePlan](t, raw)
}

func parseNamedConformanceSuiteResults(raw map[string]any) NamedConformanceSuiteResults {
	return decodeFixtureValueUntyped[NamedConformanceSuiteResults](raw)
}

func parseNamedConformanceSuiteReport(raw map[string]any) NamedConformanceSuiteReport {
	return decodeFixtureValueUntyped[NamedConformanceSuiteReport](raw)
}

func parseConformanceSuiteReport(raw map[string]any) ConformanceSuiteReport {
	resultsRaw := raw["results"].([]any)
	results := make([]ConformanceCaseResult, 0, len(resultsRaw))
	for _, item := range resultsRaw {
		results = append(results, parseConformanceCaseResult(item.(map[string]any)))
	}

	summaryRaw := raw["summary"].(map[string]any)
	return ConformanceSuiteReport{
		Results: results,
		Summary: ConformanceSuiteSummary{
			Total:   int(summaryRaw["total"].(float64)),
			Passed:  int(summaryRaw["passed"].(float64)),
			Failed:  int(summaryRaw["failed"].(float64)),
			Skipped: int(summaryRaw["skipped"].(float64)),
		},
	}
}

func parseNamedConformanceSuiteReportEnvelope(raw map[string]any) NamedConformanceSuiteReportEnvelope {
	entriesRaw := raw["entries"].([]any)
	entries := make([]NamedConformanceSuiteReport, 0, len(entriesRaw))
	for _, item := range entriesRaw {
		entries = append(entries, parseNamedConformanceSuiteReport(item.(map[string]any)))
	}

	summaryRaw := raw["summary"].(map[string]any)
	return NamedConformanceSuiteReportEnvelope{
		Entries: entries,
		Summary: ConformanceSuiteSummary{
			Total:   int(summaryRaw["total"].(float64)),
			Passed:  int(summaryRaw["passed"].(float64)),
			Failed:  int(summaryRaw["failed"].(float64)),
			Skipped: int(summaryRaw["skipped"].(float64)),
		},
	}
}

func parseDiagnostic(raw map[string]any) Diagnostic {
	diagnostic := Diagnostic{
		Severity: DiagnosticSeverity(raw["severity"].(string)),
		Category: DiagnosticCategory(raw["category"].(string)),
		Message:  raw["message"].(string),
	}
	if path, ok := raw["path"]; ok {
		diagnostic.Path = path.(string)
	}
	if rawReview, ok := raw["review"]; ok {
		review := ReviewDiagnosticDetail{}
		reviewRaw := rawReview.(map[string]any)
		if requestID, ok := reviewRaw["request_id"]; ok {
			review.RequestID = requestID.(string)
		}
		if action, ok := reviewRaw["action"]; ok {
			review.Action = ReviewDecisionAction(action.(string))
		}
		if reason, ok := reviewRaw["reason"]; ok {
			review.Reason = ReviewDiagnosticReason(reason.(string))
		}
		if payloadKind, ok := reviewRaw["payload_kind"]; ok {
			review.PayloadKind = payloadKind.(string)
		}
		if expectedFamily, ok := reviewRaw["expected_family"]; ok {
			review.ExpectedFamily = expectedFamily.(string)
		}
		if providedFamily, ok := reviewRaw["provided_family"]; ok {
			review.ProvidedFamily = providedFamily.(string)
		}
		diagnostic.Review = &review
	} else if requestID, ok := raw["request_id"]; ok {
		review := ReviewDiagnosticDetail{RequestID: requestID.(string)}
		if action, ok := raw["action"]; ok {
			review.Action = ReviewDecisionAction(action.(string))
		}
		if reason, ok := raw["reason"]; ok {
			review.Reason = ReviewDiagnosticReason(reason.(string))
		}
		if payloadKind, ok := raw["payload_kind"]; ok {
			review.PayloadKind = payloadKind.(string)
		}
		if expectedFamily, ok := raw["expected_family"]; ok {
			review.ExpectedFamily = expectedFamily.(string)
		}
		if providedFamily, ok := raw["provided_family"]; ok {
			review.ProvidedFamily = providedFamily.(string)
		}
		diagnostic.Review = &review
	}

	return diagnostic
}

func parseConformanceManifestPlanningOptions(raw map[string]any) ConformanceManifestPlanningOptions {
	options := ConformanceManifestPlanningOptions{}

	if rawContexts, ok := raw["contexts"]; ok {
		options.Contexts = make(map[string]ConformanceFamilyPlanContext, len(rawContexts.(map[string]any)))
		for family, value := range rawContexts.(map[string]any) {
			options.Contexts[family] = parseConformanceFamilyPlanContext(value.(map[string]any))
		}
	}

	if rawFamilyProfiles, ok := raw["family_profiles"]; ok {
		options.FamilyProfiles = make(map[string]FamilyFeatureProfile, len(rawFamilyProfiles.(map[string]any)))
		for family, value := range rawFamilyProfiles.(map[string]any) {
			options.FamilyProfiles[family] = parseFamilyFeatureProfile(value.(map[string]any))
		}
	}

	if rawRequireExplicitContexts, ok := raw["require_explicit_contexts"]; ok {
		options.RequireExplicitContexts = rawRequireExplicitContexts.(bool)
	}

	return options
}

func parseConformanceManifestReviewOptions(raw map[string]any) ConformanceManifestReviewOptions {
	options := ConformanceManifestReviewOptions{
		Contexts:       map[string]ConformanceFamilyPlanContext{},
		FamilyProfiles: map[string]FamilyFeatureProfile{},
	}

	if rawContexts, ok := raw["contexts"]; ok {
		for family, value := range rawContexts.(map[string]any) {
			options.Contexts[family] = parseConformanceFamilyPlanContext(value.(map[string]any))
		}
	}
	if len(options.Contexts) == 0 {
		options.Contexts = nil
	}

	if rawFamilyProfiles, ok := raw["family_profiles"]; ok {
		for family, value := range rawFamilyProfiles.(map[string]any) {
			options.FamilyProfiles[family] = parseFamilyFeatureProfile(value.(map[string]any))
		}
	}
	if len(options.FamilyProfiles) == 0 {
		options.FamilyProfiles = nil
	}

	if rawRequireExplicitContexts, ok := raw["require_explicit_contexts"]; ok {
		options.RequireExplicitContexts = rawRequireExplicitContexts.(bool)
	}
	if rawInteractive, ok := raw["interactive"]; ok {
		options.Interactive = rawInteractive.(bool)
	}
	if rawReviewDecisions, ok := raw["review_decisions"]; ok {
		options.ReviewDecisions = make([]ReviewDecision, 0, len(rawReviewDecisions.([]any)))
		for _, item := range rawReviewDecisions.([]any) {
			options.ReviewDecisions = append(options.ReviewDecisions, parseReviewDecision(item.(map[string]any)))
		}
	}
	if rawReviewReplayContext, ok := raw["review_replay_context"]; ok {
		context := parseReviewReplayContext(rawReviewReplayContext.(map[string]any))
		options.ReviewReplayContext = &context
	}
	if rawReviewReplayBundle, ok := raw["review_replay_bundle"]; ok {
		bundle := parseReviewReplayBundle(rawReviewReplayBundle.(map[string]any))
		options.ReviewReplayBundle = &bundle
	}

	return options
}

func parseConformanceManifestReport(raw map[string]any) ConformanceManifestReport {
	report := parseNamedConformanceSuiteReportEnvelope(raw["report"].(map[string]any))
	diagnosticsRaw := raw["diagnostics"].([]any)
	diagnostics := make([]Diagnostic, 0, len(diagnosticsRaw))
	for _, item := range diagnosticsRaw {
		diagnostics = append(diagnostics, parseDiagnostic(item.(map[string]any)))
	}

	return ConformanceManifestReport{
		Report:      report,
		Diagnostics: diagnostics,
	}
}

func parseReviewRequest(raw map[string]any) ReviewRequest {
	request := ReviewRequest{
		ID:           raw["id"].(string),
		Kind:         ReviewRequestKind(raw["kind"].(string)),
		Family:       raw["family"].(string),
		Message:      raw["message"].(string),
		Blocking:     raw["blocking"].(bool),
		ActionOffers: []ReviewActionOffer{},
	}
	if rawProposedContext, ok := raw["proposed_context"]; ok {
		context := parseConformanceFamilyPlanContext(rawProposedContext.(map[string]any))
		request.ProposedContext = &context
	}
	if rawDelegatedGroup, ok := raw["delegated_group"]; ok {
		group := parseProjectedChildReviewGroup(rawDelegatedGroup.(map[string]any))
		request.DelegatedGroup = &group
	}
	if rawActionOffers, ok := raw["action_offers"]; ok {
		request.ActionOffers = make([]ReviewActionOffer, 0, len(rawActionOffers.([]any)))
		for _, item := range rawActionOffers.([]any) {
			offer := item.(map[string]any)
			request.ActionOffers = append(request.ActionOffers, ReviewActionOffer{
				Action:          ReviewDecisionAction(offer["action"].(string)),
				RequiresContext: offer["requires_context"].(bool),
				PayloadKind: func() string {
					if payloadKind, ok := offer["payload_kind"]; ok {
						return payloadKind.(string)
					}
					return ""
				}(),
			})
		}
	}
	if rawDefaultAction, ok := raw["default_action"]; ok {
		request.DefaultAction = ReviewDecisionAction(rawDefaultAction.(string))
	}

	return request
}

func parseReviewDecision(raw map[string]any) ReviewDecision {
	decision := ReviewDecision{
		RequestID: raw["request_id"].(string),
		Action:    ReviewDecisionAction(raw["action"].(string)),
	}
	if rawContext, ok := raw["context"]; ok {
		context := parseConformanceFamilyPlanContext(rawContext.(map[string]any))
		decision.Context = &context
	}

	return decision
}

func parseProjectedChildReviewGroup(raw map[string]any) ProjectedChildReviewGroup {
	return ProjectedChildReviewGroup{
		DelegatedApplyGroup:         raw["delegated_apply_group"].(string),
		ParentOperationID:           raw["parent_operation_id"].(string),
		ChildOperationID:            raw["child_operation_id"].(string),
		DelegatedRuntimeSurfacePath: raw["delegated_runtime_surface_path"].(string),
		CaseIDs:                     parseStringSlice(raw["case_ids"].([]any)),
		DelegatedCaseIDs:            parseStringSlice(raw["delegated_case_ids"].([]any)),
	}
}

func parseDelegatedChildGroupReviewState(raw map[string]any) DelegatedChildGroupReviewState {
	requestsRaw := raw["requests"].([]any)
	requests := make([]ReviewRequest, 0, len(requestsRaw))
	for _, item := range requestsRaw {
		requests = append(requests, parseReviewRequest(item.(map[string]any)))
	}

	groupsRaw := raw["accepted_groups"].([]any)
	acceptedGroups := make([]ProjectedChildReviewGroup, 0, len(groupsRaw))
	for _, item := range groupsRaw {
		acceptedGroups = append(acceptedGroups, parseProjectedChildReviewGroup(item.(map[string]any)))
	}

	decisionsRaw := raw["applied_decisions"].([]any)
	appliedDecisions := make([]ReviewDecision, 0, len(decisionsRaw))
	for _, item := range decisionsRaw {
		appliedDecisions = append(appliedDecisions, parseReviewDecision(item.(map[string]any)))
	}

	diagnosticsRaw := raw["diagnostics"].([]any)
	diagnostics := make([]Diagnostic, 0, len(diagnosticsRaw))
	for _, item := range diagnosticsRaw {
		diagnostics = append(diagnostics, parseDiagnostic(item.(map[string]any)))
	}

	return DelegatedChildGroupReviewState{
		Requests:         requests,
		AcceptedGroups:   acceptedGroups,
		AppliedDecisions: appliedDecisions,
		Diagnostics:      diagnostics,
	}
}

func parseStringSlice(raw []any) []string {
	values := make([]string, 0, len(raw))
	for _, item := range raw {
		values = append(values, item.(string))
	}
	return values
}

func parseReviewReplayBundle(raw map[string]any) ReviewReplayBundle {
	bundle := ReviewReplayBundle{
		ReplayContext: parseReviewReplayContext(raw["replay_context"].(map[string]any)),
		Decisions: func() []ReviewDecision {
			decisionsRaw := raw["decisions"].([]any)
			decisions := make([]ReviewDecision, 0, len(decisionsRaw))
			for _, item := range decisionsRaw {
				decisions = append(decisions, parseReviewDecision(item.(map[string]any)))
			}
			return decisions
		}(),
	}
	if rawReviewedNestedExecutions, ok := raw["reviewed_nested_executions"]; ok {
		bundle.ReviewedNestedExecutions = make([]ReviewedNestedExecution, 0, len(rawReviewedNestedExecutions.([]any)))
		for _, item := range rawReviewedNestedExecutions.([]any) {
			bundle.ReviewedNestedExecutions = append(bundle.ReviewedNestedExecutions, parseReviewedNestedExecution(item.(map[string]any)))
		}
	}
	return bundle
}

func parseConformanceManifestReviewStateEnvelope(raw map[string]any) ConformanceManifestReviewStateEnvelope {
	return ConformanceManifestReviewStateEnvelope{
		Kind:    raw["kind"].(string),
		Version: int(raw["version"].(float64)),
		State:   parseConformanceManifestReviewState(raw["state"].(map[string]any)),
	}
}

func parseReviewReplayBundleEnvelope(raw map[string]any) ReviewReplayBundleEnvelope {
	return ReviewReplayBundleEnvelope{
		Kind:         raw["kind"].(string),
		Version:      int(raw["version"].(float64)),
		ReplayBundle: parseReviewReplayBundle(raw["replay_bundle"].(map[string]any)),
	}
}

func parseReviewedNestedExecution(raw map[string]any) ReviewedNestedExecution {
	appliedChildrenRaw := raw["applied_children"].([]any)
	appliedChildren := make([]AppliedDelegatedChildOutput, 0, len(appliedChildrenRaw))
	for _, item := range appliedChildrenRaw {
		entry := item.(map[string]any)
		appliedChildren = append(appliedChildren, AppliedDelegatedChildOutput{
			OperationID: entry["operation_id"].(string),
			Output:      entry["output"].(string),
		})
	}

	return ReviewedNestedExecution{
		Family:          raw["family"].(string),
		ReviewState:     parseDelegatedChildGroupReviewState(raw["review_state"].(map[string]any)),
		AppliedChildren: appliedChildren,
	}
}

func parseReviewedNestedExecutionEnvelope(raw map[string]any) ReviewedNestedExecutionEnvelope {
	return ReviewedNestedExecutionEnvelope{
		Kind:      raw["kind"].(string),
		Version:   int(raw["version"].(float64)),
		Execution: parseReviewedNestedExecution(raw["execution"].(map[string]any)),
	}
}

func parseReviewTransportImportError(raw map[string]any) ReviewTransportImportError {
	return ReviewTransportImportError{
		Category: ReviewTransportImportErrorCategory(raw["category"].(string)),
		Message:  raw["message"].(string),
	}
}

func parseReviewHostHints(raw map[string]any) ReviewHostHints {
	return ReviewHostHints{
		Interactive:             raw["interactive"].(bool),
		RequireExplicitContexts: raw["require_explicit_contexts"].(bool),
	}
}

func parseReviewReplayContext(raw map[string]any) ReviewReplayContext {
	context := ReviewReplayContext{
		Surface:                 raw["surface"].(string),
		Families:                []string{},
		RequireExplicitContexts: raw["require_explicit_contexts"].(bool),
	}
	for _, family := range raw["families"].([]any) {
		context.Families = append(context.Families, family.(string))
	}

	return context
}

func parseConformanceManifestReviewState(raw map[string]any) ConformanceManifestReviewState {
	state := ConformanceManifestReviewState{
		Report:           parseNamedConformanceSuiteReportEnvelope(raw["report"].(map[string]any)),
		Diagnostics:      []Diagnostic{},
		Requests:         []ReviewRequest{},
		AppliedDecisions: []ReviewDecision{},
		HostHints:        parseReviewHostHints(raw["host_hints"].(map[string]any)),
		ReplayContext:    parseReviewReplayContext(raw["replay_context"].(map[string]any)),
	}
	for _, item := range raw["diagnostics"].([]any) {
		state.Diagnostics = append(state.Diagnostics, parseDiagnostic(item.(map[string]any)))
	}
	for _, item := range raw["requests"].([]any) {
		state.Requests = append(state.Requests, parseReviewRequest(item.(map[string]any)))
	}
	for _, item := range raw["applied_decisions"].([]any) {
		state.AppliedDecisions = append(state.AppliedDecisions, parseReviewDecision(item.(map[string]any)))
	}
	if rawReviewedNestedExecutions, ok := raw["reviewed_nested_executions"]; ok {
		for _, item := range rawReviewedNestedExecutions.([]any) {
			state.ReviewedNestedExecutions = append(state.ReviewedNestedExecutions, parseReviewedNestedExecution(item.(map[string]any)))
		}
	}

	return state
}

func parseConformanceCaseRequirements(raw map[string]any) ConformanceCaseRequirements {
	requirements := ConformanceCaseRequirements{}
	if backend, ok := raw["backend"]; ok {
		requirements.Backend = backend.(string)
	}
	if dialect, ok := raw["dialect"]; ok {
		requirements.Dialect = dialect.(string)
	}
	if rawPolicies, ok := raw["policies"]; ok {
		requirements.Policies = parsePolicyReferences(rawPolicies.([]any))
	}

	return requirements
}

func parseFamilyFeatureProfile(raw map[string]any) FamilyFeatureProfile {
	profile := FamilyFeatureProfile{
		Family:            raw["family"].(string),
		SupportedDialects: []string{},
		SupportedPolicies: parsePolicyReferences(raw["supported_policies"].([]any)),
	}
	for _, dialect := range raw["supported_dialects"].([]any) {
		profile.SupportedDialects = append(profile.SupportedDialects, dialect.(string))
	}

	return profile
}

func parseFeatureProfilePointer(raw any) *ConformanceFeatureProfileView {
	if raw == nil {
		return nil
	}

	featureProfile := raw.(map[string]any)
	return &ConformanceFeatureProfileView{
		Backend:           featureProfile["backend"].(string),
		SupportsDialects:  featureProfile["supports_dialects"].(bool),
		SupportedPolicies: parsePolicyReferences(featureProfile["supported_policies"].([]any)),
	}
}

func parsePolicyReferences(raw []any) []PolicyReference {
	policies := make([]PolicyReference, 0, len(raw))
	for _, item := range raw {
		policy := item.(map[string]any)
		policies = append(policies, PolicyReference{
			Surface: PolicySurface(policy["surface"].(string)),
			Name:    policy["name"].(string),
		})
	}

	return policies
}

func parseConformanceCaseExecution(raw map[string]any) ConformanceCaseExecution {
	messages := raw["messages"].([]any)
	normalizedMessages := make([]string, 0, len(messages))
	for _, message := range messages {
		normalizedMessages = append(normalizedMessages, message.(string))
	}

	return ConformanceCaseExecution{
		Outcome:  ConformanceOutcome(raw["outcome"].(string)),
		Messages: normalizedMessages,
	}
}

func parseConformanceCaseResult(raw map[string]any) ConformanceCaseResult {
	rawRef := raw["ref"].(map[string]any)
	messages := raw["messages"].([]any)
	normalizedMessages := make([]string, 0, len(messages))
	for _, message := range messages {
		normalizedMessages = append(normalizedMessages, message.(string))
	}

	return ConformanceCaseResult{
		Ref: ConformanceCaseRef{
			Family: rawRef["family"].(string),
			Role:   rawRef["role"].(string),
			Case:   rawRef["case"].(string),
		},
		Outcome:  ConformanceOutcome(raw["outcome"].(string)),
		Messages: normalizedMessages,
	}
}

func parseConformanceSuitePlan(t *testing.T, raw map[string]any) ConformanceSuitePlan {
	t.Helper()

	entriesRaw := raw["entries"].([]any)
	entries := make([]ConformanceSuitePlanEntry, 0, len(entriesRaw))
	for _, item := range entriesRaw {
		entry := item.(map[string]any)
		refRaw := entry["ref"].(map[string]any)
		pathRaw := entry["path"].([]any)
		path := make([]string, 0, len(pathRaw))
		for _, segment := range pathRaw {
			path = append(path, segment.(string))
		}

		entries = append(entries, ConformanceSuitePlanEntry{
			Ref: ConformanceCaseRef{
				Family: refRaw["family"].(string),
				Role:   refRaw["role"].(string),
				Case:   refRaw["case"].(string),
			},
			Path: path,
			Run:  parseConformanceCaseRun(t, entry["run"].(map[string]any)),
		})
	}

	missingRaw := raw["missing_roles"].([]any)
	missingRoles := make([]string, 0, len(missingRaw))
	for _, item := range missingRaw {
		missingRoles = append(missingRoles, item.(string))
	}

	return ConformanceSuitePlan{
		Family:       raw["family"].(string),
		Entries:      entries,
		MissingRoles: missingRoles,
	}
}
