package astmerge

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// parity anchors:
// diagnosticsFixturePath(t, "mini_template_tree_family_merge_callback")
// diagnosticsFixturePath(t, "mini_template_tree_multi_family_merge_callback")
// diagnosticsFixturePath(t, "mini_template_tree_multi_family_run_report")
// diagnosticsFixturePath(t, "mini_template_tree_directory_run_report")
// diagnosticsFixturePath(t, "mini_template_tree_directory_apply_convergence")
// diagnosticsFixturePath(t, "mini_template_tree_directory_apply_report")
// diagnosticsFixturePath(t, "mini_template_tree_directory_plan_report")
// diagnosticsFixturePath(t, "mini_template_tree_directory_runner_report")

func readRelativeFileTree(t *testing.T, root string) map[string]string {
	t.Helper()

	files := map[string]string{}
	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relativePath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(relativePath)] = string(source)
		return nil
	}); err != nil {
		t.Fatalf("walk fixture tree: %v", err)
	}

	return files
}

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

func TestSharedFixtureGenericMergeIR(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-790-generic-merge-ir", "generic-merge-ir.json"))
	mergeIR := decodeFixtureValue[MergeIR](t, fixture["merge_ir"])
	expected := fixture["expected"].(map[string]any)

	changeKinds := make([]string, 0, len(mergeIR.Changes))
	for _, change := range mergeIR.Changes {
		changeKinds = append(changeKinds, change.Kind)
	}

	if mergeIR.Version != expected["version"].(string) ||
		len(mergeIR.NodeClasses) != int(expected["node_class_count"].(float64)) ||
		len(mergeIR.OrderedNodes) != int(expected["ordered_node_count"].(float64)) ||
		!reflect.DeepEqual(changeKinds, decodeFixtureValue[[]string](t, expected["change_kinds"])) ||
		mergeIR.NodeClasses[0].NodeIDs["left"] != "left-import-fmt" ||
		mergeIR.Changes[1].ClassID == nil ||
		*mergeIR.Changes[1].ClassID != "class-import-strings" {
		t.Fatalf("unexpected generic merge IR: %+v", mergeIR)
	}
}

func TestSharedFixturePairwiseMatchings(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-791-pairwise-matchings", "pairwise-matchings.json"))
	matchings := decodeFixtureValue[[]PairwiseMatching](t, fixture["pairwise_matchings"])
	expected := fixture["expected"].(map[string]any)

	matchingIDs := make([]string, 0, len(matchings))
	totalMatchCount := 0
	for _, matching := range matchings {
		matchingIDs = append(matchingIDs, matching.MatchingID)
		totalMatchCount += len(matching.Matches)
	}

	if !reflect.DeepEqual(matchingIDs, decodeFixtureValue[[]string](t, expected["matching_ids"])) ||
		totalMatchCount != int(expected["total_match_count"].(float64)) ||
		matchings[0].UnmatchedTo[0] != "left-import-os" ||
		matchings[1].UnmatchedFrom[0] != "base-decl-greet" ||
		matchings[2].Matches[1].Diagnostics[0] != "sibling position changed" {
		t.Fatalf("unexpected pairwise matchings: %+v", matchings)
	}
}

func TestSharedFixtureClassMapping(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-792-class-mapping", "class-mapping.json"))
	report := decodeFixtureValue[ClassMappingReport](t, fixture["class_mapping"])
	expected := fixture["expected"].(map[string]any)

	categories := make([]string, 0, len(report.Diagnostics))
	classIDs := make([]string, 0, len(report.Diagnostics))
	for _, diagnostic := range report.Diagnostics {
		categories = append(categories, diagnostic.Category)
		classIDs = append(classIDs, diagnostic.ClassID)
	}

	if len(report.NodeClasses) != int(expected["class_count"].(float64)) ||
		!reflect.DeepEqual(categories, decodeFixtureValue[[]string](t, expected["diagnostic_categories"])) ||
		!reflect.DeepEqual(classIDs, decodeFixtureValue[[]string](t, expected["conflicted_class_ids"])) ||
		report.NodeClasses[2].NodeIDs["right"] != "" ||
		report.Diagnostics[1].Category != "delete_edit_disagreement" {
		t.Fatalf("unexpected class mapping report: %+v", report)
	}
}

func TestSharedFixturePCSChangeSetGeneration(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-793-pcs-change-set-generation", "pcs-change-set-generation.json"))
	pcs := decodeFixtureValue[PCS](t, fixture["pcs"])
	changeSets := decodeFixtureValue[[]ChangeSet](t, fixture["change_sets"])
	expected := fixture["expected"].(map[string]any)

	changeKinds := make([]string, 0)
	diagnosticCount := 0
	for _, changeSet := range changeSets {
		diagnosticCount += len(changeSet.Diagnostics)
		for _, change := range changeSet.Changes {
			changeKinds = append(changeKinds, change.Kind)
		}
	}

	if len(pcs.Constraints) != int(expected["pcs_constraint_count"].(float64)) ||
		len(changeSets) != int(expected["change_set_count"].(float64)) ||
		!reflect.DeepEqual(changeKinds, decodeFixtureValue[[]string](t, expected["change_kinds"])) ||
		diagnosticCount != int(expected["diagnostic_count"].(float64)) ||
		pcs.Constraints[2].PredecessorClassID == nil ||
		*pcs.Constraints[2].PredecessorClassID != "class-import-strings" ||
		changeSets[1].Changes[1].Kind != "delete" {
		t.Fatalf("unexpected PCS/change-set generation: pcs=%+v changeSets=%+v", pcs, changeSets)
	}
}

func TestSharedFixtureRawMergeChangeSetUnion(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-794-raw-merge-change-set-union", "raw-merge-change-set-union.json"))
	rawMerge := decodeFixtureValue[RawMerge](t, fixture["raw_merge"])
	expected := fixture["expected"].(map[string]any)

	sides := []string{}
	seenSides := map[string]bool{}
	classChangeCount := map[string]int{}
	for _, change := range rawMerge.Changes {
		if !seenSides[change.Side] {
			seenSides[change.Side] = true
			sides = append(sides, change.Side)
		}
		classChangeCount[change.ClassID]++
	}

	if len(rawMerge.Changes) != int(expected["raw_change_count"].(float64)) ||
		len(rawMerge.InputChangeSetIDs) != int(expected["input_change_set_count"].(float64)) ||
		!reflect.DeepEqual(sides, decodeFixtureValue[[]string](t, expected["sides"])) ||
		classChangeCount["class-decl-greet"] != 2 ||
		rawMerge.Diagnostics[0] != "raw merge intentionally preserves both sides before inconsistency detection" {
		t.Fatalf("unexpected raw merge union: %+v", rawMerge)
	}
}

func TestSharedFixtureInconsistencyDetection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-795-inconsistency-detection", "inconsistency-detection.json"))
	report := decodeFixtureValue[InconsistencyReport](t, fixture["inconsistency_report"])
	expected := fixture["expected"].(map[string]any)

	categories := make([]string, 0, len(report.Inconsistencies))
	blockingCount := 0
	for _, inconsistency := range report.Inconsistencies {
		categories = append(categories, inconsistency.Category)
		if inconsistency.Severity == "error" {
			blockingCount++
		}
	}

	if len(report.Inconsistencies) != int(expected["inconsistency_count"].(float64)) ||
		!reflect.DeepEqual(categories, decodeFixtureValue[[]string](t, expected["categories"])) ||
		blockingCount != int(expected["blocking_count"].(float64)) ||
		report.Inconsistencies[1].ChangeIDs[1] != "right-delete-greet" {
		t.Fatalf("unexpected inconsistency report: %+v", report)
	}
}

func TestSharedFixtureMergeIRComparison(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-796-merge-ir-comparison", "merge-ir-comparison.json"))
	report := decodeFixtureValue[MergeIRComparisonReport](t, fixture["comparison"])
	expected := fixture["expected"].(map[string]any)

	families := make([]string, 0, len(report.Cases))
	for _, testCase := range report.Cases {
		families = append(families, testCase.Family)
	}

	if len(report.Cases) != int(expected["case_count"].(float64)) ||
		!reflect.DeepEqual(families, decodeFixtureValue[[]string](t, expected["families"])) ||
		report.Summary.MergeIRWins != int(expected["merge_ir_wins"].(float64)) ||
		report.Summary.Recommendation != expected["recommendation"].(string) ||
		report.Cases[4].MergeIRAdvantage != "defer" {
		t.Fatalf("unexpected merge IR comparison report: %+v", report)
	}
}

func TestSharedFixtureStructuralMatchingBaseline(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-797-structural-matching-baseline", "structural-matching-baseline.json"))
	report := decodeFixtureValue[StructuralMatchingReport](t, fixture["matching"])
	expected := fixture["expected"].(map[string]any)

	if report.Strategy != expected["strategy"].(string) ||
		len(report.Matches) != int(expected["match_count"].(float64)) ||
		len(report.UnmatchedFrom) != int(expected["unmatched_from_count"].(float64)) ||
		len(report.UnmatchedTo) != int(expected["unmatched_to_count"].(float64)) ||
		expected["move_detection"].(bool) ||
		report.Matches[1].FromPath != "/declarations/Greet" {
		t.Fatalf("unexpected structural matching baseline: %+v", report)
	}
}

func TestSharedFixtureSignatureMatchingCommutativeParent(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-798-signature-matching-commutative-parent", "signature-matching-commutative-parent.json"))
	parent := decodeFixtureValue[SignatureMatchingParent](t, fixture["parent"])
	report := decodeFixtureValue[SignatureMatchingReport](t, fixture["matching"])
	expected := fixture["expected"].(map[string]any)

	if parent.ChildOrder != expected["parent_policy"].(string) ||
		report.Strategy != expected["strategy"].(string) ||
		report.ParentPolicy != expected["parent_policy"].(string) ||
		!reflect.DeepEqual(report.SignatureComponents, decodeFixtureValue[[]string](t, expected["signature_components"])) ||
		len(report.Matches) != int(expected["match_count"].(float64)) ||
		len(report.UnmatchedFrom) != int(expected["unmatched_from_count"].(float64)) ||
		len(report.UnmatchedTo) != int(expected["unmatched_to_count"].(float64)) ||
		expected["order_sensitive"].(bool) ||
		report.Matches[0].Signature != expected["first_match_signature"].(string) ||
		report.Matches[0].ToPath != expected["first_match_to_path"].(string) {
		t.Fatalf("unexpected signature matching report: parent=%+v report=%+v", parent, report)
	}
}

func TestSharedFixtureSourceTextNormalizedLeafMatching(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-799-source-text-normalized-leaf-matching", "source-text-normalized-leaf-matching.json"))
	report := decodeFixtureValue[SourceTextNormalizedMatchingReport](t, fixture["matching"])
	expected := fixture["expected"].(map[string]any)

	if report.Strategy != expected["strategy"].(string) ||
		!reflect.DeepEqual(report.Normalization, decodeFixtureValue[[]string](t, expected["normalization"])) ||
		!reflect.DeepEqual(report.LeafKinds, decodeFixtureValue[[]string](t, expected["leaf_kinds"])) ||
		len(report.Matches) != int(expected["match_count"].(float64)) ||
		len(report.UnmatchedFrom) != int(expected["unmatched_from_count"].(float64)) ||
		len(report.UnmatchedTo) != int(expected["unmatched_to_count"].(float64)) ||
		report.Matches[0].NormalizedText != expected["first_match_normalized_text"].(string) ||
		report.Matches[0].Confidence < expected["minimum_confidence"].(float64) {
		t.Fatalf("unexpected source-text normalized matching report: %+v", report)
	}
}

func TestSharedFixtureMoveDetectionOptIn(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-800-move-detection-opt-in", "move-detection-opt-in.json"))
	report := decodeFixtureValue[MoveDetectionMatchingReport](t, fixture["matching"])
	expected := fixture["expected"].(map[string]any)
	moveCount := 0
	for _, match := range report.Matches {
		if match.Moved {
			moveCount++
		}
	}

	if report.Strategy != expected["strategy"].(string) ||
		report.Capability.Name != expected["capability"].(string) ||
		report.Capability.Enabled != expected["enabled"].(bool) ||
		report.Capability.DefaultEnabled != expected["default_enabled"].(bool) ||
		report.Capability.RequiresStableNodeIdentity != expected["requires_stable_node_identity"].(bool) ||
		len(report.Matches) != int(expected["match_count"].(float64)) ||
		moveCount != int(expected["move_count"].(float64)) ||
		report.Matches[0].Signature != expected["first_moved_signature"].(string) ||
		report.Matches[0].FromIndex != int(expected["first_moved_from_index"].(float64)) ||
		report.Matches[0].ToIndex != int(expected["first_moved_to_index"].(float64)) {
		t.Fatalf("unexpected move detection matching report: %+v", report)
	}
}

func TestSharedFixtureRenameAwareMatchingGated(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-801-rename-aware-matching-gated", "rename-aware-matching-gated.json"))
	report := decodeFixtureValue[RenameAwareMatchingReport](t, fixture["matching"])
	expected := fixture["expected"].(map[string]any)

	if report.Strategy != expected["strategy"].(string) ||
		report.Capability.Name != expected["capability"].(string) ||
		report.Capability.Status != expected["status"].(string) ||
		report.Capability.Enabled != expected["enabled"].(bool) ||
		report.Capability.RequiresExplicitProfile != expected["requires_explicit_profile"].(bool) ||
		report.Capability.RequiresDiagnostics != expected["requires_diagnostics"].(bool) ||
		len(report.Candidates) != int(expected["candidate_count"].(float64)) ||
		len(report.Matches) != int(expected["match_count"].(float64)) ||
		report.Candidates[0].Selected != expected["first_candidate_selected"].(bool) ||
		report.Candidates[0].StableBodyHash != expected["first_candidate_body_hash"].(string) {
		t.Fatalf("unexpected rename-aware matching report: %+v", report)
	}
}

func TestSharedFixtureAmbiguityDiagnostics(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-802-ambiguity-diagnostics", "ambiguity-diagnostics.json"))
	report := decodeFixtureValue[AmbiguityMatchingReport](t, fixture["matching"])
	expected := fixture["expected"].(map[string]any)

	if report.Strategy != expected["strategy"].(string) ||
		report.ScopePath != expected["scope_path"].(string) ||
		report.Ambiguous != expected["ambiguous"].(bool) ||
		len(report.Matches) != int(expected["match_count"].(float64)) ||
		len(report.Ambiguities) != int(expected["ambiguity_count"].(float64)) ||
		string(report.Diagnostics[0].Category) != expected["diagnostic_category"].(string) ||
		report.Ambiguities[0].Signature != expected["first_ambiguity_signature"].(string) ||
		report.Ambiguities[0].Reason != expected["first_ambiguity_reason"].(string) ||
		report.Ambiguities[0].Selected != expected["first_ambiguity_selected"].(bool) {
		t.Fatalf("unexpected ambiguity matching report: %+v", report)
	}
}

func TestSharedFixtureDuplicateSignatureTieBreak(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-803-duplicate-signature-tie-break", "duplicate-signature-tie-break.json"))
	report := decodeFixtureValue[TieBreakMatchingReport](t, fixture["matching"])
	expected := fixture["expected"].(map[string]any)

	if report.Strategy != expected["strategy"].(string) ||
		report.ScopePath != expected["scope_path"].(string) ||
		!reflect.DeepEqual(report.TieBreakRules, decodeFixtureValue[[]string](t, expected["tie_break_rules"])) ||
		len(report.Matches) != int(expected["match_count"].(float64)) ||
		report.Matches[0].Signature != expected["first_match_signature"].(string) ||
		report.Matches[0].SelectedBy != expected["first_match_selected_by"].(string) ||
		len(report.Matches[0].RejectedCandidates) != int(expected["rejected_candidate_count"].(float64)) ||
		report.Matches[0].RejectedCandidates[0].RejectedBy != expected["first_rejected_by"].(string) {
		t.Fatalf("unexpected tie-break matching report: %+v", report)
	}
}

func TestSharedFixtureMatchingDebugArtifacts(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, filepath.Join("..", "..", "fixtures", "diagnostics", "slice-804-matching-debug-artifacts", "matching-debug-artifacts.json"))
	artifacts := decodeFixtureValue[MatchingDebugArtifacts](t, fixture["debug_artifacts"])
	expected := fixture["expected"].(map[string]any)

	if artifacts.Enabled != expected["enabled"].(bool) ||
		len(artifacts.OwnerSets) != int(expected["owner_set_count"].(float64)) ||
		len(artifacts.Candidates) != int(expected["candidate_count"].(float64)) ||
		len(artifacts.SelectedMatches) != int(expected["selected_count"].(float64)) ||
		len(artifacts.RejectedMatches) != int(expected["rejected_count"].(float64)) ||
		artifacts.RejectedMatches[0].Reason != expected["first_rejection_reason"].(string) {
		t.Fatalf("unexpected matching debug artifacts: %+v", artifacts)
	}
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

func TestTemplateTokenKeysFixture(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "template_token_keys"))

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		content := testCase["content"].(string)
		var config *TemplateTokenConfig
		if rawConfig, ok := testCase["config"]; ok {
			decoded := decodeFixtureValueUntyped[TemplateTokenConfig](rawConfig)
			config = &decoded
		}

		actual := TemplateTokenKeys(content, config)
		expected := decodeFixtureValue[[]string](t, testCase["expected_token_keys"])
		if !reflect.DeepEqual(actual, expected) {
			t.Fatalf("expected token keys for %q to match fixture", content)
		}
	}
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

func TestTemplateEntryPlanStateFixture(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "template_entry_plan_state"))

	plannedEntries := decodeFixtureValue[[]TemplatePlanEntry](t, fixture["planned_entries"])
	existingDestinationPaths := decodeFixtureValueUntyped[[]string](fixture["existing_destination_paths"])
	actual := EnrichTemplatePlanEntries(plannedEntries, existingDestinationPaths)
	expected := decodeFixtureValue[[]TemplatePlanStateEntry](t, fixture["expected_entries"])
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("expected template entry plan state to match fixture")
	}
}

func TestTemplateEntryTokenStateFixture(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "template_entry_token_state"))

	plannedEntries := decodeFixtureValue[[]TemplatePlanStateEntry](t, fixture["planned_entries"])
	templateContents := decodeFixtureValueUntyped[map[string]string](fixture["template_contents"])
	replacements := decodeFixtureValueUntyped[map[string]string](fixture["replacements"])
	actual := EnrichTemplatePlanEntriesWithTokenState(plannedEntries, templateContents, replacements, nil)
	expected := decodeFixtureValue[[]TemplatePlanTokenStateEntry](t, fixture["expected_entries"])
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("expected template entry token state to match fixture")
	}
}

func TestTemplateEntryPreparedContentFixture(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "template_entry_prepared_content"))

	plannedEntries := decodeFixtureValue[[]TemplatePlanTokenStateEntry](t, fixture["planned_entries"])
	templateContents := decodeFixtureValueUntyped[map[string]string](fixture["template_contents"])
	replacements := decodeFixtureValueUntyped[map[string]string](fixture["replacements"])
	actual := PrepareTemplateEntries(plannedEntries, templateContents, replacements, nil)
	expected := decodeFixtureValue[[]TemplatePreparedEntry](t, fixture["expected_entries"])
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("expected template entry prepared content to match fixture")
	}
}

func TestTemplateExecutionPlanFixture(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "template_execution_plan"))

	preparedEntries := decodeFixtureValue[[]TemplatePreparedEntry](t, fixture["prepared_entries"])
	destinationContents := decodeFixtureValueUntyped[map[string]string](fixture["destination_contents"])
	actual := PlanTemplateExecution(preparedEntries, destinationContents)
	expected := decodeFixtureValue[[]TemplateExecutionPlanEntry](t, fixture["expected_entries"])
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("expected template execution plan to match fixture")
	}
}

func TestMiniTemplateTreePlanFixture(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "mini_template_tree_plan"))
	fixtureDir := filepath.Dir(diagnosticsFixturePath(t, "mini_template_tree_plan"))
	templateContents := readRelativeFileTree(t, filepath.Join(fixtureDir, "template"))
	destinationContents := readRelativeFileTree(t, filepath.Join(fixtureDir, "destination"))
	templateSourcePaths := mapsKeys(templateContents)
	slices.Sort(templateSourcePaths)
	existingDestinationPaths := mapsKeys(destinationContents)
	slices.Sort(existingDestinationPaths)
	context := decodeFixtureValueUntyped[TemplateDestinationContext](fixture["context"])
	overrides := decodeFixtureValueUntyped[[]TemplateStrategyOverride](fixture["overrides"])
	replacements := decodeFixtureValueUntyped[map[string]string](fixture["replacements"])
	actual := PlanTemplateTreeExecution(
		templateSourcePaths,
		templateContents,
		existingDestinationPaths,
		destinationContents,
		&context,
		TemplateStrategy(fixture["default_strategy"].(string)),
		overrides,
		replacements,
		nil,
	)
	expected := decodeFixtureValue[[]TemplateExecutionPlanEntry](t, fixture["expected_entries"])
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("expected mini template tree plan to match fixture")
	}
}

func TestMiniTemplateTreePreviewFixture(t *testing.T) {
	planFixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "mini_template_tree_plan"))
	previewFixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "mini_template_tree_preview"))
	fixtureDir := filepath.Dir(diagnosticsFixturePath(t, "mini_template_tree_plan"))
	templateContents := readRelativeFileTree(t, filepath.Join(fixtureDir, "template"))
	destinationContents := readRelativeFileTree(t, filepath.Join(fixtureDir, "destination"))
	templateSourcePaths := mapsKeys(templateContents)
	slices.Sort(templateSourcePaths)
	existingDestinationPaths := mapsKeys(destinationContents)
	slices.Sort(existingDestinationPaths)
	context := decodeFixtureValueUntyped[TemplateDestinationContext](planFixture["context"])
	overrides := decodeFixtureValueUntyped[[]TemplateStrategyOverride](planFixture["overrides"])
	replacements := decodeFixtureValueUntyped[map[string]string](planFixture["replacements"])
	executionPlan := PlanTemplateTreeExecution(
		templateSourcePaths,
		templateContents,
		existingDestinationPaths,
		destinationContents,
		&context,
		TemplateStrategy(planFixture["default_strategy"].(string)),
		overrides,
		replacements,
		nil,
	)
	actual := PreviewTemplateExecution(executionPlan)
	expected := decodeFixtureValue[TemplatePreviewResult](t, previewFixture["expected_preview"])
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("expected mini template tree preview to match fixture")
	}
}

func TestMiniTemplateTreeApplyFixture(t *testing.T) {
	planFixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "mini_template_tree_plan"))
	applyFixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "mini_template_tree_apply"))
	fixtureDir := filepath.Dir(diagnosticsFixturePath(t, "mini_template_tree_plan"))
	templateContents := readRelativeFileTree(t, filepath.Join(fixtureDir, "template"))
	destinationContents := readRelativeFileTree(t, filepath.Join(fixtureDir, "destination"))
	templateSourcePaths := mapsKeys(templateContents)
	slices.Sort(templateSourcePaths)
	existingDestinationPaths := mapsKeys(destinationContents)
	slices.Sort(existingDestinationPaths)
	context := decodeFixtureValueUntyped[TemplateDestinationContext](planFixture["context"])
	overrides := decodeFixtureValueUntyped[[]TemplateStrategyOverride](planFixture["overrides"])
	replacements := decodeFixtureValueUntyped[map[string]string](planFixture["replacements"])
	mergeResults := decodeFixtureValueUntyped[map[string]MergeResult[string]](applyFixture["merge_results"])

	executionPlan := PlanTemplateTreeExecution(
		templateSourcePaths,
		templateContents,
		existingDestinationPaths,
		destinationContents,
		&context,
		TemplateStrategy(planFixture["default_strategy"].(string)),
		overrides,
		replacements,
		nil,
	)
	actual := ApplyTemplateExecution(executionPlan, func(entry TemplateExecutionPlanEntry) MergeResult[string] {
		if entry.DestinationPath == nil {
			return MergeResult[string]{OK: false}
		}

		return mergeResults[*entry.DestinationPath]
	})
	expected := decodeFixtureValue[TemplateApplyResult](t, applyFixture["expected_result"])
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("expected mini template tree apply to match fixture")
	}
}

func TestMiniTemplateTreeConvergenceFixture(t *testing.T) {
	planFixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "mini_template_tree_plan"))
	applyFixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "mini_template_tree_apply"))
	convergenceFixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "mini_template_tree_convergence"))
	fixtureDir := filepath.Dir(diagnosticsFixturePath(t, "mini_template_tree_plan"))
	templateContents := readRelativeFileTree(t, filepath.Join(fixtureDir, "template"))
	destinationContents := readRelativeFileTree(t, filepath.Join(fixtureDir, "destination"))
	templateSourcePaths := mapsKeys(templateContents)
	slices.Sort(templateSourcePaths)
	existingDestinationPaths := mapsKeys(destinationContents)
	slices.Sort(existingDestinationPaths)
	context := decodeFixtureValueUntyped[TemplateDestinationContext](planFixture["context"])
	overrides := decodeFixtureValueUntyped[[]TemplateStrategyOverride](planFixture["overrides"])
	replacements := decodeFixtureValueUntyped[map[string]string](planFixture["replacements"])
	mergeResults := decodeFixtureValueUntyped[map[string]MergeResult[string]](applyFixture["merge_results"])

	executionPlan := PlanTemplateTreeExecution(
		templateSourcePaths,
		templateContents,
		existingDestinationPaths,
		destinationContents,
		&context,
		TemplateStrategy(planFixture["default_strategy"].(string)),
		overrides,
		replacements,
		nil,
	)
	applyResult := ApplyTemplateExecution(executionPlan, func(entry TemplateExecutionPlanEntry) MergeResult[string] {
		if entry.DestinationPath == nil {
			return MergeResult[string]{OK: false}
		}

		return mergeResults[*entry.DestinationPath]
	})
	convergenceReplacements := decodeFixtureValueUntyped[map[string]string](convergenceFixture["replacements"])
	actual := EvaluateTemplateTreeConvergence(
		templateSourcePaths,
		templateContents,
		applyResult.ResultFiles,
		&context,
		TemplateStrategy(planFixture["default_strategy"].(string)),
		overrides,
		convergenceReplacements,
		nil,
	)
	expected := decodeFixtureValue[TemplateConvergenceResult](t, convergenceFixture["expected"])
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("expected mini template tree convergence to match fixture")
	}
}

