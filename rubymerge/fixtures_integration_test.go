package rubymerge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
)

func readRubyFixture(t *testing.T, parts ...string) map[string]any {
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

func jsonReadyRuby(t *testing.T, value any) any {
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

func TestRubyFixtures(t *testing.T) {
	profileFixture := readRubyFixture(t, "diagnostics", "slice-214-ruby-family-feature-profile", "ruby-feature-profile.json")
	if RubyFeatureProfileInfo().Family != profileFixture["feature_profile"].(map[string]any)["family"].(string) {
		t.Fatal("unexpected profile family")
	}

	backendFixture := readRubyFixture(t, "diagnostics", "slice-215-ruby-family-backend-feature-profiles", "ruby-ruby-backend-feature-profiles.json")
	backends := AvailableRubyBackends()
	if len(backends) != 1 || backends[0] != "kreuzberg-language-pack" {
		t.Fatalf("unexpected backends: %+v", backends)
	}
	if backend := RubyBackendFeatureProfileInfo(); backend.Backend != backendFixture["tree_sitter"].(map[string]any)["backend"].(string) {
		t.Fatalf("unexpected backend profile: %+v", backend)
	}
	if actual := jsonReadyRuby(t, map[string]any{
		"backend":            RubyBackendFeatureProfileInfo().Backend,
		"supports_dialects":  RubyBackendFeatureProfileInfo().SupportsDialects,
		"supported_policies": RubyBackendFeatureProfileInfo().SupportedPolicies,
		"backend_ref": map[string]any{
			"id":     RubyBackendFeatureProfileInfo().BackendRef.ID,
			"family": RubyBackendFeatureProfileInfo().BackendRef.Family,
		},
	}); !reflect.DeepEqual(actual, backendFixture["tree_sitter"]) {
		t.Fatalf("unexpected backend fixture projection: %+v", actual)
	}

	planFixture := readRubyFixture(t, "diagnostics", "slice-216-ruby-family-plan-contexts", "ruby-ruby-plan-contexts.json")
	planContext := RubyPlanContext()
	if planContext.FamilyProfile.Family != planFixture["tree_sitter"].(map[string]any)["family_profile"].(map[string]any)["family"].(string) {
		t.Fatalf("unexpected plan family profile: %+v", planContext)
	}
	if planContext.FeatureProfile == nil || planContext.FeatureProfile.Backend != planFixture["tree_sitter"].(map[string]any)["feature_profile"].(map[string]any)["backend"].(string) {
		t.Fatalf("unexpected plan feature profile: %+v", planContext.FeatureProfile)
	}

	manifestFixture := readRubyFixture(t, "conformance", "slice-217-ruby-family-manifest", "ruby-family-manifest.json")
	manifestSource, err := json.Marshal(manifestFixture)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	var manifest astmerge.ConformanceManifest
	if err := json.Unmarshal(manifestSource, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}
	if path := astmerge.ConformanceFamilyFeatureProfilePath(manifest, "ruby"); path == nil || filepath.Join(path...) != filepath.Join("diagnostics", "slice-214-ruby-family-feature-profile", "ruby-feature-profile.json") {
		t.Fatalf("unexpected ruby family profile path: %v", path)
	}
	if path := astmerge.ConformanceFixturePath(manifest, "ruby", "analysis"); path == nil || filepath.Join(path...) != filepath.Join("ruby", "slice-218-analysis", "module-owners.json") {
		t.Fatalf("unexpected ruby analysis path: %v", path)
	}
	if path := astmerge.ConformanceFixturePath(manifest, "ruby", "matching"); path == nil || filepath.Join(path...) != filepath.Join("ruby", "slice-219-matching", "path-equality.json") {
		t.Fatalf("unexpected ruby matching path: %v", path)
	}
	if path := astmerge.ConformanceFixturePath(manifest, "ruby", "merge"); path == nil || filepath.Join(path...) != filepath.Join("ruby", "slice-287-merge", "module-merge.json") {
		t.Fatalf("unexpected ruby merge path: %v", path)
	}

	analysisFixture := readRubyFixture(t, "ruby", "slice-218-analysis", "module-owners.json")
	analysis := ParseRuby(analysisFixture["source"].(string), DialectRuby)
	if !analysis.OK || analysis.Analysis == nil {
		t.Fatalf("unexpected analysis: %+v", analysis)
	}
	if len(analysis.Analysis.Owners) != len(analysisFixture["expected"].(map[string]any)["owners"].([]any)) {
		t.Fatalf("unexpected owners: %+v", analysis.Analysis.Owners)
	}

	matchingFixture := readRubyFixture(t, "ruby", "slice-219-matching", "path-equality.json")
	template := ParseRuby(matchingFixture["template"].(string), DialectRuby)
	destination := ParseRuby(matchingFixture["destination"].(string), DialectRuby)
	match := MatchRubyOwners(*template.Analysis, *destination.Analysis)
	if len(match.Matched) != len(matchingFixture["expected"].(map[string]any)["matched"].([]any)) {
		t.Fatalf("unexpected matches: %+v", match)
	}

	mergeFixture := readRubyFixture(t, "ruby", "slice-287-merge", "module-merge.json")
	mergeResult := MergeRuby(mergeFixture["template"].(string), mergeFixture["destination"].(string), DialectRuby)
	if !mergeResult.OK || mergeResult.Output == nil {
		t.Fatalf("expected merge success: %+v", mergeResult)
	}
	if *mergeResult.Output != mergeFixture["expected"].(map[string]any)["output"].(string) {
		t.Fatalf("unexpected merge output:\n%s", *mergeResult.Output)
	}

	applyOutputFixture := readRubyFixture(t, "ruby", "slice-289-delegated-child-apply-output", "yard-example-applied-output.json")
	operationsSource, err := json.Marshal(applyOutputFixture["delegated_operations"])
	if err != nil {
		t.Fatalf("marshal delegated operations: %v", err)
	}
	var operations []astmerge.DelegatedChildOperation
	if err := json.Unmarshal(operationsSource, &operations); err != nil {
		t.Fatalf("decode delegated operations: %v", err)
	}
	applyPlanSource, err := json.Marshal(applyOutputFixture["apply_plan"])
	if err != nil {
		t.Fatalf("marshal apply plan: %v", err)
	}
	var applyPlan astmerge.DelegatedChildApplyPlan
	if err := json.Unmarshal(applyPlanSource, &applyPlan); err != nil {
		t.Fatalf("decode apply plan: %v", err)
	}
	childrenSource, err := json.Marshal(applyOutputFixture["applied_children"])
	if err != nil {
		t.Fatalf("marshal applied children: %v", err)
	}
	var appliedChildren []AppliedChildOutput
	if err := json.Unmarshal(childrenSource, &appliedChildren); err != nil {
		t.Fatalf("decode applied children: %v", err)
	}
	applyOutputResult := ApplyRubyDelegatedChildOutputs(
		applyOutputFixture["source"].(string),
		operations,
		applyPlan,
		appliedChildren,
	)
	if !applyOutputResult.OK || applyOutputResult.Output == nil {
		t.Fatalf("expected delegated child apply output success: %+v", applyOutputResult)
	}
	if *applyOutputResult.Output != applyOutputFixture["expected"].(map[string]any)["output"].(string) {
		t.Fatalf("unexpected delegated child applied output:\n%s", *applyOutputResult.Output)
	}

	nestedMergeFixture := readRubyFixture(t, "ruby", "slice-291-nested-merge", "yard-example-nested-merge.json")
	nestedOutputsSource, err := json.Marshal(nestedMergeFixture["nested_outputs"])
	if err != nil {
		t.Fatalf("marshal nested outputs: %v", err)
	}
	var nestedOutputs []NestedChildOutput
	if err := json.Unmarshal(nestedOutputsSource, &nestedOutputs); err != nil {
		t.Fatalf("decode nested outputs: %v", err)
	}
	nestedMergeResult := MergeRubyWithNestedOutputs(
		nestedMergeFixture["template"].(string),
		nestedMergeFixture["destination"].(string),
		DialectRuby,
		nestedOutputs,
	)
	if !nestedMergeResult.OK || nestedMergeResult.Output == nil {
		t.Fatalf("expected nested merge success: %+v", nestedMergeResult)
	}
	if *nestedMergeResult.Output != nestedMergeFixture["expected"].(map[string]any)["output"].(string) {
		t.Fatalf("unexpected nested merge output:\n%s", *nestedMergeResult.Output)
	}

	reviewedNestedMergeFixture := readRubyFixture(t, "ruby", "slice-299-reviewed-nested-merge", "yard-example-reviewed-nested-merge.json")
	reviewedStateSource, err := json.Marshal(reviewedNestedMergeFixture["review_state"])
	if err != nil {
		t.Fatalf("marshal reviewed nested merge state: %v", err)
	}
	var reviewedState astmerge.DelegatedChildGroupReviewState
	if err := json.Unmarshal(reviewedStateSource, &reviewedState); err != nil {
		t.Fatalf("decode reviewed nested merge state: %v", err)
	}
	reviewedChildrenSource, err := json.Marshal(reviewedNestedMergeFixture["applied_children"])
	if err != nil {
		t.Fatalf("marshal reviewed nested merge applied children: %v", err)
	}
	var reviewedChildren []AppliedChildOutput
	if err := json.Unmarshal(reviewedChildrenSource, &reviewedChildren); err != nil {
		t.Fatalf("decode reviewed nested merge applied children: %v", err)
	}
	reviewedNestedMergeResult := MergeRubyWithReviewedNestedOutputs(
		reviewedNestedMergeFixture["template"].(string),
		reviewedNestedMergeFixture["destination"].(string),
		DialectRuby,
		reviewedState,
		reviewedChildren,
	)
	if !reviewedNestedMergeResult.OK || reviewedNestedMergeResult.Output == nil {
		t.Fatalf("expected reviewed nested merge success: %+v", reviewedNestedMergeResult)
	}
	if *reviewedNestedMergeResult.Output != reviewedNestedMergeFixture["expected"].(map[string]any)["output"].(string) {
		t.Fatalf("unexpected reviewed nested merge output:\n%s", *reviewedNestedMergeResult.Output)
	}

	invalidTemplateFixture := readRubyFixture(t, "ruby", "slice-287-merge", "invalid-template.json")
	invalidTemplateResult := MergeRuby(invalidTemplateFixture["template"].(string), invalidTemplateFixture["destination"].(string), DialectRuby)
	if invalidTemplateResult.OK {
		t.Fatalf("expected invalid template merge failure: %+v", invalidTemplateResult)
	}
	if actual := jsonReadyRuby(t, []map[string]any{{
		"severity": invalidTemplateResult.Diagnostics[0].Severity,
		"category": invalidTemplateResult.Diagnostics[0].Category,
	}}); !reflect.DeepEqual(actual, invalidTemplateFixture["expected"].(map[string]any)["diagnostics"]) {
		t.Fatalf("unexpected invalid template diagnostics: %+v", actual)
	}

	invalidDestinationFixture := readRubyFixture(t, "ruby", "slice-287-merge", "invalid-destination.json")
	invalidDestinationResult := MergeRuby(invalidDestinationFixture["template"].(string), invalidDestinationFixture["destination"].(string), DialectRuby)
	if invalidDestinationResult.OK {
		t.Fatalf("expected invalid destination merge failure: %+v", invalidDestinationResult)
	}
	if actual := jsonReadyRuby(t, []map[string]any{{
		"severity": invalidDestinationResult.Diagnostics[0].Severity,
		"category": invalidDestinationResult.Diagnostics[0].Category,
	}}); !reflect.DeepEqual(actual, invalidDestinationFixture["expected"].(map[string]any)["diagnostics"]) {
		t.Fatalf("unexpected invalid destination diagnostics: %+v", actual)
	}

	surfacesFixture := readRubyFixture(t, "ruby", "slice-220-discovered-surfaces", "doc-comment-surfaces.json")
	surfaceAnalysis := ParseRuby(surfacesFixture["source"].(string), DialectRuby)
	encodedSurfaces, err := json.Marshal(RubyDiscoveredSurfaces(*surfaceAnalysis.Analysis))
	if err != nil {
		t.Fatalf("marshal surfaces: %v", err)
	}
	var surfaceValue any
	if err := json.Unmarshal(encodedSurfaces, &surfaceValue); err != nil {
		t.Fatalf("unmarshal surfaces: %v", err)
	}
	if !deepEqualJSON(surfaceValue, surfacesFixture["expected"]) {
		t.Fatalf("unexpected surfaces: %+v", surfaceValue)
	}

	childFixture := readRubyFixture(t, "ruby", "slice-221-delegated-child-operations", "yard-example-child-operations.json")
	childAnalysis := ParseRuby(childFixture["source"].(string), DialectRuby)
	encodedChildren, err := json.Marshal(RubyDelegatedChildOperations(*childAnalysis.Analysis, childFixture["parent_operation_id"].(string)))
	if err != nil {
		t.Fatalf("marshal child operations: %v", err)
	}
	var childValue any
	if err := json.Unmarshal(encodedChildren, &childValue); err != nil {
		t.Fatalf("unmarshal child operations: %v", err)
	}
	if !deepEqualJSON(childValue, childFixture["expected"]) {
		t.Fatalf("unexpected child operations: %+v", childValue)
	}

	groupedFixture := readRubyFixture(t, "ruby", "slice-229-projected-child-review-groups", "yard-example-review-groups.json")
	groupedSource, err := json.Marshal(groupedFixture["cases"])
	if err != nil {
		t.Fatalf("marshal projected cases: %v", err)
	}
	var groupedCases []astmerge.ProjectedChildReviewCase
	if err := json.Unmarshal(groupedSource, &groupedCases); err != nil {
		t.Fatalf("unmarshal projected cases: %v", err)
	}
	groupedValue, err := json.Marshal(astmerge.GroupProjectedChildReviewCases(groupedCases))
	if err != nil {
		t.Fatalf("marshal projected groups: %v", err)
	}
	var decodedGroups any
	if err := json.Unmarshal(groupedValue, &decodedGroups); err != nil {
		t.Fatalf("unmarshal projected groups: %v", err)
	}
	if !deepEqualJSON(decodedGroups, groupedFixture["expected_groups"]) {
		t.Fatalf("unexpected projected groups: %+v", decodedGroups)
	}

	progressFixture := readRubyFixture(t, "ruby", "slice-232-projected-child-review-group-progress", "yard-example-review-progress.json")
	progressSource, err := json.Marshal(progressFixture["groups"])
	if err != nil {
		t.Fatalf("marshal projected groups: %v", err)
	}
	var progressGroups []astmerge.ProjectedChildReviewGroup
	if err := json.Unmarshal(progressSource, &progressGroups); err != nil {
		t.Fatalf("unmarshal projected groups: %v", err)
	}
	resolvedSource, err := json.Marshal(progressFixture["resolved_case_ids"])
	if err != nil {
		t.Fatalf("marshal resolved case ids: %v", err)
	}
	var resolvedCaseIDs []string
	if err := json.Unmarshal(resolvedSource, &resolvedCaseIDs); err != nil {
		t.Fatalf("unmarshal resolved case ids: %v", err)
	}
	progressValue, err := json.Marshal(astmerge.SummarizeProjectedChildReviewGroupProgress(progressGroups, resolvedCaseIDs))
	if err != nil {
		t.Fatalf("marshal projected progress: %v", err)
	}
	var decodedProgress any
	if err := json.Unmarshal(progressValue, &decodedProgress); err != nil {
		t.Fatalf("unmarshal projected progress: %v", err)
	}
	if !deepEqualJSON(decodedProgress, progressFixture["expected_progress"]) {
		t.Fatalf("unexpected projected progress: %+v", decodedProgress)
	}

	readyFixture := readRubyFixture(t, "ruby", "slice-235-projected-child-review-groups-ready-for-apply", "yard-example-ready-groups.json")
	readySource, err := json.Marshal(readyFixture["groups"])
	if err != nil {
		t.Fatalf("marshal ready groups: %v", err)
	}
	var readyGroups []astmerge.ProjectedChildReviewGroup
	if err := json.Unmarshal(readySource, &readyGroups); err != nil {
		t.Fatalf("unmarshal ready groups: %v", err)
	}
	readyResolvedSource, err := json.Marshal(readyFixture["resolved_case_ids"])
	if err != nil {
		t.Fatalf("marshal ready resolved case ids: %v", err)
	}
	var readyResolvedCaseIDs []string
	if err := json.Unmarshal(readyResolvedSource, &readyResolvedCaseIDs); err != nil {
		t.Fatalf("unmarshal ready resolved case ids: %v", err)
	}
	readyValue, err := json.Marshal(astmerge.SelectProjectedChildReviewGroupsReadyForApply(readyGroups, readyResolvedCaseIDs))
	if err != nil {
		t.Fatalf("marshal ready groups: %v", err)
	}
	var decodedReady any
	if err := json.Unmarshal(readyValue, &decodedReady); err != nil {
		t.Fatalf("unmarshal ready groups: %v", err)
	}
	if !deepEqualJSON(decodedReady, readyFixture["expected_ready_groups"]) {
		t.Fatalf("unexpected ready groups: %+v", decodedReady)
	}

	transportFixture := readRubyFixture(t, "ruby", "slice-239-delegated-child-review-transport", "yard-example-review-transport.json")
	transportGroup := parseProjectedChildReviewGroup(transportFixture["group"].(map[string]any))
	expectedRequest := parseReviewRequest(transportFixture["expected_request"].(map[string]any))
	if actual := astmerge.ProjectedChildGroupReviewRequest(transportGroup, transportFixture["family"].(string)); !deepEqualJSON(actual, expectedRequest) {
		t.Fatalf("unexpected delegated child review request: %+v", actual)
	}
	transportSource, err := json.Marshal(transportFixture["groups"])
	if err != nil {
		t.Fatalf("marshal transport groups: %v", err)
	}
	var transportGroups []astmerge.ProjectedChildReviewGroup
	if err := json.Unmarshal(transportSource, &transportGroups); err != nil {
		t.Fatalf("unmarshal transport groups: %v", err)
	}
	transportDecisions := make([]astmerge.ReviewDecision, 0, len(transportFixture["decisions"].([]any)))
	for _, item := range transportFixture["decisions"].([]any) {
		transportDecisions = append(transportDecisions, parseReviewDecision(item.(map[string]any)))
	}
	acceptedValue, err := json.Marshal(astmerge.SelectProjectedChildReviewGroupsAcceptedForApply(transportGroups, transportFixture["family"].(string), transportDecisions))
	if err != nil {
		t.Fatalf("marshal accepted groups: %v", err)
	}
	var decodedAccepted any
	if err := json.Unmarshal(acceptedValue, &decodedAccepted); err != nil {
		t.Fatalf("unmarshal accepted groups: %v", err)
	}
	if !deepEqualJSON(decodedAccepted, transportFixture["expected_accepted_groups"]) {
		t.Fatalf("unexpected accepted groups: %+v", decodedAccepted)
	}

	stateFixture := readRubyFixture(t, "ruby", "slice-242-delegated-child-review-state", "yard-example-review-state.json")
	stateSource, err := json.Marshal(stateFixture["groups"])
	if err != nil {
		t.Fatalf("marshal state groups: %v", err)
	}
	var stateGroups []astmerge.ProjectedChildReviewGroup
	if err := json.Unmarshal(stateSource, &stateGroups); err != nil {
		t.Fatalf("unmarshal state groups: %v", err)
	}
	stateDecisions := make([]astmerge.ReviewDecision, 0, len(stateFixture["decisions"].([]any)))
	for _, item := range stateFixture["decisions"].([]any) {
		stateDecisions = append(stateDecisions, parseReviewDecision(item.(map[string]any)))
	}
	stateValue, err := json.Marshal(astmerge.ReviewProjectedChildGroups(stateGroups, stateFixture["family"].(string), stateDecisions))
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}
	var decodedState any
	if err := json.Unmarshal(stateValue, &decodedState); err != nil {
		t.Fatalf("unmarshal state: %v", err)
	}
	if !deepEqualJSON(decodedState, stateFixture["expected_state"]) {
		t.Fatalf("unexpected delegated child review state: %+v", decodedState)
	}

	applyPlanFixture := readRubyFixture(t, "ruby", "slice-245-delegated-child-apply-plan", "yard-example-apply-plan.json")
	applyPlanSource, err = json.Marshal(applyPlanFixture["review_state"])
	if err != nil {
		t.Fatalf("marshal delegated child review state: %v", err)
	}
	var applyPlanState astmerge.DelegatedChildGroupReviewState
	if err := json.Unmarshal(applyPlanSource, &applyPlanState); err != nil {
		t.Fatalf("unmarshal delegated child review state: %v", err)
	}
	applyPlanValue, err := json.Marshal(astmerge.DelegatedChildApplyPlanForState(applyPlanState, applyPlanFixture["family"].(string)))
	if err != nil {
		t.Fatalf("marshal delegated child apply plan: %v", err)
	}
	var decodedApplyPlan any
	if err := json.Unmarshal(applyPlanValue, &decodedApplyPlan); err != nil {
		t.Fatalf("unmarshal delegated child apply plan: %v", err)
	}
	if !deepEqualJSON(decodedApplyPlan, applyPlanFixture["expected_plan"]) {
		t.Fatalf("unexpected delegated child apply plan: %+v", decodedApplyPlan)
	}
}

func deepEqualJSON(left any, right any) bool {
	leftJSON, _ := json.Marshal(left)
	rightJSON, _ := json.Marshal(right)
	return string(leftJSON) == string(rightJSON)
}
