package astmerge

import (
	"encoding/json"
	"slices"
	"strconv"
	"strings"
)

type DiagnosticSeverity string

const (
	SeverityInfo    DiagnosticSeverity = "info"
	SeverityWarning DiagnosticSeverity = "warning"
	SeverityError   DiagnosticSeverity = "error"
)

type DiagnosticCategory string

const (
	CategoryParseError            DiagnosticCategory = "parse_error"
	CategoryDestinationParseError DiagnosticCategory = "destination_parse_error"
	CategoryUnsupportedFeature    DiagnosticCategory = "unsupported_feature"
	CategoryFallbackApplied       DiagnosticCategory = "fallback_applied"
	CategoryAmbiguity             DiagnosticCategory = "ambiguity"
	CategoryAssumedDefault        DiagnosticCategory = "assumed_default"
	CategoryConfigurationError    DiagnosticCategory = "configuration_error"
	CategoryReplayRejected        DiagnosticCategory = "replay_rejected"
)

type ReviewDiagnosticReason string

const (
	ReasonMissingRequiredPayload ReviewDiagnosticReason = "missing_required_payload"
	ReasonFamilyMismatch         ReviewDiagnosticReason = "family_mismatch"
	ReasonRequestNotFound        ReviewDiagnosticReason = "request_not_found"
)

type ReviewDiagnosticDetail struct {
	RequestID      string                 `json:"request_id,omitempty"`
	Action         ReviewDecisionAction   `json:"action,omitempty"`
	Reason         ReviewDiagnosticReason `json:"reason,omitempty"`
	PayloadKind    string                 `json:"payload_kind,omitempty"`
	ExpectedFamily string                 `json:"expected_family,omitempty"`
	ProvidedFamily string                 `json:"provided_family,omitempty"`
}

type Diagnostic struct {
	Severity DiagnosticSeverity      `json:"severity"`
	Category DiagnosticCategory      `json:"category"`
	Message  string                  `json:"message"`
	Path     string                  `json:"path,omitempty"`
	Review   *ReviewDiagnosticDetail `json:"review,omitempty"`
}

type SurfaceOwnerKind string

const (
	SurfaceOwnerStructuralOwner SurfaceOwnerKind = "structural_owner"
	SurfaceOwnerOwnedRegion     SurfaceOwnerKind = "owned_region"
	SurfaceOwnerParentSurface   SurfaceOwnerKind = "parent_surface"
)

type SurfaceOwnerRef struct {
	Kind    SurfaceOwnerKind `json:"kind"`
	Address string           `json:"address"`
}

type SurfaceSpan struct {
	StartLine int `json:"start_line"`
	EndLine   int `json:"end_line"`
}

type DiscoveredSurface struct {
	SurfaceKind            string          `json:"surface_kind"`
	DeclaredLanguage       string          `json:"declared_language,omitempty"`
	EffectiveLanguage      string          `json:"effective_language"`
	Address                string          `json:"address"`
	ParentAddress          string          `json:"parent_address,omitempty"`
	Span                   *SurfaceSpan    `json:"span,omitempty"`
	Owner                  SurfaceOwnerRef `json:"owner"`
	ReconstructionStrategy string          `json:"reconstruction_strategy"`
	Metadata               map[string]any  `json:"metadata,omitempty"`
}

type DelegatedChildOperation struct {
	OperationID       string            `json:"operation_id"`
	ParentOperationID string            `json:"parent_operation_id"`
	RequestedStrategy string            `json:"requested_strategy"`
	LanguageChain     []string          `json:"language_chain"`
	Surface           DiscoveredSurface `json:"surface"`
}

type ProjectedChildReviewCase struct {
	CaseID                      string `json:"case_id"`
	ParentOperationID           string `json:"parent_operation_id"`
	ChildOperationID            string `json:"child_operation_id"`
	SurfacePath                 string `json:"surface_path"`
	DelegatedCaseID             string `json:"delegated_case_id"`
	DelegatedApplyGroup         string `json:"delegated_apply_group"`
	DelegatedRuntimeSurfacePath string `json:"delegated_runtime_surface_path"`
}

type ProjectedChildReviewGroup struct {
	DelegatedApplyGroup         string   `json:"delegated_apply_group"`
	ParentOperationID           string   `json:"parent_operation_id"`
	ChildOperationID            string   `json:"child_operation_id"`
	DelegatedRuntimeSurfacePath string   `json:"delegated_runtime_surface_path"`
	CaseIDs                     []string `json:"case_ids"`
	DelegatedCaseIDs            []string `json:"delegated_case_ids"`
}

type ProjectedChildReviewGroupProgress struct {
	DelegatedApplyGroup         string   `json:"delegated_apply_group"`
	ParentOperationID           string   `json:"parent_operation_id"`
	ChildOperationID            string   `json:"child_operation_id"`
	DelegatedRuntimeSurfacePath string   `json:"delegated_runtime_surface_path"`
	ResolvedCaseIDs             []string `json:"resolved_case_ids"`
	PendingCaseIDs              []string `json:"pending_case_ids"`
	Complete                    bool     `json:"complete"`
}

type ParseResult[T any] struct {
	OK          bool
	Diagnostics []Diagnostic
	Analysis    *T
	Policies    []PolicyReference
}

type MergeResult[T any] struct {
	OK          bool
	Diagnostics []Diagnostic
	Output      *T
	Policies    []PolicyReference
}

type PolicySurface string

const (
	PolicySurfaceFallback PolicySurface = "fallback"
	PolicySurfaceArray    PolicySurface = "array"
)

type PolicyReference struct {
	Surface PolicySurface `json:"surface"`
	Name    string        `json:"name"`
}

type FamilyFeatureProfile struct {
	Family            string            `json:"family"`
	SupportedDialects []string          `json:"supported_dialects"`
	SupportedPolicies []PolicyReference `json:"supported_policies"`
}

type ConformanceOutcome string

const (
	ConformancePassed  ConformanceOutcome = "passed"
	ConformanceFailed  ConformanceOutcome = "failed"
	ConformanceSkipped ConformanceOutcome = "skipped"
)

type ConformanceCaseRef struct {
	Family string `json:"family"`
	Role   string `json:"role"`
	Case   string `json:"case"`
}

type ConformanceCaseResult struct {
	Ref      ConformanceCaseRef `json:"ref"`
	Outcome  ConformanceOutcome `json:"outcome"`
	Messages []string           `json:"messages"`
}

type ConformanceCaseRequirements struct {
	Backend  string            `json:"backend,omitempty"`
	Dialect  string            `json:"dialect,omitempty"`
	Policies []PolicyReference `json:"policies,omitempty"`
}

type ConformanceSelectionStatus string

const (
	ConformanceSelected         ConformanceSelectionStatus = "selected"
	ConformanceSelectionSkipped ConformanceSelectionStatus = "skipped"
)

type ConformanceCaseSelection struct {
	Ref      ConformanceCaseRef         `json:"ref"`
	Status   ConformanceSelectionStatus `json:"status"`
	Messages []string                   `json:"messages"`
}

type ConformanceManifestEntry struct {
	Role         string                       `json:"role"`
	Path         []string                     `json:"path"`
	Requirements *ConformanceCaseRequirements `json:"requirements,omitempty"`
}

type ConformanceFamilyFeatureProfileEntry struct {
	Family string   `json:"family"`
	Role   string   `json:"role"`
	Path   []string `json:"path"`
}

type ConformanceManifest struct {
	FamilyFeatureProfiles []ConformanceFamilyFeatureProfileEntry `json:"family_feature_profiles"`
	SuiteDescriptors      []ConformanceSuiteDefinition           `json:"suite_descriptors,omitempty"`
	Families              map[string][]ConformanceManifestEntry  `json:"families"`
}

type ConformanceSuiteSubject struct {
	Grammar string `json:"grammar"`
	Variant string `json:"variant,omitempty"`
}

type ConformanceSuiteSelector struct {
	Kind    string                  `json:"kind"`
	Subject ConformanceSuiteSubject `json:"subject"`
}

type ConformanceSuiteDefinition struct {
	Kind    string                  `json:"kind"`
	Subject ConformanceSuiteSubject `json:"subject"`
	Roles   []string                `json:"roles"`
}