func TestMiniTemplateTreeRunFixture(t *testing.T) {
	planFixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "mini_template_tree_plan"))
	runFixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "mini_template_tree_run"))
	fixtureDir := filepath.Dir(diagnosticsFixturePath(t, "mini_template_tree_plan"))
	templateContents := readRelativeFileTree(t, filepath.Join(fixtureDir, "template"))
	destinationContents := readRelativeFileTree(t, filepath.Join(fixtureDir, "destination"))
	templateSourcePaths := mapsKeys(templateContents)
	slices.Sort(templateSourcePaths)
	context := decodeFixtureValueUntyped[TemplateDestinationContext](planFixture["context"])
	overrides := decodeFixtureValueUntyped[[]TemplateStrategyOverride](planFixture["overrides"])
	replacements := decodeFixtureValueUntyped[map[string]string](planFixture["replacements"])
	mergeResults := decodeFixtureValueUntyped[map[string]MergeResult[string]](runFixture["merge_results"])

	actual := RunTemplateTreeExecution(
		templateSourcePaths,
		templateContents,
		destinationContents,
		&context,
		TemplateStrategy(planFixture["default_strategy"].(string)),
		overrides,
		replacements,
		func(entry TemplateExecutionPlanEntry) MergeResult[string] {
			if entry.DestinationPath == nil {
				return MergeResult[string]{OK: false}
			}

			return mergeResults[*entry.DestinationPath]
		},
		nil,
	)
	expected := decodeFixtureValue[TemplateTreeRunResult](t, runFixture["expected"])
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("expected mini template tree run to match fixture")
	}
}

func TestMiniTemplateTreeRunReportFixture(t *testing.T) {
	planFixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "mini_template_tree_plan"))
	runFixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "mini_template_tree_run"))
	reportFixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "mini_template_tree_run_report"))
	fixtureDir := filepath.Dir(diagnosticsFixturePath(t, "mini_template_tree_plan"))
	templateContents := readRelativeFileTree(t, filepath.Join(fixtureDir, "template"))
	destinationContents := readRelativeFileTree(t, filepath.Join(fixtureDir, "destination"))
	templateSourcePaths := mapsKeys(templateContents)
	slices.Sort(templateSourcePaths)
	context := decodeFixtureValueUntyped[TemplateDestinationContext](planFixture["context"])
	overrides := decodeFixtureValueUntyped[[]TemplateStrategyOverride](planFixture["overrides"])
	replacements := decodeFixtureValueUntyped[map[string]string](planFixture["replacements"])
	mergeResults := decodeFixtureValueUntyped[map[string]MergeResult[string]](runFixture["merge_results"])

	runResult := RunTemplateTreeExecution(
		templateSourcePaths,
		templateContents,
		destinationContents,
		&context,
		TemplateStrategy(planFixture["default_strategy"].(string)),
		overrides,
		replacements,
		func(entry TemplateExecutionPlanEntry) MergeResult[string] {
			if entry.DestinationPath == nil {
				return MergeResult[string]{OK: false}
			}

			return mergeResults[*entry.DestinationPath]
		},
		nil,
	)
	actual := ReportTemplateTreeRun(runResult)
	expected := decodeFixtureValue[TemplateTreeRunReport](t, reportFixture["expected"])
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("expected mini template tree run report to match fixture")
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

