package astmerge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
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

func TestSharedFixtureConformanceSuiteDefinitions(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "suite_definitions"))
	manifest := readManifest(t)
	suiteName := fixture["suite_name"].(string)
	expectedRaw := fixture["expected"].(map[string]any)
	expected := ConformanceSuiteDefinition{
		Family: expectedRaw["family"].(string),
		Roles:  make([]string, 0, len(expectedRaw["roles"].([]any))),
	}
	for _, role := range expectedRaw["roles"].([]any) {
		expected.Roles = append(expected.Roles, role.(string))
	}

	definition := ConformanceSuiteDefinitionByName(manifest, suiteName)
	if definition == nil || !reflect.DeepEqual(*definition, expected) {
		t.Fatalf("unexpected suite definition: %+v", definition)
	}

	planned := PlanNamedConformanceSuite(
		manifest,
		suiteName,
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
		expected.Family,
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
	suiteName := fixture["suite_name"].(string)
	executionsRaw := fixture["executions"].(map[string]any)
	expected := fixture["expected_report"].(map[string]any)

	report := ReportNamedConformanceSuite(
		manifest,
		suiteName,
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
	suiteName := fixture["suite_name"].(string)
	executionsRaw := fixture["executions"].(map[string]any)
	expectedResults := fixture["expected_results"].([]any)

	results := RunNamedConformanceSuite(
		manifest,
		suiteName,
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

	expectedRaw := fixture["suite_names"].([]any)
	expected := make([]string, 0, len(expectedRaw))
	for _, name := range expectedRaw {
		expected = append(expected, name.(string))
	}

	if names := ConformanceSuiteNames(manifest); !reflect.DeepEqual(names, expected) {
		t.Fatalf("unexpected suite names: %+v", names)
	}
}

func TestSharedFixtureNamedConformanceSuiteEntry(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "named_suite_entry"))
	manifest := readManifest(t)
	suiteName := fixture["suite_name"].(string)
	executionsRaw := fixture["executions"].(map[string]any)
	expectedRaw := fixture["expected_entry"].(map[string]any)

	entry := ReportNamedConformanceSuiteEntry(
		manifest,
		suiteName,
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
	if entry.Suite != expectedRaw["suite"].(string) {
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
	suiteName := fixture["suite_name"].(string)

	context := parseConformanceFamilyPlanContext(fixture["context"].(map[string]any))
	entry := PlanNamedConformanceSuiteEntry(manifest, suiteName, context)
	if entry == nil {
		t.Fatalf("expected named suite plan entry")
	}

	expected := parseNamedConformanceSuitePlan(t, fixture["expected_entry"].(map[string]any))
	if !reflect.DeepEqual(*entry, expected) {
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

	if plans := PlanNamedConformanceSuites(manifest, contexts); !reflect.DeepEqual(plans, expected) {
		t.Fatalf("unexpected named suite plans: %+v", plans)
	}
}

func TestSharedFixtureNamedConformanceSuiteResults(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "named_suite_results"))
	manifest := readManifest(t)
	suiteName := fixture["suite_name"].(string)
	executionsRaw := fixture["executions"].(map[string]any)

	entry := RunNamedConformanceSuiteEntry(
		manifest,
		suiteName,
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
	if !reflect.DeepEqual(*entry, expected) {
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
	return NamedConformanceSuitePlan{
		Suite: raw["suite"].(string),
		Plan:  parseConformanceSuitePlan(t, raw["plan"].(map[string]any)),
	}
}

func parseNamedConformanceSuiteResults(raw map[string]any) NamedConformanceSuiteResults {
	resultsRaw := raw["results"].([]any)
	results := make([]ConformanceCaseResult, 0, len(resultsRaw))
	for _, item := range resultsRaw {
		results = append(results, parseConformanceCaseResult(item.(map[string]any)))
	}

	return NamedConformanceSuiteResults{
		Suite:   raw["suite"].(string),
		Results: results,
	}
}

func parseNamedConformanceSuiteReport(raw map[string]any) NamedConformanceSuiteReport {
	return NamedConformanceSuiteReport{
		Suite:  raw["suite"].(string),
		Report: parseConformanceSuiteReport(raw["report"].(map[string]any)),
	}
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

func parseConformanceCaseRequirements(raw map[string]any) ConformanceCaseRequirements {
	requirements := ConformanceCaseRequirements{}
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