type NamedConformanceSuiteReport struct {
	Suite  ConformanceSuiteDefinition `json:"suite"`
	Report ConformanceSuiteReport     `json:"report"`
}

type ConformanceFamilyPlanContext struct {
	FamilyProfile  FamilyFeatureProfile           `json:"family_profile"`
	FeatureProfile *ConformanceFeatureProfileView `json:"feature_profile,omitempty"`
}

type NamedConformanceSuitePlan struct {
	Suite ConformanceSuiteDefinition `json:"suite"`
	Plan  ConformanceSuitePlan       `json:"plan"`
}

type NamedConformanceSuiteResults struct {
	Suite   ConformanceSuiteDefinition `json:"suite"`
	Results []ConformanceCaseResult    `json:"results"`
}

type NamedConformanceSuiteReportEnvelope struct {
	Entries []NamedConformanceSuiteReport `json:"entries"`
	Summary ConformanceSuiteSummary       `json:"summary"`
}

type ConformanceManifestPlanningOptions struct {
	Contexts                map[string]ConformanceFamilyPlanContext `json:"contexts,omitempty"`
	FamilyProfiles          map[string]FamilyFeatureProfile         `json:"family_profiles,omitempty"`
	RequireExplicitContexts bool                                    `json:"require_explicit_contexts,omitempty"`
}

type ConformanceManifestReport struct {
	Report      NamedConformanceSuiteReportEnvelope `json:"report"`
	Diagnostics []Diagnostic                        `json:"diagnostics"`
}

type ReviewRequestKind string

const (
	ReviewRequestFamilyContext       ReviewRequestKind = "family_context"
	ReviewRequestDelegatedChildGroup ReviewRequestKind = "delegated_child_group"
)

type ReviewDecisionAction string

const (
	ReviewDecisionAcceptDefaultContext     ReviewDecisionAction = "accept_default_context"
	ReviewDecisionProvideExplicitContext   ReviewDecisionAction = "provide_explicit_context"
	ReviewDecisionApplyDelegatedChildGroup ReviewDecisionAction = "apply_delegated_child_group"
)

type ReviewActionOffer struct {
	Action          ReviewDecisionAction `json:"action"`
	RequiresContext bool                 `json:"requires_context"`
	PayloadKind     string               `json:"payload_kind,omitempty"`
}

type ReviewRequest struct {
	ID              string                        `json:"id"`
	Kind            ReviewRequestKind             `json:"kind"`
	Family          string                        `json:"family"`
	Message         string                        `json:"message"`
	Blocking        bool                          `json:"blocking"`
	ProposedContext *ConformanceFamilyPlanContext `json:"proposed_context,omitempty"`
	DelegatedGroup  *ProjectedChildReviewGroup    `json:"delegated_group,omitempty"`
	ActionOffers    []ReviewActionOffer           `json:"action_offers"`
	DefaultAction   ReviewDecisionAction          `json:"default_action,omitempty"`
}

type ReviewDecision struct {
	RequestID string                        `json:"request_id"`
	Action    ReviewDecisionAction          `json:"action"`
	Context   *ConformanceFamilyPlanContext `json:"context,omitempty"`
}

type DelegatedChildGroupReviewState struct {
	Requests         []ReviewRequest             `json:"requests"`
	AcceptedGroups   []ProjectedChildReviewGroup `json:"accepted_groups"`
	AppliedDecisions []ReviewDecision            `json:"applied_decisions"`
	Diagnostics      []Diagnostic                `json:"diagnostics"`
}

type DelegatedChildApplyPlanEntry struct {
	RequestID      string                    `json:"request_id"`
	Family         string                    `json:"family"`
	DelegatedGroup ProjectedChildReviewGroup `json:"delegated_group"`
	Decision       ReviewDecision            `json:"decision"`
}

type DelegatedChildApplyPlan struct {
	Entries []DelegatedChildApplyPlanEntry `json:"entries"`
}

type DelegatedChildSurfaceOutput struct {
	SurfaceAddress string `json:"surface_address"`
	Output         string `json:"output"`
}

type AppliedDelegatedChildOutput struct {
	OperationID string `json:"operation_id"`
	Output      string `json:"output"`
}

type DelegatedChildOutputResolutionOptions struct {
	DefaultFamily   string `json:"default_family"`
	RequestIDPrefix string `json:"request_id_prefix"`
}

type DelegatedChildOutputResolution struct {
	OK              bool                          `json:"ok"`
	Diagnostics     []Diagnostic                  `json:"diagnostics"`
	ApplyPlan       *DelegatedChildApplyPlan      `json:"apply_plan,omitempty"`
	AppliedChildren []AppliedDelegatedChildOutput `json:"applied_children,omitempty"`
}

type NestedMergeDiscoveryResult struct {
	OK          bool                      `json:"ok"`
	Diagnostics []Diagnostic              `json:"diagnostics"`
	Operations  []DelegatedChildOperation `json:"operations,omitempty"`
}

type NestedMergeExecutionCallbacks[T any] struct {
	MergeParent          func() MergeResult[T]
	DiscoverOperations   func(mergedOutput T) NestedMergeDiscoveryResult
	ApplyResolvedOutputs func(mergedOutput T, operations []DelegatedChildOperation, applyPlan DelegatedChildApplyPlan, appliedChildren []AppliedDelegatedChildOutput) MergeResult[T]
}

type ReviewReplayBundle struct {
	ReplayContext ReviewReplayContext `json:"replay_context"`
	Decisions     []ReviewDecision    `json:"decisions"`
}

const ReviewTransportVersion = 1

type ReviewTransportImportErrorCategory string

const (
	ReviewTransportKindMismatch       ReviewTransportImportErrorCategory = "kind_mismatch"
	ReviewTransportUnsupportedVersion ReviewTransportImportErrorCategory = "unsupported_version"
)

type ReviewTransportImportError struct {
	Category ReviewTransportImportErrorCategory `json:"category"`
	Message  string                             `json:"message"`
}

type ConformanceManifestReviewStateEnvelope struct {
	Kind    string                         `json:"kind"`
	Version int                            `json:"version"`
	State   ConformanceManifestReviewState `json:"state"`
}

type ReviewReplayBundleEnvelope struct {
	Kind         string             `json:"kind"`
	Version      int                `json:"version"`
	ReplayBundle ReviewReplayBundle `json:"replay_bundle"`
}

type ReviewedNestedExecution struct {
	Family          string                         `json:"family"`
	ReviewState     DelegatedChildGroupReviewState `json:"review_state"`
	AppliedChildren []AppliedDelegatedChildOutput  `json:"applied_children"`
}

type ReviewedNestedExecutionEnvelope struct {
	Kind      string                  `json:"kind"`
	Version   int                     `json:"version"`
	Execution ReviewedNestedExecution `json:"execution"`
}

type ReviewHostHints struct {
	Interactive             bool `json:"interactive"`
	RequireExplicitContexts bool `json:"require_explicit_contexts"`
}

type ReviewReplayContext struct {
	Surface                 string   `json:"surface"`
	Families                []string `json:"families"`
	RequireExplicitContexts bool     `json:"require_explicit_contexts"`
}

type ConformanceManifestReviewOptions struct {
	Contexts                map[string]ConformanceFamilyPlanContext `json:"contexts,omitempty"`
	FamilyProfiles          map[string]FamilyFeatureProfile         `json:"family_profiles,omitempty"`
	RequireExplicitContexts bool                                    `json:"require_explicit_contexts,omitempty"`
	ReviewDecisions         []ReviewDecision                        `json:"review_decisions,omitempty"`
	ReviewReplayContext     *ReviewReplayContext                    `json:"review_replay_context,omitempty"`
	ReviewReplayBundle      *ReviewReplayBundle                     `json:"review_replay_bundle,omitempty"`
	Interactive             bool                                    `json:"interactive,omitempty"`
}

type ConformanceManifestReviewState struct {
	Report           NamedConformanceSuiteReportEnvelope `json:"report"`
	Diagnostics      []Diagnostic                        `json:"diagnostics"`
	Requests         []ReviewRequest                     `json:"requests"`
	AppliedDecisions []ReviewDecision                    `json:"applied_decisions"`
	HostHints        ReviewHostHints                     `json:"host_hints"`
	ReplayContext    ReviewReplayContext                 `json:"replay_context"`
}