func TestSharedFixtureStructuredEditStructureProfile(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_structure_profile"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var profile StructuredEditStructureProfile
		if raw, err := json.Marshal(testCase["profile"]); err != nil {
			t.Fatalf("marshal structured edit structure profile: %v", err)
		} else if err := json.Unmarshal(raw, &profile); err != nil {
			t.Fatalf("unmarshal structured edit structure profile: %v", err)
		}

		roundtrip, err := json.Marshal(profile)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit structure profile: %v", err)
		}
		var decoded StructuredEditStructureProfile
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit structure profile: %v", err)
		}

		if !reflect.DeepEqual(decoded, profile) {
			t.Fatalf("unexpected structured edit structure profile roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditSelectionProfile(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_selection_profile"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var profile StructuredEditSelectionProfile
		if raw, err := json.Marshal(testCase["profile"]); err != nil {
			t.Fatalf("marshal structured edit selection profile: %v", err)
		} else if err := json.Unmarshal(raw, &profile); err != nil {
			t.Fatalf("unmarshal structured edit selection profile: %v", err)
		}

		roundtrip, err := json.Marshal(profile)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit selection profile: %v", err)
		}
		var decoded StructuredEditSelectionProfile
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit selection profile: %v", err)
		}

		if !reflect.DeepEqual(decoded, profile) {
			t.Fatalf("unexpected structured edit selection profile roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditMatchProfile(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_match_profile"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var profile StructuredEditMatchProfile
		if raw, err := json.Marshal(testCase["profile"]); err != nil {
			t.Fatalf("marshal structured edit match profile: %v", err)
		} else if err := json.Unmarshal(raw, &profile); err != nil {
			t.Fatalf("unmarshal structured edit match profile: %v", err)
		}

		roundtrip, err := json.Marshal(profile)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit match profile: %v", err)
		}
		var decoded StructuredEditMatchProfile
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit match profile: %v", err)
		}

		if !reflect.DeepEqual(decoded, profile) {
			t.Fatalf("unexpected structured edit match profile roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditOperationProfile(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_operation_profile"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var profile StructuredEditOperationProfile
		if raw, err := json.Marshal(testCase["profile"]); err != nil {
			t.Fatalf("marshal structured edit operation profile: %v", err)
		} else if err := json.Unmarshal(raw, &profile); err != nil {
			t.Fatalf("unmarshal structured edit operation profile: %v", err)
		}

		roundtrip, err := json.Marshal(profile)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit operation profile: %v", err)
		}
		var decoded StructuredEditOperationProfile
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit operation profile: %v", err)
		}

		if !reflect.DeepEqual(decoded, profile) {
			t.Fatalf("unexpected structured edit operation profile roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditDestinationProfile(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_destination_profile"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var profile StructuredEditDestinationProfile
		if raw, err := json.Marshal(testCase["profile"]); err != nil {
			t.Fatalf("marshal structured edit destination profile: %v", err)
		} else if err := json.Unmarshal(raw, &profile); err != nil {
			t.Fatalf("unmarshal structured edit destination profile: %v", err)
		}

		roundtrip, err := json.Marshal(profile)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit destination profile: %v", err)
		}
		var decoded StructuredEditDestinationProfile
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit destination profile: %v", err)
		}

		if !reflect.DeepEqual(decoded, profile) {
			t.Fatalf("unexpected structured edit destination profile roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditRequest(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_request"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var request StructuredEditRequest
		if raw, err := json.Marshal(testCase["request"]); err != nil {
			t.Fatalf("marshal structured edit request: %v", err)
		} else if err := json.Unmarshal(raw, &request); err != nil {
			t.Fatalf("unmarshal structured edit request: %v", err)
		}

		roundtrip, err := json.Marshal(request)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit request: %v", err)
		}
		var decoded StructuredEditRequest
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit request: %v", err)
		}

		if !reflect.DeepEqual(decoded, request) {
			t.Fatalf("unexpected structured edit request roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditResult(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_result"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var result StructuredEditResult
		if raw, err := json.Marshal(testCase["result"]); err != nil {
			t.Fatalf("marshal structured edit result: %v", err)
		} else if err := json.Unmarshal(raw, &result); err != nil {
			t.Fatalf("unmarshal structured edit result: %v", err)
		}

		roundtrip, err := json.Marshal(result)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit result: %v", err)
		}
		var decoded StructuredEditResult
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit result: %v", err)
		}

		if !reflect.DeepEqual(decoded, result) {
			t.Fatalf("unexpected structured edit result roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_application"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var application StructuredEditApplication
		if raw, err := json.Marshal(testCase["application"]); err != nil {
			t.Fatalf("marshal structured edit application: %v", err)
		} else if err := json.Unmarshal(raw, &application); err != nil {
			t.Fatalf("unmarshal structured edit application: %v", err)
		}

		roundtrip, err := json.Marshal(application)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit application: %v", err)
		}
		var decoded StructuredEditApplication
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit application: %v", err)
		}

		if !reflect.DeepEqual(decoded, application) {
			t.Fatalf("unexpected structured edit application roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditApplicationEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_application_envelope"))
	application := decodeFixtureValue[StructuredEditApplication](t, fixture["structured_edit_application"])
	expected := decodeFixtureValue[StructuredEditApplicationEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditApplicationEnvelopeFor(application); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit application envelope: %+v", envelope)
	}

	if imported, importErr := ImportStructuredEditApplicationEnvelope(expected); importErr != nil {
		t.Fatalf("unexpected structured edit application envelope import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, application) {
		t.Fatalf("unexpected structured edit application envelope import: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditApplicationEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_application_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeFixtureValue[StructuredEditApplicationEnvelope](t, testCase["envelope"])
		expected := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditApplicationEnvelope(envelope); importErr == nil {
			t.Fatalf("expected structured edit application envelope rejection, got application: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expected) {
			t.Fatalf("unexpected structured edit application envelope rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditApplicationEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_application_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditApplicationEnvelope](t, fixture["structured_edit_application_envelope"])
	expected := decodeFixtureValue[StructuredEditApplication](t, fixture["expected_application"])

	if imported, importErr := ImportStructuredEditApplicationEnvelope(envelope); importErr != nil {
		t.Fatalf("unexpected structured edit application envelope application import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected structured edit application envelope application: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditApplicationEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditApplicationEnvelope(rejectedEnvelope); importErr == nil {
			t.Fatalf("expected structured edit application envelope application rejection, got application: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit application envelope application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditRequestEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_request_envelope"))
	request := decodeFixtureValue[StructuredEditRequest](t, fixture["structured_edit_request"])
	expected := decodeFixtureValue[StructuredEditRequestEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditRequestEnvelopeFor(request); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit request envelope: %+v", envelope)
	}

	if imported, importErr := ImportStructuredEditRequestEnvelope(expected); importErr != nil {
		t.Fatalf("unexpected structured edit request envelope import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, request) {
		t.Fatalf("unexpected structured edit request envelope import: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditRequestEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_request_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeFixtureValue[StructuredEditRequestEnvelope](t, testCase["envelope"])
		expected := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditRequestEnvelope(envelope); importErr == nil {
			t.Fatalf("expected structured edit request envelope rejection, got request: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expected) {
			t.Fatalf("unexpected structured edit request envelope rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditRequestEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_request_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditRequestEnvelope](t, fixture["structured_edit_request_envelope"])
	expected := decodeFixtureValue[StructuredEditRequest](t, fixture["expected_request"])

	if imported, importErr := ImportStructuredEditRequestEnvelope(envelope); importErr != nil {
		t.Fatalf("unexpected structured edit request envelope application import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected structured edit request envelope application: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditRequestEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditRequestEnvelope(rejectedEnvelope); importErr == nil {
			t.Fatalf("expected structured edit request envelope application rejection, got request: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit request envelope application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditExecutionReport(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_execution_report"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var report StructuredEditExecutionReport
		if raw, err := json.Marshal(testCase["report"]); err != nil {
			t.Fatalf("marshal structured edit execution report: %v", err)
		} else if err := json.Unmarshal(raw, &report); err != nil {
			t.Fatalf("unmarshal structured edit execution report: %v", err)
		}

		roundtrip, err := json.Marshal(report)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit execution report: %v", err)
		}
		var decoded StructuredEditExecutionReport
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit execution report: %v", err)
		}

		if !reflect.DeepEqual(decoded, report) {
			t.Fatalf("unexpected structured edit execution report roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionRequest(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_request"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var executionRequest StructuredEditProviderExecutionRequest
		if raw, err := json.Marshal(testCase["execution_request"]); err != nil {
			t.Fatalf("marshal structured edit provider execution request: %v", err)
		} else if err := json.Unmarshal(raw, &executionRequest); err != nil {
			t.Fatalf("unmarshal structured edit provider execution request: %v", err)
		}

		roundtrip, err := json.Marshal(executionRequest)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider execution request: %v", err)
		}
		var decoded StructuredEditProviderExecutionRequest
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider execution request: %v", err)
		}

		if !reflect.DeepEqual(decoded, executionRequest) {
			t.Fatalf("unexpected structured edit provider execution request roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionRequestEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_request_envelope"))
	executionRequest := decodeFixtureValue[StructuredEditProviderExecutionRequest](t, fixture["structured_edit_provider_execution_request"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionRequestEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderExecutionRequestEnvelopeFor(executionRequest); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider execution request envelope: %+v", envelope)
	}

	if imported, importErr := ImportStructuredEditProviderExecutionRequestEnvelope(expected); importErr != nil {
		t.Fatalf("unexpected structured edit provider execution request envelope import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, executionRequest) {
		t.Fatalf("unexpected structured edit provider execution request envelope import: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderExecutionRequestEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_request_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeFixtureValue[StructuredEditProviderExecutionRequestEnvelope](t, testCase["envelope"])
		expected := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderExecutionRequestEnvelope(envelope); importErr == nil {
			t.Fatalf("expected structured edit provider execution request envelope rejection, got request: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expected) {
			t.Fatalf("unexpected structured edit provider execution request envelope rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionRequestEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_request_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderExecutionRequestEnvelope](t, fixture["structured_edit_provider_execution_request_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionRequest](t, fixture["expected_execution_request"])

	if imported, importErr := ImportStructuredEditProviderExecutionRequestEnvelope(envelope); importErr != nil {
		t.Fatalf("unexpected structured edit provider execution request envelope application import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected structured edit provider execution request envelope application: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderExecutionRequestEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderExecutionRequestEnvelope(rejectedEnvelope); importErr == nil {
			t.Fatalf("expected structured edit provider execution request envelope application rejection, got request: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider execution request envelope application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_application"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var application StructuredEditProviderExecutionApplication
		if raw, err := json.Marshal(testCase["application"]); err != nil {
			t.Fatalf("marshal structured edit provider execution application: %v", err)
		} else if err := json.Unmarshal(raw, &application); err != nil {
			t.Fatalf("unmarshal structured edit provider execution application: %v", err)
		}

		roundtrip, err := json.Marshal(application)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider execution application: %v", err)
		}
		var decoded StructuredEditProviderExecutionApplication
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider execution application: %v", err)
		}

		if !reflect.DeepEqual(decoded, application) {
			t.Fatalf("unexpected structured edit provider execution application roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionDispatch(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_dispatch"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var dispatch StructuredEditProviderExecutionDispatch
		if raw, err := json.Marshal(testCase["dispatch"]); err != nil {
			t.Fatalf("marshal structured edit provider execution dispatch: %v", err)
		} else if err := json.Unmarshal(raw, &dispatch); err != nil {
			t.Fatalf("unmarshal structured edit provider execution dispatch: %v", err)
		}

		roundtrip, err := json.Marshal(dispatch)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider execution dispatch: %v", err)
		}
		var decoded StructuredEditProviderExecutionDispatch
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider execution dispatch: %v", err)
		}

		if !reflect.DeepEqual(decoded, dispatch) {
			t.Fatalf("unexpected structured edit provider execution dispatch roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionDispatchEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_dispatch_envelope"))
	dispatch := decodeFixtureValue[StructuredEditProviderExecutionDispatch](t, fixture["structured_edit_provider_execution_dispatch"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionDispatchEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderExecutionDispatchEnvelopeFor(dispatch); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider execution dispatch envelope: %+v", envelope)
	}

	if imported, importErr := ImportStructuredEditProviderExecutionDispatchEnvelope(expected); importErr != nil {
		t.Fatalf("unexpected structured edit provider execution dispatch envelope import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, dispatch) {
		t.Fatalf("unexpected structured edit provider execution dispatch envelope import: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderExecutionDispatchEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_dispatch_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeFixtureValue[StructuredEditProviderExecutionDispatchEnvelope](t, testCase["envelope"])
		expected := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderExecutionDispatchEnvelope(envelope); importErr == nil {
			t.Fatalf("expected structured edit provider execution dispatch envelope rejection, got dispatch: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expected) {
			t.Fatalf("unexpected structured edit provider execution dispatch envelope rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionDispatchEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_dispatch_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderExecutionDispatchEnvelope](t, fixture["structured_edit_provider_execution_dispatch_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionDispatch](t, fixture["expected_dispatch"])

	if imported, importErr := ImportStructuredEditProviderExecutionDispatchEnvelope(envelope); importErr != nil {
		t.Fatalf("unexpected structured edit provider execution dispatch envelope application import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected structured edit provider execution dispatch envelope application: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderExecutionDispatchEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderExecutionDispatchEnvelope(rejectedEnvelope); importErr == nil {
			t.Fatalf("expected structured edit provider execution dispatch envelope application rejection, got dispatch: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider execution dispatch envelope application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionOutcome(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_outcome"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var outcome StructuredEditProviderExecutionOutcome
		if raw, err := json.Marshal(testCase["outcome"]); err != nil {
			t.Fatalf("marshal structured edit provider execution outcome: %v", err)
		} else if err := json.Unmarshal(raw, &outcome); err != nil {
			t.Fatalf("unmarshal structured edit provider execution outcome: %v", err)
		}

		roundtrip, err := json.Marshal(outcome)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider execution outcome: %v", err)
		}
		var decoded StructuredEditProviderExecutionOutcome
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider execution outcome: %v", err)
		}

		if !reflect.DeepEqual(decoded, outcome) {
			t.Fatalf("unexpected structured edit provider execution outcome roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionOutcomeEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_outcome_envelope"))
	outcome := decodeFixtureValue[StructuredEditProviderExecutionOutcome](t, fixture["structured_edit_provider_execution_outcome"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionOutcomeEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderExecutionOutcomeEnvelopeFor(outcome); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider execution outcome envelope: %+v", envelope)
	}

	if imported, importErr := ImportStructuredEditProviderExecutionOutcomeEnvelope(expected); importErr != nil {
		t.Fatalf("unexpected structured edit provider execution outcome envelope import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, outcome) {
		t.Fatalf("unexpected structured edit provider execution outcome envelope import: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderExecutionOutcomeEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_outcome_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeFixtureValue[StructuredEditProviderExecutionOutcomeEnvelope](t, testCase["envelope"])
		expected := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderExecutionOutcomeEnvelope(envelope); importErr == nil {
			t.Fatalf("expected structured edit provider execution outcome envelope rejection, got outcome: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expected) {
			t.Fatalf("unexpected structured edit provider execution outcome envelope rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionOutcomeEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_outcome_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderExecutionOutcomeEnvelope](t, fixture["structured_edit_provider_execution_outcome_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionOutcome](t, fixture["expected_outcome"])

	if imported, importErr := ImportStructuredEditProviderExecutionOutcomeEnvelope(envelope); importErr != nil {
		t.Fatalf("unexpected structured edit provider execution outcome envelope application import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected structured edit provider execution outcome envelope application: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderExecutionOutcomeEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderExecutionOutcomeEnvelope(rejectedEnvelope); importErr == nil {
			t.Fatalf("expected structured edit provider execution outcome envelope application rejection, got outcome: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider execution outcome envelope application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionOutcome(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_outcome"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var batchOutcome StructuredEditProviderBatchExecutionOutcome
		if raw, err := json.Marshal(testCase["batch_outcome"]); err != nil {
			t.Fatalf("marshal structured edit provider batch execution outcome: %v", err)
		} else if err := json.Unmarshal(raw, &batchOutcome); err != nil {
			t.Fatalf("unmarshal structured edit provider batch execution outcome: %v", err)
		}

		roundtrip, err := json.Marshal(batchOutcome)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider batch execution outcome: %v", err)
		}
		var decoded StructuredEditProviderBatchExecutionOutcome
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider batch execution outcome: %v", err)
		}

		if !reflect.DeepEqual(decoded, batchOutcome) {
			t.Fatalf("unexpected structured edit provider batch execution outcome roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionOutcomeEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_outcome_envelope"))
	batchOutcome := decodeFixtureValue[StructuredEditProviderBatchExecutionOutcome](t, fixture["structured_edit_provider_batch_execution_outcome"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionOutcomeEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderBatchExecutionOutcomeEnvelopeFor(batchOutcome); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider batch execution outcome envelope: %+v", envelope)
	}

	if imported, importErr := ImportStructuredEditProviderBatchExecutionOutcomeEnvelope(expected); importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution outcome envelope import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, batchOutcome) {
		t.Fatalf("unexpected structured edit provider batch execution outcome envelope import: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionOutcomeEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_outcome_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionOutcomeEnvelope](t, testCase["envelope"])
		expected := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderBatchExecutionOutcomeEnvelope(envelope); importErr == nil {
			t.Fatalf("expected structured edit provider batch execution outcome envelope rejection, got batch outcome: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expected) {
			t.Fatalf("unexpected structured edit provider batch execution outcome envelope rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionOutcomeEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_outcome_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionOutcomeEnvelope](t, fixture["structured_edit_provider_batch_execution_outcome_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionOutcome](t, fixture["expected_batch_outcome"])

	if imported, importErr := ImportStructuredEditProviderBatchExecutionOutcomeEnvelope(envelope); importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution outcome envelope application import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected structured edit provider batch execution outcome envelope application: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderBatchExecutionOutcomeEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderBatchExecutionOutcomeEnvelope(rejectedEnvelope); importErr == nil {
			t.Fatalf("expected structured edit provider batch execution outcome envelope application rejection, got batch outcome: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider batch execution outcome envelope application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionProvenance(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_provenance"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var provenance StructuredEditProviderExecutionProvenance
		if raw, err := json.Marshal(testCase["provenance"]); err != nil {
			t.Fatalf("marshal structured edit provider execution provenance: %v", err)
		} else if err := json.Unmarshal(raw, &provenance); err != nil {
			t.Fatalf("unmarshal structured edit provider execution provenance: %v", err)
		}

		roundtrip, err := json.Marshal(provenance)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider execution provenance: %v", err)
		}
		var decoded StructuredEditProviderExecutionProvenance
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider execution provenance: %v", err)
		}

		if !reflect.DeepEqual(decoded, provenance) {
			t.Fatalf("unexpected structured edit provider execution provenance roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionProvenanceEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_provenance_envelope"))
	provenance := decodeFixtureValue[StructuredEditProviderExecutionProvenance](t, fixture["structured_edit_provider_execution_provenance"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionProvenanceEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderExecutionProvenanceEnvelopeFor(provenance); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider execution provenance envelope: %+v", envelope)
	}

	if imported, importErr := ImportStructuredEditProviderExecutionProvenanceEnvelope(expected); importErr != nil {
		t.Fatalf("unexpected structured edit provider execution provenance envelope import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, provenance) {
		t.Fatalf("unexpected structured edit provider execution provenance envelope import: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderExecutionProvenanceEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_provenance_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeFixtureValue[StructuredEditProviderExecutionProvenanceEnvelope](t, testCase["envelope"])
		expected := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderExecutionProvenanceEnvelope(envelope); importErr == nil {
			t.Fatalf("expected structured edit provider execution provenance envelope rejection, got provenance: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expected) {
			t.Fatalf("unexpected structured edit provider execution provenance envelope rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionProvenanceEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_provenance_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderExecutionProvenanceEnvelope](t, fixture["structured_edit_provider_execution_provenance_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionProvenance](t, fixture["expected_provenance"])

	if imported, importErr := ImportStructuredEditProviderExecutionProvenanceEnvelope(envelope); importErr != nil {
		t.Fatalf("unexpected structured edit provider execution provenance envelope application import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected structured edit provider execution provenance envelope application: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderExecutionProvenanceEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderExecutionProvenanceEnvelope(rejectedEnvelope); importErr == nil {
			t.Fatalf("expected structured edit provider execution provenance envelope application rejection, got provenance: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider execution provenance envelope application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionProvenance(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_provenance"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var batchProvenance StructuredEditProviderBatchExecutionProvenance
		if raw, err := json.Marshal(testCase["batch_provenance"]); err != nil {
			t.Fatalf("marshal structured edit provider batch execution provenance: %v", err)
		} else if err := json.Unmarshal(raw, &batchProvenance); err != nil {
			t.Fatalf("unmarshal structured edit provider batch execution provenance: %v", err)
		}

		roundtrip, err := json.Marshal(batchProvenance)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider batch execution provenance: %v", err)
		}
		var decoded StructuredEditProviderBatchExecutionProvenance
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider batch execution provenance: %v", err)
		}

		if !reflect.DeepEqual(decoded, batchProvenance) {
			t.Fatalf("unexpected structured edit provider batch execution provenance roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionProvenanceEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_provenance_envelope"))
	batchProvenance := decodeFixtureValue[StructuredEditProviderBatchExecutionProvenance](t, fixture["structured_edit_provider_batch_execution_provenance"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionProvenanceEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderBatchExecutionProvenanceEnvelopeFor(batchProvenance); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider batch execution provenance envelope: %+v", envelope)
	}

	if imported, importErr := ImportStructuredEditProviderBatchExecutionProvenanceEnvelope(expected); importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution provenance envelope import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, batchProvenance) {
		t.Fatalf("unexpected structured edit provider batch execution provenance envelope import: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionProvenanceEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_provenance_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionProvenanceEnvelope](t, testCase["envelope"])
		expected := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderBatchExecutionProvenanceEnvelope(envelope); importErr == nil {
			t.Fatalf("expected structured edit provider batch execution provenance envelope rejection, got batch provenance: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expected) {
			t.Fatalf("unexpected structured edit provider batch execution provenance envelope rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionProvenanceEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_provenance_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionProvenanceEnvelope](t, fixture["structured_edit_provider_batch_execution_provenance_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionProvenance](t, fixture["expected_batch_provenance"])

	if imported, importErr := ImportStructuredEditProviderBatchExecutionProvenanceEnvelope(envelope); importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution provenance envelope application import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected structured edit provider batch execution provenance envelope application: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderBatchExecutionProvenanceEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderBatchExecutionProvenanceEnvelope(rejectedEnvelope); importErr == nil {
			t.Fatalf("expected structured edit provider batch execution provenance envelope application rejection, got batch provenance: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider batch execution provenance envelope application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReplayBundle(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_replay_bundle"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var replayBundle StructuredEditProviderExecutionReplayBundle
		if raw, err := json.Marshal(testCase["replay_bundle"]); err != nil {
			t.Fatalf("marshal structured edit provider execution replay bundle: %v", err)
		} else if err := json.Unmarshal(raw, &replayBundle); err != nil {
			t.Fatalf("unmarshal structured edit provider execution replay bundle: %v", err)
		}

		roundtrip, err := json.Marshal(replayBundle)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider execution replay bundle: %v", err)
		}
		var decoded StructuredEditProviderExecutionReplayBundle
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider execution replay bundle: %v", err)
		}

		if !reflect.DeepEqual(decoded, replayBundle) {
			t.Fatalf("unexpected structured edit provider execution replay bundle roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReplayBundleEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_replay_bundle_envelope"))
	replayBundle := decodeFixtureValue[StructuredEditProviderExecutionReplayBundle](t, fixture["structured_edit_provider_execution_replay_bundle"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionReplayBundleEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderExecutionReplayBundleEnvelopeFor(replayBundle); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider execution replay bundle envelope: %+v", envelope)
	}

	if imported, importErr := ImportStructuredEditProviderExecutionReplayBundleEnvelope(expected); importErr != nil {
		t.Fatalf("unexpected structured edit provider execution replay bundle envelope import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, replayBundle) {
		t.Fatalf("unexpected structured edit provider execution replay bundle envelope import: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReplayBundleEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_replay_bundle_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeFixtureValue[StructuredEditProviderExecutionReplayBundleEnvelope](t, testCase["envelope"])
		expected := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderExecutionReplayBundleEnvelope(envelope); importErr == nil {
			t.Fatalf("expected structured edit provider execution replay bundle envelope rejection, got replay bundle: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expected) {
			t.Fatalf("unexpected structured edit provider execution replay bundle envelope rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReplayBundleEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_replay_bundle_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderExecutionReplayBundleEnvelope](t, fixture["structured_edit_provider_execution_replay_bundle_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionReplayBundle](t, fixture["expected_replay_bundle"])

	if imported, importErr := ImportStructuredEditProviderExecutionReplayBundleEnvelope(envelope); importErr != nil {
		t.Fatalf("unexpected structured edit provider execution replay bundle envelope application import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected structured edit provider execution replay bundle envelope application: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderExecutionReplayBundleEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderExecutionReplayBundleEnvelope(rejectedEnvelope); importErr == nil {
			t.Fatalf("expected structured edit provider execution replay bundle envelope application rejection, got replay bundle: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider execution replay bundle envelope application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReplayBundle(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_replay_bundle"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var batchReplayBundle StructuredEditProviderBatchExecutionReplayBundle
		if raw, err := json.Marshal(testCase["batch_replay_bundle"]); err != nil {
			t.Fatalf("marshal structured edit provider batch execution replay bundle: %v", err)
		} else if err := json.Unmarshal(raw, &batchReplayBundle); err != nil {
			t.Fatalf("unmarshal structured edit provider batch execution replay bundle: %v", err)
		}

		roundtrip, err := json.Marshal(batchReplayBundle)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider batch execution replay bundle: %v", err)
		}
		var decoded StructuredEditProviderBatchExecutionReplayBundle
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider batch execution replay bundle: %v", err)
		}

		if !reflect.DeepEqual(decoded, batchReplayBundle) {
			t.Fatalf("unexpected structured edit provider batch execution replay bundle roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReplayBundleEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_replay_bundle_envelope"))
	batchReplayBundle := decodeFixtureValue[StructuredEditProviderBatchExecutionReplayBundle](t, fixture["structured_edit_provider_batch_execution_replay_bundle"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionReplayBundleEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderBatchExecutionReplayBundleEnvelopeFor(batchReplayBundle); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider batch execution replay bundle envelope: %+v", envelope)
	}

	if imported, importErr := ImportStructuredEditProviderBatchExecutionReplayBundleEnvelope(expected); importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution replay bundle envelope import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, batchReplayBundle) {
		t.Fatalf("unexpected structured edit provider batch execution replay bundle envelope import: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReplayBundleEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_replay_bundle_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReplayBundleEnvelope](t, testCase["envelope"])
		expected := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderBatchExecutionReplayBundleEnvelope(envelope); importErr == nil {
			t.Fatalf("expected structured edit provider batch execution replay bundle envelope rejection, got batch replay bundle: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expected) {
			t.Fatalf("unexpected structured edit provider batch execution replay bundle envelope rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReplayBundleEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_replay_bundle_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReplayBundleEnvelope](t, fixture["structured_edit_provider_batch_execution_replay_bundle_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionReplayBundle](t, fixture["expected_batch_replay_bundle"])

	if imported, importErr := ImportStructuredEditProviderBatchExecutionReplayBundleEnvelope(envelope); importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution replay bundle envelope application import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected structured edit provider batch execution replay bundle envelope application: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReplayBundleEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderBatchExecutionReplayBundleEnvelope(rejectedEnvelope); importErr == nil {
			t.Fatalf("expected structured edit provider batch execution replay bundle envelope application rejection, got batch replay bundle: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider batch execution replay bundle envelope application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutorProfile(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_executor_profile"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var executorProfile StructuredEditProviderExecutorProfile
		if raw, err := json.Marshal(testCase["executor_profile"]); err != nil {
			t.Fatalf("marshal structured edit provider executor profile: %v", err)
		} else if err := json.Unmarshal(raw, &executorProfile); err != nil {
			t.Fatalf("unmarshal structured edit provider executor profile: %v", err)
		}

		roundtrip, err := json.Marshal(executorProfile)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider executor profile: %v", err)
		}
		var decoded StructuredEditProviderExecutorProfile
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider executor profile: %v", err)
		}

		if !reflect.DeepEqual(decoded, executorProfile) {
			t.Fatalf("unexpected structured edit provider executor profile roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutorOperationTriadProfile(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_executor_operation_triad_profile"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var executorProfile StructuredEditProviderExecutorProfile
		if raw, err := json.Marshal(testCase["executor_profile"]); err != nil {
			t.Fatalf("marshal structured edit provider executor operation triad profile: %v", err)
		} else if err := json.Unmarshal(raw, &executorProfile); err != nil {
			t.Fatalf("unmarshal structured edit provider executor operation triad profile: %v", err)
		}

		roundtrip, err := json.Marshal(executorProfile)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider executor operation triad profile: %v", err)
		}
		var decoded StructuredEditProviderExecutorProfile
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider executor operation triad profile: %v", err)
		}

		if !reflect.DeepEqual(decoded, executorProfile) {
			t.Fatalf("unexpected structured edit provider executor operation triad profile roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutorProfileEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_executor_profile_envelope"))
	executorProfile := decodeFixtureValue[StructuredEditProviderExecutorProfile](t, fixture["structured_edit_provider_executor_profile"])
	expected := decodeFixtureValue[StructuredEditProviderExecutorProfileEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderExecutorProfileEnvelopeFor(executorProfile); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider executor profile envelope: %+v", envelope)
	}

	if imported, importErr := ImportStructuredEditProviderExecutorProfileEnvelope(expected); importErr != nil {
		t.Fatalf("unexpected structured edit provider executor profile envelope import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, executorProfile) {
		t.Fatalf("unexpected structured edit provider executor profile envelope import: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderExecutorProfileEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_executor_profile_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeFixtureValue[StructuredEditProviderExecutorProfileEnvelope](t, testCase["envelope"])
		expected := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderExecutorProfileEnvelope(envelope); importErr == nil {
			t.Fatalf("expected structured edit provider executor profile envelope rejection, got executor profile: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expected) {
			t.Fatalf("unexpected structured edit provider executor profile envelope rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutorProfileEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_executor_profile_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderExecutorProfileEnvelope](t, fixture["structured_edit_provider_executor_profile_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderExecutorProfile](t, fixture["expected_executor_profile"])

	if imported, importErr := ImportStructuredEditProviderExecutorProfileEnvelope(envelope); importErr != nil {
		t.Fatalf("unexpected structured edit provider executor profile envelope application import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected structured edit provider executor profile envelope application: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderExecutorProfileEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderExecutorProfileEnvelope(rejectedEnvelope); importErr == nil {
			t.Fatalf("expected structured edit provider executor profile envelope application rejection, got executor profile: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider executor profile envelope application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutorRegistry(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_executor_registry"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var executorRegistry StructuredEditProviderExecutorRegistry
		if raw, err := json.Marshal(testCase["executor_registry"]); err != nil {
			t.Fatalf("marshal structured edit provider executor registry: %v", err)
		} else if err := json.Unmarshal(raw, &executorRegistry); err != nil {
			t.Fatalf("unmarshal structured edit provider executor registry: %v", err)
		}

		roundtrip, err := json.Marshal(executorRegistry)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider executor registry: %v", err)
		}
		var decoded StructuredEditProviderExecutorRegistry
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider executor registry: %v", err)
		}

		if !reflect.DeepEqual(decoded, executorRegistry) {
			t.Fatalf("unexpected structured edit provider executor registry roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutorRegistryEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_executor_registry_envelope"))
	executorRegistry := decodeFixtureValue[StructuredEditProviderExecutorRegistry](t, fixture["structured_edit_provider_executor_registry"])
	expected := decodeFixtureValue[StructuredEditProviderExecutorRegistryEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderExecutorRegistryEnvelopeFor(executorRegistry); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider executor registry envelope: %+v", envelope)
	}

	if imported, importErr := ImportStructuredEditProviderExecutorRegistryEnvelope(expected); importErr != nil {
		t.Fatalf("unexpected structured edit provider executor registry envelope import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, executorRegistry) {
		t.Fatalf("unexpected structured edit provider executor registry envelope import: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderExecutorRegistryEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_executor_registry_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeFixtureValue[StructuredEditProviderExecutorRegistryEnvelope](t, testCase["envelope"])
		expected := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderExecutorRegistryEnvelope(envelope); importErr == nil {
			t.Fatalf("expected structured edit provider executor registry envelope rejection, got executor registry: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expected) {
			t.Fatalf("unexpected structured edit provider executor registry envelope rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutorRegistryEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_executor_registry_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderExecutorRegistryEnvelope](t, fixture["structured_edit_provider_executor_registry_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderExecutorRegistry](t, fixture["expected_executor_registry"])

	if imported, importErr := ImportStructuredEditProviderExecutorRegistryEnvelope(envelope); importErr != nil {
		t.Fatalf("unexpected structured edit provider executor registry envelope application import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected structured edit provider executor registry envelope application: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderExecutorRegistryEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderExecutorRegistryEnvelope(rejectedEnvelope); importErr == nil {
			t.Fatalf("expected structured edit provider executor registry envelope application rejection, got executor registry: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider executor registry envelope application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutorSelectionPolicy(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_executor_selection_policy"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var selectionPolicy StructuredEditProviderExecutorSelectionPolicy
		if raw, err := json.Marshal(testCase["selection_policy"]); err != nil {
			t.Fatalf("marshal structured edit provider executor selection policy: %v", err)
		} else if err := json.Unmarshal(raw, &selectionPolicy); err != nil {
			t.Fatalf("unmarshal structured edit provider executor selection policy: %v", err)
		}

		roundtrip, err := json.Marshal(selectionPolicy)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider executor selection policy: %v", err)
		}
		var decoded StructuredEditProviderExecutorSelectionPolicy
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider executor selection policy: %v", err)
		}

		if !reflect.DeepEqual(decoded, selectionPolicy) {
			t.Fatalf("unexpected structured edit provider executor selection policy roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutorSelectionPolicyEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_executor_selection_policy_envelope"))
	selectionPolicy := decodeFixtureValue[StructuredEditProviderExecutorSelectionPolicy](t, fixture["structured_edit_provider_executor_selection_policy"])
	expected := decodeFixtureValue[StructuredEditProviderExecutorSelectionPolicyEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderExecutorSelectionPolicyEnvelopeFor(selectionPolicy); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider executor selection policy envelope: %+v", envelope)
	}

	if imported, importErr := ImportStructuredEditProviderExecutorSelectionPolicyEnvelope(expected); importErr != nil {
		t.Fatalf("unexpected structured edit provider executor selection policy envelope import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, selectionPolicy) {
		t.Fatalf("unexpected structured edit provider executor selection policy envelope import: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderExecutorSelectionPolicyEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_executor_selection_policy_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeFixtureValue[StructuredEditProviderExecutorSelectionPolicyEnvelope](t, testCase["envelope"])
		expected := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderExecutorSelectionPolicyEnvelope(envelope); importErr == nil {
			t.Fatalf("expected structured edit provider executor selection policy envelope rejection, got selection policy: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expected) {
			t.Fatalf("unexpected structured edit provider executor selection policy envelope rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutorSelectionPolicyEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_executor_selection_policy_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderExecutorSelectionPolicyEnvelope](t, fixture["structured_edit_provider_executor_selection_policy_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderExecutorSelectionPolicy](t, fixture["expected_selection_policy"])

	if imported, importErr := ImportStructuredEditProviderExecutorSelectionPolicyEnvelope(envelope); importErr != nil {
		t.Fatalf("unexpected structured edit provider executor selection policy envelope application import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected structured edit provider executor selection policy envelope application: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderExecutorSelectionPolicyEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderExecutorSelectionPolicyEnvelope(rejectedEnvelope); importErr == nil {
			t.Fatalf("expected structured edit provider executor selection policy envelope application rejection, got selection policy: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider executor selection policy envelope application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutorResolution(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_executor_resolution"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var executorResolution StructuredEditProviderExecutorResolution
		if raw, err := json.Marshal(testCase["executor_resolution"]); err != nil {
			t.Fatalf("marshal structured edit provider executor resolution: %v", err)
		} else if err := json.Unmarshal(raw, &executorResolution); err != nil {
			t.Fatalf("unmarshal structured edit provider executor resolution: %v", err)
		}

		roundtrip, err := json.Marshal(executorResolution)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider executor resolution: %v", err)
		}
		var decoded StructuredEditProviderExecutorResolution
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider executor resolution: %v", err)
		}

		if !reflect.DeepEqual(decoded, executorResolution) {
			t.Fatalf("unexpected structured edit provider executor resolution roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutorResolutionEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_executor_resolution_envelope"))
	executorResolution := decodeFixtureValue[StructuredEditProviderExecutorResolution](t, fixture["structured_edit_provider_executor_resolution"])
	expected := decodeFixtureValue[StructuredEditProviderExecutorResolutionEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderExecutorResolutionEnvelopeFor(executorResolution); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider executor resolution envelope: %+v", envelope)
	}

	if imported, importErr := ImportStructuredEditProviderExecutorResolutionEnvelope(expected); importErr != nil {
		t.Fatalf("unexpected structured edit provider executor resolution envelope import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, executorResolution) {
		t.Fatalf("unexpected structured edit provider executor resolution envelope import: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderExecutorResolutionEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_executor_resolution_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeFixtureValue[StructuredEditProviderExecutorResolutionEnvelope](t, testCase["envelope"])
		expected := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderExecutorResolutionEnvelope(envelope); importErr == nil {
			t.Fatalf("expected structured edit provider executor resolution envelope rejection, got executor resolution: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expected) {
			t.Fatalf("unexpected structured edit provider executor resolution envelope rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutorResolutionEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_executor_resolution_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderExecutorResolutionEnvelope](t, fixture["structured_edit_provider_executor_resolution_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderExecutorResolution](t, fixture["expected_executor_resolution"])

	if imported, importErr := ImportStructuredEditProviderExecutorResolutionEnvelope(envelope); importErr != nil {
		t.Fatalf("unexpected structured edit provider executor resolution envelope application import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected structured edit provider executor resolution envelope application: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderExecutorResolutionEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderExecutorResolutionEnvelope(rejectedEnvelope); importErr == nil {
			t.Fatalf("expected structured edit provider executor resolution envelope application rejection, got executor resolution: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider executor resolution envelope application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionPlan(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_plan"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var executionPlan StructuredEditProviderExecutionPlan
		if raw, err := json.Marshal(testCase["execution_plan"]); err != nil {
			t.Fatalf("marshal structured edit provider execution plan: %v", err)
		} else if err := json.Unmarshal(raw, &executionPlan); err != nil {
			t.Fatalf("unmarshal structured edit provider execution plan: %v", err)
		}

		roundtrip, err := json.Marshal(executionPlan)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider execution plan: %v", err)
		}
		var decoded StructuredEditProviderExecutionPlan
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider execution plan: %v", err)
		}

		if !reflect.DeepEqual(decoded, executionPlan) {
			t.Fatalf("unexpected structured edit provider execution plan roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionHandoff(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_handoff"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var executionHandoff StructuredEditProviderExecutionHandoff
		if raw, err := json.Marshal(testCase["execution_handoff"]); err != nil {
			t.Fatalf("marshal structured edit provider execution handoff: %v", err)
		} else if err := json.Unmarshal(raw, &executionHandoff); err != nil {
			t.Fatalf("unmarshal structured edit provider execution handoff: %v", err)
		}

		roundtrip, err := json.Marshal(executionHandoff)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider execution handoff: %v", err)
		}
		var decoded StructuredEditProviderExecutionHandoff
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider execution handoff: %v", err)
		}

		if !reflect.DeepEqual(decoded, executionHandoff) {
			t.Fatalf("unexpected structured edit provider execution handoff roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionHandoffEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_handoff_envelope"))
	executionHandoff := decodeFixtureValue[StructuredEditProviderExecutionHandoff](t, fixture["structured_edit_provider_execution_handoff"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionHandoffEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderExecutionHandoffEnvelopeFor(executionHandoff); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider execution handoff envelope: %+v", envelope)
	}

	if imported, importErr := ImportStructuredEditProviderExecutionHandoffEnvelope(expected); importErr != nil {
		t.Fatalf("unexpected structured edit provider execution handoff envelope import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, executionHandoff) {
		t.Fatalf("unexpected structured edit provider execution handoff envelope import: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderExecutionHandoffEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_handoff_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeFixtureValue[StructuredEditProviderExecutionHandoffEnvelope](t, testCase["envelope"])
		expected := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderExecutionHandoffEnvelope(envelope); importErr == nil {
			t.Fatalf("expected structured edit provider execution handoff envelope rejection, got execution handoff: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expected) {
			t.Fatalf("unexpected structured edit provider execution handoff envelope rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionHandoffEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_handoff_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderExecutionHandoffEnvelope](t, fixture["structured_edit_provider_execution_handoff_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionHandoff](t, fixture["expected_execution_handoff"])

	if imported, importErr := ImportStructuredEditProviderExecutionHandoffEnvelope(envelope); importErr != nil {
		t.Fatalf("unexpected structured edit provider execution handoff envelope application import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected structured edit provider execution handoff envelope application: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderExecutionHandoffEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderExecutionHandoffEnvelope(rejectedEnvelope); importErr == nil {
			t.Fatalf("expected structured edit provider execution handoff envelope application rejection, got execution handoff: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider execution handoff envelope application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionInvocation(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_invocation"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var executionInvocation StructuredEditProviderExecutionInvocation
		if raw, err := json.Marshal(testCase["execution_invocation"]); err != nil {
			t.Fatalf("marshal structured edit provider execution invocation: %v", err)
		} else if err := json.Unmarshal(raw, &executionInvocation); err != nil {
			t.Fatalf("unmarshal structured edit provider execution invocation: %v", err)
		}

		roundtrip, err := json.Marshal(executionInvocation)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider execution invocation: %v", err)
		}
		var decoded StructuredEditProviderExecutionInvocation
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider execution invocation: %v", err)
		}

		if !reflect.DeepEqual(decoded, executionInvocation) {
			t.Fatalf("unexpected structured edit provider execution invocation roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionInvocationEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_invocation_envelope"))
	executionInvocation := decodeFixtureValue[StructuredEditProviderExecutionInvocation](t, fixture["structured_edit_provider_execution_invocation"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionInvocationEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderExecutionInvocationEnvelopeFor(executionInvocation); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider execution invocation envelope: %+v", envelope)
	}

	if imported, importErr := ImportStructuredEditProviderExecutionInvocationEnvelope(expected); importErr != nil {
		t.Fatalf("unexpected structured edit provider execution invocation envelope import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, executionInvocation) {
		t.Fatalf("unexpected structured edit provider execution invocation envelope import: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderExecutionInvocationEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_invocation_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeFixtureValue[StructuredEditProviderExecutionInvocationEnvelope](t, testCase["envelope"])
		expected := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderExecutionInvocationEnvelope(envelope); importErr == nil {
			t.Fatalf("expected structured edit provider execution invocation envelope rejection, got execution invocation: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expected) {
			t.Fatalf("unexpected structured edit provider execution invocation envelope rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionInvocationEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_invocation_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderExecutionInvocationEnvelope](t, fixture["structured_edit_provider_execution_invocation_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionInvocation](t, fixture["expected_execution_invocation"])

	if imported, importErr := ImportStructuredEditProviderExecutionInvocationEnvelope(envelope); importErr != nil {
		t.Fatalf("unexpected structured edit provider execution invocation envelope application import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected structured edit provider execution invocation envelope application: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderExecutionInvocationEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderExecutionInvocationEnvelope(rejectedEnvelope); importErr == nil {
			t.Fatalf("expected structured edit provider execution invocation envelope application rejection, got execution invocation: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider execution invocation envelope application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionInvocation(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_invocation"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var batchExecutionInvocation StructuredEditProviderBatchExecutionInvocation
		if raw, err := json.Marshal(testCase["batch_execution_invocation"]); err != nil {
			t.Fatalf("marshal structured edit provider batch execution invocation: %v", err)
		} else if err := json.Unmarshal(raw, &batchExecutionInvocation); err != nil {
			t.Fatalf("unmarshal structured edit provider batch execution invocation: %v", err)
		}

		roundtrip, err := json.Marshal(batchExecutionInvocation)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider batch execution invocation: %v", err)
		}
		var decoded StructuredEditProviderBatchExecutionInvocation
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider batch execution invocation: %v", err)
		}

		if !reflect.DeepEqual(decoded, batchExecutionInvocation) {
			t.Fatalf("unexpected structured edit provider batch execution invocation roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionInvocationEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_invocation_envelope"))
	batchExecutionInvocation := decodeFixtureValue[StructuredEditProviderBatchExecutionInvocation](t, fixture["structured_edit_provider_batch_execution_invocation"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionInvocationEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderBatchExecutionInvocationEnvelopeFor(batchExecutionInvocation); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider batch execution invocation envelope: %+v", envelope)
	}

	if imported, importErr := ImportStructuredEditProviderBatchExecutionInvocationEnvelope(expected); importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution invocation envelope import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, batchExecutionInvocation) {
		t.Fatalf("unexpected structured edit provider batch execution invocation envelope import: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionInvocationEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_invocation_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionInvocationEnvelope](t, testCase["envelope"])
		expected := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderBatchExecutionInvocationEnvelope(envelope); importErr == nil {
			t.Fatalf("expected structured edit provider batch execution invocation envelope rejection, got batch execution invocation: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expected) {
			t.Fatalf("unexpected structured edit provider batch execution invocation envelope rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionInvocationEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_invocation_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionInvocationEnvelope](t, fixture["structured_edit_provider_batch_execution_invocation_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionInvocation](t, fixture["expected_batch_execution_invocation"])

	if imported, importErr := ImportStructuredEditProviderBatchExecutionInvocationEnvelope(envelope); importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution invocation envelope application import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected structured edit provider batch execution invocation envelope application: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderBatchExecutionInvocationEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderBatchExecutionInvocationEnvelope(rejectedEnvelope); importErr == nil {
			t.Fatalf("expected structured edit provider batch execution invocation envelope application rejection, got batch execution invocation: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider batch execution invocation envelope application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionRunResult(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_run_result"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var executionRunResult StructuredEditProviderExecutionRunResult
		if raw, err := json.Marshal(testCase["execution_run_result"]); err != nil {
			t.Fatalf("marshal structured edit provider execution run result: %v", err)
		} else if err := json.Unmarshal(raw, &executionRunResult); err != nil {
			t.Fatalf("unmarshal structured edit provider execution run result: %v", err)
		}

		roundtrip, err := json.Marshal(executionRunResult)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider execution run result: %v", err)
		}
		var decoded StructuredEditProviderExecutionRunResult
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider execution run result: %v", err)
		}

		if !reflect.DeepEqual(decoded, executionRunResult) {
			t.Fatalf("unexpected structured edit provider execution run result roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionRunResultEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_run_result_envelope"))
	executionRunResult := decodeFixtureValue[StructuredEditProviderExecutionRunResult](t, fixture["structured_edit_provider_execution_run_result"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionRunResultEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderExecutionRunResultEnvelopeFor(executionRunResult); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider execution run result envelope: %+v", envelope)
	}

	if imported, importErr := ImportStructuredEditProviderExecutionRunResultEnvelope(expected); importErr != nil {
		t.Fatalf("unexpected structured edit provider execution run result envelope import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, executionRunResult) {
		t.Fatalf("unexpected structured edit provider execution run result envelope import: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderExecutionRunResultEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_run_result_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeFixtureValue[StructuredEditProviderExecutionRunResultEnvelope](t, testCase["envelope"])
		expected := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderExecutionRunResultEnvelope(envelope); importErr == nil {
			t.Fatalf("expected structured edit provider execution run result envelope rejection, got execution run result: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expected) {
			t.Fatalf("unexpected structured edit provider execution run result envelope rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionRunResultEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_run_result_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderExecutionRunResultEnvelope](t, fixture["structured_edit_provider_execution_run_result_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionRunResult](t, fixture["expected_execution_run_result"])

	if imported, importErr := ImportStructuredEditProviderExecutionRunResultEnvelope(envelope); importErr != nil {
		t.Fatalf("unexpected structured edit provider execution run result envelope application import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected structured edit provider execution run result envelope application: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderExecutionRunResultEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderExecutionRunResultEnvelope(rejectedEnvelope); importErr == nil {
			t.Fatalf("expected structured edit provider execution run result envelope application rejection, got execution run result: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider execution run result envelope application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionRunResult(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_run_result"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var batchExecutionRunResult StructuredEditProviderBatchExecutionRunResult
		if raw, err := json.Marshal(testCase["batch_execution_run_result"]); err != nil {
			t.Fatalf("marshal structured edit provider batch execution run result: %v", err)
		} else if err := json.Unmarshal(raw, &batchExecutionRunResult); err != nil {
			t.Fatalf("unmarshal structured edit provider batch execution run result: %v", err)
		}

		roundtrip, err := json.Marshal(batchExecutionRunResult)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider batch execution run result: %v", err)
		}
		var decoded StructuredEditProviderBatchExecutionRunResult
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider batch execution run result: %v", err)
		}

		if !reflect.DeepEqual(decoded, batchExecutionRunResult) {
			t.Fatalf("unexpected structured edit provider batch execution run result roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionRunResultEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_run_result_envelope"))
	batchExecutionRunResult := decodeFixtureValue[StructuredEditProviderBatchExecutionRunResult](t, fixture["structured_edit_provider_batch_execution_run_result"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionRunResultEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderBatchExecutionRunResultEnvelopeFor(batchExecutionRunResult); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider batch execution run result envelope: %+v", envelope)
	}

	if imported, importErr := ImportStructuredEditProviderBatchExecutionRunResultEnvelope(expected); importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution run result envelope import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, batchExecutionRunResult) {
		t.Fatalf("unexpected structured edit provider batch execution run result import: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionRunResultEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_run_result_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionRunResultEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderBatchExecutionRunResultEnvelope(envelope); importErr == nil {
			t.Fatalf("expected structured edit provider batch execution run result envelope rejection, got batch execution run result: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider batch execution run result envelope rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionRunResultEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_run_result_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionRunResultEnvelope](t, fixture["structured_edit_provider_batch_execution_run_result_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionRunResult](t, fixture["expected_batch_execution_run_result"])

	if imported, importErr := ImportStructuredEditProviderBatchExecutionRunResultEnvelope(envelope); importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution run result envelope application import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected structured edit provider batch execution run result envelope application import: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderBatchExecutionRunResultEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderBatchExecutionRunResultEnvelope(rejectedEnvelope); importErr == nil {
			t.Fatalf("expected structured edit provider batch execution run result envelope application rejection, got batch execution run result: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider batch execution run result envelope application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceipt(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var executionReceipt StructuredEditProviderExecutionReceipt
		if raw, err := json.Marshal(testCase["execution_receipt"]); err != nil {
			t.Fatalf("marshal structured edit provider execution receipt: %v", err)
		} else if err := json.Unmarshal(raw, &executionReceipt); err != nil {
			t.Fatalf("unmarshal structured edit provider execution receipt: %v", err)
		}

		roundtrip, err := json.Marshal(executionReceipt)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider execution receipt: %v", err)
		}
		var decoded StructuredEditProviderExecutionReceipt
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider execution receipt: %v", err)
		}

		if !reflect.DeepEqual(decoded, executionReceipt) {
			t.Fatalf("unexpected structured edit provider execution receipt roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceipt(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var batchExecutionReceipt StructuredEditProviderBatchExecutionReceipt
		if raw, err := json.Marshal(testCase["batch_execution_receipt"]); err != nil {
			t.Fatalf("marshal structured edit provider batch execution receipt: %v", err)
		} else if err := json.Unmarshal(raw, &batchExecutionReceipt); err != nil {
			t.Fatalf("unmarshal structured edit provider batch execution receipt: %v", err)
		}

		roundtrip, err := json.Marshal(batchExecutionReceipt)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider batch execution receipt: %v", err)
		}
		var decoded StructuredEditProviderBatchExecutionReceipt
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider batch execution receipt: %v", err)
		}

		if !reflect.DeepEqual(decoded, batchExecutionReceipt) {
			t.Fatalf("unexpected structured edit provider batch execution receipt roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_envelope"))
	batchExecutionReceipt := decodeFixtureValue[StructuredEditProviderBatchExecutionReceipt](t, fixture["structured_edit_provider_batch_execution_receipt"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderBatchExecutionReceiptEnvelopeFor(batchExecutionReceipt); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider batch execution receipt envelope: %+v", envelope)
	}

	imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptEnvelope(expected)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution receipt envelope import error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, batchExecutionReceipt) {
		t.Fatalf("unexpected imported structured edit provider batch execution receipt: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptEnvelope(envelope)
		if imported != nil {
			t.Fatalf("expected no structured edit provider batch execution receipt for rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider batch execution receipt import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider batch execution receipt rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptEnvelope](t, fixture["structured_edit_provider_batch_execution_receipt_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionReceipt](t, fixture["expected_batch_execution_receipt"])

	imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptEnvelope(envelope)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution receipt envelope application error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected applied structured edit provider batch execution receipt: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		rejected, importErr := ImportStructuredEditProviderBatchExecutionReceiptEnvelope(rejectedEnvelope)
		if rejected != nil {
			t.Fatalf("expected no structured edit provider batch execution receipt for application rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider batch execution receipt envelope application import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider batch execution receipt envelope application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayRequest(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_request"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var receiptReplayRequest StructuredEditProviderExecutionReceiptReplayRequest
		if raw, err := json.Marshal(testCase["receipt_replay_request"]); err != nil {
			t.Fatalf("marshal structured edit provider execution receipt replay request: %v", err)
		} else if err := json.Unmarshal(raw, &receiptReplayRequest); err != nil {
			t.Fatalf("unmarshal structured edit provider execution receipt replay request: %v", err)
		}

		roundtrip, err := json.Marshal(receiptReplayRequest)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider execution receipt replay request: %v", err)
		}
		var decoded StructuredEditProviderExecutionReceiptReplayRequest
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider execution receipt replay request: %v", err)
		}

		if !reflect.DeepEqual(decoded, receiptReplayRequest) {
			t.Fatalf("unexpected structured edit provider execution receipt replay request roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayRequestEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_request_envelope"))
	receiptReplayRequest := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayRequest](t, fixture["structured_edit_provider_execution_receipt_replay_request"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayRequestEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderExecutionReceiptReplayRequestEnvelopeFor(receiptReplayRequest); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider execution receipt replay request envelope: %+v", envelope)
	}

	imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayRequestEnvelope(expected)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider execution receipt replay request envelope import error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, receiptReplayRequest) {
		t.Fatalf("unexpected imported structured edit provider execution receipt replay request: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayRequestEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_request_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		envelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayRequestEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayRequestEnvelope(envelope)
		if imported != nil {
			t.Fatalf("expected no structured edit provider execution receipt replay request for rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider execution receipt replay request import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider execution receipt replay request rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayRequestEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_request_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayRequestEnvelope](t, fixture["structured_edit_provider_execution_receipt_replay_request_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayRequest](t, fixture["expected_receipt_replay_request"])

	imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayRequestEnvelope(envelope)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider execution receipt replay request envelope application error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected applied structured edit provider execution receipt replay request: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayRequestEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		rejected, importErr := ImportStructuredEditProviderExecutionReceiptReplayRequestEnvelope(rejectedEnvelope)
		if rejected != nil {
			t.Fatalf("expected no structured edit provider execution receipt replay request for application rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider execution receipt replay request application import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider execution receipt replay request application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayRequest(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_request"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var batchReceiptReplayRequest StructuredEditProviderBatchExecutionReceiptReplayRequest
		if raw, err := json.Marshal(testCase["batch_receipt_replay_request"]); err != nil {
			t.Fatalf("marshal structured edit provider batch execution receipt replay request: %v", err)
		} else if err := json.Unmarshal(raw, &batchReceiptReplayRequest); err != nil {
			t.Fatalf("unmarshal structured edit provider batch execution receipt replay request: %v", err)
		}

		roundtrip, err := json.Marshal(batchReceiptReplayRequest)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider batch execution receipt replay request: %v", err)
		}
		var decoded StructuredEditProviderBatchExecutionReceiptReplayRequest
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider batch execution receipt replay request: %v", err)
		}

		if !reflect.DeepEqual(decoded, batchReceiptReplayRequest) {
			t.Fatalf("unexpected structured edit provider batch execution receipt replay request roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayRequestEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_request_envelope"))
	batchReceiptReplayRequest := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayRequest](t, fixture["structured_edit_provider_batch_execution_receipt_replay_request"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayRequestEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderBatchExecutionReceiptReplayRequestEnvelopeFor(batchReceiptReplayRequest); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider batch execution receipt replay request envelope: %+v", envelope)
	}

	imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayRequestEnvelope(expected)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution receipt replay request envelope import error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, batchReceiptReplayRequest) {
		t.Fatalf("unexpected imported structured edit provider batch execution receipt replay request: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayRequestEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_request_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayRequestEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayRequestEnvelope(envelope)
		if imported != nil {
			t.Fatalf("expected no structured edit provider batch execution receipt replay request for rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider batch execution receipt replay request import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider batch execution receipt replay request rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayRequestEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_request_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayRequestEnvelope](t, fixture["structured_edit_provider_batch_execution_receipt_replay_request_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayRequest](t, fixture["expected_batch_receipt_replay_request"])

	imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayRequestEnvelope(envelope)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution receipt replay request envelope application error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected applied structured edit provider batch execution receipt replay request: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayRequestEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		rejected, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayRequestEnvelope(rejectedEnvelope)
		if rejected != nil {
			t.Fatalf("expected no structured edit provider batch execution receipt replay request for application rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider batch execution receipt replay request application import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider batch execution receipt replay request application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_application"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var receiptReplayApplication StructuredEditProviderExecutionReceiptReplayApplication
		if raw, err := json.Marshal(testCase["receipt_replay_application"]); err != nil {
			t.Fatalf("marshal structured edit provider execution receipt replay application: %v", err)
		} else if err := json.Unmarshal(raw, &receiptReplayApplication); err != nil {
			t.Fatalf("unmarshal structured edit provider execution receipt replay application: %v", err)
		}

		roundtrip, err := json.Marshal(receiptReplayApplication)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider execution receipt replay application: %v", err)
		}
		var decoded StructuredEditProviderExecutionReceiptReplayApplication
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider execution receipt replay application: %v", err)
		}

		if !reflect.DeepEqual(decoded, receiptReplayApplication) {
			t.Fatalf("unexpected structured edit provider execution receipt replay application roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayApplicationEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_application_envelope"))
	receiptReplayApplication := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayApplication](t, fixture["structured_edit_provider_execution_receipt_replay_application"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayApplicationEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderExecutionReceiptReplayApplicationEnvelopeFor(receiptReplayApplication); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider execution receipt replay application envelope: %+v", envelope)
	}

	imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayApplicationEnvelope(expected)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider execution receipt replay application envelope import error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, receiptReplayApplication) {
		t.Fatalf("unexpected imported structured edit provider execution receipt replay application: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayApplicationEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_application_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		envelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayApplicationEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayApplicationEnvelope(envelope)
		if imported != nil {
			t.Fatalf("expected no structured edit provider execution receipt replay application for rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider execution receipt replay application import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider execution receipt replay application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayApplicationEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_application_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayApplicationEnvelope](t, fixture["structured_edit_provider_execution_receipt_replay_application_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayApplication](t, fixture["expected_receipt_replay_application"])

	imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayApplicationEnvelope(envelope)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider execution receipt replay application envelope application error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected applied structured edit provider execution receipt replay application: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayApplicationEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		rejected, importErr := ImportStructuredEditProviderExecutionReceiptReplayApplicationEnvelope(rejectedEnvelope)
		if rejected != nil {
			t.Fatalf("expected no structured edit provider execution receipt replay application for application rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider execution receipt replay application application import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider execution receipt replay application application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_application"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var batchReceiptReplayApplication StructuredEditProviderBatchExecutionReceiptReplayApplication
		if raw, err := json.Marshal(testCase["batch_receipt_replay_application"]); err != nil {
			t.Fatalf("marshal structured edit provider batch execution receipt replay application: %v", err)
		} else if err := json.Unmarshal(raw, &batchReceiptReplayApplication); err != nil {
			t.Fatalf("unmarshal structured edit provider batch execution receipt replay application: %v", err)
		}

		roundtrip, err := json.Marshal(batchReceiptReplayApplication)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider batch execution receipt replay application: %v", err)
		}
		var decoded StructuredEditProviderBatchExecutionReceiptReplayApplication
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider batch execution receipt replay application: %v", err)
		}

		if !reflect.DeepEqual(decoded, batchReceiptReplayApplication) {
			t.Fatalf("unexpected structured edit provider batch execution receipt replay application roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayApplicationEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_application_envelope"))
	batchReceiptReplayApplication := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayApplication](t, fixture["structured_edit_provider_batch_execution_receipt_replay_application"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayApplicationEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderBatchExecutionReceiptReplayApplicationEnvelopeFor(batchReceiptReplayApplication); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider batch execution receipt replay application envelope: %+v", envelope)
	}

	imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayApplicationEnvelope(expected)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution receipt replay application envelope import error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, batchReceiptReplayApplication) {
		t.Fatalf("unexpected imported structured edit provider batch execution receipt replay application: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayApplicationEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_application_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayApplicationEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayApplicationEnvelope(envelope)
		if imported != nil {
			t.Fatalf("expected no structured edit provider batch execution receipt replay application for rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider batch execution receipt replay application import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider batch execution receipt replay application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayApplicationEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_application_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayApplicationEnvelope](t, fixture["structured_edit_provider_batch_execution_receipt_replay_application_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayApplication](t, fixture["expected_batch_receipt_replay_application"])

	imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayApplicationEnvelope(envelope)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution receipt replay application envelope application error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected applied structured edit provider batch execution receipt replay application: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayApplicationEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		rejected, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayApplicationEnvelope(rejectedEnvelope)
		if rejected != nil {
			t.Fatalf("expected no structured edit provider batch execution receipt replay application for application rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider batch execution receipt replay application application import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider batch execution receipt replay application application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplaySession(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_session"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var receiptReplaySession StructuredEditProviderExecutionReceiptReplaySession
		if raw, err := json.Marshal(testCase["receipt_replay_session"]); err != nil {
			t.Fatalf("marshal structured edit provider execution receipt replay session: %v", err)
		} else if err := json.Unmarshal(raw, &receiptReplaySession); err != nil {
			t.Fatalf("unmarshal structured edit provider execution receipt replay session: %v", err)
		}

		roundtrip, err := json.Marshal(receiptReplaySession)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider execution receipt replay session: %v", err)
		}
		var decoded StructuredEditProviderExecutionReceiptReplaySession
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider execution receipt replay session: %v", err)
		}

		if !reflect.DeepEqual(decoded, receiptReplaySession) {
			t.Fatalf("unexpected structured edit provider execution receipt replay session roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplaySessionEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_session_envelope"))
	receiptReplaySession := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplaySession](t, fixture["structured_edit_provider_execution_receipt_replay_session"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplaySessionEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderExecutionReceiptReplaySessionEnvelopeFor(receiptReplaySession); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider execution receipt replay session envelope: %+v", envelope)
	}

	imported, importErr := ImportStructuredEditProviderExecutionReceiptReplaySessionEnvelope(expected)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider execution receipt replay session envelope import error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, receiptReplaySession) {
		t.Fatalf("unexpected imported structured edit provider execution receipt replay session: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplaySessionEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_session_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		envelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplaySessionEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		imported, importErr := ImportStructuredEditProviderExecutionReceiptReplaySessionEnvelope(envelope)
		if imported != nil {
			t.Fatalf("expected no structured edit provider execution receipt replay session for rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider execution receipt replay session import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider execution receipt replay session rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplaySessionEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_session_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplaySessionEnvelope](t, fixture["structured_edit_provider_execution_receipt_replay_session_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplaySession](t, fixture["expected_receipt_replay_session"])

	imported, importErr := ImportStructuredEditProviderExecutionReceiptReplaySessionEnvelope(envelope)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider execution receipt replay session envelope application error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected applied structured edit provider execution receipt replay session: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplaySessionEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		rejected, importErr := ImportStructuredEditProviderExecutionReceiptReplaySessionEnvelope(rejectedEnvelope)
		if rejected != nil {
			t.Fatalf("expected no structured edit provider execution receipt replay session for application rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider execution receipt replay session application import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider execution receipt replay session application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplaySession(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_session"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var batchReceiptReplaySession StructuredEditProviderBatchExecutionReceiptReplaySession
		if raw, err := json.Marshal(testCase["batch_receipt_replay_session"]); err != nil {
			t.Fatalf("marshal structured edit provider batch execution receipt replay session: %v", err)
		} else if err := json.Unmarshal(raw, &batchReceiptReplaySession); err != nil {
			t.Fatalf("unmarshal structured edit provider batch execution receipt replay session: %v", err)
		}

		roundtrip, err := json.Marshal(batchReceiptReplaySession)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider batch execution receipt replay session: %v", err)
		}
		var decoded StructuredEditProviderBatchExecutionReceiptReplaySession
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider batch execution receipt replay session: %v", err)
		}

		if !reflect.DeepEqual(decoded, batchReceiptReplaySession) {
			t.Fatalf("unexpected structured edit provider batch execution receipt replay session roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplaySessionEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_session_envelope"))
	batchReceiptReplaySession := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplaySession](t, fixture["structured_edit_provider_batch_execution_receipt_replay_session"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplaySessionEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderBatchExecutionReceiptReplaySessionEnvelopeFor(batchReceiptReplaySession); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider batch execution receipt replay session envelope: %+v", envelope)
	}

	imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplaySessionEnvelope(expected)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution receipt replay session envelope import error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, batchReceiptReplaySession) {
		t.Fatalf("unexpected imported structured edit provider batch execution receipt replay session: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplaySessionEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_session_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplaySessionEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplaySessionEnvelope(envelope)
		if imported != nil {
			t.Fatalf("expected no structured edit provider batch execution receipt replay session for rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider batch execution receipt replay session import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider batch execution receipt replay session rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplaySessionEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_session_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplaySessionEnvelope](t, fixture["structured_edit_provider_batch_execution_receipt_replay_session_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplaySession](t, fixture["expected_batch_receipt_replay_session"])

	imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplaySessionEnvelope(envelope)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution receipt replay session envelope application error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected applied structured edit provider batch execution receipt replay session: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplaySessionEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		rejected, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplaySessionEnvelope(rejectedEnvelope)
		if rejected != nil {
			t.Fatalf("expected no structured edit provider batch execution receipt replay session for application rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider batch execution receipt replay session application import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider batch execution receipt replay session application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflow(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var receiptReplayWorkflow StructuredEditProviderExecutionReceiptReplayWorkflow
		if raw, err := json.Marshal(testCase["receipt_replay_workflow"]); err != nil {
			t.Fatalf("marshal structured edit provider execution receipt replay workflow: %v", err)
		} else if err := json.Unmarshal(raw, &receiptReplayWorkflow); err != nil {
			t.Fatalf("unmarshal structured edit provider execution receipt replay workflow: %v", err)
		}

		roundtrip, err := json.Marshal(receiptReplayWorkflow)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider execution receipt replay workflow: %v", err)
		}
		var decoded StructuredEditProviderExecutionReceiptReplayWorkflow
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider execution receipt replay workflow: %v", err)
		}

		if !reflect.DeepEqual(decoded, receiptReplayWorkflow) {
			t.Fatalf("unexpected structured edit provider execution receipt replay workflow roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_envelope"))
	receiptReplayWorkflow := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflow](t, fixture["structured_edit_provider_execution_receipt_replay_workflow"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderExecutionReceiptReplayWorkflowEnvelopeFor(receiptReplayWorkflow); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider execution receipt replay workflow envelope: %+v", envelope)
	}

	imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowEnvelope(expected)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider execution receipt replay workflow envelope import error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, receiptReplayWorkflow) {
		t.Fatalf("unexpected imported structured edit provider execution receipt replay workflow: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		envelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowEnvelope(envelope)
		if imported != nil {
			t.Fatalf("expected no structured edit provider execution receipt replay workflow for rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider execution receipt replay workflow import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider execution receipt replay workflow rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowEnvelope](t, fixture["structured_edit_provider_execution_receipt_replay_workflow_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflow](t, fixture["expected_receipt_replay_workflow"])

	imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowEnvelope(envelope)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider execution receipt replay workflow envelope application error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected applied structured edit provider execution receipt replay workflow: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		rejected, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowEnvelope(rejectedEnvelope)
		if rejected != nil {
			t.Fatalf("expected no structured edit provider execution receipt replay workflow for application rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider execution receipt replay workflow application import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider execution receipt replay workflow application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayWorkflow(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_workflow"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var batchReceiptReplayWorkflow StructuredEditProviderBatchExecutionReceiptReplayWorkflow
		if raw, err := json.Marshal(testCase["batch_receipt_replay_workflow"]); err != nil {
			t.Fatalf("marshal structured edit provider batch execution receipt replay workflow: %v", err)
		} else if err := json.Unmarshal(raw, &batchReceiptReplayWorkflow); err != nil {
			t.Fatalf("unmarshal structured edit provider batch execution receipt replay workflow: %v", err)
		}

		roundtrip, err := json.Marshal(batchReceiptReplayWorkflow)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider batch execution receipt replay workflow: %v", err)
		}
		var decoded StructuredEditProviderBatchExecutionReceiptReplayWorkflow
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider batch execution receipt replay workflow: %v", err)
		}

		if !reflect.DeepEqual(decoded, batchReceiptReplayWorkflow) {
			t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayWorkflowEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_workflow_envelope"))
	batchReceiptReplayWorkflow := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflow](t, fixture["structured_edit_provider_batch_execution_receipt_replay_workflow"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderBatchExecutionReceiptReplayWorkflowEnvelopeFor(batchReceiptReplayWorkflow); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow envelope: %+v", envelope)
	}

	imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayWorkflowEnvelope(expected)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow envelope import error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, batchReceiptReplayWorkflow) {
		t.Fatalf("unexpected imported structured edit provider batch execution receipt replay workflow: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayWorkflowEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_workflow_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayWorkflowEnvelope(envelope)
		if imported != nil {
			t.Fatalf("expected no structured edit provider batch execution receipt replay workflow for rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider batch execution receipt replay workflow import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayWorkflowEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_workflow_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowEnvelope](t, fixture["structured_edit_provider_batch_execution_receipt_replay_workflow_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflow](t, fixture["expected_batch_receipt_replay_workflow"])

	imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayWorkflowEnvelope(envelope)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow envelope application error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected applied structured edit provider batch execution receipt replay workflow: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		rejected, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayWorkflowEnvelope(rejectedEnvelope)
		if rejected != nil {
			t.Fatalf("expected no structured edit provider batch execution receipt replay workflow for application rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider batch execution receipt replay workflow application import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowResult(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_result"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var receiptReplayWorkflowResult StructuredEditProviderExecutionReceiptReplayWorkflowResult
		if raw, err := json.Marshal(testCase["receipt_replay_workflow_result"]); err != nil {
			t.Fatalf("marshal structured edit provider execution receipt replay workflow result: %v", err)
		} else if err := json.Unmarshal(raw, &receiptReplayWorkflowResult); err != nil {
			t.Fatalf("unmarshal structured edit provider execution receipt replay workflow result: %v", err)
		}

		roundtrip, err := json.Marshal(receiptReplayWorkflowResult)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider execution receipt replay workflow result: %v", err)
		}
		var decoded StructuredEditProviderExecutionReceiptReplayWorkflowResult
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider execution receipt replay workflow result: %v", err)
		}

		if !reflect.DeepEqual(decoded, receiptReplayWorkflowResult) {
			t.Fatalf("unexpected structured edit provider execution receipt replay workflow result roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowResultEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_result_envelope"))
	receiptReplayWorkflowResult := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowResult](t, fixture["structured_edit_provider_execution_receipt_replay_workflow_result"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowResultEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderExecutionReceiptReplayWorkflowResultEnvelopeFor(receiptReplayWorkflowResult); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider execution receipt replay workflow result envelope: %+v", envelope)
	}

	imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowResultEnvelope(expected)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider execution receipt replay workflow result envelope import error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, receiptReplayWorkflowResult) {
		t.Fatalf("unexpected imported structured edit provider execution receipt replay workflow result: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowResultEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_result_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		envelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowResultEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowResultEnvelope(envelope)
		if imported != nil {
			t.Fatalf("expected no structured edit provider execution receipt replay workflow result for rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider execution receipt replay workflow result import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider execution receipt replay workflow result rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowResultEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_result_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowResultEnvelope](t, fixture["structured_edit_provider_execution_receipt_replay_workflow_result_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowResult](t, fixture["expected_receipt_replay_workflow_result"])

	imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowResultEnvelope(envelope)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider execution receipt replay workflow result envelope application error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected applied structured edit provider execution receipt replay workflow result: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowResultEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		rejected, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowResultEnvelope(rejectedEnvelope)
		if rejected != nil {
			t.Fatalf("expected no structured edit provider execution receipt replay workflow result for application rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider execution receipt replay workflow result application import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider execution receipt replay workflow result application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayWorkflowResult(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_workflow_result"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var batchReceiptReplayWorkflowResult StructuredEditProviderBatchExecutionReceiptReplayWorkflowResult
		if raw, err := json.Marshal(testCase["batch_receipt_replay_workflow_result"]); err != nil {
			t.Fatalf("marshal structured edit provider batch execution receipt replay workflow result: %v", err)
		} else if err := json.Unmarshal(raw, &batchReceiptReplayWorkflowResult); err != nil {
			t.Fatalf("unmarshal structured edit provider batch execution receipt replay workflow result: %v", err)
		}

		roundtrip, err := json.Marshal(batchReceiptReplayWorkflowResult)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider batch execution receipt replay workflow result: %v", err)
		}
		var decoded StructuredEditProviderBatchExecutionReceiptReplayWorkflowResult
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider batch execution receipt replay workflow result: %v", err)
		}

		if !reflect.DeepEqual(decoded, batchReceiptReplayWorkflowResult) {
			t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow result roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayWorkflowResultEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_workflow_result_envelope"))
	batchReceiptReplayWorkflowResult := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowResult](t, fixture["structured_edit_provider_batch_execution_receipt_replay_workflow_result"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowResultEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderBatchExecutionReceiptReplayWorkflowResultEnvelopeFor(batchReceiptReplayWorkflowResult); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow result envelope: %+v", envelope)
	}

	imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayWorkflowResultEnvelope(expected)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow result envelope import error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, batchReceiptReplayWorkflowResult) {
		t.Fatalf("unexpected imported structured edit provider batch execution receipt replay workflow result: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayWorkflowResultEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_workflow_result_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowResultEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayWorkflowResultEnvelope(envelope)
		if imported != nil {
			t.Fatalf("expected no structured edit provider batch execution receipt replay workflow result for rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider batch execution receipt replay workflow result import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow result rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayWorkflowResultEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_workflow_result_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowResultEnvelope](t, fixture["structured_edit_provider_batch_execution_receipt_replay_workflow_result_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowResult](t, fixture["expected_batch_receipt_replay_workflow_result"])

	imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayWorkflowResultEnvelope(envelope)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow result envelope application error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected applied structured edit provider batch execution receipt replay workflow result: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowResultEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		rejected, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayWorkflowResultEnvelope(rejectedEnvelope)
		if rejected != nil {
			t.Fatalf("expected no structured edit provider batch execution receipt replay workflow result for application rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider batch execution receipt replay workflow result application import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow result application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowReviewRequest(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_review_request"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var receiptReplayWorkflowReviewRequest StructuredEditProviderExecutionReceiptReplayWorkflowReviewRequest
		if raw, err := json.Marshal(testCase["receipt_replay_workflow_review_request"]); err != nil {
			t.Fatalf("marshal structured edit provider execution receipt replay workflow review request: %v", err)
		} else if err := json.Unmarshal(raw, &receiptReplayWorkflowReviewRequest); err != nil {
			t.Fatalf("unmarshal structured edit provider execution receipt replay workflow review request: %v", err)
		}

		roundtrip, err := json.Marshal(receiptReplayWorkflowReviewRequest)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider execution receipt replay workflow review request: %v", err)
		}
		var decoded StructuredEditProviderExecutionReceiptReplayWorkflowReviewRequest
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider execution receipt replay workflow review request: %v", err)
		}

		if !reflect.DeepEqual(decoded, receiptReplayWorkflowReviewRequest) {
			t.Fatalf("unexpected structured edit provider execution receipt replay workflow review request roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowApplyRequest(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_apply_request"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var receiptReplayWorkflowApplyRequest StructuredEditProviderExecutionReceiptReplayWorkflowApplyRequest
		if raw, err := json.Marshal(testCase["receipt_replay_workflow_apply_request"]); err != nil {
			t.Fatalf("marshal structured edit provider execution receipt replay workflow apply request: %v", err)
		} else if err := json.Unmarshal(raw, &receiptReplayWorkflowApplyRequest); err != nil {
			t.Fatalf("unmarshal structured edit provider execution receipt replay workflow apply request: %v", err)
		}

		roundtrip, err := json.Marshal(receiptReplayWorkflowApplyRequest)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider execution receipt replay workflow apply request: %v", err)
		}
		var decoded StructuredEditProviderExecutionReceiptReplayWorkflowApplyRequest
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider execution receipt replay workflow apply request: %v", err)
		}

		if !reflect.DeepEqual(decoded, receiptReplayWorkflowApplyRequest) {
			t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply request roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowApplySession(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_apply_session"))

	cases := fixture["cases"].([]any)
	for _, rawEntry := range cases {
		entry := rawEntry.(map[string]any)

		var receiptReplayWorkflowApplySession StructuredEditProviderExecutionReceiptReplayWorkflowApplySession
		if raw, err := json.Marshal(entry["receipt_replay_workflow_apply_session"]); err != nil {
			t.Fatalf("marshal apply session: %v", err)
		} else if err := json.Unmarshal(raw, &receiptReplayWorkflowApplySession); err != nil {
			t.Fatalf("unmarshal apply session: %v", err)
		}

		payload, err := json.Marshal(receiptReplayWorkflowApplySession)
		if err != nil {
			t.Fatalf("marshal apply session payload: %v", err)
		}

		var decoded StructuredEditProviderExecutionReceiptReplayWorkflowApplySession
		if err := json.Unmarshal(payload, &decoded); err != nil {
			t.Fatalf("decode apply session payload: %v", err)
		}

		if !reflect.DeepEqual(decoded, receiptReplayWorkflowApplySession) {
			t.Fatalf("apply session mismatch: %+v", decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowApplyResult(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_apply_result"))

	cases := fixture["cases"].([]any)
	for _, rawEntry := range cases {
		entry := rawEntry.(map[string]any)

		var receiptReplayWorkflowApplyResult StructuredEditProviderExecutionReceiptReplayWorkflowApplyResult
		if raw, err := json.Marshal(entry["receipt_replay_workflow_apply_result"]); err != nil {
			t.Fatalf("marshal apply result: %v", err)
		} else if err := json.Unmarshal(raw, &receiptReplayWorkflowApplyResult); err != nil {
			t.Fatalf("unmarshal apply result: %v", err)
		}

		payload, err := json.Marshal(receiptReplayWorkflowApplyResult)
		if err != nil {
			t.Fatalf("marshal apply result payload: %v", err)
		}

		var decoded StructuredEditProviderExecutionReceiptReplayWorkflowApplyResult
		if err := json.Unmarshal(payload, &decoded); err != nil {
			t.Fatalf("decode apply result payload: %v", err)
		}

		if !reflect.DeepEqual(decoded, receiptReplayWorkflowApplyResult) {
			t.Fatalf("apply result mismatch: %+v", decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowApplyResultEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_apply_result_envelope"))
	receiptReplayWorkflowApplyResult := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyResult](t, fixture["structured_edit_provider_execution_receipt_replay_workflow_apply_result"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyResultEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderExecutionReceiptReplayWorkflowApplyResultEnvelopeFor(receiptReplayWorkflowApplyResult); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply result envelope: %+v", envelope)
	}

	imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowApplyResultEnvelope(expected)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply result envelope import error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, receiptReplayWorkflowApplyResult) {
		t.Fatalf("unexpected imported structured edit provider execution receipt replay workflow apply result: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowApplyResultEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_apply_result_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		envelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyResultEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowApplyResultEnvelope(envelope)
		if imported != nil {
			t.Fatalf("expected no structured edit provider execution receipt replay workflow apply result for rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider execution receipt replay workflow apply result import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply result rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowApplyResultEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_apply_result_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyResultEnvelope](t, fixture["structured_edit_provider_execution_receipt_replay_workflow_apply_result_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyResult](t, fixture["expected_receipt_replay_workflow_apply_result"])

	imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowApplyResultEnvelope(envelope)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply result envelope application error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected applied structured edit provider execution receipt replay workflow apply result: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyResultEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		rejected, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowApplyResultEnvelope(rejectedEnvelope)
		if rejected != nil {
			t.Fatalf("expected no structured edit provider execution receipt replay workflow apply result for application rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider execution receipt replay workflow apply result application import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply result application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowApplySessionEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_apply_session_envelope"))
	receiptReplayWorkflowApplySession := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplySession](t, fixture["structured_edit_provider_execution_receipt_replay_workflow_apply_session"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplySessionEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderExecutionReceiptReplayWorkflowApplySessionEnvelopeFor(receiptReplayWorkflowApplySession); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply session envelope: %+v", envelope)
	}

	imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowApplySessionEnvelope(expected)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply session envelope import error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, receiptReplayWorkflowApplySession) {
		t.Fatalf("unexpected imported structured edit provider execution receipt replay workflow apply session: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowApplySessionEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_apply_session_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		envelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplySessionEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowApplySessionEnvelope(envelope)
		if imported != nil {
			t.Fatalf("expected no structured edit provider execution receipt replay workflow apply session for rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider execution receipt replay workflow apply session import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply session rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowApplySessionEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_apply_session_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplySessionEnvelope](t, fixture["structured_edit_provider_execution_receipt_replay_workflow_apply_session_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplySession](t, fixture["expected_receipt_replay_workflow_apply_session"])

	imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowApplySessionEnvelope(envelope)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply session envelope application error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected applied structured edit provider execution receipt replay workflow apply session: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplySessionEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		rejected, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowApplySessionEnvelope(rejectedEnvelope)
		if rejected != nil {
			t.Fatalf("expected no structured edit provider execution receipt replay workflow apply session for envelope application rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider execution receipt replay workflow apply session rejection error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply session envelope rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowApplyRequestEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_apply_request_envelope"))
	receiptReplayWorkflowApplyRequest := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyRequest](t, fixture["structured_edit_provider_execution_receipt_replay_workflow_apply_request"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyRequestEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderExecutionReceiptReplayWorkflowApplyRequestEnvelopeFor(receiptReplayWorkflowApplyRequest); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply request envelope: %+v", envelope)
	}

	imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowApplyRequestEnvelope(expected)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply request envelope import error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, receiptReplayWorkflowApplyRequest) {
		t.Fatalf("unexpected imported structured edit provider execution receipt replay workflow apply request: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowApplyRequestEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_apply_request_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		envelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyRequestEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowApplyRequestEnvelope(envelope)
		if imported != nil {
			t.Fatalf("expected no structured edit provider execution receipt replay workflow apply request for rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider execution receipt replay workflow apply request import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply request rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowApplyRequestEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_apply_request_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyRequestEnvelope](t, fixture["structured_edit_provider_execution_receipt_replay_workflow_apply_request_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyRequest](t, fixture["expected_receipt_replay_workflow_apply_request"])

	imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowApplyRequestEnvelope(envelope)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply request envelope application error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected applied structured edit provider execution receipt replay workflow apply request: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyRequestEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		rejected, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowApplyRequestEnvelope(rejectedEnvelope)
		if rejected != nil {
			t.Fatalf("expected no structured edit provider execution receipt replay workflow apply request for application rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider execution receipt replay workflow apply request application import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply request application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyRequest(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_workflow_apply_request"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var batchReceiptReplayWorkflowApplyRequest StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyRequest
		if raw, err := json.Marshal(testCase["batch_receipt_replay_workflow_apply_request"]); err != nil {
			t.Fatalf("marshal structured edit provider batch execution receipt replay workflow apply request: %v", err)
		} else if err := json.Unmarshal(raw, &batchReceiptReplayWorkflowApplyRequest); err != nil {
			t.Fatalf("unmarshal structured edit provider batch execution receipt replay workflow apply request: %v", err)
		}

		roundtrip, err := json.Marshal(batchReceiptReplayWorkflowApplyRequest)
		if err != nil {
			t.Fatalf("marshal structured edit provider batch execution receipt replay workflow apply request roundtrip: %v", err)
		}

		var decoded StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyRequest
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider batch execution receipt replay workflow apply request: %v", err)
		}

		if !reflect.DeepEqual(decoded, batchReceiptReplayWorkflowApplyRequest) {
			t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply request roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyRequestEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_workflow_apply_request_envelope"))
	batchReceiptReplayWorkflowApplyRequest := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyRequest](t, fixture["structured_edit_provider_batch_execution_receipt_replay_workflow_apply_request"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyRequestEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyRequestEnvelopeFor(batchReceiptReplayWorkflowApplyRequest); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply request envelope: %+v", envelope)
	}

	imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyRequestEnvelope(expected)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply request envelope import error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, batchReceiptReplayWorkflowApplyRequest) {
		t.Fatalf("unexpected imported structured edit provider batch execution receipt replay workflow apply request: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyRequestEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_workflow_apply_request_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyRequestEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyRequestEnvelope(envelope)
		if imported != nil {
			t.Fatalf("expected no structured edit provider batch execution receipt replay workflow apply request for rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider batch execution receipt replay workflow apply request import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply request rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyRequestEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_workflow_apply_request_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyRequestEnvelope](t, fixture["structured_edit_provider_batch_execution_receipt_replay_workflow_apply_request_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyRequest](t, fixture["expected_batch_receipt_replay_workflow_apply_request"])

	imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyRequestEnvelope(envelope)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply request envelope application error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected applied structured edit provider batch execution receipt replay workflow apply request: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyRequestEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		rejected, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyRequestEnvelope(rejectedEnvelope)
		if rejected != nil {
			t.Fatalf("expected no structured edit provider batch execution receipt replay workflow apply request for application rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider batch execution receipt replay workflow apply request application import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply request application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplySession(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_workflow_apply_session"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var batchReceiptReplayWorkflowApplySession StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplySession
		if raw, err := json.Marshal(testCase["batch_receipt_replay_workflow_apply_session"]); err != nil {
			t.Fatalf("marshal structured edit provider batch execution receipt replay workflow apply session: %v", err)
		} else if err := json.Unmarshal(raw, &batchReceiptReplayWorkflowApplySession); err != nil {
			t.Fatalf("unmarshal structured edit provider batch execution receipt replay workflow apply session: %v", err)
		}

		roundtrip, err := json.Marshal(batchReceiptReplayWorkflowApplySession)
		if err != nil {
			t.Fatalf("marshal structured edit provider batch execution receipt replay workflow apply session roundtrip: %v", err)
		}

		var decoded StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplySession
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider batch execution receipt replay workflow apply session: %v", err)
		}

		if !reflect.DeepEqual(decoded, batchReceiptReplayWorkflowApplySession) {
			t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply session roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplySessionEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_workflow_apply_session_envelope"))
	batchReceiptReplayWorkflowApplySession := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplySession](t, fixture["structured_edit_provider_batch_execution_receipt_replay_workflow_apply_session"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplySessionEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplySessionEnvelopeFor(batchReceiptReplayWorkflowApplySession); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply session envelope: %+v", envelope)
	}

	imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplySessionEnvelope(expected)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply session envelope import error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, batchReceiptReplayWorkflowApplySession) {
		t.Fatalf("unexpected imported structured edit provider batch execution receipt replay workflow apply session: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplySessionEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_workflow_apply_session_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplySessionEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplySessionEnvelope(envelope)
		if imported != nil {
			t.Fatalf("expected no structured edit provider batch execution receipt replay workflow apply session for rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider batch execution receipt replay workflow apply session import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply session rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplySessionEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_workflow_apply_session_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplySessionEnvelope](t, fixture["structured_edit_provider_batch_execution_receipt_replay_workflow_apply_session_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplySession](t, fixture["expected_batch_receipt_replay_workflow_apply_session"])

	imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplySessionEnvelope(envelope)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply session envelope application error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected applied structured edit provider batch execution receipt replay workflow apply session: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplySessionEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		rejected, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplySessionEnvelope(rejectedEnvelope)
		if rejected != nil {
			t.Fatalf("expected no structured edit provider batch execution receipt replay workflow apply session for application rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider batch execution receipt replay workflow apply session application import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply session application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyResult(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_workflow_apply_result"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var batchReceiptReplayWorkflowApplyResult StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyResult
		if raw, err := json.Marshal(testCase["batch_receipt_replay_workflow_apply_result"]); err != nil {
			t.Fatalf("marshal structured edit provider batch execution receipt replay workflow apply result: %v", err)
		} else if err := json.Unmarshal(raw, &batchReceiptReplayWorkflowApplyResult); err != nil {
			t.Fatalf("unmarshal structured edit provider batch execution receipt replay workflow apply result: %v", err)
		}

		roundtrip, err := json.Marshal(batchReceiptReplayWorkflowApplyResult)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider batch execution receipt replay workflow apply result: %v", err)
		}
		var decoded StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyResult
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider batch execution receipt replay workflow apply result: %v", err)
		}

		if !reflect.DeepEqual(decoded, batchReceiptReplayWorkflowApplyResult) {
			t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply result roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyResultEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_workflow_apply_result_envelope"))
	batchReceiptReplayWorkflowApplyResult := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyResult](t, fixture["structured_edit_provider_batch_execution_receipt_replay_workflow_apply_result"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyResultEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyResultEnvelopeFor(batchReceiptReplayWorkflowApplyResult); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply result envelope: %+v", envelope)
	}

	imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyResultEnvelope(expected)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply result envelope import error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, batchReceiptReplayWorkflowApplyResult) {
		t.Fatalf("unexpected imported structured edit provider batch execution receipt replay workflow apply result: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyResultEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_workflow_apply_result_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyResultEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyResultEnvelope(envelope)
		if imported != nil {
			t.Fatalf("expected no structured edit provider batch execution receipt replay workflow apply result for rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider batch execution receipt replay workflow apply result import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply result rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyResultEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_workflow_apply_result_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyResultEnvelope](t, fixture["structured_edit_provider_batch_execution_receipt_replay_workflow_apply_result_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyResult](t, fixture["expected_batch_receipt_replay_workflow_apply_result"])

	imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyResultEnvelope(envelope)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply result envelope application error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected applied structured edit provider batch execution receipt replay workflow apply result: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyResultEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		rejected, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyResultEnvelope(rejectedEnvelope)
		if rejected != nil {
			t.Fatalf("expected no structured edit provider batch execution receipt replay workflow apply result for application rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider batch execution receipt replay workflow apply result application import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply result application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecision(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_apply_decision"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var receiptReplayWorkflowApplyDecision StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecision
		if raw, err := json.Marshal(testCase["receipt_replay_workflow_apply_decision"]); err != nil {
			t.Fatalf("marshal structured edit provider execution receipt replay workflow apply decision: %v", err)
		} else if err := json.Unmarshal(raw, &receiptReplayWorkflowApplyDecision); err != nil {
			t.Fatalf("unmarshal structured edit provider execution receipt replay workflow apply decision: %v", err)
		}

		roundtrip, err := json.Marshal(receiptReplayWorkflowApplyDecision)
		if err != nil {
			t.Fatalf("marshal structured edit provider execution receipt replay workflow apply decision roundtrip: %v", err)
		}

		var decoded StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecision
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider execution receipt replay workflow apply decision: %v", err)
		}

		if !reflect.DeepEqual(decoded, receiptReplayWorkflowApplyDecision) {
			t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply decision roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionOutcome(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_apply_decision_outcome"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var applyDecisionOutcome StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionOutcome
		if raw, err := json.Marshal(testCase["receipt_replay_workflow_apply_decision_outcome"]); err != nil {
			t.Fatalf("marshal structured edit provider execution receipt replay workflow apply decision outcome: %v", err)
		} else if err := json.Unmarshal(raw, &applyDecisionOutcome); err != nil {
			t.Fatalf("unmarshal structured edit provider execution receipt replay workflow apply decision outcome: %v", err)
		}

		roundtrip, err := json.Marshal(applyDecisionOutcome)
		if err != nil {
			t.Fatalf("marshal structured edit provider execution receipt replay workflow apply decision outcome roundtrip: %v", err)
		}

		var decoded StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionOutcome
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider execution receipt replay workflow apply decision outcome: %v", err)
		}

		if !reflect.DeepEqual(decoded, applyDecisionOutcome) {
			t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply decision outcome roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionSettlement(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_apply_decision_settlement"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var settlement StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionSettlement
		if raw, err := json.Marshal(testCase["receipt_replay_workflow_apply_decision_settlement"]); err != nil {
			t.Fatalf("marshal structured edit provider execution receipt replay workflow apply decision settlement: %v", err)
		} else if err := json.Unmarshal(raw, &settlement); err != nil {
			t.Fatalf("unmarshal structured edit provider execution receipt replay workflow apply decision settlement: %v", err)
		}

		roundtrip, err := json.Marshal(settlement)
		if err != nil {
			t.Fatalf("marshal structured edit provider execution receipt replay workflow apply decision settlement roundtrip: %v", err)
		}

		var decoded StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionSettlement
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider execution receipt replay workflow apply decision settlement: %v", err)
		}

		if !reflect.DeepEqual(decoded, settlement) {
			t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply decision settlement roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionConfirmation(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_apply_decision_confirmation"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var confirmation StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionConfirmation
		if raw, err := json.Marshal(testCase["receipt_replay_workflow_apply_decision_confirmation"]); err != nil {
			t.Fatalf("marshal structured edit provider execution receipt replay workflow apply decision confirmation: %v", err)
		} else if err := json.Unmarshal(raw, &confirmation); err != nil {
			t.Fatalf("unmarshal structured edit provider execution receipt replay workflow apply decision confirmation: %v", err)
		}

		roundtrip, err := json.Marshal(confirmation)
		if err != nil {
			t.Fatalf("marshal structured edit provider execution receipt replay workflow apply decision confirmation roundtrip: %v", err)
		}

		var decoded StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionConfirmation
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider execution receipt replay workflow apply decision confirmation: %v", err)
		}

		if !reflect.DeepEqual(decoded, confirmation) {
			t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply decision confirmation roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionConfirmationEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_apply_decision_confirmation_envelope"))
	confirmation := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionConfirmation](t, fixture["structured_edit_provider_execution_receipt_replay_workflow_apply_decision_confirmation"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionConfirmationEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionConfirmationEnvelopeFor(confirmation); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply decision confirmation envelope: %+v", envelope)
	}

	imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionConfirmationEnvelope(expected)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply decision confirmation envelope import error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, confirmation) {
		t.Fatalf("unexpected imported structured edit provider execution receipt replay workflow apply decision confirmation: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionConfirmationEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_apply_decision_confirmation_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionConfirmationEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionConfirmationEnvelope(envelope)
		if imported != nil {
			t.Fatalf("expected no structured edit provider execution receipt replay workflow apply decision confirmation for rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider execution receipt replay workflow apply decision confirmation import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply decision confirmation rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionConfirmationEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_apply_decision_confirmation_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionConfirmationEnvelope](t, fixture["structured_edit_provider_execution_receipt_replay_workflow_apply_decision_confirmation_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionConfirmation](t, fixture["expected_receipt_replay_workflow_apply_decision_confirmation"])

	imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionConfirmationEnvelope(envelope)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply decision confirmation envelope application error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected applied structured edit provider execution receipt replay workflow apply decision confirmation: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionConfirmationEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionConfirmationEnvelope(rejectedEnvelope)
		if imported != nil {
			t.Fatalf("expected no structured edit provider execution receipt replay workflow apply decision confirmation for application rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider execution receipt replay workflow apply decision confirmation envelope application error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply decision confirmation application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionConfirmation(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_workflow_apply_decision_confirmation"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var confirmation StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionConfirmation
		if raw, err := json.Marshal(testCase["batch_receipt_replay_workflow_apply_decision_confirmation"]); err != nil {
			t.Fatalf("marshal structured edit provider batch execution receipt replay workflow apply decision confirmation: %v", err)
		} else if err := json.Unmarshal(raw, &confirmation); err != nil {
			t.Fatalf("unmarshal structured edit provider batch execution receipt replay workflow apply decision confirmation: %v", err)
		}

		roundtrip, err := json.Marshal(confirmation)
		if err != nil {
			t.Fatalf("marshal structured edit provider batch execution receipt replay workflow apply decision confirmation roundtrip: %v", err)
		}

		var decoded StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionConfirmation
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider batch execution receipt replay workflow apply decision confirmation: %v", err)
		}

		if !reflect.DeepEqual(decoded, confirmation) {
			t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply decision confirmation roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionConfirmationEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_workflow_apply_decision_confirmation_envelope"))
	confirmation := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionConfirmation](t, fixture["batch_receipt_replay_workflow_apply_decision_confirmation"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionConfirmationEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionConfirmationEnvelopeFor(confirmation); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply decision confirmation envelope: %+v", envelope)
	}

	imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionConfirmationEnvelope(expected)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply decision confirmation envelope import error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, confirmation) {
		t.Fatalf("unexpected imported structured edit provider batch execution receipt replay workflow apply decision confirmation: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionConfirmationEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_workflow_apply_decision_confirmation_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionConfirmationEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionConfirmationEnvelope(envelope)
		if imported != nil {
			t.Fatalf("expected no structured edit provider batch execution receipt replay workflow apply decision confirmation for rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider batch execution receipt replay workflow apply decision confirmation import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply decision confirmation rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionConfirmationEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_workflow_apply_decision_confirmation_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionConfirmationEnvelope](t, fixture["batch_receipt_replay_workflow_apply_decision_confirmation_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionConfirmation](t, fixture["expected_batch_receipt_replay_workflow_apply_decision_confirmation"])

	imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionConfirmationEnvelope(envelope)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply decision confirmation envelope application error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected applied structured edit provider batch execution receipt replay workflow apply decision confirmation: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionConfirmationEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionConfirmationEnvelope(rejectedEnvelope)
		if imported != nil {
			t.Fatalf("expected no structured edit provider batch execution receipt replay workflow apply decision confirmation for application rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider batch execution receipt replay workflow apply decision confirmation envelope application error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply decision confirmation application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionClosureReport(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_apply_decision_closure_report"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var closureReport StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionClosureReport
		if raw, err := json.Marshal(testCase["receipt_replay_workflow_apply_decision_closure_report"]); err != nil {
			t.Fatalf("marshal structured edit provider execution receipt replay workflow apply decision closure report: %v", err)
		} else if err := json.Unmarshal(raw, &closureReport); err != nil {
			t.Fatalf("unmarshal structured edit provider execution receipt replay workflow apply decision closure report: %v", err)
		}

		roundtrip, err := json.Marshal(closureReport)
		if err != nil {
			t.Fatalf("marshal structured edit provider execution receipt replay workflow apply decision closure report roundtrip: %v", err)
		}

		var decoded StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionClosureReport
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider execution receipt replay workflow apply decision closure report: %v", err)
		}

		if !reflect.DeepEqual(decoded, closureReport) {
			t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply decision closure report roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionAuditRecord(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_apply_decision_audit_record"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var auditRecord StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionAuditRecord
		if raw, err := json.Marshal(testCase["receipt_replay_workflow_apply_decision_audit_record"]); err != nil {
			t.Fatalf("marshal structured edit provider execution receipt replay workflow apply decision audit record: %v", err)
		} else if err := json.Unmarshal(raw, &auditRecord); err != nil {
			t.Fatalf("unmarshal structured edit provider execution receipt replay workflow apply decision audit record: %v", err)
		}

		roundtrip, err := json.Marshal(auditRecord)
		if err != nil {
			t.Fatalf("marshal structured edit provider execution receipt replay workflow apply decision audit record roundtrip: %v", err)
		}

		var decoded StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionAuditRecord
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider execution receipt replay workflow apply decision audit record: %v", err)
		}

		if !reflect.DeepEqual(decoded, auditRecord) {
			t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply decision audit record roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionClosureReportEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_apply_decision_closure_report_envelope"))
	closureReport := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionClosureReport](t, fixture["structured_edit_provider_execution_receipt_replay_workflow_apply_decision_closure_report"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionClosureReportEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionClosureReportEnvelopeFor(closureReport); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply decision closure report envelope: %+v", envelope)
	}

	imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionClosureReportEnvelope(expected)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply decision closure report envelope import error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, closureReport) {
		t.Fatalf("unexpected imported structured edit provider execution receipt replay workflow apply decision closure report: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionClosureReportEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_apply_decision_closure_report_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionClosureReportEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionClosureReportEnvelope(envelope)
		if imported != nil {
			t.Fatalf("expected no structured edit provider execution receipt replay workflow apply decision closure report for rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider execution receipt replay workflow apply decision closure report import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply decision closure report rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionClosureReportEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_apply_decision_closure_report_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionClosureReportEnvelope](t, fixture["structured_edit_provider_execution_receipt_replay_workflow_apply_decision_closure_report_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionClosureReport](t, fixture["expected_receipt_replay_workflow_apply_decision_closure_report"])

	imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionClosureReportEnvelope(envelope)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply decision closure report envelope application error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected applied structured edit provider execution receipt replay workflow apply decision closure report: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionClosureReportEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionClosureReportEnvelope(rejectedEnvelope)
		if imported != nil {
			t.Fatalf("expected no structured edit provider execution receipt replay workflow apply decision closure report for application rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider execution receipt replay workflow apply decision closure report envelope application error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply decision closure report application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionClosureReport(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_workflow_apply_decision_closure_report"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var closureReport StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionClosureReport
		if raw, err := json.Marshal(testCase["batch_receipt_replay_workflow_apply_decision_closure_report"]); err != nil {
			t.Fatalf("marshal structured edit provider batch execution receipt replay workflow apply decision closure report: %v", err)
		} else if err := json.Unmarshal(raw, &closureReport); err != nil {
			t.Fatalf("unmarshal structured edit provider batch execution receipt replay workflow apply decision closure report: %v", err)
		}

		roundtrip, err := json.Marshal(closureReport)
		if err != nil {
			t.Fatalf("marshal structured edit provider batch execution receipt replay workflow apply decision closure report roundtrip: %v", err)
		}

		var decoded StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionClosureReport
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider batch execution receipt replay workflow apply decision closure report: %v", err)
		}

		if !reflect.DeepEqual(decoded, closureReport) {
			t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply decision closure report roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionClosureReportEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_workflow_apply_decision_closure_report_envelope"))
	closureReport := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionClosureReport](t, fixture["batch_receipt_replay_workflow_apply_decision_closure_report"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionClosureReportEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionClosureReportEnvelopeFor(closureReport); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply decision closure report envelope: %+v", envelope)
	}

	imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionClosureReportEnvelope(expected)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply decision closure report envelope import error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, closureReport) {
		t.Fatalf("unexpected imported structured edit provider batch execution receipt replay workflow apply decision closure report: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionClosureReportEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_workflow_apply_decision_closure_report_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionClosureReportEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionClosureReportEnvelope(envelope)
		if imported != nil {
			t.Fatalf("expected no structured edit provider batch execution receipt replay workflow apply decision closure report for rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider batch execution receipt replay workflow apply decision closure report import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply decision closure report rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionClosureReportEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_workflow_apply_decision_closure_report_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionClosureReportEnvelope](t, fixture["batch_receipt_replay_workflow_apply_decision_closure_report_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionClosureReport](t, fixture["expected_batch_receipt_replay_workflow_apply_decision_closure_report"])

	imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionClosureReportEnvelope(envelope)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply decision closure report envelope application error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected applied structured edit provider batch execution receipt replay workflow apply decision closure report: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionClosureReportEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionClosureReportEnvelope(rejectedEnvelope)
		if imported != nil {
			t.Fatalf("expected no structured edit provider batch execution receipt replay workflow apply decision closure report for application rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider batch execution receipt replay workflow apply decision closure report envelope application error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply decision closure report application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionSettlementEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_apply_decision_settlement_envelope"))
	settlement := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionSettlement](t, fixture["structured_edit_provider_execution_receipt_replay_workflow_apply_decision_settlement"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionSettlementEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionSettlementEnvelopeFor(settlement); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply decision settlement envelope: %+v", envelope)
	}

	imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionSettlementEnvelope(expected)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply decision settlement envelope import error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, settlement) {
		t.Fatalf("unexpected imported structured edit provider execution receipt replay workflow apply decision settlement: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionSettlementEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_apply_decision_settlement_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionSettlementEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionSettlementEnvelope(envelope)
		if imported != nil {
			t.Fatalf("expected no structured edit provider execution receipt replay workflow apply decision settlement for rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider execution receipt replay workflow apply decision settlement import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply decision settlement rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionSettlementEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_apply_decision_settlement_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionSettlementEnvelope](t, fixture["structured_edit_provider_execution_receipt_replay_workflow_apply_decision_settlement_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionSettlement](t, fixture["expected_receipt_replay_workflow_apply_decision_settlement"])

	imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionSettlementEnvelope(envelope)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply decision settlement envelope application error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected applied structured edit provider execution receipt replay workflow apply decision settlement: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionSettlementEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		rejected, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionSettlementEnvelope(rejectedEnvelope)
		if rejected != nil {
			t.Fatalf("expected no structured edit provider execution receipt replay workflow apply decision settlement for application rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider execution receipt replay workflow apply decision settlement application import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply decision settlement application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionOutcomeEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_apply_decision_outcome_envelope"))
	applyDecisionOutcome := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionOutcome](t, fixture["structured_edit_provider_execution_receipt_replay_workflow_apply_decision_outcome"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionOutcomeEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionOutcomeEnvelopeFor(applyDecisionOutcome); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply decision outcome envelope: %+v", envelope)
	}

	imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionOutcomeEnvelope(expected)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply decision outcome envelope import error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, applyDecisionOutcome) {
		t.Fatalf("unexpected imported structured edit provider execution receipt replay workflow apply decision outcome: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionOutcomeEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_apply_decision_outcome_envelope_rejection"))

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionOutcomeEnvelope](t, testCase["envelope"])
		expectedErr := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionOutcomeEnvelope(envelope)
		if imported != nil {
			t.Fatalf("expected no structured edit provider execution receipt replay workflow apply decision outcome for rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider execution receipt replay workflow apply decision outcome import error for %s", testCase["label"])
		}
		if !reflect.DeepEqual(*importErr, expectedErr) {
			t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply decision outcome rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionOutcomeEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_apply_decision_outcome_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionOutcomeEnvelope](t, fixture["structured_edit_provider_execution_receipt_replay_workflow_apply_decision_outcome_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionOutcome](t, fixture["expected_receipt_replay_workflow_apply_decision_outcome"])

	imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionOutcomeEnvelope(envelope)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply decision outcome envelope application error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected applied structured edit provider execution receipt replay workflow apply decision outcome: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionOutcomeEnvelope](t, testCase["envelope"])
		expectedErr := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		rejected, rejectionErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionOutcomeEnvelope(rejectedEnvelope)
		if rejected != nil {
			t.Fatalf("expected no structured edit provider execution receipt replay workflow apply decision outcome for application rejection %s", testCase["label"])
		}
		if rejectionErr == nil {
			t.Fatalf("expected structured edit provider execution receipt replay workflow apply decision outcome application import error for %s", testCase["label"])
		}
		if !reflect.DeepEqual(*rejectionErr, expectedErr) {
			t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply decision outcome application rejection for %s: %+v", testCase["label"], *rejectionErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_apply_decision_envelope"))
	receiptReplayWorkflowApplyDecision := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecision](t, fixture["structured_edit_provider_execution_receipt_replay_workflow_apply_decision"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionEnvelopeFor(receiptReplayWorkflowApplyDecision); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply decision envelope: %+v", envelope)
	}

	imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionEnvelope(expected)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply decision envelope import error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, receiptReplayWorkflowApplyDecision) {
		t.Fatalf("unexpected imported structured edit provider execution receipt replay workflow apply decision: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_apply_decision_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		envelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionEnvelope(envelope)
		if imported != nil {
			t.Fatalf("expected no structured edit provider execution receipt replay workflow apply decision for rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider execution receipt replay workflow apply decision import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply decision rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_apply_decision_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionEnvelope](t, fixture["structured_edit_provider_execution_receipt_replay_workflow_apply_decision_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecision](t, fixture["expected_receipt_replay_workflow_apply_decision"])

	imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionEnvelope(envelope)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply decision envelope application error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected applied structured edit provider execution receipt replay workflow apply decision: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		rejected, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowApplyDecisionEnvelope(rejectedEnvelope)
		if rejected != nil {
			t.Fatalf("expected no structured edit provider execution receipt replay workflow apply decision for application rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider execution receipt replay workflow apply decision application import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider execution receipt replay workflow apply decision application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecision(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_workflow_apply_decision"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var batchReceiptReplayWorkflowApplyDecision StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecision
		if raw, err := json.Marshal(testCase["batch_receipt_replay_workflow_apply_decision"]); err != nil {
			t.Fatalf("marshal structured edit provider batch execution receipt replay workflow apply decision: %v", err)
		} else if err := json.Unmarshal(raw, &batchReceiptReplayWorkflowApplyDecision); err != nil {
			t.Fatalf("unmarshal structured edit provider batch execution receipt replay workflow apply decision: %v", err)
		}

		roundtrip, err := json.Marshal(batchReceiptReplayWorkflowApplyDecision)
		if err != nil {
			t.Fatalf("marshal structured edit provider batch execution receipt replay workflow apply decision roundtrip: %v", err)
		}

		var decoded StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecision
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider batch execution receipt replay workflow apply decision: %v", err)
		}

		if !reflect.DeepEqual(decoded, batchReceiptReplayWorkflowApplyDecision) {
			t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply decision roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_workflow_apply_decision_envelope"))
	batchReceiptReplayWorkflowApplyDecision := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecision](t, fixture["structured_edit_provider_batch_execution_receipt_replay_workflow_apply_decision"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionEnvelopeFor(batchReceiptReplayWorkflowApplyDecision); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply decision envelope: %+v", envelope)
	}

	imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionEnvelope(expected)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply decision envelope import error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, batchReceiptReplayWorkflowApplyDecision) {
		t.Fatalf("unexpected imported structured edit provider batch execution receipt replay workflow apply decision: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_workflow_apply_decision_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionEnvelope(envelope)
		if imported != nil {
			t.Fatalf("expected no structured edit provider batch execution receipt replay workflow apply decision for rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider batch execution receipt replay workflow apply decision import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply decision rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_workflow_apply_decision_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionEnvelope](t, fixture["structured_edit_provider_batch_execution_receipt_replay_workflow_apply_decision_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecision](t, fixture["expected_batch_receipt_replay_workflow_apply_decision"])

	imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionEnvelope(envelope)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply decision envelope application error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected applied structured edit provider batch execution receipt replay workflow apply decision: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		rejected, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionEnvelope(rejectedEnvelope)
		if rejected != nil {
			t.Fatalf("expected no structured edit provider batch execution receipt replay workflow apply decision for application rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider batch execution receipt replay workflow apply decision application import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply decision application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionOutcome(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_workflow_apply_decision_outcome"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var batchOutcome StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionOutcome
		if raw, err := json.Marshal(testCase["batch_receipt_replay_workflow_apply_decision_outcome"]); err != nil {
			t.Fatalf("marshal structured edit provider batch execution receipt replay workflow apply decision outcome: %v", err)
		} else if err := json.Unmarshal(raw, &batchOutcome); err != nil {
			t.Fatalf("unmarshal structured edit provider batch execution receipt replay workflow apply decision outcome: %v", err)
		}

		roundtrip, err := json.Marshal(batchOutcome)
		if err != nil {
			t.Fatalf("marshal structured edit provider batch execution receipt replay workflow apply decision outcome roundtrip: %v", err)
		}

		var decoded StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionOutcome
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider batch execution receipt replay workflow apply decision outcome: %v", err)
		}

		if !reflect.DeepEqual(decoded, batchOutcome) {
			t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply decision outcome roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionOutcomeEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_workflow_apply_decision_outcome_envelope"))
	batchOutcome := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionOutcome](t, fixture["structured_edit_provider_batch_execution_receipt_replay_workflow_apply_decision_outcome"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionOutcomeEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionOutcomeEnvelopeFor(batchOutcome); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply decision outcome envelope: %+v", envelope)
	}

	imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionOutcomeEnvelope(expected)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply decision outcome envelope import error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, batchOutcome) {
		t.Fatalf("unexpected imported structured edit provider batch execution receipt replay workflow apply decision outcome: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionOutcomeEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_workflow_apply_decision_outcome_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionOutcomeEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionOutcomeEnvelope(envelope)
		if imported != nil {
			t.Fatalf("expected no structured edit provider batch execution receipt replay workflow apply decision outcome for rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider batch execution receipt replay workflow apply decision outcome import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply decision outcome rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionOutcomeEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_workflow_apply_decision_outcome_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionOutcomeEnvelope](t, fixture["structured_edit_provider_batch_execution_receipt_replay_workflow_apply_decision_outcome_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionOutcome](t, fixture["expected_batch_receipt_replay_workflow_apply_decision_outcome"])

	imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionOutcomeEnvelope(envelope)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply decision outcome envelope application error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected applied structured edit provider batch execution receipt replay workflow apply decision outcome: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionOutcomeEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		rejected, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayWorkflowApplyDecisionOutcomeEnvelope(rejectedEnvelope)
		if rejected != nil {
			t.Fatalf("expected no structured edit provider batch execution receipt replay workflow apply decision outcome for application rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider batch execution receipt replay workflow apply decision outcome application import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow apply decision outcome application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowReviewRequestEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_review_request_envelope"))
	receiptReplayWorkflowReviewRequest := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowReviewRequest](t, fixture["structured_edit_provider_execution_receipt_replay_workflow_review_request"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowReviewRequestEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderExecutionReceiptReplayWorkflowReviewRequestEnvelopeFor(receiptReplayWorkflowReviewRequest); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider execution receipt replay workflow review request envelope: %+v", envelope)
	}

	imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowReviewRequestEnvelope(expected)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider execution receipt replay workflow review request envelope import error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, receiptReplayWorkflowReviewRequest) {
		t.Fatalf("unexpected imported structured edit provider execution receipt replay workflow review request: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowReviewRequestEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_review_request_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		envelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowReviewRequestEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowReviewRequestEnvelope(envelope)
		if imported != nil {
			t.Fatalf("expected no structured edit provider execution receipt replay workflow review request for rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider execution receipt replay workflow review request import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider execution receipt replay workflow review request rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionReceiptReplayWorkflowReviewRequestEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_receipt_replay_workflow_review_request_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowReviewRequestEnvelope](t, fixture["structured_edit_provider_execution_receipt_replay_workflow_review_request_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowReviewRequest](t, fixture["expected_receipt_replay_workflow_review_request"])

	imported, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowReviewRequestEnvelope(envelope)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider execution receipt replay workflow review request envelope application error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected applied structured edit provider execution receipt replay workflow review request: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderExecutionReceiptReplayWorkflowReviewRequestEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		rejected, importErr := ImportStructuredEditProviderExecutionReceiptReplayWorkflowReviewRequestEnvelope(rejectedEnvelope)
		if rejected != nil {
			t.Fatalf("expected no structured edit provider execution receipt replay workflow review request for application rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider execution receipt replay workflow review request application import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider execution receipt replay workflow review request application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayWorkflowReviewRequest(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_workflow_review_request"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var batchReceiptReplayWorkflowReviewRequest StructuredEditProviderBatchExecutionReceiptReplayWorkflowReviewRequest
		if raw, err := json.Marshal(testCase["batch_receipt_replay_workflow_review_request"]); err != nil {
			t.Fatalf("marshal structured edit provider batch execution receipt replay workflow review request: %v", err)
		} else if err := json.Unmarshal(raw, &batchReceiptReplayWorkflowReviewRequest); err != nil {
			t.Fatalf("unmarshal structured edit provider batch execution receipt replay workflow review request: %v", err)
		}

		roundtrip, err := json.Marshal(batchReceiptReplayWorkflowReviewRequest)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider batch execution receipt replay workflow review request: %v", err)
		}
		var decoded StructuredEditProviderBatchExecutionReceiptReplayWorkflowReviewRequest
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider batch execution receipt replay workflow review request: %v", err)
		}

		if !reflect.DeepEqual(decoded, batchReceiptReplayWorkflowReviewRequest) {
			t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow review request roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayWorkflowReviewRequestEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_workflow_review_request_envelope"))
	batchReceiptReplayWorkflowReviewRequest := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowReviewRequest](t, fixture["structured_edit_provider_batch_execution_receipt_replay_workflow_review_request"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowReviewRequestEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderBatchExecutionReceiptReplayWorkflowReviewRequestEnvelopeFor(batchReceiptReplayWorkflowReviewRequest); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow review request envelope: %+v", envelope)
	}

	imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayWorkflowReviewRequestEnvelope(expected)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow review request envelope import error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, batchReceiptReplayWorkflowReviewRequest) {
		t.Fatalf("unexpected imported structured edit provider batch execution receipt replay workflow review request: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayWorkflowReviewRequestEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_workflow_review_request_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowReviewRequestEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayWorkflowReviewRequestEnvelope(envelope)
		if imported != nil {
			t.Fatalf("expected no structured edit provider batch execution receipt replay workflow review request for rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider batch execution receipt replay workflow review request import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow review request rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReceiptReplayWorkflowReviewRequestEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_receipt_replay_workflow_review_request_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowReviewRequestEnvelope](t, fixture["structured_edit_provider_batch_execution_receipt_replay_workflow_review_request_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowReviewRequest](t, fixture["expected_batch_receipt_replay_workflow_review_request"])

	imported, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayWorkflowReviewRequestEnvelope(envelope)
	if importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow review request envelope application error: %+v", *importErr)
	}
	if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected applied structured edit provider batch execution receipt replay workflow review request: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReceiptReplayWorkflowReviewRequestEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		rejected, importErr := ImportStructuredEditProviderBatchExecutionReceiptReplayWorkflowReviewRequestEnvelope(rejectedEnvelope)
		if rejected != nil {
			t.Fatalf("expected no structured edit provider batch execution receipt replay workflow review request for application rejection %s", testCase["label"])
		}
		if importErr == nil {
			t.Fatalf("expected structured edit provider batch execution receipt replay workflow review request application import error for %s", testCase["label"])
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider batch execution receipt replay workflow review request application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionHandoff(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_handoff"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var batchExecutionHandoff StructuredEditProviderBatchExecutionHandoff
		if raw, err := json.Marshal(testCase["batch_execution_handoff"]); err != nil {
			t.Fatalf("marshal structured edit provider batch execution handoff: %v", err)
		} else if err := json.Unmarshal(raw, &batchExecutionHandoff); err != nil {
			t.Fatalf("unmarshal structured edit provider batch execution handoff: %v", err)
		}

		roundtrip, err := json.Marshal(batchExecutionHandoff)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider batch execution handoff: %v", err)
		}
		var decoded StructuredEditProviderBatchExecutionHandoff
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider batch execution handoff: %v", err)
		}

		if !reflect.DeepEqual(decoded, batchExecutionHandoff) {
			t.Fatalf("unexpected structured edit provider batch execution handoff roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionHandoffEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_handoff_envelope"))
	batchExecutionHandoff := decodeFixtureValue[StructuredEditProviderBatchExecutionHandoff](t, fixture["structured_edit_provider_batch_execution_handoff"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionHandoffEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderBatchExecutionHandoffEnvelopeFor(batchExecutionHandoff); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider batch execution handoff envelope: %+v", envelope)
	}

	if imported, importErr := ImportStructuredEditProviderBatchExecutionHandoffEnvelope(expected); importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution handoff envelope import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, batchExecutionHandoff) {
		t.Fatalf("unexpected structured edit provider batch execution handoff envelope import: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionHandoffEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_handoff_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionHandoffEnvelope](t, testCase["envelope"])
		expected := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderBatchExecutionHandoffEnvelope(envelope); importErr == nil {
			t.Fatalf("expected structured edit provider batch execution handoff envelope rejection, got batch execution handoff: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expected) {
			t.Fatalf("unexpected structured edit provider batch execution handoff envelope rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionHandoffEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_handoff_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionHandoffEnvelope](t, fixture["structured_edit_provider_batch_execution_handoff_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionHandoff](t, fixture["expected_batch_execution_handoff"])

	if imported, importErr := ImportStructuredEditProviderBatchExecutionHandoffEnvelope(envelope); importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution handoff envelope application import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected structured edit provider batch execution handoff envelope application: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderBatchExecutionHandoffEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderBatchExecutionHandoffEnvelope(rejectedEnvelope); importErr == nil {
			t.Fatalf("expected structured edit provider batch execution handoff envelope application rejection, got batch execution handoff: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider batch execution handoff envelope application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionPlanEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_plan_envelope"))
	executionPlan := decodeFixtureValue[StructuredEditProviderExecutionPlan](t, fixture["structured_edit_provider_execution_plan"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionPlanEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderExecutionPlanEnvelopeFor(executionPlan); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider execution plan envelope: %+v", envelope)
	}

	if imported, importErr := ImportStructuredEditProviderExecutionPlanEnvelope(expected); importErr != nil {
		t.Fatalf("unexpected structured edit provider execution plan envelope import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, executionPlan) {
		t.Fatalf("unexpected structured edit provider execution plan envelope import: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderExecutionPlanEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_plan_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeFixtureValue[StructuredEditProviderExecutionPlanEnvelope](t, testCase["envelope"])
		expected := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderExecutionPlanEnvelope(envelope); importErr == nil {
			t.Fatalf("expected structured edit provider execution plan envelope rejection, got execution plan: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expected) {
			t.Fatalf("unexpected structured edit provider execution plan envelope rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionPlanEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_plan_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderExecutionPlanEnvelope](t, fixture["structured_edit_provider_execution_plan_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionPlan](t, fixture["expected_execution_plan"])

	if imported, importErr := ImportStructuredEditProviderExecutionPlanEnvelope(envelope); importErr != nil {
		t.Fatalf("unexpected structured edit provider execution plan envelope application import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected structured edit provider execution plan envelope application: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderExecutionPlanEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderExecutionPlanEnvelope(rejectedEnvelope); importErr == nil {
			t.Fatalf("expected structured edit provider execution plan envelope application rejection, got execution plan: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider execution plan envelope application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionPlan(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_plan"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var batchExecutionPlan StructuredEditProviderBatchExecutionPlan
		if raw, err := json.Marshal(testCase["batch_execution_plan"]); err != nil {
			t.Fatalf("marshal structured edit provider batch execution plan: %v", err)
		} else if err := json.Unmarshal(raw, &batchExecutionPlan); err != nil {
			t.Fatalf("unmarshal structured edit provider batch execution plan: %v", err)
		}

		roundtrip, err := json.Marshal(batchExecutionPlan)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider batch execution plan: %v", err)
		}
		var decoded StructuredEditProviderBatchExecutionPlan
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider batch execution plan: %v", err)
		}

		if !reflect.DeepEqual(decoded, batchExecutionPlan) {
			t.Fatalf("unexpected structured edit provider batch execution plan roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionPlanEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_plan_envelope"))
	batchExecutionPlan := decodeFixtureValue[StructuredEditProviderBatchExecutionPlan](t, fixture["structured_edit_provider_batch_execution_plan"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionPlanEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderBatchExecutionPlanEnvelopeFor(batchExecutionPlan); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider batch execution plan envelope: %+v", envelope)
	}

	if imported, importErr := ImportStructuredEditProviderBatchExecutionPlanEnvelope(expected); importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution plan envelope import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, batchExecutionPlan) {
		t.Fatalf("unexpected structured edit provider batch execution plan envelope import: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionPlanEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_plan_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionPlanEnvelope](t, testCase["envelope"])
		expected := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderBatchExecutionPlanEnvelope(envelope); importErr == nil {
			t.Fatalf("expected structured edit provider batch execution plan envelope rejection, got batch execution plan: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expected) {
			t.Fatalf("unexpected structured edit provider batch execution plan envelope rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionPlanEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_plan_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionPlanEnvelope](t, fixture["structured_edit_provider_batch_execution_plan_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionPlan](t, fixture["expected_batch_execution_plan"])

	if imported, importErr := ImportStructuredEditProviderBatchExecutionPlanEnvelope(envelope); importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution plan envelope application import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected structured edit provider batch execution plan envelope application: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderBatchExecutionPlanEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderBatchExecutionPlanEnvelope(rejectedEnvelope); importErr == nil {
			t.Fatalf("expected structured edit provider batch execution plan envelope application rejection, got batch execution plan: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider batch execution plan envelope application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionApplicationEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_application_envelope"))
	application := decodeFixtureValue[StructuredEditProviderExecutionApplication](t, fixture["structured_edit_provider_execution_application"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionApplicationEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderExecutionApplicationEnvelopeFor(application); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider execution application envelope: %+v", envelope)
	}

	if imported, importErr := ImportStructuredEditProviderExecutionApplicationEnvelope(expected); importErr != nil {
		t.Fatalf("unexpected structured edit provider execution application envelope import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, application) {
		t.Fatalf("unexpected structured edit provider execution application envelope import: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderExecutionApplicationEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_application_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeFixtureValue[StructuredEditProviderExecutionApplicationEnvelope](t, testCase["envelope"])
		expected := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderExecutionApplicationEnvelope(envelope); importErr == nil {
			t.Fatalf("expected structured edit provider execution application envelope rejection, got application: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expected) {
			t.Fatalf("unexpected structured edit provider execution application envelope rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderExecutionApplicationEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_execution_application_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderExecutionApplicationEnvelope](t, fixture["structured_edit_provider_execution_application_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderExecutionApplication](t, fixture["expected_application"])

	if imported, importErr := ImportStructuredEditProviderExecutionApplicationEnvelope(envelope); importErr != nil {
		t.Fatalf("unexpected structured edit provider execution application envelope application import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected structured edit provider execution application envelope application: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderExecutionApplicationEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderExecutionApplicationEnvelope(rejectedEnvelope); importErr == nil {
			t.Fatalf("expected structured edit provider execution application envelope application rejection, got application: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider execution application envelope application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditCrisprOvermatchFailClosed(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_crispr_overmatch_fail_closed"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var report StructuredEditExecutionReport
		if raw, err := json.Marshal(testCase["report"]); err != nil {
			t.Fatalf("marshal structured edit crispr overmatch fail closed report: %v", err)
		} else if err := json.Unmarshal(raw, &report); err != nil {
			t.Fatalf("unmarshal structured edit crispr overmatch fail closed report: %v", err)
		}

		roundtrip, err := json.Marshal(report)
		if err != nil {
			t.Fatalf("marshal structured edit crispr overmatch fail closed report roundtrip: %v", err)
		}

		var decoded StructuredEditExecutionReport
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit crispr overmatch fail closed report: %v", err)
		}

		if !reflect.DeepEqual(decoded, report) {
			t.Fatalf("unexpected structured edit crispr overmatch fail closed report roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditCrisprAcceptanceScenario(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_crispr_acceptance_scenario"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var report StructuredEditExecutionReport
		if raw, err := json.Marshal(testCase["report"]); err != nil {
			t.Fatalf("marshal structured edit crispr acceptance scenario report: %v", err)
		} else if err := json.Unmarshal(raw, &report); err != nil {
			t.Fatalf("unmarshal structured edit crispr acceptance scenario report: %v", err)
		}

		roundtrip, err := json.Marshal(report)
		if err != nil {
			t.Fatalf("marshal structured edit crispr acceptance scenario report roundtrip: %v", err)
		}

		var decoded StructuredEditExecutionReport
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit crispr acceptance scenario report: %v", err)
		}

		if !reflect.DeepEqual(decoded, report) {
			t.Fatalf("unexpected structured edit crispr acceptance scenario report roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditCrisprAppendFallbackInsert(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_crispr_append_fallback_insert"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var report StructuredEditExecutionReport
		if raw, err := json.Marshal(testCase["report"]); err != nil {
			t.Fatalf("marshal structured edit crispr append fallback insert report: %v", err)
		} else if err := json.Unmarshal(raw, &report); err != nil {
			t.Fatalf("unmarshal structured edit crispr append fallback insert report: %v", err)
		}

		roundtrip, err := json.Marshal(report)
		if err != nil {
			t.Fatalf("marshal structured edit crispr append fallback insert report roundtrip: %v", err)
		}

		var decoded StructuredEditExecutionReport
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit crispr append fallback insert report: %v", err)
		}

		if !reflect.DeepEqual(decoded, report) {
			t.Fatalf("unexpected structured edit crispr append fallback insert report roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditCrisprRubyCommentOwnedRewriteDeleteParity(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_crispr_ruby_comment_owned_rewrite_delete_parity"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var report StructuredEditExecutionReport
		if raw, err := json.Marshal(testCase["report"]); err != nil {
			t.Fatalf("marshal structured edit crispr ruby comment owned rewrite delete parity report: %v", err)
		} else if err := json.Unmarshal(raw, &report); err != nil {
			t.Fatalf("unmarshal structured edit crispr ruby comment owned rewrite delete parity report: %v", err)
		}

		roundtrip, err := json.Marshal(report)
		if err != nil {
			t.Fatalf("marshal structured edit crispr ruby comment owned rewrite delete parity report roundtrip: %v", err)
		}

		var decoded StructuredEditExecutionReport
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit crispr ruby comment owned rewrite delete parity report: %v", err)
		}

		if !reflect.DeepEqual(decoded, report) {
			t.Fatalf("unexpected structured edit crispr ruby comment owned rewrite delete parity report roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditCrisprRubyCallableDestinationMoveParity(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_crispr_ruby_callable_destination_move_parity"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var report StructuredEditExecutionReport
		if raw, err := json.Marshal(testCase["report"]); err != nil {
			t.Fatalf("marshal structured edit crispr ruby callable destination move parity report: %v", err)
		} else if err := json.Unmarshal(raw, &report); err != nil {
			t.Fatalf("unmarshal structured edit crispr ruby callable destination move parity report: %v", err)
		}

		roundtrip, err := json.Marshal(report)
		if err != nil {
			t.Fatalf("marshal structured edit crispr ruby callable destination move parity report roundtrip: %v", err)
		}

		var decoded StructuredEditExecutionReport
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit crispr ruby callable destination move parity report: %v", err)
		}

		if !reflect.DeepEqual(decoded, report) {
			t.Fatalf("unexpected structured edit crispr ruby callable destination move parity report roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditCrisprMarkdownHeadingSectionReplaceParity(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_crispr_markdown_heading_section_replace_parity"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var report StructuredEditExecutionReport
		if raw, err := json.Marshal(testCase["report"]); err != nil {
			t.Fatalf("marshal structured edit crispr markdown heading section replace parity report: %v", err)
		} else if err := json.Unmarshal(raw, &report); err != nil {
			t.Fatalf("unmarshal structured edit crispr markdown heading section replace parity report: %v", err)
		}

		roundtrip, err := json.Marshal(report)
		if err != nil {
			t.Fatalf("marshal structured edit crispr markdown heading section replace parity report roundtrip: %v", err)
		}

		var decoded StructuredEditExecutionReport
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit crispr markdown heading section replace parity report: %v", err)
		}

		if !reflect.DeepEqual(decoded, report) {
			t.Fatalf("unexpected structured edit crispr markdown heading section replace parity report roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditCrisprExampleParityReport(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_crispr_example_parity_report"))

	var report StructuredEditCrisprExampleParityReport
	if raw, err := json.Marshal(fixture["report"]); err != nil {
		t.Fatalf("marshal structured edit crispr example parity report: %v", err)
	} else if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("unmarshal structured edit crispr example parity report: %v", err)
	}

	roundtrip, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal structured edit crispr example parity report roundtrip: %v", err)
	}

	var decoded StructuredEditCrisprExampleParityReport
	if err := json.Unmarshal(roundtrip, &decoded); err != nil {
		t.Fatalf("unmarshal roundtrip structured edit crispr example parity report: %v", err)
	}

	if !reflect.DeepEqual(decoded, report) {
		t.Fatalf("unexpected structured edit crispr example parity report roundtrip: %+v", decoded)
	}
}

func TestSharedFixtureStructuredEditCrisprParitySubstrateReport(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_crispr_parity_substrate_report"))

	var report StructuredEditCrisprExampleParityReport
	if raw, err := json.Marshal(fixture["report"]); err != nil {
		t.Fatalf("marshal structured edit crispr parity substrate report: %v", err)
	} else if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("unmarshal structured edit crispr parity substrate report: %v", err)
	}

	roundtrip, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal structured edit crispr parity substrate report roundtrip: %v", err)
	}

	var decoded StructuredEditCrisprExampleParityReport
	if err := json.Unmarshal(roundtrip, &decoded); err != nil {
		t.Fatalf("unmarshal roundtrip structured edit crispr parity substrate report: %v", err)
	}

	if !reflect.DeepEqual(decoded, report) {
		t.Fatalf("unexpected structured edit crispr parity substrate report roundtrip: %+v", decoded)
	}
}

func TestSharedFixtureStructuredEditKettleJemPrimitiveGapReport(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_kettle_jem_primitive_gap_report"))

	var report StructuredEditKettleJemPrimitiveGapReport
	if raw, err := json.Marshal(fixture["report"]); err != nil {
		t.Fatalf("marshal structured edit kettle-jem primitive gap report: %v", err)
	} else if err := json.Unmarshal(raw, &report); err != nil {
		t.Fatalf("unmarshal structured edit kettle-jem primitive gap report: %v", err)
	}

	roundtrip, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal structured edit kettle-jem primitive gap report roundtrip: %v", err)
	}

	var decoded StructuredEditKettleJemPrimitiveGapReport
	if err := json.Unmarshal(roundtrip, &decoded); err != nil {
		t.Fatalf("unmarshal roundtrip structured edit kettle-jem primitive gap report: %v", err)
	}

	if !reflect.DeepEqual(decoded, report) {
		t.Fatalf("unexpected structured edit kettle-jem primitive gap report roundtrip: %+v", decoded)
	}
}

func TestSharedFixtureContentRecipeExecutionEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "content_recipe_execution_envelope"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var requestEnvelope ContentRecipeExecutionRequestEnvelope
		if raw, err := json.Marshal(testCase["request_envelope"]); err != nil {
			t.Fatalf("marshal content recipe execution request envelope: %v", err)
		} else if err := json.Unmarshal(raw, &requestEnvelope); err != nil {
			t.Fatalf("unmarshal content recipe execution request envelope: %v", err)
		}

		var reportEnvelope ContentRecipeExecutionReportEnvelope
		if raw, err := json.Marshal(testCase["report_envelope"]); err != nil {
			t.Fatalf("marshal content recipe execution report envelope: %v", err)
		} else if err := json.Unmarshal(raw, &reportEnvelope); err != nil {
			t.Fatalf("unmarshal content recipe execution report envelope: %v", err)
		}

		if reportEnvelope.Report.FinalContent == reportEnvelope.Report.Request.DestinationContent {
			t.Fatalf("expected content recipe report to carry changed final content")
		}
		if len(reportEnvelope.Report.StepReports) != len(reportEnvelope.Report.Request.Steps) {
			t.Fatalf("expected one step report per request step")
		}
	}
}

func TestSharedFixtureSingleFileReadmeHeadingSectionAcceptance(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "single_file_readme_heading_section_acceptance"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var requestEnvelope ContentRecipeExecutionRequestEnvelope
		if raw, err := json.Marshal(testCase["request_envelope"]); err != nil {
			t.Fatalf("marshal README acceptance request envelope: %v", err)
		} else if err := json.Unmarshal(raw, &requestEnvelope); err != nil {
			t.Fatalf("unmarshal README acceptance request envelope: %v", err)
		}

		var reportEnvelope ContentRecipeExecutionReportEnvelope
		if raw, err := json.Marshal(testCase["report_envelope"]); err != nil {
			t.Fatalf("marshal README acceptance report envelope: %v", err)
		} else if err := json.Unmarshal(raw, &reportEnvelope); err != nil {
			t.Fatalf("unmarshal README acceptance report envelope: %v", err)
		}

		if reportEnvelope.Report.FinalContent == "" {
			t.Fatalf("expected README acceptance final content")
		}
		if len(reportEnvelope.Report.StepReports) != len(reportEnvelope.Report.Request.Steps) {
			t.Fatalf("expected README acceptance to include one step report per request step")
		}
	}
}

func TestSharedFixtureNativeStructuredEditRecipeSteps(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "native_structured_edit_recipe_steps"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var reportEnvelope ContentRecipeExecutionReportEnvelope
		if raw, err := json.Marshal(testCase["report_envelope"]); err != nil {
			t.Fatalf("marshal native structured edit recipe steps report envelope: %v", err)
		} else if err := json.Unmarshal(raw, &reportEnvelope); err != nil {
			t.Fatalf("unmarshal native structured edit recipe steps report envelope: %v", err)
		}

		kinds := []string{}
		for _, step := range reportEnvelope.Report.StepReports {
			if step.Application == nil {
				t.Fatalf("expected structured edit step report to include application")
			}
			kinds = append(kinds, step.Application.Request.OperationKind)
		}
		if !reflect.DeepEqual(kinds, []string{"replace", "insert", "delete"}) {
			t.Fatalf("unexpected structured edit recipe operation order: %v", kinds)
		}
	}
}

func TestSharedFixtureRubyGemfileSignatureMergeAcceptance(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "ruby_gemfile_signature_merge_acceptance"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		label := testCase["label"].(string)

		var reportEnvelope ContentRecipeExecutionReportEnvelope
		if raw, err := json.Marshal(testCase["report_envelope"]); err != nil {
			t.Fatalf("marshal Ruby Gemfile signature merge report envelope: %v", err)
		} else if err := json.Unmarshal(raw, &reportEnvelope); err != nil {
			t.Fatalf("unmarshal Ruby Gemfile signature merge report envelope: %v", err)
		}

		step := reportEnvelope.Report.Request.Steps[0]
		if step.MergeProfile["signature_profile"] != "gemfile_declarations" {
			t.Fatalf("expected gemfile_declarations signature profile")
		}
		if label == "gemfile-cross-nesting-duplicates-fail-closed" && reportEnvelope.Report.Changed {
			t.Fatalf("cross-nesting duplicate case should fail closed without changes")
		}
	}
}

func TestSharedFixtureRubyGemspecNativeBoundaryReport(t *testing.T) {
	report := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "ruby_gemspec_native_boundary_report"))
	if report["kind"] != "ruby_gemspec_native_boundary_report" {
		t.Fatalf("unexpected gemspec native boundary report kind: %v", report["kind"])
	}

	nativeSurface := report["native_recipe_surface"].(map[string]any)
	if nativeSurface["signature_profile"] != "gemspec_declarations" {
		t.Fatalf("expected gemspec_declarations signature profile")
	}

	wrapperNames := map[string]bool{}
	for _, rawBehavior := range report["wrapper_required_behaviors"].([]any) {
		behavior := rawBehavior.(map[string]any)
		wrapperNames[behavior["name"].(string)] = true
	}
	if !wrapperNames["dependency_ruby_floor_comment_alignment"] {
		t.Fatalf("expected resolver-backed dependency floor comment alignment to require wrapper")
	}
}

func TestSharedFixtureRubyGemspecSignatureMergeAcceptance(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "ruby_gemspec_signature_merge_acceptance"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var reportEnvelope ContentRecipeExecutionReportEnvelope
		if raw, err := json.Marshal(testCase["report_envelope"]); err != nil {
			t.Fatalf("marshal Ruby gemspec signature merge report envelope: %v", err)
		} else if err := json.Unmarshal(raw, &reportEnvelope); err != nil {
			t.Fatalf("unmarshal Ruby gemspec signature merge report envelope: %v", err)
		}

		step := reportEnvelope.Report.Request.Steps[0]
		if step.MergeProfile["signature_profile"] != "gemspec_declarations" {
			t.Fatalf("expected gemspec_declarations signature profile")
		}
		if !strings.Contains(reportEnvelope.Report.FinalContent, "spec.add_development_dependency(\"rubocop\"") {
			t.Fatalf("expected destination-only development dependency to be preserved")
		}
	}
}

func TestSharedFixtureRubyGemspecFieldPolicyAcceptance(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "ruby_gemspec_field_policy_acceptance"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var reportEnvelope ContentRecipeExecutionReportEnvelope
		if raw, err := json.Marshal(testCase["report_envelope"]); err != nil {
			t.Fatalf("marshal Ruby gemspec field policy report envelope: %v", err)
		} else if err := json.Unmarshal(raw, &reportEnvelope); err != nil {
			t.Fatalf("unmarshal Ruby gemspec field policy report envelope: %v", err)
		}

		if !strings.Contains(reportEnvelope.Report.FinalContent, "Real project summary") {
			t.Fatalf("expected non-placeholder summary to be preserved")
		}
		if strings.Contains(reportEnvelope.Report.FinalContent, "spec.license =") {
			t.Fatalf("expected singular license field to be deleted")
		}
	}
}

func TestSharedFixtureRubyGemspecDependencySectionPolicyAcceptance(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "ruby_gemspec_dependency_section_policy_acceptance"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var reportEnvelope ContentRecipeExecutionReportEnvelope
		if raw, err := json.Marshal(testCase["report_envelope"]); err != nil {
			t.Fatalf("marshal Ruby gemspec dependency section policy report envelope: %v", err)
		} else if err := json.Unmarshal(raw, &reportEnvelope); err != nil {
			t.Fatalf("unmarshal Ruby gemspec dependency section policy report envelope: %v", err)
		}

		if strings.Contains(reportEnvelope.Report.FinalContent, "add_development_dependency(\"json\"") {
			t.Fatalf("expected runtime-shadowed development dependency to be removed")
		}
		if !strings.Contains(reportEnvelope.Report.FinalContent, "add_development_dependency(\"rubocop\"") {
			t.Fatalf("expected destination-only development dependency to be preserved")
		}
	}
}

func TestSharedFixtureRubyGemspecFilesPolicyAcceptance(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "ruby_gemspec_files_policy_acceptance"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var reportEnvelope ContentRecipeExecutionReportEnvelope
		if raw, err := json.Marshal(testCase["report_envelope"]); err != nil {
			t.Fatalf("marshal Ruby gemspec files policy report envelope: %v", err)
		} else if err := json.Unmarshal(raw, &reportEnvelope); err != nil {
			t.Fatalf("unmarshal Ruby gemspec files policy report envelope: %v", err)
		}

		if testCase["label"] == "gemspec-files-literal-dir-union-and-duplicate-cleanup" {
			if strings.Count(reportEnvelope.Report.FinalContent, "spec.files =") != 1 {
				t.Fatalf("expected duplicate spec.files assignments to be removed")
			}
			if !strings.Contains(reportEnvelope.Report.FinalContent, "sig/**/*.rbs") {
				t.Fatalf("expected template-only files entry to be unioned")
			}
		}
	}
}

func TestSharedFixtureRubyGemspecVersionLoaderPolicyAcceptance(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "ruby_gemspec_version_loader_policy_acceptance"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var reportEnvelope ContentRecipeExecutionReportEnvelope
		if raw, err := json.Marshal(testCase["report_envelope"]); err != nil {
			t.Fatalf("marshal Ruby gemspec version-loader policy report envelope: %v", err)
		} else if err := json.Unmarshal(raw, &reportEnvelope); err != nil {
			t.Fatalf("unmarshal Ruby gemspec version-loader policy report envelope: %v", err)
		}

		finalContent := reportEnvelope.Report.FinalContent
		if testCase["label"] == "modern-ruby-inline-version-loader" && strings.Contains(finalContent, "gem_version =") {
			t.Fatalf("expected modern version loader to remove gem_version preamble")
		}
		if testCase["label"] == "legacy-ruby-gem-version-preamble" && !strings.Contains(finalContent, "spec.version = gem_version") {
			t.Fatalf("expected legacy version loader to use gem_version preamble")
		}
	}
}

func TestSharedFixtureRuntimeFactsContext(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "runtime_facts_context"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var reportEnvelope ContentRecipeExecutionReportEnvelope
		if raw, err := json.Marshal(testCase["report_envelope"]); err != nil {
			t.Fatalf("marshal runtime facts runtime context report envelope: %v", err)
		} else if err := json.Unmarshal(raw, &reportEnvelope); err != nil {
			t.Fatalf("unmarshal runtime facts runtime context report envelope: %v", err)
		}

		runtimeFacts := reportEnvelope.Report.Request.RuntimeContext["facts"].(map[string]any)
		if runtimeFacts["schema"] != "runtime_facts.v1" {
			t.Fatalf("expected runtime_facts.v1 schema")
		}

		if testCase["label"] == "dependency-floor-comments-from-project-facts" {
			if !strings.Contains(reportEnvelope.Report.FinalContent, "# Required for Ruby < 3.4.") {
				t.Fatalf("expected dependency floor comment from runtime facts")
			}
			if reportEnvelope.Report.StepReports[0].Metadata["consumed_fact_id"] != "dependency.ruby_floor" {
				t.Fatalf("expected dependency.ruby_floor facts to be consumed")
			}
		}
		if testCase["label"] == "dependency-floor-comments-missing-project-facts-fail-closed" {
			if reportEnvelope.Report.Changed {
				t.Fatalf("expected missing dependency facts to fail closed without changes")
			}
			if reportEnvelope.Report.StepReports[0].Status != "failed" {
				t.Fatalf("expected missing dependency facts to fail the policy step")
			}
		}
	}
}

func TestSharedFixtureRubyGemspecSelfDependencyPolicyAcceptance(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "ruby_gemspec_self_dependency_policy_acceptance"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var reportEnvelope ContentRecipeExecutionReportEnvelope
		if raw, err := json.Marshal(testCase["report_envelope"]); err != nil {
			t.Fatalf("marshal Ruby gemspec self-dependency policy report envelope: %v", err)
		} else if err := json.Unmarshal(raw, &reportEnvelope); err != nil {
			t.Fatalf("unmarshal Ruby gemspec self-dependency policy report envelope: %v", err)
		}

		if testCase["label"] == "delete-active-self-dependencies-preserve-comments" {
			finalContent := reportEnvelope.Report.FinalContent
			if strings.Contains(finalContent, "spec.add_dependency \"demo\", \"~> 1.0\"") {
				t.Fatalf("expected active self dependency to be deleted")
			}
			if !strings.Contains(finalContent, "# spec.add_dependency \"demo\", \"~> 0\"") {
				t.Fatalf("expected commented self dependency to be preserved")
			}
			if reportEnvelope.Report.StepReports[0].Metadata["operation"] != "delete" {
				t.Fatalf("expected canonical delete operation")
			}
		}
		if testCase["label"] == "missing-project-identity-fails-closed" && reportEnvelope.Report.StepReports[0].Status != "failed" {
			t.Fatalf("expected missing package identity to fail closed")
		}
	}
}

func TestSharedFixtureRubyGemfileSelfDependencyPolicyAcceptance(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "ruby_gemfile_self_dependency_policy_acceptance"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var reportEnvelope ContentRecipeExecutionReportEnvelope
		if raw, err := json.Marshal(testCase["report_envelope"]); err != nil {
			t.Fatalf("marshal Ruby Gemfile self-dependency policy report envelope: %v", err)
		} else if err := json.Unmarshal(raw, &reportEnvelope); err != nil {
			t.Fatalf("unmarshal Ruby Gemfile self-dependency policy report envelope: %v", err)
		}

		if testCase["label"] == "delete-gemfile-self-dependencies-across-nesting" {
			finalContent := reportEnvelope.Report.FinalContent
			if strings.Contains(finalContent, "gem \"demo\", \"~> 1.0\"") || strings.Contains(finalContent, "path: \"../dev/demo\"") {
				t.Fatalf("expected active Gemfile self dependencies to be deleted")
			}
			if !strings.Contains(finalContent, "# gem \"demo\", \"~> 0\"") || !strings.Contains(finalContent, "gem \"fallback-gem\"") {
				t.Fatalf("expected comments and unrelated dependencies to be preserved")
			}
			if reportEnvelope.Report.StepReports[0].Metadata["operation"] != "delete" {
				t.Fatalf("expected canonical delete operation")
			}
		}
		if testCase["label"] == "missing-project-identity-fails-closed" && reportEnvelope.Report.StepReports[0].Status != "failed" {
			t.Fatalf("expected missing package identity to fail closed")
		}
	}
}

func TestSharedFixtureRubyAppraisalsSelfDependencyPolicyAcceptance(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "ruby_appraisals_self_dependency_policy_acceptance"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var reportEnvelope ContentRecipeExecutionReportEnvelope
		if raw, err := json.Marshal(testCase["report_envelope"]); err != nil {
			t.Fatalf("marshal Ruby Appraisals self-dependency policy report envelope: %v", err)
		} else if err := json.Unmarshal(raw, &reportEnvelope); err != nil {
			t.Fatalf("unmarshal Ruby Appraisals self-dependency policy report envelope: %v", err)
		}

		if testCase["label"] == "delete-appraisals-self-dependencies" {
			finalContent := reportEnvelope.Report.FinalContent
			if strings.Contains(finalContent, "gem \"demo\"") {
				t.Fatalf("expected Appraisals self dependencies to be deleted")
			}
			if !strings.Contains(finalContent, "appraise(\"rails-6\")") || !strings.Contains(finalContent, "gem \"rspec\" # Testing") {
				t.Fatalf("expected unrelated appraisals and dependencies to be preserved")
			}
			if reportEnvelope.Report.StepReports[0].Metadata["operation"] != "delete" {
				t.Fatalf("expected canonical delete operation")
			}
		}
		if testCase["label"] == "missing-project-identity-fails-closed" && reportEnvelope.Report.StepReports[0].Status != "failed" {
			t.Fatalf("expected missing package identity to fail closed")
		}
	}
}

func TestSharedFixtureRubyAppraisalsMinRubyPrunePolicyAcceptance(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "ruby_appraisals_min_ruby_prune_policy_acceptance"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var reportEnvelope ContentRecipeExecutionReportEnvelope
		if raw, err := json.Marshal(testCase["report_envelope"]); err != nil {
			t.Fatalf("marshal Ruby Appraisals min-Ruby prune policy report envelope: %v", err)
		} else if err := json.Unmarshal(raw, &reportEnvelope); err != nil {
			t.Fatalf("unmarshal Ruby Appraisals min-Ruby prune policy report envelope: %v", err)
		}

		if testCase["label"] == "delete-ruby-appraisals-below-min-ruby" {
			finalContent := reportEnvelope.Report.FinalContent
			if strings.Contains(finalContent, "ruby-2-3") || strings.Contains(finalContent, "ruby-2-7") || strings.Contains(finalContent, "ruby-3-0") {
				t.Fatalf("expected appraisals below min_ruby to be deleted")
			}
			if !strings.Contains(finalContent, "ruby-3-2") || !strings.Contains(finalContent, "appraise \"style\"") {
				t.Fatalf("expected appraisals at min_ruby and non-Ruby appraisals to be preserved")
			}
			if reportEnvelope.Report.StepReports[0].Metadata["operation"] != "delete" {
				t.Fatalf("expected canonical delete operation")
			}
			if strings.Contains(finalContent, "\n\n\n") {
				t.Fatalf("expected excessive blank lines to be normalized")
			}
		}
		if testCase["label"] == "missing-min-ruby-fails-closed" && reportEnvelope.Report.StepReports[0].Status != "failed" {
			t.Fatalf("expected missing min_ruby to fail closed")
		}
	}
}

func TestSharedFixtureChangelogUnreleasedNormalizationAcceptance(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "changelog_unreleased_normalization_acceptance"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var reportEnvelope ContentRecipeExecutionReportEnvelope
		if raw, err := json.Marshal(testCase["report_envelope"]); err != nil {
			t.Fatalf("marshal CHANGELOG Unreleased normalization report envelope: %v", err)
		} else if err := json.Unmarshal(raw, &reportEnvelope); err != nil {
			t.Fatalf("unmarshal CHANGELOG Unreleased normalization report envelope: %v", err)
		}

		if testCase["label"] == "create-unreleased-section-from-supplied-entries" {
			finalContent := reportEnvelope.Report.FinalContent
			unreleasedIndex := strings.Index(finalContent, "## Unreleased")
			releaseIndex := strings.Index(finalContent, "## 1.2.0")
			if unreleasedIndex == -1 || releaseIndex == -1 || unreleasedIndex > releaseIndex {
				t.Fatalf("expected Unreleased section before first release heading")
			}
			if !strings.Contains(finalContent, "- Added native Markdown recipe boundary.") || !strings.Contains(finalContent, "- Existing release.") {
				t.Fatalf("expected supplied entries and release history to be preserved")
			}
			if reportEnvelope.Report.StepReports[0].Metadata["operation"] != "insert_or_replace_section" {
				t.Fatalf("expected canonical insert_or_replace_section operation")
			}
		}
		if testCase["label"] == "missing-entries-fails-closed" && reportEnvelope.Report.StepReports[0].Status != "failed" {
			t.Fatalf("expected missing entries to fail closed")
		}
	}
}

func TestSharedFixtureReadmeSuppliedMetadataSynchronizationAcceptance(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "readme_supplied_metadata_synchronization_acceptance"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var reportEnvelope ContentRecipeExecutionReportEnvelope
		if raw, err := json.Marshal(testCase["report_envelope"]); err != nil {
			t.Fatalf("marshal README supplied metadata synchronization report envelope: %v", err)
		} else if err := json.Unmarshal(raw, &reportEnvelope); err != nil {
			t.Fatalf("unmarshal README supplied metadata synchronization report envelope: %v", err)
		}

		if testCase["label"] == "sync-readme-heading-and-summary-from-supplied-metadata" {
			finalContent := reportEnvelope.Report.FinalContent
			if !strings.HasPrefix(finalContent, "# Demo Toolkit\n") {
				t.Fatalf("expected README H1 to come from supplied metadata")
			}
			if !strings.Contains(finalContent, "A deterministic toolkit for structured merges.") || !strings.Contains(finalContent, "Destination usage.") {
				t.Fatalf("expected supplied summary and unrelated destination section to be preserved")
			}
			if reportEnvelope.Report.StepReports[0].Metadata["consumed_context"] != "readme_metadata.title" {
				t.Fatalf("expected title context to be consumed")
			}
			if reportEnvelope.Report.StepReports[1].Metadata["consumed_context"] != "readme_metadata.summary" {
				t.Fatalf("expected summary context to be consumed")
			}
		}
		if testCase["label"] == "missing-readme-metadata-fails-closed" && reportEnvelope.Report.StepReports[0].Status != "failed" {
			t.Fatalf("expected missing README metadata to fail closed")
		}
	}
}

func TestSharedFixtureSuppliedMarkdownPruningAcceptance(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "supplied_markdown_pruning_acceptance"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var reportEnvelope ContentRecipeExecutionReportEnvelope
		if raw, err := json.Marshal(testCase["report_envelope"]); err != nil {
			t.Fatalf("marshal supplied Markdown pruning report envelope: %v", err)
		} else if err := json.Unmarshal(raw, &reportEnvelope); err != nil {
			t.Fatalf("unmarshal supplied Markdown pruning report envelope: %v", err)
		}

		if testCase["label"] == "prune-supplied-table-rows-and-reference-definitions" {
			finalContent := reportEnvelope.Report.FinalContent
			if strings.Contains(finalContent, "Works with JRuby") || strings.Contains(finalContent, "[jruby-9.4]:") || strings.Contains(finalContent, "[jruby-head]:") {
				t.Fatalf("expected supplied Markdown selectors to be pruned")
			}
			if !strings.Contains(finalContent, "Works with MRI Ruby") || !strings.Contains(finalContent, "[ruby-3.2]:") {
				t.Fatalf("expected unmatched Markdown content to be preserved")
			}
			if reportEnvelope.Report.StepReports[0].Metadata["deleted_rows"] != float64(1) {
				t.Fatalf("expected one table row to be deleted")
			}
			if reportEnvelope.Report.StepReports[1].Metadata["deleted_reference_definitions"] != float64(2) {
				t.Fatalf("expected two reference definitions to be deleted")
			}
		}
		if testCase["label"] == "missing-prune-selectors-fails-closed" && reportEnvelope.Report.StepReports[0].Status != "failed" {
			t.Fatalf("expected missing prune selectors to fail closed")
		}
	}
}

func TestSharedFixtureSuppliedSourceSelectorDeletionAcceptance(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "supplied_source_selector_deletion_acceptance"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var reportEnvelope ContentRecipeExecutionReportEnvelope
		if raw, err := json.Marshal(testCase["report_envelope"]); err != nil {
			t.Fatalf("marshal supplied source selector deletion report envelope: %v", err)
		} else if err := json.Unmarshal(raw, &reportEnvelope); err != nil {
			t.Fatalf("unmarshal supplied source selector deletion report envelope: %v", err)
		}

		if testCase["label"] == "delete-supplied-structural-owner-ranges" {
			finalContent := reportEnvelope.Report.FinalContent
			if strings.Contains(finalContent, "kettle/scaffold") || strings.Contains(finalContent, "task :scaffold") {
				t.Fatalf("expected supplied source selectors to be deleted")
			}
			if !strings.Contains(finalContent, "require \"bundler/gem_tasks\"") || !strings.Contains(finalContent, "task :spec") {
				t.Fatalf("expected unmatched source content to be preserved")
			}
			if strings.Contains(finalContent, "\n\n\n") {
				t.Fatalf("expected deletion gaps to be normalized")
			}
			if reportEnvelope.Report.StepReports[0].Metadata["deleted_ranges"] != float64(2) {
				t.Fatalf("expected two structural owner ranges to be deleted")
			}
		}
		if testCase["label"] == "missing-delete-selectors-fails-closed" && reportEnvelope.Report.StepReports[0].Status != "failed" {
			t.Fatalf("expected missing delete selectors to fail closed")
		}
	}
}

func TestSharedFixtureSuppliedYAMLSnippetSynchronizationAcceptance(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "supplied_yaml_snippet_synchronization_acceptance"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var reportEnvelope ContentRecipeExecutionReportEnvelope
		if raw, err := json.Marshal(testCase["report_envelope"]); err != nil {
			t.Fatalf("marshal supplied YAML snippet synchronization report envelope: %v", err)
		} else if err := json.Unmarshal(raw, &reportEnvelope); err != nil {
			t.Fatalf("unmarshal supplied YAML snippet synchronization report envelope: %v", err)
		}

		if testCase["label"] == "apply-supplied-sections-and-scalar-pins" {
			finalContent := reportEnvelope.Report.FinalContent
			if !strings.Contains(finalContent, "concurrency:") || !strings.Contains(finalContent, "permissions:") {
				t.Fatalf("expected supplied YAML sections to be applied")
			}
			if !strings.Contains(finalContent, "actions/checkout@de0fac2e4500dabe0009e67214ff5f5447ce83dd") || !strings.Contains(finalContent, "ruby/setup-ruby@e65c17d16e57e481586a6a5a0282698790062f92") {
				t.Fatalf("expected supplied YAML scalar pins to be applied")
			}
			if strings.Contains(finalContent, "actions/checkout@v3") || strings.Contains(finalContent, "ruby/setup-ruby@v1") {
				t.Fatalf("expected old action pins to be replaced")
			}
			if !strings.Contains(finalContent, "gemfiles/current.gemfile") || !strings.Contains(finalContent, "ruby-version: ${{ matrix.ruby }}") {
				t.Fatalf("expected unmatched workflow YAML to be preserved")
			}
			if reportEnvelope.Report.StepReports[0].Metadata["updated_sections"] != float64(2) {
				t.Fatalf("expected two YAML sections to be updated")
			}
			if reportEnvelope.Report.StepReports[1].Metadata["updated_scalars"] != float64(2) {
				t.Fatalf("expected two YAML scalars to be updated")
			}
		}
		if testCase["label"] == "missing-yaml-updates-fails-closed" && reportEnvelope.Report.StepReports[0].Status != "failed" {
			t.Fatalf("expected missing YAML updates to fail closed")
		}
	}
}

func TestSharedFixtureSuppliedManagedTextBlockReplacementAcceptance(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "supplied_managed_text_block_replacement_acceptance"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var reportEnvelope ContentRecipeExecutionReportEnvelope
		if raw, err := json.Marshal(testCase["report_envelope"]); err != nil {
			t.Fatalf("marshal supplied managed text block replacement report envelope: %v", err)
		} else if err := json.Unmarshal(raw, &reportEnvelope); err != nil {
			t.Fatalf("unmarshal supplied managed text block replacement report envelope: %v", err)
		}

		if testCase["label"] == "replace-existing-managed-text-block" {
			finalContent := reportEnvelope.Report.FinalContent
			if !strings.Contains(finalContent, "gem \"debug\", \"~> 1.9\"") || !strings.Contains(finalContent, "gem \"irb\", \"~> 1.15\"") {
				t.Fatalf("expected generated block content to replace old content")
			}
			if strings.Contains(finalContent, "old-debug") {
				t.Fatalf("expected old generated block content to be removed")
			}
			if !strings.Contains(finalContent, "gem \"rake\"") || !strings.Contains(finalContent, "gem \"rspec\"") {
				t.Fatalf("expected content outside managed block to be preserved")
			}
			if reportEnvelope.Report.StepReports[0].Metadata["replaced_blocks"] != float64(1) {
				t.Fatalf("expected one managed block to be replaced")
			}
		}
		if testCase["label"] == "append-missing-managed-text-block" {
			finalContent := reportEnvelope.Report.FinalContent
			if !strings.Contains(finalContent, "# <<kettle-jem:generated>>") || !strings.Contains(finalContent, "# (no shunted dependencies)") {
				t.Fatalf("expected missing managed block to be appended")
			}
			if reportEnvelope.Report.StepReports[0].Metadata["appended_blocks"] != float64(1) {
				t.Fatalf("expected one managed block to be appended")
			}
		}
		if testCase["label"] == "missing-managed-block-updates-fails-closed" && reportEnvelope.Report.StepReports[0].Status != "failed" {
			t.Fatalf("expected missing managed block updates to fail closed")
		}
	}
}

func TestSharedFixtureSuppliedYAMLPlaceholderScalarBackfillAcceptance(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "supplied_yaml_placeholder_scalar_backfill_acceptance"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var reportEnvelope ContentRecipeExecutionReportEnvelope
		if raw, err := json.Marshal(testCase["report_envelope"]); err != nil {
			t.Fatalf("marshal supplied YAML placeholder scalar backfill report envelope: %v", err)
		} else if err := json.Unmarshal(raw, &reportEnvelope); err != nil {
			t.Fatalf("unmarshal supplied YAML placeholder scalar backfill report envelope: %v", err)
		}

		if testCase["label"] == "backfill-placeholder-and-blank-scalars" {
			finalContent := reportEnvelope.Report.FinalContent
			if !strings.Contains(finalContent, "name: \"demo-toolkit\"") || !strings.Contains(finalContent, "namespace: 'Demo::Toolkit'") {
				t.Fatalf("expected placeholder and blank YAML scalars to be backfilled")
			}
			if !strings.Contains(finalContent, "homepage: \"https://example.invalid/existing\"") {
				t.Fatalf("expected concrete YAML scalar to be preserved")
			}
			if !strings.Contains(finalContent, "# ENV: KJ_GEM_NAME") || !strings.Contains(finalContent, "# keep concrete value") {
				t.Fatalf("expected YAML comments to be preserved")
			}
			if reportEnvelope.Report.StepReports[0].Metadata["updated_scalars"] != float64(2) {
				t.Fatalf("expected two YAML scalars to be updated")
			}
			if reportEnvelope.Report.StepReports[0].Metadata["preserved_scalars"] != float64(1) {
				t.Fatalf("expected one concrete YAML scalar to be preserved")
			}
		}
		if testCase["label"] == "missing-yaml-scalar-backfills-fails-closed" && reportEnvelope.Report.StepReports[0].Status != "failed" {
			t.Fatalf("expected missing YAML scalar backfills to fail closed")
		}
	}
}

func TestSharedFixtureStructuredEditCallableDestinationRequest(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_callable_destination_request"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var request StructuredEditRequest
		if raw, err := json.Marshal(testCase["request"]); err != nil {
			t.Fatalf("marshal structured edit callable destination request: %v", err)
		} else if err := json.Unmarshal(raw, &request); err != nil {
			t.Fatalf("unmarshal structured edit callable destination request: %v", err)
		}

		roundtrip, err := json.Marshal(request)
		if err != nil {
			t.Fatalf("marshal structured edit callable destination request roundtrip: %v", err)
		}

		var decoded StructuredEditRequest
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit callable destination request: %v", err)
		}

		if !reflect.DeepEqual(decoded, request) {
			t.Fatalf("unexpected structured edit callable destination request roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditParitySelectionSemantics(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_parity_selection_semantics"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var request StructuredEditRequest
		if raw, err := json.Marshal(testCase["request"]); err != nil {
			t.Fatalf("marshal structured edit parity selection semantics request: %v", err)
		} else if err := json.Unmarshal(raw, &request); err != nil {
			t.Fatalf("unmarshal structured edit parity selection semantics request: %v", err)
		}

		roundtrip, err := json.Marshal(request)
		if err != nil {
			t.Fatalf("marshal structured edit parity selection semantics request roundtrip: %v", err)
		}

		var decoded StructuredEditRequest
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit parity selection semantics request: %v", err)
		}

		if !reflect.DeepEqual(decoded, request) {
			t.Fatalf("unexpected structured edit parity selection semantics request roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditParityMatchSemantics(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_parity_match_semantics"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var request StructuredEditRequest
		if raw, err := json.Marshal(testCase["request"]); err != nil {
			t.Fatalf("marshal structured edit parity match semantics request: %v", err)
		} else if err := json.Unmarshal(raw, &request); err != nil {
			t.Fatalf("unmarshal structured edit parity match semantics request: %v", err)
		}

		roundtrip, err := json.Marshal(request)
		if err != nil {
			t.Fatalf("marshal structured edit parity match semantics request roundtrip: %v", err)
		}

		var decoded StructuredEditRequest
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit parity match semantics request: %v", err)
		}

		if !reflect.DeepEqual(decoded, request) {
			t.Fatalf("unexpected structured edit parity match semantics request roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditOperationTriadParity(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_operation_triad_parity"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		applicationValue := testCase["application"].(map[string]any)

		var application StructuredEditApplication
		if raw, err := json.Marshal(applicationValue); err != nil {
			t.Fatalf("marshal structured edit operation triad application: %v", err)
		} else if err := json.Unmarshal(raw, &application); err != nil {
			t.Fatalf("unmarshal structured edit operation triad application: %v", err)
		}

		roundtrip, err := json.Marshal(application)
		if err != nil {
			t.Fatalf("marshal structured edit operation triad application roundtrip: %v", err)
		}

		var decoded StructuredEditApplication
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit operation triad application: %v", err)
		}

		if !reflect.DeepEqual(decoded, application) {
			t.Fatalf("unexpected structured edit operation triad application roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditExecutionReportEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_execution_report_envelope"))
	report := decodeFixtureValue[StructuredEditExecutionReport](t, fixture["structured_edit_execution_report"])
	expected := decodeFixtureValue[StructuredEditExecutionReportEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditExecutionReportEnvelopeFor(report); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit execution report envelope: %+v", envelope)
	}

	if imported, importErr := ImportStructuredEditExecutionReportEnvelope(expected); importErr != nil {
		t.Fatalf("unexpected structured edit execution report envelope import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, report) {
		t.Fatalf("unexpected structured edit execution report envelope import: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditExecutionReportEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_execution_report_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeFixtureValue[StructuredEditExecutionReportEnvelope](t, testCase["envelope"])
		expected := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditExecutionReportEnvelope(envelope); importErr == nil {
			t.Fatalf("expected structured edit execution report envelope rejection, got report: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expected) {
			t.Fatalf("unexpected structured edit execution report envelope rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditExecutionReportEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_execution_report_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditExecutionReportEnvelope](t, fixture["structured_edit_execution_report_envelope"])
	expected := decodeFixtureValue[StructuredEditExecutionReport](t, fixture["expected_report"])

	if imported, importErr := ImportStructuredEditExecutionReportEnvelope(envelope); importErr != nil {
		t.Fatalf("unexpected structured edit execution report envelope application import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected structured edit execution report envelope application: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditExecutionReportEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditExecutionReportEnvelope(rejectedEnvelope); importErr == nil {
			t.Fatalf("expected structured edit execution report envelope application rejection, got report: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit execution report envelope application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditBatchRequest(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_batch_request"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var batch StructuredEditBatchRequest
		if raw, err := json.Marshal(testCase["batch_request"]); err != nil {
			t.Fatalf("marshal structured edit batch request: %v", err)
		} else if err := json.Unmarshal(raw, &batch); err != nil {
			t.Fatalf("unmarshal structured edit batch request: %v", err)
		}

		roundtrip, err := json.Marshal(batch)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit batch request: %v", err)
		}
		var decoded StructuredEditBatchRequest
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit batch request: %v", err)
		}

		if !reflect.DeepEqual(decoded, batch) {
			t.Fatalf("unexpected structured edit batch request roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionRequest(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_request"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var batch StructuredEditProviderBatchExecutionRequest
		if raw, err := json.Marshal(testCase["batch_execution_request"]); err != nil {
			t.Fatalf("marshal structured edit provider batch execution request: %v", err)
		} else if err := json.Unmarshal(raw, &batch); err != nil {
			t.Fatalf("unmarshal structured edit provider batch execution request: %v", err)
		}

		roundtrip, err := json.Marshal(batch)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider batch execution request: %v", err)
		}
		var decoded StructuredEditProviderBatchExecutionRequest
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider batch execution request: %v", err)
		}

		if !reflect.DeepEqual(decoded, batch) {
			t.Fatalf("unexpected structured edit provider batch execution request roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionRequestEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_request_envelope"))
	batchExecutionRequest := decodeFixtureValue[StructuredEditProviderBatchExecutionRequest](t, fixture["structured_edit_provider_batch_execution_request"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionRequestEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderBatchExecutionRequestEnvelopeFor(batchExecutionRequest); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider batch execution request envelope: %+v", envelope)
	}

	if imported, importErr := ImportStructuredEditProviderBatchExecutionRequestEnvelope(expected); importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution request envelope import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, batchExecutionRequest) {
		t.Fatalf("unexpected structured edit provider batch execution request envelope import: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionRequestEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_request_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionRequestEnvelope](t, testCase["envelope"])
		expected := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderBatchExecutionRequestEnvelope(envelope); importErr == nil {
			t.Fatalf("expected structured edit provider batch execution request envelope rejection, got batch: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expected) {
			t.Fatalf("unexpected structured edit provider batch execution request envelope rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionRequestEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_request_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionRequestEnvelope](t, fixture["structured_edit_provider_batch_execution_request_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionRequest](t, fixture["expected_batch_execution_request"])

	if imported, importErr := ImportStructuredEditProviderBatchExecutionRequestEnvelope(envelope); importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution request envelope application import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected structured edit provider batch execution request envelope application: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderBatchExecutionRequestEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderBatchExecutionRequestEnvelope(rejectedEnvelope); importErr == nil {
			t.Fatalf("expected structured edit provider batch execution request envelope application rejection, got batch: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider batch execution request envelope application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionDispatch(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_dispatch"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var batch StructuredEditProviderBatchExecutionDispatch
		if raw, err := json.Marshal(testCase["batch_dispatch"]); err != nil {
			t.Fatalf("marshal structured edit provider batch execution dispatch: %v", err)
		} else if err := json.Unmarshal(raw, &batch); err != nil {
			t.Fatalf("unmarshal structured edit provider batch execution dispatch: %v", err)
		}

		roundtrip, err := json.Marshal(batch)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider batch execution dispatch: %v", err)
		}
		var decoded StructuredEditProviderBatchExecutionDispatch
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider batch execution dispatch: %v", err)
		}

		if !reflect.DeepEqual(decoded, batch) {
			t.Fatalf("unexpected structured edit provider batch execution dispatch roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionDispatchEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_dispatch_envelope"))
	batchDispatch := decodeFixtureValue[StructuredEditProviderBatchExecutionDispatch](t, fixture["structured_edit_provider_batch_execution_dispatch"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionDispatchEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderBatchExecutionDispatchEnvelopeFor(batchDispatch); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider batch execution dispatch envelope: %+v", envelope)
	}

	if imported, importErr := ImportStructuredEditProviderBatchExecutionDispatchEnvelope(expected); importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution dispatch envelope import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, batchDispatch) {
		t.Fatalf("unexpected structured edit provider batch execution dispatch envelope import: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionDispatchEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_dispatch_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionDispatchEnvelope](t, testCase["envelope"])
		expected := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderBatchExecutionDispatchEnvelope(envelope); importErr == nil {
			t.Fatalf("expected structured edit provider batch execution dispatch envelope rejection, got batch: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expected) {
			t.Fatalf("unexpected structured edit provider batch execution dispatch envelope rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionDispatchEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_dispatch_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionDispatchEnvelope](t, fixture["structured_edit_provider_batch_execution_dispatch_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionDispatch](t, fixture["expected_batch_dispatch"])

	if imported, importErr := ImportStructuredEditProviderBatchExecutionDispatchEnvelope(envelope); importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution dispatch envelope application import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected structured edit provider batch execution dispatch envelope application: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderBatchExecutionDispatchEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderBatchExecutionDispatchEnvelope(rejectedEnvelope); importErr == nil {
			t.Fatalf("expected structured edit provider batch execution dispatch envelope application rejection, got batch: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider batch execution dispatch envelope application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReport(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_report"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var batch StructuredEditProviderBatchExecutionReport
		if raw, err := json.Marshal(testCase["batch_report"]); err != nil {
			t.Fatalf("marshal structured edit provider batch execution report: %v", err)
		} else if err := json.Unmarshal(raw, &batch); err != nil {
			t.Fatalf("unmarshal structured edit provider batch execution report: %v", err)
		}

		roundtrip, err := json.Marshal(batch)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit provider batch execution report: %v", err)
		}
		var decoded StructuredEditProviderBatchExecutionReport
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit provider batch execution report: %v", err)
		}

		if !reflect.DeepEqual(decoded, batch) {
			t.Fatalf("unexpected structured edit provider batch execution report roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReportEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_report_envelope"))
	batchReport := decodeFixtureValue[StructuredEditProviderBatchExecutionReport](t, fixture["structured_edit_provider_batch_execution_report"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionReportEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditProviderBatchExecutionReportEnvelopeFor(batchReport); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit provider batch execution report envelope: %+v", envelope)
	}

	if imported, importErr := ImportStructuredEditProviderBatchExecutionReportEnvelope(expected); importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution report envelope import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, batchReport) {
		t.Fatalf("unexpected structured edit provider batch execution report envelope import: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReportEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_report_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReportEnvelope](t, testCase["envelope"])
		expected := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderBatchExecutionReportEnvelope(envelope); importErr == nil {
			t.Fatalf("expected structured edit provider batch execution report envelope rejection, got batch report: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expected) {
			t.Fatalf("unexpected structured edit provider batch execution report envelope rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditProviderBatchExecutionReportEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_provider_batch_execution_report_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReportEnvelope](t, fixture["structured_edit_provider_batch_execution_report_envelope"])
	expected := decodeFixtureValue[StructuredEditProviderBatchExecutionReport](t, fixture["expected_batch_report"])

	if imported, importErr := ImportStructuredEditProviderBatchExecutionReportEnvelope(envelope); importErr != nil {
		t.Fatalf("unexpected structured edit provider batch execution report envelope application import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected structured edit provider batch execution report envelope application: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditProviderBatchExecutionReportEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditProviderBatchExecutionReportEnvelope(rejectedEnvelope); importErr == nil {
			t.Fatalf("expected structured edit provider batch execution report envelope application rejection, got batch report: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit provider batch execution report envelope application rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditBatchReport(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_batch_report"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)

		var batch StructuredEditBatchReport
		if raw, err := json.Marshal(testCase["batch_report"]); err != nil {
			t.Fatalf("marshal structured edit batch report: %v", err)
		} else if err := json.Unmarshal(raw, &batch); err != nil {
			t.Fatalf("unmarshal structured edit batch report: %v", err)
		}

		roundtrip, err := json.Marshal(batch)
		if err != nil {
			t.Fatalf("marshal roundtrip structured edit batch report: %v", err)
		}
		var decoded StructuredEditBatchReport
		if err := json.Unmarshal(roundtrip, &decoded); err != nil {
			t.Fatalf("unmarshal roundtrip structured edit batch report: %v", err)
		}

		if !reflect.DeepEqual(decoded, batch) {
			t.Fatalf("unexpected structured edit batch report roundtrip for %s: %+v", testCase["label"], decoded)
		}
	}
}

func TestSharedFixtureStructuredEditBatchReportEnvelope(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_batch_report_envelope"))
	batchReport := decodeFixtureValue[StructuredEditBatchReport](t, fixture["structured_edit_batch_report"])
	expected := decodeFixtureValue[StructuredEditBatchReportEnvelope](t, fixture["expected_envelope"])

	if envelope := StructuredEditBatchReportEnvelopeFor(batchReport); !reflect.DeepEqual(envelope, expected) {
		t.Fatalf("unexpected structured edit batch report envelope: %+v", envelope)
	}

	if imported, importErr := ImportStructuredEditBatchReportEnvelope(expected); importErr != nil {
		t.Fatalf("unexpected structured edit batch report envelope import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, batchReport) {
		t.Fatalf("unexpected structured edit batch report envelope import: %+v", *imported)
	}
}

func TestSharedFixtureStructuredEditBatchReportEnvelopeRejection(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_batch_report_envelope_rejection"))
	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		envelope := decodeFixtureValue[StructuredEditBatchReportEnvelope](t, testCase["envelope"])
		expected := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditBatchReportEnvelope(envelope); importErr == nil {
			t.Fatalf("expected structured edit batch report envelope rejection, got batch report: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expected) {
			t.Fatalf("unexpected structured edit batch report envelope rejection for %s: %+v", testCase["label"], *importErr)
		}
	}
}

func TestSharedFixtureStructuredEditBatchReportEnvelopeApplication(t *testing.T) {
	fixture := readDiagnosticFixtureFromPath(t, diagnosticsFixturePath(t, "structured_edit_batch_report_envelope_application"))
	envelope := decodeFixtureValue[StructuredEditBatchReportEnvelope](t, fixture["structured_edit_batch_report_envelope"])
	expected := decodeFixtureValue[StructuredEditBatchReport](t, fixture["expected_batch_report"])

	if imported, importErr := ImportStructuredEditBatchReportEnvelope(envelope); importErr != nil {
		t.Fatalf("unexpected structured edit batch report envelope application import error: %+v", importErr)
	} else if !reflect.DeepEqual(*imported, expected) {
		t.Fatalf("unexpected structured edit batch report envelope application: %+v", *imported)
	}

	for _, rawCase := range fixture["cases"].([]any) {
		testCase := rawCase.(map[string]any)
		rejectedEnvelope := decodeFixtureValue[StructuredEditBatchReportEnvelope](t, testCase["envelope"])
		expectedError := decodeFixtureValue[StructuredEditTransportImportError](t, testCase["expected_error"])

		if imported, importErr := ImportStructuredEditBatchReportEnvelope(rejectedEnvelope); importErr == nil {
			t.Fatalf("expected structured edit batch report envelope application rejection, got batch report: %+v", imported)
		} else if !reflect.DeepEqual(*importErr, expectedError) {
			t.Fatalf("unexpected structured edit batch report envelope application rejection for %s: %+v", testCase["label"], *importErr)
		}
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
