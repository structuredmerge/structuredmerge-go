package markdownmerge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/treehaver"
)

func readMarkdownFixture(t *testing.T, parts ...string) map[string]any {
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

func jsonReadyMarkdown(t *testing.T, value any) any {
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

func parseProjectedChildReviewGroup(raw map[string]any) astmerge.ProjectedChildReviewGroup {
	return astmerge.ProjectedChildReviewGroup{
		DelegatedApplyGroup:         raw["delegated_apply_group"].(string),
		ParentOperationID:           raw["parent_operation_id"].(string),
		ChildOperationID:            raw["child_operation_id"].(string),
		DelegatedRuntimeSurfacePath: raw["delegated_runtime_surface_path"].(string),
		CaseIDs:                     parseStringSlice(raw["case_ids"].([]any)),
		DelegatedCaseIDs:            parseStringSlice(raw["delegated_case_ids"].([]any)),
	}
}

func parseReviewRequest(raw map[string]any) astmerge.ReviewRequest {
	request := astmerge.ReviewRequest{
		ID:           raw["id"].(string),
		Kind:         astmerge.ReviewRequestKind(raw["kind"].(string)),
		Family:       raw["family"].(string),
		Message:      raw["message"].(string),
		Blocking:     raw["blocking"].(bool),
		ActionOffers: []astmerge.ReviewActionOffer{},
	}
	if rawDelegatedGroup, ok := raw["delegated_group"]; ok {
		group := parseProjectedChildReviewGroup(rawDelegatedGroup.(map[string]any))
		request.DelegatedGroup = &group
	}
	if rawActionOffers, ok := raw["action_offers"]; ok {
		request.ActionOffers = make([]astmerge.ReviewActionOffer, 0, len(rawActionOffers.([]any)))
		for _, item := range rawActionOffers.([]any) {
			offer := item.(map[string]any)
			request.ActionOffers = append(request.ActionOffers, astmerge.ReviewActionOffer{
				Action:          astmerge.ReviewDecisionAction(offer["action"].(string)),
				RequiresContext: offer["requires_context"].(bool),
			})
		}
	}
	if rawDefaultAction, ok := raw["default_action"]; ok {
		request.DefaultAction = astmerge.ReviewDecisionAction(rawDefaultAction.(string))
	}
	return request
}

func parseReviewDecision(raw map[string]any) astmerge.ReviewDecision {
	return astmerge.ReviewDecision{
		RequestID: raw["request_id"].(string),
		Action:    astmerge.ReviewDecisionAction(raw["action"].(string)),
	}
}

func parseStringSlice(raw []any) []string {
	values := make([]string, 0, len(raw))
	for _, item := range raw {
		values = append(values, item.(string))
	}
	return values
}

func TestSharedFixtureMarkdownFeatureProfile(t *testing.T) {
	fixture := readMarkdownFixture(t, "diagnostics", "slice-194-markdown-family-feature-profile", "markdown-feature-profile.json")
	profile := MarkdownFeatureProfileInfo()
	if profile.Family != fixture["feature_profile"].(map[string]any)["family"].(string) {
		t.Fatalf("unexpected family: %+v", profile)
	}
}

func TestSharedFixtureMarkdownBackendFeatureProfiles(t *testing.T) {
	fixture := readMarkdownFixture(t, "diagnostics", "slice-195-markdown-family-backend-feature-profiles", "go-markdown-backend-feature-profiles.json")
	backends := AvailableMarkdownBackends()
	if len(backends) != 1 || backends[0] != BackendKreuzberg {
		t.Fatalf("unexpected backends: %+v", backends)
	}
	if treeSitter := MarkdownBackendFeatureProfileInfo(BackendKreuzberg); treeSitter.Backend != fixture["tree_sitter"].(map[string]any)["backend"].(string) {
		t.Fatalf("unexpected tree-sitter backend profile: %+v", treeSitter)
	}
	if actual := jsonReadyMarkdown(t, map[string]any{
		"backend":            MarkdownBackendFeatureProfileInfo(BackendKreuzberg).Backend,
		"supported_policies": MarkdownBackendFeatureProfileInfo(BackendKreuzberg).SupportedPolicies,
		"backend_ref": map[string]any{
			"id":     MarkdownBackendFeatureProfileInfo(BackendKreuzberg).BackendRef.ID,
			"family": MarkdownBackendFeatureProfileInfo(BackendKreuzberg).BackendRef.Family,
		},
	}); !reflect.DeepEqual(actual, fixture["tree_sitter"]) {
		t.Fatalf("unexpected tree-sitter backend fixture projection: %+v", actual)
	}
	if backend := treehaver.BackendReferenceByID(string(BackendKreuzberg)); backend == nil || backend.ID != string(BackendKreuzberg) || backend.Family != "tree-sitter" {
		t.Fatalf("unexpected registered backend: %+v", backend)
	}
}

func TestSharedFixtureMarkdownPlanContexts(t *testing.T) {
	fixture := readMarkdownFixture(t, "diagnostics", "slice-196-markdown-family-plan-contexts", "go-markdown-plan-contexts.json")
	treeSitter := MarkdownPlanContextWithBackend(BackendKreuzberg)
	if treeSitter.FamilyProfile.Family != fixture["tree_sitter"].(map[string]any)["family_profile"].(map[string]any)["family"].(string) {
		t.Fatalf("unexpected tree-sitter family profile: %+v", treeSitter)
	}
	if treeSitter.FeatureProfile == nil || treeSitter.FeatureProfile.Backend != fixture["tree_sitter"].(map[string]any)["feature_profile"].(map[string]any)["backend"].(string) {
		t.Fatalf("unexpected tree-sitter feature profile: %+v", treeSitter.FeatureProfile)
	}
}

func TestSharedFixtureMarkdownManifest(t *testing.T) {
	fixture := readMarkdownFixture(t, "conformance", "slice-197-markdown-family-manifest", "markdown-family-manifest.json")
	source, err := json.Marshal(fixture)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}

	var manifest astmerge.ConformanceManifest
	if err := json.Unmarshal(source, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}

	if path := astmerge.ConformanceFamilyFeatureProfilePath(manifest, "markdown"); path == nil || filepath.Join(path...) != filepath.Join("diagnostics", "slice-194-markdown-family-feature-profile", "markdown-feature-profile.json") {
		t.Fatalf("unexpected family feature profile path: %v", path)
	}
	if path := astmerge.ConformanceFixturePath(manifest, "markdown", "analysis"); path == nil || filepath.Join(path...) != filepath.Join("markdown", "slice-198-analysis", "headings-and-code-fences.json") {
		t.Fatalf("unexpected analysis fixture path: %v", path)
	}
	if path := astmerge.ConformanceFixturePath(manifest, "markdown", "merge"); path == nil || filepath.Join(path...) != filepath.Join("markdown", "slice-286-merge", "section-merge.json") {
		t.Fatalf("unexpected merge fixture path: %v", path)
	}
}

func TestSharedFixtureMarkdownAnalysis(t *testing.T) {
	fixture := readMarkdownFixture(t, "markdown", "slice-198-analysis", "headings-and-code-fences.json")
	expectedOwners := fixture["expected"].(map[string]any)["owners"].([]any)

	result := ParseMarkdownWithBackend(fixture["source"].(string), DialectMarkdown, BackendKreuzberg)
	if !result.OK || result.Analysis == nil {
		t.Fatalf("expected parse success: %+v", result)
	}
	if result.Analysis.RootKind != RootDocument {
		t.Fatalf("unexpected root kind: %+v", result.Analysis.RootKind)
	}
	if len(result.Analysis.Owners) != len(expectedOwners) {
		t.Fatalf("unexpected owner count: %+v", result.Analysis.Owners)
	}
}

func TestSharedFixtureMarkdownMatching(t *testing.T) {
	fixture := readMarkdownFixture(t, "markdown", "slice-199-matching", "path-equality.json")
	template := ParseMarkdownWithBackend(fixture["template"].(string), DialectMarkdown, BackendKreuzberg)
	destination := ParseMarkdownWithBackend(fixture["destination"].(string), DialectMarkdown, BackendKreuzberg)
	if !template.OK || template.Analysis == nil || !destination.OK || destination.Analysis == nil {
		t.Fatalf("expected parse success")
	}

	result := MatchMarkdownOwners(*template.Analysis, *destination.Analysis)
	if len(result.Matched) != len(fixture["expected"].(map[string]any)["matched"].([]any)) {
		t.Fatalf("unexpected matches: %+v", result.Matched)
	}
	if len(result.UnmatchedTemplate) != len(fixture["expected"].(map[string]any)["unmatched_template"].([]any)) {
		t.Fatalf("unexpected unmatched template: %+v", result.UnmatchedTemplate)
	}
}

func TestSharedFixtureMarkdownMerge(t *testing.T) {
	fixture := readMarkdownFixture(t, "markdown", "slice-286-merge", "section-merge.json")
	result := MergeMarkdown(fixture["template"].(string), fixture["destination"].(string), DialectMarkdown, BackendKreuzberg)
	if !result.OK || result.Output == nil {
		t.Fatalf("expected merge success: %+v", result)
	}
	if *result.Output != fixture["expected"].(map[string]any)["output"].(string) {
		t.Fatalf("unexpected merge output:\n%s", *result.Output)
	}
}

func TestSharedFixtureMarkdownDelegatedChildApplyOutput(t *testing.T) {
	fixture := readMarkdownFixture(t, "markdown", "slice-288-delegated-child-apply-output", "fenced-code-applied-output.json")
	operationsSource, err := json.Marshal(fixture["delegated_operations"])
	if err != nil {
		t.Fatalf("marshal delegated operations: %v", err)
	}
	var operations []astmerge.DelegatedChildOperation
	if err := json.Unmarshal(operationsSource, &operations); err != nil {
		t.Fatalf("decode delegated operations: %v", err)
	}
	applyPlanSource, err := json.Marshal(fixture["apply_plan"])
	if err != nil {
		t.Fatalf("marshal apply plan: %v", err)
	}
	var applyPlan astmerge.DelegatedChildApplyPlan
	if err := json.Unmarshal(applyPlanSource, &applyPlan); err != nil {
		t.Fatalf("decode apply plan: %v", err)
	}
	childrenSource, err := json.Marshal(fixture["applied_children"])
	if err != nil {
		t.Fatalf("marshal applied children: %v", err)
	}
	var appliedChildren []AppliedChildOutput
	if err := json.Unmarshal(childrenSource, &appliedChildren); err != nil {
		t.Fatalf("decode applied children: %v", err)
	}

	result := ApplyMarkdownDelegatedChildOutputs(fixture["source"].(string), operations, applyPlan, appliedChildren)
	if !result.OK || result.Output == nil {
		t.Fatalf("expected apply output success: %+v", result)
	}
	if *result.Output != fixture["expected"].(map[string]any)["output"].(string) {
		t.Fatalf("unexpected delegated child applied output:\n%s", *result.Output)
	}
}

func TestSharedFixtureMarkdownNestedMerge(t *testing.T) {
	fixture := readMarkdownFixture(t, "markdown", "slice-290-nested-merge", "fenced-code-nested-merge.json")
	nestedOutputsSource, err := json.Marshal(fixture["nested_outputs"])
	if err != nil {
		t.Fatalf("marshal nested outputs: %v", err)
	}
	var nestedOutputs []NestedChildOutput
	if err := json.Unmarshal(nestedOutputsSource, &nestedOutputs); err != nil {
		t.Fatalf("decode nested outputs: %v", err)
	}
	result := MergeMarkdownWithNestedOutputs(
		fixture["template"].(string),
		fixture["destination"].(string),
		DialectMarkdown,
		nestedOutputs,
		BackendKreuzberg,
	)
	if !result.OK || result.Output == nil {
		t.Fatalf("expected nested merge success: %+v", result)
	}
	if *result.Output != fixture["expected"].(map[string]any)["output"].(string) {
		t.Fatalf("unexpected nested merge output:\n%s", *result.Output)
	}
}

func TestSharedFixtureMarkdownReviewedNestedMerge(t *testing.T) {
	fixture := readMarkdownFixture(t, "markdown", "slice-298-reviewed-nested-merge", "fenced-code-reviewed-nested-merge.json")
	reviewStateSource, err := json.Marshal(fixture["review_state"])
	if err != nil {
		t.Fatalf("marshal review state: %v", err)
	}
	var reviewState astmerge.DelegatedChildGroupReviewState
	if err := json.Unmarshal(reviewStateSource, &reviewState); err != nil {
		t.Fatalf("decode review state: %v", err)
	}
	appliedChildrenSource, err := json.Marshal(fixture["applied_children"])
	if err != nil {
		t.Fatalf("marshal applied children: %v", err)
	}
	var appliedChildren []AppliedChildOutput
	if err := json.Unmarshal(appliedChildrenSource, &appliedChildren); err != nil {
		t.Fatalf("decode applied children: %v", err)
	}

	result := MergeMarkdownWithReviewedNestedOutputs(
		fixture["template"].(string),
		fixture["destination"].(string),
		DialectMarkdown,
		reviewState,
		appliedChildren,
		BackendKreuzberg,
	)
	if !result.OK || result.Output == nil {
		t.Fatalf("expected reviewed nested merge success: %+v", result)
	}
	if *result.Output != fixture["expected"].(map[string]any)["output"].(string) {
		t.Fatalf("unexpected reviewed nested merge output:\n%s", *result.Output)
	}
}

func TestSharedFixtureMarkdownReviewedNestedReviewArtifactApplication(t *testing.T) {
	fixture := readMarkdownFixture(t, "markdown", "slice-309-reviewed-nested-review-artifact-application", "fenced-code-reviewed-nested-review-artifact-application.json")
	replayBundleSource, err := json.Marshal(fixture["replay_bundle"])
	if err != nil {
		t.Fatalf("marshal replay bundle: %v", err)
	}
	var replayBundle astmerge.ReviewReplayBundle
	if err := json.Unmarshal(replayBundleSource, &replayBundle); err != nil {
		t.Fatalf("decode replay bundle: %v", err)
	}
	reviewStateSource, err := json.Marshal(fixture["review_state"])
	if err != nil {
		t.Fatalf("marshal review state: %v", err)
	}
	var reviewState astmerge.ConformanceManifestReviewState
	if err := json.Unmarshal(reviewStateSource, &reviewState); err != nil {
		t.Fatalf("decode review state: %v", err)
	}

	replayResult := MergeMarkdownWithReviewedNestedOutputsFromReplayBundle(
		fixture["template"].(string),
		fixture["destination"].(string),
		DialectMarkdown,
		replayBundle,
		BackendKreuzberg,
	)
	if !replayResult.OK || replayResult.Output == nil {
		t.Fatalf("expected replay-bundle reviewed nested merge success: %+v", replayResult)
	}
	if *replayResult.Output != fixture["expected"].(map[string]any)["output"].(string) {
		t.Fatalf("unexpected replay-bundle reviewed nested merge output:\n%s", *replayResult.Output)
	}

	stateResult := MergeMarkdownWithReviewedNestedOutputsFromReviewState(
		fixture["template"].(string),
		fixture["destination"].(string),
		DialectMarkdown,
		reviewState,
		BackendKreuzberg,
	)
	if !stateResult.OK || stateResult.Output == nil {
		t.Fatalf("expected review-state reviewed nested merge success: %+v", stateResult)
	}
	if *stateResult.Output != fixture["expected"].(map[string]any)["output"].(string) {
		t.Fatalf("unexpected review-state reviewed nested merge output:\n%s", *stateResult.Output)
	}
}

func TestSharedFixtureMarkdownReviewedNestedReviewArtifactRejection(t *testing.T) {
	fixture := readMarkdownFixture(t, "markdown", "slice-311-reviewed-nested-review-artifact-rejection", "fenced-code-reviewed-nested-review-artifact-rejection.json")
	replayBundleSource, err := json.Marshal(fixture["replay_bundle"])
	if err != nil {
		t.Fatalf("marshal replay bundle: %v", err)
	}
	var replayBundle astmerge.ReviewReplayBundle
	if err := json.Unmarshal(replayBundleSource, &replayBundle); err != nil {
		t.Fatalf("decode replay bundle: %v", err)
	}
	reviewStateSource, err := json.Marshal(fixture["review_state"])
	if err != nil {
		t.Fatalf("marshal review state: %v", err)
	}
	var reviewState astmerge.ConformanceManifestReviewState
	if err := json.Unmarshal(reviewStateSource, &reviewState); err != nil {
		t.Fatalf("decode review state: %v", err)
	}

	replayResult := MergeMarkdownWithReviewedNestedOutputsFromReplayBundle(
		fixture["template"].(string),
		fixture["destination"].(string),
		DialectMarkdown,
		replayBundle,
		BackendKreuzberg,
	)
	if replayResult.OK || replayResult.Output != nil || len(replayResult.Diagnostics) != 1 || replayResult.Diagnostics[0].Message != fixture["expected"].(map[string]any)["diagnostics"].([]any)[0].(map[string]any)["message"].(string) {
		t.Fatalf("unexpected replay-bundle rejection: %+v", replayResult)
	}

	stateResult := MergeMarkdownWithReviewedNestedOutputsFromReviewState(
		fixture["template"].(string),
		fixture["destination"].(string),
		DialectMarkdown,
		reviewState,
		BackendKreuzberg,
	)
	if stateResult.OK || stateResult.Output != nil || len(stateResult.Diagnostics) != 1 || stateResult.Diagnostics[0].Message != fixture["expected_review_state"].(map[string]any)["diagnostics"].([]any)[0].(map[string]any)["message"].(string) {
		t.Fatalf("unexpected review-state rejection: %+v", stateResult)
	}
}

func TestSharedFixtureMarkdownReviewedNestedReviewArtifactEnvelopeApplication(t *testing.T) {
	fixture := readMarkdownFixture(t, "markdown", "slice-313-reviewed-nested-review-artifact-envelope-application", "fenced-code-reviewed-nested-review-artifact-envelope-application.json")
	replayEnvelopeSource, err := json.Marshal(fixture["replay_bundle_envelope"])
	if err != nil {
		t.Fatalf("marshal replay bundle envelope: %v", err)
	}
	var replayEnvelope astmerge.ReviewReplayBundleEnvelope
	if err := json.Unmarshal(replayEnvelopeSource, &replayEnvelope); err != nil {
		t.Fatalf("decode replay bundle envelope: %v", err)
	}
	reviewStateEnvelopeSource, err := json.Marshal(fixture["review_state_envelope"])
	if err != nil {
		t.Fatalf("marshal review state envelope: %v", err)
	}
	var reviewStateEnvelope astmerge.ConformanceManifestReviewStateEnvelope
	if err := json.Unmarshal(reviewStateEnvelopeSource, &reviewStateEnvelope); err != nil {
		t.Fatalf("decode review state envelope: %v", err)
	}

	replayResult := MergeMarkdownWithReviewedNestedOutputsFromReplayBundleEnvelope(
		fixture["template"].(string),
		fixture["destination"].(string),
		DialectMarkdown,
		replayEnvelope,
		BackendKreuzberg,
	)
	if !replayResult.OK || replayResult.Output == nil || *replayResult.Output != fixture["expected"].(map[string]any)["output"].(string) {
		t.Fatalf("unexpected replay-bundle-envelope reviewed nested merge: %+v", replayResult)
	}

	stateResult := MergeMarkdownWithReviewedNestedOutputsFromReviewStateEnvelope(
		fixture["template"].(string),
		fixture["destination"].(string),
		DialectMarkdown,
		reviewStateEnvelope,
		BackendKreuzberg,
	)
	if !stateResult.OK || stateResult.Output == nil || *stateResult.Output != fixture["expected"].(map[string]any)["output"].(string) {
		t.Fatalf("unexpected review-state-envelope reviewed nested merge: %+v", stateResult)
	}
}

func TestSharedFixtureMarkdownReviewedNestedReviewArtifactEnvelopeRejection(t *testing.T) {
	fixture := readMarkdownFixture(t, "markdown", "slice-315-reviewed-nested-review-artifact-envelope-rejection", "fenced-code-reviewed-nested-review-artifact-envelope-rejection.json")
	replayEnvelopeSource, err := json.Marshal(fixture["replay_bundle_envelope"])
	if err != nil {
		t.Fatalf("marshal replay bundle envelope: %v", err)
	}
	var replayEnvelope astmerge.ReviewReplayBundleEnvelope
	if err := json.Unmarshal(replayEnvelopeSource, &replayEnvelope); err != nil {
		t.Fatalf("decode replay bundle envelope: %v", err)
	}
	reviewStateEnvelopeSource, err := json.Marshal(fixture["review_state_envelope"])
	if err != nil {
		t.Fatalf("marshal review state envelope: %v", err)
	}
	var reviewStateEnvelope astmerge.ConformanceManifestReviewStateEnvelope
	if err := json.Unmarshal(reviewStateEnvelopeSource, &reviewStateEnvelope); err != nil {
		t.Fatalf("decode review state envelope: %v", err)
	}

	replayResult := MergeMarkdownWithReviewedNestedOutputsFromReplayBundleEnvelope(
		fixture["template"].(string),
		fixture["destination"].(string),
		DialectMarkdown,
		replayEnvelope,
		BackendKreuzberg,
	)
	if replayResult.OK || replayResult.Output != nil || len(replayResult.Diagnostics) != 1 || replayResult.Diagnostics[0].Message != fixture["expected_replay_bundle"].(map[string]any)["diagnostics"].([]any)[0].(map[string]any)["message"].(string) {
		t.Fatalf("unexpected replay-bundle-envelope rejection: %+v", replayResult)
	}

	stateResult := MergeMarkdownWithReviewedNestedOutputsFromReviewStateEnvelope(
		fixture["template"].(string),
		fixture["destination"].(string),
		DialectMarkdown,
		reviewStateEnvelope,
		BackendKreuzberg,
	)
	if stateResult.OK || stateResult.Output != nil || len(stateResult.Diagnostics) != 1 || stateResult.Diagnostics[0].Message != fixture["expected_review_state"].(map[string]any)["diagnostics"].([]any)[0].(map[string]any)["message"].(string) {
		t.Fatalf("unexpected review-state-envelope rejection: %+v", stateResult)
	}
}

func TestSharedFixtureMarkdownEmbeddedFamilies(t *testing.T) {
	fixture := readMarkdownFixture(t, "markdown", "slice-208-embedded-families", "code-fence-families.json")
	result := ParseMarkdownWithBackend(fixture["source"].(string), DialectMarkdown, BackendKreuzberg)
	if !result.OK || result.Analysis == nil {
		t.Fatalf("expected parse success: %+v", result)
	}
	if actual := jsonReadyMarkdown(t, MarkdownEmbeddedFamilies(*result.Analysis)); !reflect.DeepEqual(actual, fixture["expected"]) {
		t.Fatalf("unexpected embedded families: %+v", actual)
	}
}

func TestSharedFixtureMarkdownDiscoveredSurfaces(t *testing.T) {
	fixture := readMarkdownFixture(t, "markdown", "slice-212-discovered-surfaces", "fenced-code-surfaces.json")
	result := ParseMarkdownWithBackend(fixture["source"].(string), DialectMarkdown, BackendKreuzberg)
	if !result.OK || result.Analysis == nil {
		t.Fatalf("expected parse success: %+v", result)
	}

	if actual := jsonReadyMarkdown(t, MarkdownDiscoveredSurfaces(*result.Analysis)); !reflect.DeepEqual(actual, fixture["expected"]) {
		t.Fatalf("unexpected discovered surfaces: %+v", actual)
	}
}

func TestSharedFixtureMarkdownDelegatedChildOperations(t *testing.T) {
	fixture := readMarkdownFixture(t, "markdown", "slice-213-delegated-child-operations", "fenced-code-child-operations.json")
	result := ParseMarkdownWithBackend(fixture["source"].(string), DialectMarkdown, BackendKreuzberg)
	if !result.OK || result.Analysis == nil {
		t.Fatalf("expected parse success: %+v", result)
	}

	if actual := jsonReadyMarkdown(t, MarkdownDelegatedChildOperations(*result.Analysis, fixture["parent_operation_id"].(string))); !reflect.DeepEqual(actual, fixture["expected"]) {
		t.Fatalf("unexpected delegated child operations: %+v", actual)
	}
}

func TestSharedFixtureMarkdownProjectedChildReviewGroups(t *testing.T) {
	fixture := readMarkdownFixture(t, "markdown", "slice-228-projected-child-review-groups", "fenced-code-review-groups.json")
	source, err := json.Marshal(fixture["cases"])
	if err != nil {
		t.Fatalf("marshal projected cases: %v", err)
	}
	var cases []astmerge.ProjectedChildReviewCase
	if err := json.Unmarshal(source, &cases); err != nil {
		t.Fatalf("unmarshal projected cases: %v", err)
	}
	if actual := jsonReadyMarkdown(t, astmerge.GroupProjectedChildReviewCases(cases)); !reflect.DeepEqual(actual, fixture["expected_groups"]) {
		t.Fatalf("unexpected projected child review groups: %+v", actual)
	}
}

func TestSharedFixtureMarkdownProjectedChildReviewGroupProgress(t *testing.T) {
	fixture := readMarkdownFixture(t, "markdown", "slice-231-projected-child-review-group-progress", "fenced-code-review-progress.json")
	groupSource, err := json.Marshal(fixture["groups"])
	if err != nil {
		t.Fatalf("marshal projected groups: %v", err)
	}
	var groups []astmerge.ProjectedChildReviewGroup
	if err := json.Unmarshal(groupSource, &groups); err != nil {
		t.Fatalf("unmarshal projected groups: %v", err)
	}
	resolvedSource, err := json.Marshal(fixture["resolved_case_ids"])
	if err != nil {
		t.Fatalf("marshal resolved case ids: %v", err)
	}
	var resolvedCaseIDs []string
	if err := json.Unmarshal(resolvedSource, &resolvedCaseIDs); err != nil {
		t.Fatalf("unmarshal resolved case ids: %v", err)
	}
	if actual := jsonReadyMarkdown(t, astmerge.SummarizeProjectedChildReviewGroupProgress(groups, resolvedCaseIDs)); !reflect.DeepEqual(actual, fixture["expected_progress"]) {
		t.Fatalf("unexpected projected child review group progress: %+v", actual)
	}
}

func TestSharedFixtureMarkdownProjectedChildReviewGroupsReadyForApply(t *testing.T) {
	fixture := readMarkdownFixture(t, "markdown", "slice-234-projected-child-review-groups-ready-for-apply", "fenced-code-ready-groups.json")
	groupSource, err := json.Marshal(fixture["groups"])
	if err != nil {
		t.Fatalf("marshal projected groups: %v", err)
	}
	var groups []astmerge.ProjectedChildReviewGroup
	if err := json.Unmarshal(groupSource, &groups); err != nil {
		t.Fatalf("unmarshal projected groups: %v", err)
	}
	resolvedSource, err := json.Marshal(fixture["resolved_case_ids"])
	if err != nil {
		t.Fatalf("marshal resolved case ids: %v", err)
	}
	var resolvedCaseIDs []string
	if err := json.Unmarshal(resolvedSource, &resolvedCaseIDs); err != nil {
		t.Fatalf("unmarshal resolved case ids: %v", err)
	}
	if actual := jsonReadyMarkdown(t, astmerge.SelectProjectedChildReviewGroupsReadyForApply(groups, resolvedCaseIDs)); !reflect.DeepEqual(actual, fixture["expected_ready_groups"]) {
		t.Fatalf("unexpected ready projected child review groups: %+v", actual)
	}
}

func TestSharedFixtureMarkdownDelegatedChildReviewTransport(t *testing.T) {
	fixture := readMarkdownFixture(t, "markdown", "slice-238-delegated-child-review-transport", "fenced-code-review-transport.json")
	group := parseProjectedChildReviewGroup(fixture["group"].(map[string]any))
	expectedRequest := parseReviewRequest(fixture["expected_request"].(map[string]any))

	if actual := astmerge.ProjectedChildGroupReviewRequest(group, fixture["family"].(string)); !reflect.DeepEqual(actual, expectedRequest) {
		t.Fatalf("unexpected delegated child review request: %+v", actual)
	}

	groupSource, err := json.Marshal(fixture["groups"])
	if err != nil {
		t.Fatalf("marshal projected groups: %v", err)
	}
	var groups []astmerge.ProjectedChildReviewGroup
	if err := json.Unmarshal(groupSource, &groups); err != nil {
		t.Fatalf("unmarshal projected groups: %v", err)
	}
	decisions := make([]astmerge.ReviewDecision, 0, len(fixture["decisions"].([]any)))
	for _, item := range fixture["decisions"].([]any) {
		decisions = append(decisions, parseReviewDecision(item.(map[string]any)))
	}
	if actual := jsonReadyMarkdown(t, astmerge.SelectProjectedChildReviewGroupsAcceptedForApply(groups, fixture["family"].(string), decisions)); !reflect.DeepEqual(actual, fixture["expected_accepted_groups"]) {
		t.Fatalf("unexpected accepted projected child review groups: %+v", actual)
	}
}

func TestSharedFixtureMarkdownDelegatedChildReviewState(t *testing.T) {
	fixture := readMarkdownFixture(t, "markdown", "slice-241-delegated-child-review-state", "fenced-code-review-state.json")
	groupSource, err := json.Marshal(fixture["groups"])
	if err != nil {
		t.Fatalf("marshal projected groups: %v", err)
	}
	var groups []astmerge.ProjectedChildReviewGroup
	if err := json.Unmarshal(groupSource, &groups); err != nil {
		t.Fatalf("unmarshal projected groups: %v", err)
	}
	decisions := make([]astmerge.ReviewDecision, 0, len(fixture["decisions"].([]any)))
	for _, item := range fixture["decisions"].([]any) {
		decisions = append(decisions, parseReviewDecision(item.(map[string]any)))
	}
	if actual := jsonReadyMarkdown(t, astmerge.ReviewProjectedChildGroups(groups, fixture["family"].(string), decisions)); !reflect.DeepEqual(actual, fixture["expected_state"]) {
		t.Fatalf("unexpected delegated child review state: %+v", actual)
	}
}

func TestSharedFixtureMarkdownDelegatedChildApplyPlan(t *testing.T) {
	fixture := readMarkdownFixture(t, "markdown", "slice-244-delegated-child-apply-plan", "fenced-code-apply-plan.json")
	stateSource, err := json.Marshal(fixture["review_state"])
	if err != nil {
		t.Fatalf("marshal delegated child review state: %v", err)
	}
	var state astmerge.DelegatedChildGroupReviewState
	if err := json.Unmarshal(stateSource, &state); err != nil {
		t.Fatalf("unmarshal delegated child review state: %v", err)
	}
	if actual := jsonReadyMarkdown(t, astmerge.DelegatedChildApplyPlanForState(state, fixture["family"].(string))); !reflect.DeepEqual(actual, fixture["expected_plan"]) {
		t.Fatalf("unexpected delegated child apply plan: %+v", actual)
	}
}