type ConformanceManifestPlan struct {
	Entries     []NamedConformanceSuitePlan `json:"entries"`
	Diagnostics []Diagnostic                `json:"diagnostics"`
}

type ConformanceSuiteSummary struct {
	Total   int `json:"total"`
	Passed  int `json:"passed"`
	Failed  int `json:"failed"`
	Skipped int `json:"skipped"`
}

type ConformanceSuiteReport struct {
	Results []ConformanceCaseResult `json:"results"`
	Summary ConformanceSuiteSummary `json:"summary"`
}

type ConformanceSuitePlanEntry struct {
	Ref  ConformanceCaseRef `json:"ref"`
	Path []string           `json:"path"`
	Run  ConformanceCaseRun `json:"run"`
}

type ConformanceSuitePlan struct {
	Family       string                      `json:"family"`
	Entries      []ConformanceSuitePlanEntry `json:"entries"`
	MissingRoles []string                    `json:"missing_roles"`
}

type ConformanceFeatureProfileView struct {
	Backend           string            `json:"backend"`
	SupportsDialects  bool              `json:"supports_dialects"`
	SupportedPolicies []PolicyReference `json:"supported_policies"`
}

type ConformanceCaseRun struct {
	Ref            ConformanceCaseRef             `json:"ref"`
	Requirements   ConformanceCaseRequirements    `json:"requirements"`
	FamilyProfile  FamilyFeatureProfile           `json:"family_profile"`
	FeatureProfile *ConformanceFeatureProfileView `json:"feature_profile,omitempty"`
}

type ConformanceCaseExecution struct {
	Outcome  ConformanceOutcome `json:"outcome"`
	Messages []string           `json:"messages"`
}

func includesPolicy(supportedPolicies []PolicyReference, policy PolicyReference) bool {
	for _, supportedPolicy := range supportedPolicies {
		if supportedPolicy == policy {
			return true
		}
	}

	return false
}

func isDefaultDialect(familyProfile FamilyFeatureProfile, dialect string) bool {
	return dialect == familyProfile.Family
}

func ConformanceFamilyEntries(manifest ConformanceManifest, family string) []ConformanceManifestEntry {
	if entries, ok := manifest.Families[family]; ok {
		return entries
	}

	return []ConformanceManifestEntry{}
}

func ConformanceFixturePath(manifest ConformanceManifest, family string, role string) []string {
	for _, entry := range ConformanceFamilyEntries(manifest, family) {
		if entry.Role == role {
			return entry.Path
		}
	}

	return nil
}

func ConformanceFamilyFeatureProfilePath(manifest ConformanceManifest, family string) []string {
	for _, entry := range manifest.FamilyFeatureProfiles {
		if entry.Family == family {
			return entry.Path
		}
	}

	return nil
}

func conformanceSuiteSelectorsEqual(left ConformanceSuiteSelector, right ConformanceSuiteSelector) bool {
	return left.Kind == right.Kind &&
		left.Subject.Grammar == right.Subject.Grammar &&
		left.Subject.Variant == right.Subject.Variant
}

func compareConformanceSuiteSelectors(left ConformanceSuiteSelector, right ConformanceSuiteSelector) int {
	if compared := strings.Compare(left.Kind, right.Kind); compared != 0 {
		return compared
	}
	if compared := strings.Compare(left.Subject.Grammar, right.Subject.Grammar); compared != 0 {
		return compared
	}
	return strings.Compare(left.Subject.Variant, right.Subject.Variant)
}

func ConformanceSuiteSelectors(manifest ConformanceManifest) []ConformanceSuiteSelector {
	selectors := make([]ConformanceSuiteSelector, 0, len(manifest.SuiteDescriptors))
	for _, definition := range manifest.SuiteDescriptors {
		selectors = append(selectors, ConformanceSuiteSelector{
			Kind:    definition.Kind,
			Subject: definition.Subject,
		})
	}
	slices.SortFunc(selectors, compareConformanceSuiteSelectors)
	return selectors
}

func ConformanceSuiteDefinitionForSelector(
	manifest ConformanceManifest,
	selector ConformanceSuiteSelector,
) *ConformanceSuiteDefinition {
	for _, definition := range manifest.SuiteDescriptors {
		if conformanceSuiteSelectorsEqual(
			ConformanceSuiteSelector{Kind: definition.Kind, Subject: definition.Subject},
			selector,
		) {
			matched := definition
			return &matched
		}
	}

	return nil
}

func conformanceSuiteDescriptorString(definition ConformanceSuiteDefinition) string {
	encoded, err := json.Marshal(definition)
	if err != nil {
		return "<invalid suite descriptor>"
	}
	return string(encoded)
}

func GroupProjectedChildReviewCases(cases []ProjectedChildReviewCase) []ProjectedChildReviewGroup {
	groups := make([]ProjectedChildReviewGroup, 0)

	for _, entry := range cases {
		found := false
		for index := range groups {
			if groups[index].DelegatedApplyGroup == entry.DelegatedApplyGroup {
				groups[index].CaseIDs = append(groups[index].CaseIDs, entry.CaseID)
				groups[index].DelegatedCaseIDs = append(groups[index].DelegatedCaseIDs, entry.DelegatedCaseID)
				found = true
				break
			}
		}
		if found {
			continue
		}

		groups = append(groups, ProjectedChildReviewGroup{
			DelegatedApplyGroup:         entry.DelegatedApplyGroup,
			ParentOperationID:           entry.ParentOperationID,
			ChildOperationID:            entry.ChildOperationID,
			DelegatedRuntimeSurfacePath: entry.DelegatedRuntimeSurfacePath,
			CaseIDs:                     []string{entry.CaseID},
			DelegatedCaseIDs:            []string{entry.DelegatedCaseID},
		})
	}

	return groups
}

func SummarizeProjectedChildReviewGroupProgress(groups []ProjectedChildReviewGroup, resolvedCaseIDs []string) []ProjectedChildReviewGroupProgress {
	progress := make([]ProjectedChildReviewGroupProgress, 0, len(groups))

	for _, group := range groups {
		resolved := make([]string, 0)
		pending := make([]string, 0)
		for _, caseID := range group.CaseIDs {
			if slices.Contains(resolvedCaseIDs, caseID) {
				resolved = append(resolved, caseID)
			} else {
				pending = append(pending, caseID)
			}
		}

		progress = append(progress, ProjectedChildReviewGroupProgress{
			DelegatedApplyGroup:         group.DelegatedApplyGroup,
			ParentOperationID:           group.ParentOperationID,
			ChildOperationID:            group.ChildOperationID,
			DelegatedRuntimeSurfacePath: group.DelegatedRuntimeSurfacePath,
			ResolvedCaseIDs:             resolved,
			PendingCaseIDs:              pending,
			Complete:                    len(pending) == 0,
		})
	}

	return progress
}

func SelectProjectedChildReviewGroupsReadyForApply(groups []ProjectedChildReviewGroup, resolvedCaseIDs []string) []ProjectedChildReviewGroup {
	ready := make([]ProjectedChildReviewGroup, 0)

	for _, group := range groups {
		complete := true
		for _, caseID := range group.CaseIDs {
			if !slices.Contains(resolvedCaseIDs, caseID) {
				complete = false
				break
			}
		}

		if complete {
			ready = append(ready, group)
		}
	}

	return ready
}

func ReviewRequestIDForProjectedChildGroup(group ProjectedChildReviewGroup) string {
	return "projected_child_group:" + group.DelegatedApplyGroup
}

func ProjectedChildGroupReviewRequest(group ProjectedChildReviewGroup, family string) ReviewRequest {
	return ReviewRequest{
		ID:             ReviewRequestIDForProjectedChildGroup(group),
		Kind:           ReviewRequestDelegatedChildGroup,
		Family:         family,
		Message:        "delegated child group " + group.DelegatedApplyGroup + " is ready to apply for " + family + ".",
		Blocking:       true,
		DelegatedGroup: &group,
		ActionOffers: []ReviewActionOffer{{
			Action:          ReviewDecisionApplyDelegatedChildGroup,
			RequiresContext: false,
		}},
		DefaultAction: ReviewDecisionApplyDelegatedChildGroup,
	}
}

func SelectProjectedChildReviewGroupsAcceptedForApply(groups []ProjectedChildReviewGroup, _family string, decisions []ReviewDecision) []ProjectedChildReviewGroup {
	acceptedRequestIDs := make(map[string]bool)
	for _, decision := range decisions {
		if decision.Action == ReviewDecisionApplyDelegatedChildGroup {
			acceptedRequestIDs[decision.RequestID] = true
		}
	}

	accepted := make([]ProjectedChildReviewGroup, 0)
	for _, group := range groups {
		if acceptedRequestIDs[ReviewRequestIDForProjectedChildGroup(group)] {
			accepted = append(accepted, group)
		}
	}

	return accepted
}

func ReviewProjectedChildGroups(groups []ProjectedChildReviewGroup, family string, decisions []ReviewDecision) DelegatedChildGroupReviewState {
	requestIDs := make(map[string]bool)
	for _, group := range groups {
		requestIDs[ReviewRequestIDForProjectedChildGroup(group)] = true
	}

	appliedDecisions := make([]ReviewDecision, 0)
	diagnostics := make([]Diagnostic, 0)
	for _, decision := range decisions {
		if decision.Action != ReviewDecisionApplyDelegatedChildGroup {
			continue
		}
		if requestIDs[decision.RequestID] {
			appliedDecisions = append(appliedDecisions, decision)
		} else {
			diagnostics = append(diagnostics, Diagnostic{
				Severity: SeverityError,
				Category: CategoryReplayRejected,
				Message:  "review decision " + decision.RequestID + " does not match any current delegated child review request.",
				Review: &ReviewDiagnosticDetail{
					RequestID: decision.RequestID,
					Action:    decision.Action,
					Reason:    ReasonRequestNotFound,
				},
			})
		}
	}

	acceptedGroups := SelectProjectedChildReviewGroupsAcceptedForApply(groups, family, appliedDecisions)
	acceptedRequestIDs := make(map[string]bool)
	for _, group := range acceptedGroups {
		acceptedRequestIDs[ReviewRequestIDForProjectedChildGroup(group)] = true
	}

	requests := make([]ReviewRequest, 0)
	for _, group := range groups {
		if !acceptedRequestIDs[ReviewRequestIDForProjectedChildGroup(group)] {
			requests = append(requests, ProjectedChildGroupReviewRequest(group, family))
		}
	}

	return DelegatedChildGroupReviewState{
		Requests:         requests,
		AcceptedGroups:   acceptedGroups,
		AppliedDecisions: appliedDecisions,
		Diagnostics:      diagnostics,
	}
}

func DelegatedChildApplyPlanForState(state DelegatedChildGroupReviewState, family string) DelegatedChildApplyPlan {
	entries := make([]DelegatedChildApplyPlanEntry, 0, len(state.AcceptedGroups))
	for _, group := range state.AcceptedGroups {
		requestID := ReviewRequestIDForProjectedChildGroup(group)
		var matched *ReviewDecision
		for _, decision := range state.AppliedDecisions {
			if decision.RequestID == requestID {
				decisionCopy := decision
				matched = &decisionCopy
				break
			}
		}
		if matched == nil {
			continue
		}

		entries = append(entries, DelegatedChildApplyPlanEntry{
			RequestID:      requestID,
			Family:         family,
			DelegatedGroup: group,
			Decision:       *matched,
		})
	}

	return DelegatedChildApplyPlan{Entries: entries}
}

func ResolveDelegatedChildOutputs(
	operations []DelegatedChildOperation,
	nestedOutputs []DelegatedChildSurfaceOutput,
	options DelegatedChildOutputResolutionOptions,
) DelegatedChildOutputResolution {
	operationsBySurfaceAddress := make(map[string]DelegatedChildOperation, len(operations))
	for _, operation := range operations {
		operationsBySurfaceAddress[operation.Surface.Address] = operation
	}

	for _, nestedOutput := range nestedOutputs {
		if _, ok := operationsBySurfaceAddress[nestedOutput.SurfaceAddress]; ok {
			continue
		}

		return DelegatedChildOutputResolution{
			OK: false,
			Diagnostics: []Diagnostic{{
				Severity: SeverityError,
				Category: CategoryConfigurationError,
				Message:  "missing delegated child surface " + nestedOutput.SurfaceAddress + ".",
			}},
		}
	}

	entries := make([]DelegatedChildApplyPlanEntry, 0, len(nestedOutputs))
	appliedChildren := make([]AppliedDelegatedChildOutput, 0, len(nestedOutputs))
	for index, nestedOutput := range nestedOutputs {
		operation := operationsBySurfaceAddress[nestedOutput.SurfaceAddress]
		requestID := options.RequestIDPrefix + ":" + strconv.Itoa(index)
		family := options.DefaultFamily
		if value, ok := operation.Surface.Metadata["family"].(string); ok && value != "" {
			family = value
		}

		entries = append(entries, DelegatedChildApplyPlanEntry{
			RequestID: requestID,
			Family:    family,
			DelegatedGroup: ProjectedChildReviewGroup{
				DelegatedApplyGroup:         requestID,
				ParentOperationID:           operation.ParentOperationID,
				ChildOperationID:            operation.OperationID,
				DelegatedRuntimeSurfacePath: nestedOutput.SurfaceAddress,
				CaseIDs:                     []string{},
				DelegatedCaseIDs:            []string{},
			},
			Decision: ReviewDecision{
				RequestID: requestID,
				Action:    ReviewDecisionApplyDelegatedChildGroup,
			},
		})
		appliedChildren = append(appliedChildren, AppliedDelegatedChildOutput{
			OperationID: operation.OperationID,
			Output:      nestedOutput.Output,
		})
	}

	return DelegatedChildOutputResolution{
		OK:              true,
		Diagnostics:     []Diagnostic{},
		ApplyPlan:       &DelegatedChildApplyPlan{Entries: entries},
		AppliedChildren: appliedChildren,
	}
}

func ExecuteNestedMerge[T any](
	nestedOutputs []DelegatedChildSurfaceOutput,
	options DelegatedChildOutputResolutionOptions,
	callbacks NestedMergeExecutionCallbacks[T],
) MergeResult[T] {
	merged := callbacks.MergeParent()
	if !merged.OK || merged.Output == nil {
		return merged
	}

	discovery := callbacks.DiscoverOperations(*merged.Output)
	if !discovery.OK {
		return MergeResult[T]{
			OK:          false,
			Diagnostics: discovery.Diagnostics,
			Policies:    []PolicyReference{},
		}
	}

	resolution := ResolveDelegatedChildOutputs(discovery.Operations, nestedOutputs, options)
	if !resolution.OK || resolution.ApplyPlan == nil {
		return MergeResult[T]{
			OK:          false,
			Diagnostics: resolution.Diagnostics,
			Policies:    []PolicyReference{},
		}
	}

	return callbacks.ApplyResolvedOutputs(
		*merged.Output,
		discovery.Operations,
		*resolution.ApplyPlan,
		resolution.AppliedChildren,
	)
}

func ExecuteDelegatedChildApplyPlan[T any](
	applyPlan DelegatedChildApplyPlan,
	appliedChildren []AppliedDelegatedChildOutput,
	callbacks NestedMergeExecutionCallbacks[T],
) MergeResult[T] {
	merged := callbacks.MergeParent()
	if !merged.OK || merged.Output == nil {
		return merged
	}

	discovery := callbacks.DiscoverOperations(*merged.Output)
	if !discovery.OK {
		return MergeResult[T]{
			OK:          false,
			Diagnostics: discovery.Diagnostics,
			Policies:    []PolicyReference{},
		}
	}

	return callbacks.ApplyResolvedOutputs(
		*merged.Output,
		discovery.Operations,
		applyPlan,
		appliedChildren,
	)
}

func ExecuteReviewedNestedMerge[T any](
	state DelegatedChildGroupReviewState,
	family string,
	appliedChildren []AppliedDelegatedChildOutput,
	callbacks NestedMergeExecutionCallbacks[T],
) MergeResult[T] {
	return ExecuteDelegatedChildApplyPlan(
		DelegatedChildApplyPlanForState(state, family),
		appliedChildren,
		callbacks,
	)
}

func DefaultConformanceFamilyContext(
	familyProfile FamilyFeatureProfile,
) ConformanceFamilyPlanContext {
	return ConformanceFamilyPlanContext{
		FamilyProfile: familyProfile,
	}
}

func ReviewRequestIDForFamilyContext(family string) string {
	return "family_context:" + family
}

func ConformanceReviewHostHints(options ConformanceManifestReviewOptions) ReviewHostHints {
	return ReviewHostHints{
		Interactive:             options.Interactive,
		RequireExplicitContexts: options.RequireExplicitContexts,
	}
}

func ConformanceManifestReplayContext(
	manifest ConformanceManifest,
	options ConformanceManifestReviewOptions,
) ReviewReplayContext {
	families := make([]string, 0, len(manifest.SuiteDescriptors))
	seen := make(map[string]bool)
	for _, selector := range ConformanceSuiteSelectors(manifest) {
		definition := ConformanceSuiteDefinitionForSelector(manifest, selector)
		if definition == nil || seen[definition.Subject.Grammar] {
			continue
		}
		seen[definition.Subject.Grammar] = true
		families = append(families, definition.Subject.Grammar)
	}

	return ReviewReplayContext{
		Surface:                 "conformance_manifest",
		Families:                families,
		RequireExplicitContexts: options.RequireExplicitContexts,
	}
}

func ReviewReplayContextCompatible(
	current ReviewReplayContext,
	candidate *ReviewReplayContext,
) bool {
	if candidate == nil {
		return false
	}

	return current.Surface == candidate.Surface &&
		current.RequireExplicitContexts == candidate.RequireExplicitContexts &&
		slices.Equal(current.Families, candidate.Families)
}

func ConformanceManifestReviewRequestIDs(
	manifest ConformanceManifest,
	options ConformanceManifestReviewOptions,
) []string {
	if !options.RequireExplicitContexts {
		return []string{}
	}

	requestIDs := make([]string, 0, len(manifest.SuiteDescriptors))
	seen := make(map[string]bool)
	for _, selector := range ConformanceSuiteSelectors(manifest) {
		definition := ConformanceSuiteDefinitionForSelector(manifest, selector)
		if definition == nil || seen[definition.Subject.Grammar] {
			continue
		}
		seen[definition.Subject.Grammar] = true
		if _, ok := options.Contexts[definition.Subject.Grammar]; ok {
			continue
		}
		if _, ok := options.FamilyProfiles[definition.Subject.Grammar]; ok {
			requestIDs = append(requestIDs, ReviewRequestIDForFamilyContext(definition.Subject.Grammar))
		}
	}

	return requestIDs
}

func ReviewReplayBundleInputs(
	options ConformanceManifestReviewOptions,
) (*ReviewReplayContext, []ReviewDecision) {
	if options.ReviewReplayBundle != nil {
		return &options.ReviewReplayBundle.ReplayContext, options.ReviewReplayBundle.Decisions
	}

	return options.ReviewReplayContext, options.ReviewDecisions
}

func ConformanceManifestReviewStateEnvelopeFor(
	state ConformanceManifestReviewState,
) ConformanceManifestReviewStateEnvelope {
	return ConformanceManifestReviewStateEnvelope{
		Kind:    "conformance_manifest_review_state",
		Version: ReviewTransportVersion,
		State:   state,
	}
}

func ReviewReplayBundleEnvelopeFor(
	bundle ReviewReplayBundle,
) ReviewReplayBundleEnvelope {
	return ReviewReplayBundleEnvelope{
		Kind:         "review_replay_bundle",
		Version:      ReviewTransportVersion,
		ReplayBundle: bundle,
	}
}

func ReviewedNestedExecutionEnvelopeFor(
	execution ReviewedNestedExecution,
) ReviewedNestedExecutionEnvelope {
	return ReviewedNestedExecutionEnvelope{
		Kind:      "reviewed_nested_execution",
		Version:   ReviewTransportVersion,
		Execution: execution,
	}
}

func ImportConformanceManifestReviewStateEnvelope(
	envelope ConformanceManifestReviewStateEnvelope,
) (*ConformanceManifestReviewState, *ReviewTransportImportError) {
	if envelope.Kind != "conformance_manifest_review_state" {
		return nil, &ReviewTransportImportError{
			Category: ReviewTransportKindMismatch,
			Message:  "expected conformance_manifest_review_state envelope kind.",
		}
	}

	if envelope.Version != ReviewTransportVersion {
		return nil, &ReviewTransportImportError{
			Category: ReviewTransportUnsupportedVersion,
			Message:  "unsupported conformance_manifest_review_state envelope version " + strconv.Itoa(envelope.Version) + ".",
		}
	}

	state := envelope.State
	return &state, nil
}

func ImportReviewReplayBundleEnvelope(
	envelope ReviewReplayBundleEnvelope,
) (*ReviewReplayBundle, *ReviewTransportImportError) {
	if envelope.Kind != "review_replay_bundle" {
		return nil, &ReviewTransportImportError{
			Category: ReviewTransportKindMismatch,
			Message:  "expected review_replay_bundle envelope kind.",
		}
	}

	if envelope.Version != ReviewTransportVersion {
		return nil, &ReviewTransportImportError{
			Category: ReviewTransportUnsupportedVersion,
			Message:  "unsupported review_replay_bundle envelope version " + strconv.Itoa(envelope.Version) + ".",
		}
	}

	bundle := envelope.ReplayBundle
	return &bundle, nil
}

func ImportReviewedNestedExecutionEnvelope(
	envelope ReviewedNestedExecutionEnvelope,
) (*ReviewedNestedExecution, *ReviewTransportImportError) {
	if envelope.Kind != "reviewed_nested_execution" {
		return nil, &ReviewTransportImportError{
			Category: ReviewTransportKindMismatch,
			Message:  "expected reviewed_nested_execution envelope kind.",
		}
	}

	if envelope.Version != ReviewTransportVersion {
		return nil, &ReviewTransportImportError{
			Category: ReviewTransportUnsupportedVersion,
			Message:  "unsupported reviewed_nested_execution envelope version " + strconv.Itoa(envelope.Version) + ".",
		}
	}

	execution := envelope.Execution
	return &execution, nil
}

func ResolveConformanceFamilyContext(
	family string,
	options ConformanceManifestPlanningOptions,
) (*ConformanceFamilyPlanContext, []Diagnostic) {
	if options.Contexts != nil {
		if context, ok := options.Contexts[family]; ok {
			return &context, nil
		}
	}

	if options.RequireExplicitContexts {
		return nil, []Diagnostic{{
			Severity: SeverityError,
			Category: CategoryConfigurationError,
			Message:  "missing explicit family context for " + family + ".",
		}}
	}

	if options.FamilyProfiles != nil {
		if familyProfile, ok := options.FamilyProfiles[family]; ok {
			context := DefaultConformanceFamilyContext(familyProfile)
			return &context, []Diagnostic{{
				Severity: SeverityWarning,
				Category: CategoryAssumedDefault,
				Message:  "using default family context for " + family + ".",
			}}
		}
	}

	return nil, []Diagnostic{{
		Severity: SeverityError,
		Category: CategoryConfigurationError,
		Message:  "missing family context for " + family + " and no default family profile is available.",
	}}
}

func reviewDecisionForFamilyContext(
	family string,
	options ConformanceManifestReviewOptions,
) (*ConformanceFamilyPlanContext, *ReviewDecision, bool, []Diagnostic) {
	requestID := ReviewRequestIDForFamilyContext(family)
	var familyProfile *FamilyFeatureProfile
	if profile, ok := options.FamilyProfiles[family]; ok {
		copyProfile := profile
		familyProfile = &copyProfile
	}
	for _, decision := range options.ReviewDecisions {
		if decision.RequestID != requestID {
			continue
		}
		if decision.Action == ReviewDecisionAcceptDefaultContext && familyProfile != nil {
			copyDecision := decision
			context := DefaultConformanceFamilyContext(*familyProfile)
			return &context, &copyDecision, true, nil
		}
		if decision.Action == ReviewDecisionProvideExplicitContext && decision.Context == nil {
			return nil, nil, false, []Diagnostic{{
				Severity: SeverityError,
				Category: CategoryConfigurationError,
				Message:  "review decision " + requestID + " requires explicit context payload.",
				Review: &ReviewDiagnosticDetail{
					RequestID:   requestID,
					Action:      ReviewDecisionProvideExplicitContext,
					Reason:      ReasonMissingRequiredPayload,
					PayloadKind: "conformance_family_context",
				},
			}}
		}
		if decision.Action == ReviewDecisionProvideExplicitContext && decision.Context != nil {
			if decision.Context.FamilyProfile.Family != family {
				return nil, nil, false, []Diagnostic{{
					Severity: SeverityError,
					Category: CategoryConfigurationError,
					Message:  "review decision " + requestID + " provided context for " + decision.Context.FamilyProfile.Family + ", expected " + family + ".",
					Review: &ReviewDiagnosticDetail{
						RequestID:      requestID,
						Action:         ReviewDecisionProvideExplicitContext,
						Reason:         ReasonFamilyMismatch,
						ExpectedFamily: family,
						ProvidedFamily: decision.Context.FamilyProfile.Family,
					},
				}}
			}
			copyDecision := decision
			context := *decision.Context
			return &context, &copyDecision, false, nil
		}
	}

	return nil, nil, false, nil
}

func ReviewConformanceFamilyContext(
	family string,
	options ConformanceManifestReviewOptions,
) (*ConformanceFamilyPlanContext, []Diagnostic, []ReviewRequest, []ReviewDecision) {
	if context, ok := options.Contexts[family]; ok {
		copyContext := context
		return &copyContext, nil, nil, nil
	}

	if !options.RequireExplicitContexts {
		planningOptions := ConformanceManifestPlanningOptions{
			Contexts:                options.Contexts,
			FamilyProfiles:          options.FamilyProfiles,
			RequireExplicitContexts: false,
		}
		context, diagnostics := ResolveConformanceFamilyContext(family, planningOptions)
		return context, diagnostics, nil, nil
	}

	familyProfile, ok := options.FamilyProfiles[family]
	if !ok {
		return nil, []Diagnostic{{
			Severity: SeverityError,
			Category: CategoryConfigurationError,
			Message:  "missing family context for " + family + " and no default family profile is available.",
		}}, nil, nil
	}

	if context, decision, assumedDefault, decisionDiagnostics := reviewDecisionForFamilyContext(family, options); decision != nil {
		diagnostics := []Diagnostic{}
		if assumedDefault {
			diagnostics = append(diagnostics, Diagnostic{
				Severity: SeverityWarning,
				Category: CategoryAssumedDefault,
				Message:  "using default family context for " + family + ".",
			})
		}
		return context, diagnostics, nil, []ReviewDecision{*decision}
	} else if len(decisionDiagnostics) > 0 {
		return nil, decisionDiagnostics, []ReviewRequest{{
			ID:              ReviewRequestIDForFamilyContext(family),
			Kind:            ReviewRequestFamilyContext,
			Family:          family,
			Message:         "explicit family context is required for " + family + "; a synthesized default may be accepted by review.",
			Blocking:        true,
			ProposedContext: &ConformanceFamilyPlanContext{FamilyProfile: familyProfile},
			ActionOffers: []ReviewActionOffer{
				{Action: ReviewDecisionAcceptDefaultContext, RequiresContext: false},
				{Action: ReviewDecisionProvideExplicitContext, RequiresContext: true, PayloadKind: "conformance_family_context"},
			},
			DefaultAction: ReviewDecisionAcceptDefaultContext,
		}}, nil
	}

	return nil, []Diagnostic{{
			Severity: SeverityError,
			Category: CategoryConfigurationError,
			Message:  "missing explicit family context for " + family + ".",
		}}, []ReviewRequest{{
			ID:              ReviewRequestIDForFamilyContext(family),
			Kind:            ReviewRequestFamilyContext,
			Family:          family,
			Message:         "explicit family context is required for " + family + "; a synthesized default may be accepted by review.",
			Blocking:        true,
			ProposedContext: &ConformanceFamilyPlanContext{FamilyProfile: familyProfile},
			ActionOffers: []ReviewActionOffer{
				{Action: ReviewDecisionAcceptDefaultContext, RequiresContext: false},
				{Action: ReviewDecisionProvideExplicitContext, RequiresContext: true, PayloadKind: "conformance_family_context"},
			},
			DefaultAction: ReviewDecisionAcceptDefaultContext,
		}}, nil
}

func SummarizeConformanceResults(results []ConformanceCaseResult) ConformanceSuiteSummary {
	summary := ConformanceSuiteSummary{}
	for _, result := range results {
		summary.Total++
		switch result.Outcome {
		case ConformancePassed:
			summary.Passed++
		case ConformanceFailed:
			summary.Failed++
		case ConformanceSkipped:
			summary.Skipped++
		}
	}

	return summary
}

func SelectConformanceCase(
	ref ConformanceCaseRef,
	requirements ConformanceCaseRequirements,
	familyProfile FamilyFeatureProfile,
	featureProfile *ConformanceFeatureProfileView,
) ConformanceCaseSelection {
	messages := []string{}

	if requirements.Backend != "" {
		if featureProfile == nil {
			messages = append(
				messages,
				"case requires backend "+requirements.Backend+" but no backend feature profile is available for family "+familyProfile.Family+".",
			)
		} else if featureProfile.Backend != requirements.Backend {
			messages = append(
				messages,
				"case requires backend "+requirements.Backend+" but backend "+featureProfile.Backend+" is active for family "+familyProfile.Family+".",
			)
		}
	}

	if requirements.Dialect != "" {
		if !slices.Contains(familyProfile.SupportedDialects, requirements.Dialect) {
			messages = append(messages, "family "+familyProfile.Family+" does not support dialect "+requirements.Dialect+".")
		} else if featureProfile != nil && !featureProfile.SupportsDialects && !isDefaultDialect(familyProfile, requirements.Dialect) {
			messages = append(
				messages,
				"backend "+featureProfile.Backend+" does not support dialect "+requirements.Dialect+" for family "+familyProfile.Family+".",
			)
		}
	}

	for _, policy := range requirements.Policies {
		if !includesPolicy(familyProfile.SupportedPolicies, policy) {
			messages = append(messages, "family "+familyProfile.Family+" does not support policy "+policy.Name+".")
			continue
		}

		if featureProfile != nil && !includesPolicy(featureProfile.SupportedPolicies, policy) {
			messages = append(messages, "backend "+featureProfile.Backend+" does not support policy "+policy.Name+".")
		}
	}

	status := ConformanceSelected
	if len(messages) > 0 {
		status = ConformanceSelectionSkipped
	}

	return ConformanceCaseSelection{
		Ref:      ref,
		Status:   status,
		Messages: messages,
	}
}

func RunConformanceCase(
	run ConformanceCaseRun,
	execute func(ConformanceCaseRun) ConformanceCaseExecution,
) ConformanceCaseResult {
	selection := SelectConformanceCase(run.Ref, run.Requirements, run.FamilyProfile, run.FeatureProfile)
	if selection.Status == ConformanceSelectionSkipped {
		return ConformanceCaseResult{
			Ref:      run.Ref,
			Outcome:  ConformanceSkipped,
			Messages: selection.Messages,
		}
	}

	execution := execute(run)
	return ConformanceCaseResult{
		Ref:      run.Ref,
		Outcome:  execution.Outcome,
		Messages: execution.Messages,
	}
}

func RunConformanceSuite(
	runs []ConformanceCaseRun,
	execute func(ConformanceCaseRun) ConformanceCaseExecution,
) []ConformanceCaseResult {
	results := make([]ConformanceCaseResult, 0, len(runs))
	for _, run := range runs {
		results = append(results, RunConformanceCase(run, execute))
	}

	return results
}

func RunPlannedConformanceSuite(
	plan ConformanceSuitePlan,
	execute func(ConformanceCaseRun) ConformanceCaseExecution,
) []ConformanceCaseResult {
	results := make([]ConformanceCaseResult, 0, len(plan.Entries))
	for _, entry := range plan.Entries {
		results = append(results, RunConformanceCase(entry.Run, execute))
	}

	return results
}

func RunNamedConformanceSuite(
	manifest ConformanceManifest,
	selector ConformanceSuiteSelector,
	familyProfile FamilyFeatureProfile,
	execute func(ConformanceCaseRun) ConformanceCaseExecution,
	featureProfile *ConformanceFeatureProfileView,
) []ConformanceCaseResult {
	plan := PlanNamedConformanceSuite(
		manifest,
		selector,
		familyProfile,
		featureProfile,
	)
	if plan == nil {
		return nil
	}

	return RunPlannedConformanceSuite(*plan, execute)
}

func RunNamedConformanceSuiteEntry(
	manifest ConformanceManifest,
	selector ConformanceSuiteSelector,
	familyProfile FamilyFeatureProfile,
	execute func(ConformanceCaseRun) ConformanceCaseExecution,
	featureProfile *ConformanceFeatureProfileView,
) *NamedConformanceSuiteResults {
	results := RunNamedConformanceSuite(
		manifest,
		selector,
		familyProfile,
		execute,
		featureProfile,
	)
	if results == nil {
		return nil
	}
	definition := ConformanceSuiteDefinitionForSelector(manifest, selector)
	if definition == nil {
		return nil
	}

	return &NamedConformanceSuiteResults{
		Suite:   *definition,
		Results: results,
	}
}

func RunPlannedNamedConformanceSuites(
	entries []NamedConformanceSuitePlan,
	execute func(ConformanceCaseRun) ConformanceCaseExecution,
) []NamedConformanceSuiteResults {
	results := make([]NamedConformanceSuiteResults, 0, len(entries))
	for _, entry := range entries {
		results = append(results, NamedConformanceSuiteResults{
			Suite:   entry.Suite,
			Results: RunPlannedConformanceSuite(entry.Plan, execute),
		})
	}

	return results
}

func ReportPlannedConformanceSuite(
	plan ConformanceSuitePlan,
	execute func(ConformanceCaseRun) ConformanceCaseExecution,
) ConformanceSuiteReport {
	return ReportConformanceSuite(RunPlannedConformanceSuite(plan, execute))
}

func ReportNamedConformanceSuite(
	manifest ConformanceManifest,
	selector ConformanceSuiteSelector,
	familyProfile FamilyFeatureProfile,
	execute func(ConformanceCaseRun) ConformanceCaseExecution,
	featureProfile *ConformanceFeatureProfileView,
) *ConformanceSuiteReport {
	plan := PlanNamedConformanceSuite(
		manifest,
		selector,
		familyProfile,
		featureProfile,
	)
	if plan == nil {
		return nil
	}

	report := ReportPlannedConformanceSuite(*plan, execute)
	return &report
}

func ReportNamedConformanceSuiteEntry(
	manifest ConformanceManifest,
	selector ConformanceSuiteSelector,
	familyProfile FamilyFeatureProfile,
	execute func(ConformanceCaseRun) ConformanceCaseExecution,
	featureProfile *ConformanceFeatureProfileView,
) *NamedConformanceSuiteReport {
	report := ReportNamedConformanceSuite(
		manifest,
		selector,
		familyProfile,
		execute,
		featureProfile,
	)
	if report == nil {
		return nil
	}
	definition := ConformanceSuiteDefinitionForSelector(manifest, selector)
	if definition == nil {
		return nil
	}

	return &NamedConformanceSuiteReport{
		Suite:  *definition,
		Report: *report,
	}
}

func ReportPlannedNamedConformanceSuites(
	entries []NamedConformanceSuitePlan,
	execute func(ConformanceCaseRun) ConformanceCaseExecution,
) []NamedConformanceSuiteReport {
	reports := make([]NamedConformanceSuiteReport, 0, len(entries))
	for _, entry := range entries {
		reports = append(reports, NamedConformanceSuiteReport{
			Suite:  entry.Suite,
			Report: ReportPlannedConformanceSuite(entry.Plan, execute),
		})
	}

	return reports
}

func SummarizeNamedConformanceSuiteReports(
	entries []NamedConformanceSuiteReport,
) ConformanceSuiteSummary {
	summary := ConformanceSuiteSummary{}
	for _, entry := range entries {
		summary.Total += entry.Report.Summary.Total
		summary.Passed += entry.Report.Summary.Passed
		summary.Failed += entry.Report.Summary.Failed
		summary.Skipped += entry.Report.Summary.Skipped
	}

	return summary
}

func ReportNamedConformanceSuiteEnvelope(
	entries []NamedConformanceSuiteReport,
) NamedConformanceSuiteReportEnvelope {
	return NamedConformanceSuiteReportEnvelope{
		Entries: entries,
		Summary: SummarizeNamedConformanceSuiteReports(entries),
	}
}

func ReportNamedConformanceSuiteManifest(
	manifest ConformanceManifest,
	contexts map[string]ConformanceFamilyPlanContext,
	execute func(ConformanceCaseRun) ConformanceCaseExecution,
) NamedConformanceSuiteReportEnvelope {
	return ReportNamedConformanceSuiteEnvelope(
		ReportPlannedNamedConformanceSuites(
			PlanNamedConformanceSuites(manifest, contexts),
			execute,
		),
	)
}

func ReportConformanceManifest(
	manifest ConformanceManifest,
	options ConformanceManifestPlanningOptions,
	execute func(ConformanceCaseRun) ConformanceCaseExecution,
) ConformanceManifestReport {
	planned := PlanNamedConformanceSuitesWithDiagnostics(manifest, options)
	return ConformanceManifestReport{
		Report: ReportNamedConformanceSuiteEnvelope(
			ReportPlannedNamedConformanceSuites(planned.Entries, execute),
		),
		Diagnostics: planned.Diagnostics,
	}
}

func ReviewConformanceManifest(
	manifest ConformanceManifest,
	options ConformanceManifestReviewOptions,
	execute func(ConformanceCaseRun) ConformanceCaseExecution,
) ConformanceManifestReviewState {
	replayContext := ConformanceManifestReplayContext(manifest, options)
	entries := make([]NamedConformanceSuitePlan, 0, len(manifest.SuiteDescriptors))
	diagnostics := make([]Diagnostic, 0)
	requests := make([]ReviewRequest, 0)
	appliedDecisions := make([]ReviewDecision, 0)
	effectiveOptions := options
	replayInputContext, replayInputDecisions := ReviewReplayBundleInputs(options)
	if len(replayInputDecisions) > 0 {
		if replayInputContext == nil {
			diagnostics = append(diagnostics, Diagnostic{
				Severity: SeverityError,
				Category: CategoryReplayRejected,
				Message:  "review decisions were provided without replay context.",
			})
			effectiveOptions.ReviewReplayBundle = nil
			effectiveOptions.ReviewReplayContext = nil
			effectiveOptions.ReviewDecisions = nil
		} else if !ReviewReplayContextCompatible(replayContext, replayInputContext) {
			diagnostics = append(diagnostics, Diagnostic{
				Severity: SeverityError,
				Category: CategoryReplayRejected,
				Message:  "review replay context does not match the current conformance manifest state.",
			})
			effectiveOptions.ReviewReplayBundle = nil
			effectiveOptions.ReviewReplayContext = nil
			effectiveOptions.ReviewDecisions = nil
		} else {
			allowedRequestIDs := make(map[string]bool)
			for _, requestID := range ConformanceManifestReviewRequestIDs(manifest, options) {
				allowedRequestIDs[requestID] = true
			}
			acceptedDecisions := make([]ReviewDecision, 0, len(replayInputDecisions))
			for _, decision := range replayInputDecisions {
				if allowedRequestIDs[decision.RequestID] {
					acceptedDecisions = append(acceptedDecisions, decision)
				} else {
					diagnostics = append(diagnostics, Diagnostic{
						Severity: SeverityError,
						Category: CategoryReplayRejected,
						Message:  "review decision " + decision.RequestID + " does not match any current review request.",
						Review: &ReviewDiagnosticDetail{
							RequestID: decision.RequestID,
							Action:    decision.Action,
							Reason:    ReasonRequestNotFound,
						},
					})
				}
			}
			effectiveOptions.ReviewReplayBundle = nil
			effectiveOptions.ReviewReplayContext = replayInputContext
			effectiveOptions.ReviewDecisions = acceptedDecisions
		}
	}
	resolvedContexts := make(map[string]*ConformanceFamilyPlanContext)
	resolvedFamilies := make(map[string]bool)

	for _, selector := range ConformanceSuiteSelectors(manifest) {
		definition := ConformanceSuiteDefinitionForSelector(manifest, selector)
		if definition == nil {
			continue
		}

		var context *ConformanceFamilyPlanContext
		if resolvedFamilies[definition.Subject.Grammar] {
			context = resolvedContexts[definition.Subject.Grammar]
		} else {
			var resolvedDiagnostics []Diagnostic
			var resolvedRequests []ReviewRequest
			var resolvedDecisions []ReviewDecision
			context, resolvedDiagnostics, resolvedRequests, resolvedDecisions = ReviewConformanceFamilyContext(definition.Subject.Grammar, effectiveOptions)
			diagnostics = append(diagnostics, resolvedDiagnostics...)
			requests = append(requests, resolvedRequests...)
			appliedDecisions = append(appliedDecisions, resolvedDecisions...)
			resolvedFamilies[definition.Subject.Grammar] = true
			resolvedContexts[definition.Subject.Grammar] = context
		}
		if context == nil {
			continue
		}

		entry := PlanNamedConformanceSuiteEntry(manifest, selector, *context)
		if entry == nil {
			continue
		}

		if len(entry.Plan.MissingRoles) > 0 {
			diagnostics = append(diagnostics, Diagnostic{
				Severity: SeverityError,
				Category: CategoryConfigurationError,
				Message:  "suite " + conformanceSuiteDescriptorString(entry.Suite) + " declares missing roles: " + joinComma(entry.Plan.MissingRoles) + ".",
			})
			continue
		}

		entries = append(entries, *entry)
	}

	return ConformanceManifestReviewState{
		Report:           ReportNamedConformanceSuiteEnvelope(ReportPlannedNamedConformanceSuites(entries, execute)),
		Diagnostics:      diagnostics,
		Requests:         requests,
		AppliedDecisions: appliedDecisions,
		HostHints:        ConformanceReviewHostHints(options),
		ReplayContext:    replayContext,
	}
}

func ReportConformanceSuite(results []ConformanceCaseResult) ConformanceSuiteReport {
	return ConformanceSuiteReport{
		Results: results,
		Summary: SummarizeConformanceResults(results),
	}
}

func PlanConformanceSuite(
	manifest ConformanceManifest,
	family string,
	roles []string,
	familyProfile FamilyFeatureProfile,
	featureProfile *ConformanceFeatureProfileView,
) ConformanceSuitePlan {
	entries := make([]ConformanceSuitePlanEntry, 0, len(roles))
	missingRoles := make([]string, 0)

	for _, role := range roles {
		var entry *ConformanceManifestEntry
		for _, candidate := range ConformanceFamilyEntries(manifest, family) {
			if candidate.Role == role {
				entry = &candidate
				break
			}
		}
		if entry == nil {
			missingRoles = append(missingRoles, role)
			continue
		}

		ref := ConformanceCaseRef{
			Family: family,
			Role:   role,
			Case:   role,
		}

		entries = append(entries, ConformanceSuitePlanEntry{
			Ref:  ref,
			Path: slices.Clone(entry.Path),
			Run: ConformanceCaseRun{
				Ref:            ref,
				Requirements:   derefRequirements(entry.Requirements),
				FamilyProfile:  familyProfile,
				FeatureProfile: featureProfile,
			},
		})
	}

	return ConformanceSuitePlan{
		Family:       family,
		Entries:      entries,
		MissingRoles: missingRoles,
	}
}

func PlanNamedConformanceSuite(
	manifest ConformanceManifest,
	selector ConformanceSuiteSelector,
	familyProfile FamilyFeatureProfile,
	featureProfile *ConformanceFeatureProfileView,
) *ConformanceSuitePlan {
	definition := ConformanceSuiteDefinitionForSelector(manifest, selector)
	if definition == nil {
		return nil
	}

	plan := PlanConformanceSuite(
		manifest,
		definition.Subject.Grammar,
		definition.Roles,
		familyProfile,
		featureProfile,
	)
	return &plan
}

func PlanNamedConformanceSuiteEntry(
	manifest ConformanceManifest,
	selector ConformanceSuiteSelector,
	context ConformanceFamilyPlanContext,
) *NamedConformanceSuitePlan {
	plan := PlanNamedConformanceSuite(
		manifest,
		selector,
		context.FamilyProfile,
		context.FeatureProfile,
	)
	if plan == nil {
		return nil
	}
	definition := ConformanceSuiteDefinitionForSelector(manifest, selector)
	if definition == nil {
		return nil
	}

	return &NamedConformanceSuitePlan{
		Suite: *definition,
		Plan:  *plan,
	}
}

func PlanNamedConformanceSuites(
	manifest ConformanceManifest,
	contexts map[string]ConformanceFamilyPlanContext,
) []NamedConformanceSuitePlan {
	entries := make([]NamedConformanceSuitePlan, 0, len(manifest.SuiteDescriptors))
	for _, selector := range ConformanceSuiteSelectors(manifest) {
		definition := ConformanceSuiteDefinitionForSelector(manifest, selector)
		if definition == nil {
			continue
		}

		context, ok := contexts[definition.Subject.Grammar]
		if !ok {
			continue
		}

		entry := PlanNamedConformanceSuiteEntry(manifest, selector, context)
		if entry != nil {
			entries = append(entries, *entry)
		}
	}

	return entries
}

func PlanNamedConformanceSuitesWithDiagnostics(
	manifest ConformanceManifest,
	options ConformanceManifestPlanningOptions,
) ConformanceManifestPlan {
	entries := make([]NamedConformanceSuitePlan, 0, len(manifest.SuiteDescriptors))
	diagnostics := make([]Diagnostic, 0)
	resolvedContexts := make(map[string]*ConformanceFamilyPlanContext)
	resolvedFamilies := make(map[string]bool)

	for _, selector := range ConformanceSuiteSelectors(manifest) {
		definition := ConformanceSuiteDefinitionForSelector(manifest, selector)
		if definition == nil {
			continue
		}

		var context *ConformanceFamilyPlanContext
		if resolvedFamilies[definition.Subject.Grammar] {
			context = resolvedContexts[definition.Subject.Grammar]
		} else {
			context, diagnostics = func() (*ConformanceFamilyPlanContext, []Diagnostic) {
				resolved, resolvedDiagnostics := ResolveConformanceFamilyContext(definition.Subject.Grammar, options)
				return resolved, append(diagnostics, resolvedDiagnostics...)
			}()
			resolvedFamilies[definition.Subject.Grammar] = true
			resolvedContexts[definition.Subject.Grammar] = context
		}
		if context == nil {
			continue
		}

		entry := PlanNamedConformanceSuiteEntry(manifest, selector, *context)
		if entry == nil {
			continue
		}

		if len(entry.Plan.MissingRoles) > 0 {
			diagnostics = append(diagnostics, Diagnostic{
				Severity: SeverityError,
				Category: CategoryConfigurationError,
				Message:  "suite " + conformanceSuiteDescriptorString(entry.Suite) + " declares missing roles: " + joinComma(entry.Plan.MissingRoles) + ".",
			})
			continue
		}

		entries = append(entries, *entry)
	}

	return ConformanceManifestPlan{
		Entries:     entries,
		Diagnostics: diagnostics,
	}
}

func derefRequirements(requirements *ConformanceCaseRequirements) ConformanceCaseRequirements {
	if requirements == nil {
		return ConformanceCaseRequirements{}
	}

	return *requirements
}

func joinComma(values []string) string {
	if len(values) == 0 {
		return ""
	}

	result := values[0]
	for _, value := range values[1:] {
		result += ", " + value
	}

	return result
}
