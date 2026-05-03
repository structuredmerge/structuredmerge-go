package astmerge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
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

type StructuredEditStructureProfile struct {
	OwnerScope              string         `json:"owner_scope"`
	OwnerSelector           string         `json:"owner_selector"`
	OwnerSelectorFamily     string         `json:"owner_selector_family,omitempty"`
	KnownOwnerSelector      bool           `json:"known_owner_selector"`
	SupportedCommentRegions []string       `json:"supported_comment_regions"`
	Metadata                map[string]any `json:"metadata,omitempty"`
}

type StructuredEditSelectionProfile struct {
	OwnerScope            string         `json:"owner_scope"`
	OwnerSelector         string         `json:"owner_selector"`
	OwnerSelectorFamily   string         `json:"owner_selector_family,omitempty"`
	SelectorKind          string         `json:"selector_kind"`
	SelectionIntent       string         `json:"selection_intent"`
	SelectionIntentFamily string         `json:"selection_intent_family,omitempty"`
	KnownSelectionIntent  bool           `json:"known_selection_intent"`
	CommentRegion         *string        `json:"comment_region,omitempty"`
	IncludeTrailingGap    bool           `json:"include_trailing_gap"`
	CommentAnchored       bool           `json:"comment_anchored"`
	Metadata              map[string]any `json:"metadata,omitempty"`
}

type StructuredEditMatchProfile struct {
	StartBoundary       string         `json:"start_boundary"`
	StartBoundaryFamily string         `json:"start_boundary_family,omitempty"`
	KnownStartBoundary  bool           `json:"known_start_boundary"`
	EndBoundary         string         `json:"end_boundary"`
	EndBoundaryFamily   string         `json:"end_boundary_family,omitempty"`
	KnownEndBoundary    bool           `json:"known_end_boundary"`
	PayloadKind         string         `json:"payload_kind"`
	PayloadFamily       string         `json:"payload_family,omitempty"`
	KnownPayloadKind    bool           `json:"known_payload_kind"`
	CommentAnchored     bool           `json:"comment_anchored"`
	TrailingGapExtended bool           `json:"trailing_gap_extended"`
	Metadata            map[string]any `json:"metadata,omitempty"`
}

type StructuredEditOperationProfile struct {
	OperationKind          string         `json:"operation_kind"`
	OperationFamily        string         `json:"operation_family,omitempty"`
	KnownOperationKind     bool           `json:"known_operation_kind"`
	SourceRequirement      string         `json:"source_requirement"`
	DestinationRequirement string         `json:"destination_requirement"`
	ReplacementSource      string         `json:"replacement_source"`
	CapturesSourceText     bool           `json:"captures_source_text"`
	SupportsIfMissing      bool           `json:"supports_if_missing"`
	Metadata               map[string]any `json:"metadata,omitempty"`
}

type StructuredEditDestinationProfile struct {
	ResolutionKind         string         `json:"resolution_kind"`
	ResolutionSource       string         `json:"resolution_source"`
	AnchorBoundary         string         `json:"anchor_boundary"`
	ResolutionFamily       string         `json:"resolution_family"`
	ResolutionSourceFamily string         `json:"resolution_source_family"`
	AnchorBoundaryFamily   string         `json:"anchor_boundary_family"`
	KnownResolutionKind    bool           `json:"known_resolution_kind"`
	KnownResolutionSource  bool           `json:"known_resolution_source"`
	KnownAnchorBoundary    bool           `json:"known_anchor_boundary"`
	UsedIfMissing          bool           `json:"used_if_missing"`
	Metadata               map[string]any `json:"metadata,omitempty"`
}

type StructuredEditRequest struct {
	OperationKind             string         `json:"operation_kind"`
	Content                   string         `json:"content"`
	SourceLabel               string         `json:"source_label"`
	TargetSelector            *string        `json:"target_selector,omitempty"`
	TargetSelectorFamily      *string        `json:"target_selector_family,omitempty"`
	DestinationSelector       *string        `json:"destination_selector,omitempty"`
	DestinationSelectorFamily *string        `json:"destination_selector_family,omitempty"`
	PayloadText               *string        `json:"payload_text,omitempty"`
	IfMissing                 *string        `json:"if_missing,omitempty"`
	Metadata                  map[string]any `json:"metadata,omitempty"`
}

type StructuredEditResult struct {
	OperationKind      string                            `json:"operation_kind"`
	UpdatedContent     string                            `json:"updated_content"`
	Changed            bool                              `json:"changed"`
	CapturedText       *string                           `json:"captured_text,omitempty"`
	MatchCount         *int                              `json:"match_count,omitempty"`
	OperationProfile   StructuredEditOperationProfile    `json:"operation_profile"`
	DestinationProfile *StructuredEditDestinationProfile `json:"destination_profile,omitempty"`
	Metadata           map[string]any                    `json:"metadata,omitempty"`
}

type StructuredEditApplication struct {
	Request  StructuredEditRequest `json:"request"`
	Result   StructuredEditResult  `json:"result"`
	Metadata map[string]any        `json:"metadata,omitempty"`
}

type StructuredEditTransportImportErrorCategory string

const (
	StructuredEditTransportKindMismatch       StructuredEditTransportImportErrorCategory = "kind_mismatch"
	StructuredEditTransportUnsupportedVersion StructuredEditTransportImportErrorCategory = "unsupported_version"
)

type StructuredEditTransportImportError struct {
	Category StructuredEditTransportImportErrorCategory `json:"category"`
	Message  string                                     `json:"message"`
}

type StructuredEditApplicationEnvelope struct {
	Kind        string                    `json:"kind"`
	Version     int                       `json:"version"`
	Application StructuredEditApplication `json:"application"`
}

type StructuredEditExecutionReport struct {
	Application     StructuredEditApplication `json:"application"`
	ProviderFamily  string                    `json:"provider_family"`
	ProviderBackend *string                   `json:"provider_backend,omitempty"`
	Diagnostics     []Diagnostic              `json:"diagnostics"`
	Metadata        map[string]any            `json:"metadata,omitempty"`
}

type StructuredEditProviderExecutionRequest struct {
	Request         StructuredEditRequest `json:"request"`
	ProviderFamily  string                `json:"provider_family"`
	ProviderBackend *string               `json:"provider_backend,omitempty"`
	Metadata        map[string]any        `json:"metadata,omitempty"`
}

type StructuredEditProviderExecutionRequestEnvelope struct {
	Kind             string                                 `json:"kind"`
	Version          int                                    `json:"version"`
	ExecutionRequest StructuredEditProviderExecutionRequest `json:"execution_request"`
}

type StructuredEditProviderExecutionPlan struct {
	ExecutionRequest   StructuredEditProviderExecutionRequest   `json:"execution_request"`
	ExecutorResolution StructuredEditProviderExecutorResolution `json:"executor_resolution"`
	Metadata           map[string]any                           `json:"metadata,omitempty"`
}

type StructuredEditProviderExecutionPlanEnvelope struct {
	Kind          string                              `json:"kind"`
	Version       int                                 `json:"version"`
	ExecutionPlan StructuredEditProviderExecutionPlan `json:"execution_plan"`
}

type StructuredEditProviderExecutionHandoff struct {
	ExecutionPlan     StructuredEditProviderExecutionPlan     `json:"execution_plan"`
	ExecutionDispatch StructuredEditProviderExecutionDispatch `json:"execution_dispatch"`
	Metadata          map[string]any                          `json:"metadata,omitempty"`
}

type StructuredEditProviderExecutionHandoffEnvelope struct {
	Kind             string                                 `json:"kind"`
	Version          int                                    `json:"version"`
	ExecutionHandoff StructuredEditProviderExecutionHandoff `json:"execution_handoff"`
}

type StructuredEditProviderExecutionInvocation struct {
	ExecutionHandoff StructuredEditProviderExecutionHandoff `json:"execution_handoff"`
	Metadata         map[string]any                         `json:"metadata,omitempty"`
}

type StructuredEditProviderExecutionInvocationEnvelope struct {
	Kind                string                                    `json:"kind"`
	Version             int                                       `json:"version"`
	ExecutionInvocation StructuredEditProviderExecutionInvocation `json:"execution_invocation"`
}

type StructuredEditProviderBatchExecutionInvocation struct {
	Invocations []StructuredEditProviderExecutionInvocation `json:"invocations"`
	Metadata    map[string]any                              `json:"metadata,omitempty"`
}

type StructuredEditProviderBatchExecutionInvocationEnvelope struct {
	Kind                     string                                         `json:"kind"`
	Version                  int                                            `json:"version"`
	BatchExecutionInvocation StructuredEditProviderBatchExecutionInvocation `json:"batch_execution_invocation"`
}

type StructuredEditProviderExecutionRunResult struct {
	ExecutionInvocation StructuredEditProviderExecutionInvocation `json:"execution_invocation"`
	Outcome             StructuredEditProviderExecutionOutcome    `json:"outcome"`
	Metadata            map[string]any                            `json:"metadata,omitempty"`
}

type StructuredEditProviderExecutionRunResultEnvelope struct {
	Kind               string                                   `json:"kind"`
	Version            int                                      `json:"version"`
	ExecutionRunResult StructuredEditProviderExecutionRunResult `json:"execution_run_result"`
}

type StructuredEditProviderBatchExecutionRunResult struct {
	RunResults []StructuredEditProviderExecutionRunResult `json:"run_results"`
	Metadata   map[string]any                             `json:"metadata,omitempty"`
}

type StructuredEditProviderBatchExecutionRunResultEnvelope struct {
	Kind                    string                                        `json:"kind"`
	Version                 int                                           `json:"version"`
	BatchExecutionRunResult StructuredEditProviderBatchExecutionRunResult `json:"batch_execution_run_result"`
}

type StructuredEditProviderExecutionReceipt struct {
	RunResult    StructuredEditProviderExecutionRunResult     `json:"run_result"`
	Provenance   *StructuredEditProviderExecutionProvenance   `json:"provenance,omitempty"`
	ReplayBundle *StructuredEditProviderExecutionReplayBundle `json:"replay_bundle,omitempty"`
	Metadata     map[string]any                               `json:"metadata,omitempty"`
}

type StructuredEditProviderExecutionReceiptEnvelope struct {
	Kind             string                                 `json:"kind"`
	Version          int                                    `json:"version"`
	ExecutionReceipt StructuredEditProviderExecutionReceipt `json:"execution_receipt"`
}

type StructuredEditProviderBatchExecutionReceipt struct {
	Receipts []StructuredEditProviderExecutionReceipt `json:"receipts"`
	Metadata map[string]any                           `json:"metadata,omitempty"`
}

type StructuredEditProviderBatchExecutionReceiptEnvelope struct {
	Kind                  string                                      `json:"kind"`
	Version               int                                         `json:"version"`
	BatchExecutionReceipt StructuredEditProviderBatchExecutionReceipt `json:"batch_execution_receipt"`
}

type StructuredEditProviderExecutionReceiptReplayRequest struct {
	ExecutionReceipt StructuredEditProviderExecutionReceipt `json:"execution_receipt"`
	ReplayMode       string                                 `json:"replay_mode"`
	Metadata         map[string]any                         `json:"metadata,omitempty"`
}

type StructuredEditProviderExecutionReceiptReplayRequestEnvelope struct {
	Kind                 string                                              `json:"kind"`
	Version              int                                                 `json:"version"`
	ReceiptReplayRequest StructuredEditProviderExecutionReceiptReplayRequest `json:"receipt_replay_request"`
}

type StructuredEditProviderBatchExecutionReceiptReplayRequest struct {
	Requests []StructuredEditProviderExecutionReceiptReplayRequest `json:"requests"`
	Metadata map[string]any                                        `json:"metadata,omitempty"`
}

type StructuredEditProviderBatchExecutionReceiptReplayRequestEnvelope struct {
	Kind                      string                                                   `json:"kind"`
	Version                   int                                                      `json:"version"`
	BatchReceiptReplayRequest StructuredEditProviderBatchExecutionReceiptReplayRequest `json:"batch_receipt_replay_request"`
}

type StructuredEditProviderExecutionReceiptReplayApplication struct {
	ReceiptReplayRequest StructuredEditProviderExecutionReceiptReplayRequest `json:"receipt_replay_request"`
	RunResult            StructuredEditProviderExecutionRunResult            `json:"run_result"`
	Metadata             map[string]any                                      `json:"metadata,omitempty"`
}

type StructuredEditProviderExecutionReceiptReplayApplicationEnvelope struct {
	Kind                     string                                                  `json:"kind"`
	Version                  int                                                     `json:"version"`
	ReceiptReplayApplication StructuredEditProviderExecutionReceiptReplayApplication `json:"receipt_replay_application"`
}

type StructuredEditProviderBatchExecutionReceiptReplayApplication struct {
	Applications []StructuredEditProviderExecutionReceiptReplayApplication `json:"applications"`
	Metadata     map[string]any                                            `json:"metadata,omitempty"`
}

type StructuredEditProviderBatchExecutionReceiptReplayApplicationEnvelope struct {
	Kind                          string                                                       `json:"kind"`
	Version                       int                                                          `json:"version"`
	BatchReceiptReplayApplication StructuredEditProviderBatchExecutionReceiptReplayApplication `json:"batch_receipt_replay_application"`
}

type StructuredEditProviderExecutionApplication struct {
	ExecutionRequest StructuredEditProviderExecutionRequest `json:"execution_request"`
	Report           StructuredEditExecutionReport          `json:"report"`
	Metadata         map[string]any                         `json:"metadata,omitempty"`
}

type StructuredEditProviderExecutionDispatch struct {
	ExecutionRequest        StructuredEditProviderExecutionRequest `json:"execution_request"`
	ResolvedProviderFamily  string                                 `json:"resolved_provider_family"`
	ResolvedProviderBackend string                                 `json:"resolved_provider_backend"`
	ExecutorLabel           *string                                `json:"executor_label,omitempty"`
	Metadata                map[string]any                         `json:"metadata,omitempty"`
}

type StructuredEditProviderExecutionDispatchEnvelope struct {
	Kind                      string                                  `json:"kind"`
	Version                   int                                     `json:"version"`
	ProviderExecutionDispatch StructuredEditProviderExecutionDispatch `json:"provider_execution_dispatch"`
}

type StructuredEditProviderExecutionOutcome struct {
	Dispatch    StructuredEditProviderExecutionDispatch    `json:"dispatch"`
	Application StructuredEditProviderExecutionApplication `json:"application"`
	Metadata    map[string]any                             `json:"metadata,omitempty"`
}

type StructuredEditProviderExecutionOutcomeEnvelope struct {
	Kind                     string                                 `json:"kind"`
	Version                  int                                    `json:"version"`
	ProviderExecutionOutcome StructuredEditProviderExecutionOutcome `json:"provider_execution_outcome"`
}

type StructuredEditProviderBatchExecutionOutcome struct {
	Outcomes []StructuredEditProviderExecutionOutcome `json:"outcomes"`
	Metadata map[string]any                           `json:"metadata,omitempty"`
}

type StructuredEditProviderBatchExecutionOutcomeEnvelope struct {
	Kind         string                                      `json:"kind"`
	Version      int                                         `json:"version"`
	BatchOutcome StructuredEditProviderBatchExecutionOutcome `json:"batch_outcome"`
}

type StructuredEditProviderExecutionProvenance struct {
	Dispatch    StructuredEditProviderExecutionDispatch `json:"dispatch"`
	Outcome     StructuredEditProviderExecutionOutcome  `json:"outcome"`
	Diagnostics []Diagnostic                            `json:"diagnostics"`
	Metadata    map[string]any                          `json:"metadata,omitempty"`
}

type StructuredEditProviderExecutionProvenanceEnvelope struct {
	Kind       string                                    `json:"kind"`
	Version    int                                       `json:"version"`
	Provenance StructuredEditProviderExecutionProvenance `json:"provenance"`
}

type StructuredEditProviderBatchExecutionProvenance struct {
	Provenances []StructuredEditProviderExecutionProvenance `json:"provenances"`
	Metadata    map[string]any                              `json:"metadata,omitempty"`
}

type StructuredEditProviderBatchExecutionProvenanceEnvelope struct {
	Kind            string                                         `json:"kind"`
	Version         int                                            `json:"version"`
	BatchProvenance StructuredEditProviderBatchExecutionProvenance `json:"batch_provenance"`
}

type StructuredEditProviderExecutionReplayBundle struct {
	ExecutionRequest StructuredEditProviderExecutionRequest    `json:"execution_request"`
	Provenance       StructuredEditProviderExecutionProvenance `json:"provenance"`
	Metadata         map[string]any                            `json:"metadata,omitempty"`
}

type StructuredEditProviderExecutionReplayBundleEnvelope struct {
	Kind         string                                      `json:"kind"`
	Version      int                                         `json:"version"`
	ReplayBundle StructuredEditProviderExecutionReplayBundle `json:"replay_bundle"`
}

type StructuredEditProviderBatchExecutionReplayBundle struct {
	ReplayBundles []StructuredEditProviderExecutionReplayBundle `json:"replay_bundles"`
	Metadata      map[string]any                                `json:"metadata,omitempty"`
}

type StructuredEditProviderBatchExecutionReplayBundleEnvelope struct {
	Kind              string                                           `json:"kind"`
	Version           int                                              `json:"version"`
	BatchReplayBundle StructuredEditProviderBatchExecutionReplayBundle `json:"batch_replay_bundle"`
}

type StructuredEditProviderExecutorProfile struct {
	ProviderFamily     string                           `json:"provider_family"`
	ProviderBackend    string                           `json:"provider_backend"`
	ExecutorLabel      string                           `json:"executor_label"`
	StructureProfile   StructuredEditStructureProfile   `json:"structure_profile"`
	SelectionProfile   StructuredEditSelectionProfile   `json:"selection_profile"`
	MatchProfile       StructuredEditMatchProfile       `json:"match_profile"`
	OperationProfiles  []StructuredEditOperationProfile `json:"operation_profiles"`
	DestinationProfile StructuredEditDestinationProfile `json:"destination_profile"`
	Metadata           map[string]any                   `json:"metadata,omitempty"`
}

type StructuredEditProviderExecutorProfileEnvelope struct {
	Kind            string                                `json:"kind"`
	Version         int                                   `json:"version"`
	ExecutorProfile StructuredEditProviderExecutorProfile `json:"executor_profile"`
}

type StructuredEditProviderExecutorRegistry struct {
	ExecutorProfiles []StructuredEditProviderExecutorProfile `json:"executor_profiles"`
	Metadata         map[string]any                          `json:"metadata,omitempty"`
}

type StructuredEditProviderExecutorRegistryEnvelope struct {
	Kind             string                                 `json:"kind"`
	Version          int                                    `json:"version"`
	ExecutorRegistry StructuredEditProviderExecutorRegistry `json:"executor_registry"`
}

type StructuredEditProviderExecutorSelectionPolicy struct {
	ProviderFamily        string         `json:"provider_family"`
	ProviderBackend       *string        `json:"provider_backend,omitempty"`
	ExecutorLabel         *string        `json:"executor_label,omitempty"`
	SelectionMode         string         `json:"selection_mode"`
	AllowRegistryFallback bool           `json:"allow_registry_fallback"`
	Metadata              map[string]any `json:"metadata,omitempty"`
}

type StructuredEditProviderExecutorSelectionPolicyEnvelope struct {
	Kind            string                                        `json:"kind"`
	Version         int                                           `json:"version"`
	SelectionPolicy StructuredEditProviderExecutorSelectionPolicy `json:"selection_policy"`
}

type StructuredEditProviderExecutorResolution struct {
	ExecutorRegistry        StructuredEditProviderExecutorRegistry        `json:"executor_registry"`
	SelectionPolicy         StructuredEditProviderExecutorSelectionPolicy `json:"selection_policy"`
	SelectedExecutorProfile StructuredEditProviderExecutorProfile         `json:"selected_executor_profile"`
	Metadata                map[string]any                                `json:"metadata,omitempty"`
}

type StructuredEditProviderExecutorResolutionEnvelope struct {
	Kind               string                                   `json:"kind"`
	Version            int                                      `json:"version"`
	ExecutorResolution StructuredEditProviderExecutorResolution `json:"executor_resolution"`
}

type StructuredEditProviderExecutionApplicationEnvelope struct {
	Kind                         string                                     `json:"kind"`
	Version                      int                                        `json:"version"`
	ProviderExecutionApplication StructuredEditProviderExecutionApplication `json:"provider_execution_application"`
}

type StructuredEditExecutionReportEnvelope struct {
	Kind    string                        `json:"kind"`
	Version int                           `json:"version"`
	Report  StructuredEditExecutionReport `json:"report"`
}

type StructuredEditBatchRequest struct {
	Requests []StructuredEditRequest `json:"requests"`
	Metadata map[string]any          `json:"metadata,omitempty"`
}

type StructuredEditProviderBatchExecutionRequest struct {
	Requests []StructuredEditProviderExecutionRequest `json:"requests"`
	Metadata map[string]any                           `json:"metadata,omitempty"`
}

type StructuredEditProviderBatchExecutionRequestEnvelope struct {
	Kind                  string                                      `json:"kind"`
	Version               int                                         `json:"version"`
	BatchExecutionRequest StructuredEditProviderBatchExecutionRequest `json:"batch_execution_request"`
}

type StructuredEditProviderBatchExecutionHandoff struct {
	Handoffs []StructuredEditProviderExecutionHandoff `json:"handoffs"`
	Metadata map[string]any                           `json:"metadata,omitempty"`
}

type StructuredEditProviderBatchExecutionHandoffEnvelope struct {
	Kind                  string                                      `json:"kind"`
	Version               int                                         `json:"version"`
	BatchExecutionHandoff StructuredEditProviderBatchExecutionHandoff `json:"batch_execution_handoff"`
}

type StructuredEditProviderBatchExecutionPlan struct {
	Plans    []StructuredEditProviderExecutionPlan `json:"plans"`
	Metadata map[string]any                        `json:"metadata,omitempty"`
}

type StructuredEditProviderBatchExecutionPlanEnvelope struct {
	Kind               string                                   `json:"kind"`
	Version            int                                      `json:"version"`
	BatchExecutionPlan StructuredEditProviderBatchExecutionPlan `json:"batch_execution_plan"`
}

type StructuredEditProviderBatchExecutionDispatch struct {
	Dispatches []StructuredEditProviderExecutionDispatch `json:"dispatches"`
	Metadata   map[string]any                            `json:"metadata,omitempty"`
}

type StructuredEditProviderBatchExecutionDispatchEnvelope struct {
	Kind          string                                       `json:"kind"`
	Version       int                                          `json:"version"`
	BatchDispatch StructuredEditProviderBatchExecutionDispatch `json:"batch_dispatch"`
}

type StructuredEditProviderBatchExecutionReport struct {
	Applications []StructuredEditProviderExecutionApplication `json:"applications"`
	Diagnostics  []Diagnostic                                 `json:"diagnostics"`
	Metadata     map[string]any                               `json:"metadata,omitempty"`
}

type StructuredEditProviderBatchExecutionReportEnvelope struct {
	Kind        string                                     `json:"kind"`
	Version     int                                        `json:"version"`
	BatchReport StructuredEditProviderBatchExecutionReport `json:"batch_report"`
}

type StructuredEditBatchReport struct {
	Reports     []StructuredEditExecutionReport `json:"reports"`
	Diagnostics []Diagnostic                    `json:"diagnostics"`
	Metadata    map[string]any                  `json:"metadata,omitempty"`
}

type StructuredEditBatchReportEnvelope struct {
	Kind        string                    `json:"kind"`
	Version     int                       `json:"version"`
	BatchReport StructuredEditBatchReport `json:"batch_report"`
}

type TemplateTargetClassification struct {
	DestinationPath string `json:"destination_path"`
	FileType        string `json:"file_type"`
	Family          string `json:"family"`
	Dialect         string `json:"dialect"`
}

type TemplateDestinationContext struct {
	ProjectName string `json:"project_name,omitempty"`
}

type TemplateTokenConfig struct {
	Pre            string   `json:"pre"`
	Post           string   `json:"post"`
	Separators     []string `json:"separators"`
	MinSegments    int      `json:"min_segments"`
	MaxSegments    *int     `json:"max_segments,omitempty"`
	SegmentPattern string   `json:"segment_pattern"`
}

type TemplateStrategy string

const (
	TemplateStrategyMerge           TemplateStrategy = "merge"
	TemplateStrategyAcceptTemplate  TemplateStrategy = "accept_template"
	TemplateStrategyKeepDestination TemplateStrategy = "keep_destination"
	TemplateStrategyRawCopy         TemplateStrategy = "raw_copy"
)

type TemplateStrategyOverride struct {
	Path     string           `json:"path"`
	Strategy TemplateStrategy `json:"strategy"`
}

type TemplatePlanEntry struct {
	TemplateSourcePath     string                       `json:"template_source_path"`
	LogicalDestinationPath string                       `json:"logical_destination_path"`
	DestinationPath        *string                      `json:"destination_path"`
	Classification         TemplateTargetClassification `json:"classification"`
	Strategy               TemplateStrategy             `json:"strategy"`
	Action                 string                       `json:"action"`
}

type TemplatePlanStateEntry struct {
	TemplateSourcePath     string                       `json:"template_source_path"`
	LogicalDestinationPath string                       `json:"logical_destination_path"`
	DestinationPath        *string                      `json:"destination_path"`
	Classification         TemplateTargetClassification `json:"classification"`
	Strategy               TemplateStrategy             `json:"strategy"`
	Action                 string                       `json:"action"`
	DestinationExists      bool                         `json:"destination_exists"`
	WriteAction            string                       `json:"write_action"`
}

type TemplatePlanBlockReason string

const (
	TemplatePlanBlockReasonUnresolvedTokens TemplatePlanBlockReason = "unresolved_tokens"
)

type TemplatePlanTokenStateEntry struct {
	TemplateSourcePath      string                       `json:"template_source_path"`
	LogicalDestinationPath  string                       `json:"logical_destination_path"`
	DestinationPath         *string                      `json:"destination_path"`
	Classification          TemplateTargetClassification `json:"classification"`
	Strategy                TemplateStrategy             `json:"strategy"`
	Action                  string                       `json:"action"`
	DestinationExists       bool                         `json:"destination_exists"`
	WriteAction             string                       `json:"write_action"`
	TokenKeys               []string                     `json:"token_keys"`
	UnresolvedTokenKeys     []string                     `json:"unresolved_token_keys"`
	TokenResolutionRequired bool                         `json:"token_resolution_required"`
	Blocked                 bool                         `json:"blocked"`
	BlockReason             *TemplatePlanBlockReason     `json:"block_reason,omitempty"`
}

type TemplatePreparationAction string

const (
	TemplatePreparationBlocked       TemplatePreparationAction = "blocked"
	TemplatePreparationResolveTokens TemplatePreparationAction = "resolve_tokens"
	TemplatePreparationPassThrough   TemplatePreparationAction = "pass_through"
)

type TemplatePreparedEntry struct {
	TemplateSourcePath      string                       `json:"template_source_path"`
	LogicalDestinationPath  string                       `json:"logical_destination_path"`
	DestinationPath         *string                      `json:"destination_path"`
	Classification          TemplateTargetClassification `json:"classification"`
	Strategy                TemplateStrategy             `json:"strategy"`
	Action                  string                       `json:"action"`
	DestinationExists       bool                         `json:"destination_exists"`
	WriteAction             string                       `json:"write_action"`
	TokenKeys               []string                     `json:"token_keys"`
	UnresolvedTokenKeys     []string                     `json:"unresolved_token_keys"`
	TokenResolutionRequired bool                         `json:"token_resolution_required"`
	Blocked                 bool                         `json:"blocked"`
	BlockReason             *TemplatePlanBlockReason     `json:"block_reason,omitempty"`
	TemplateContent         string                       `json:"template_content"`
	PreparedTemplateContent *string                      `json:"prepared_template_content"`
	PreparationAction       TemplatePreparationAction    `json:"preparation_action"`
}

type TemplateExecutionAction string

const (
	TemplateExecutionBlocked       TemplateExecutionAction = "blocked"
	TemplateExecutionOmit          TemplateExecutionAction = "omit"
	TemplateExecutionKeep          TemplateExecutionAction = "keep"
	TemplateExecutionRawCopy       TemplateExecutionAction = "raw_copy"
	TemplateExecutionWritePrepared TemplateExecutionAction = "write_prepared_content"
	TemplateExecutionMergePrepared TemplateExecutionAction = "merge_prepared_content"
)

type TemplateExecutionPlanEntry struct {
	TemplateSourcePath      string                       `json:"template_source_path"`
	LogicalDestinationPath  string                       `json:"logical_destination_path"`
	DestinationPath         *string                      `json:"destination_path"`
	Classification          TemplateTargetClassification `json:"classification"`
	Strategy                TemplateStrategy             `json:"strategy"`
	Action                  string                       `json:"action"`
	DestinationExists       bool                         `json:"destination_exists"`
	WriteAction             string                       `json:"write_action"`
	TokenKeys               []string                     `json:"token_keys"`
	UnresolvedTokenKeys     []string                     `json:"unresolved_token_keys"`
	TokenResolutionRequired bool                         `json:"token_resolution_required"`
	Blocked                 bool                         `json:"blocked"`
	BlockReason             *TemplatePlanBlockReason     `json:"block_reason,omitempty"`
	TemplateContent         string                       `json:"template_content"`
	PreparedTemplateContent *string                      `json:"prepared_template_content"`
	PreparationAction       TemplatePreparationAction    `json:"preparation_action"`
	ExecutionAction         TemplateExecutionAction      `json:"execution_action"`
	Ready                   bool                         `json:"ready"`
	DestinationContent      *string                      `json:"destination_content"`
}

type TemplatePreviewResult struct {
	ResultFiles  map[string]string `json:"result_files"`
	CreatedPaths []string          `json:"created_paths"`
	UpdatedPaths []string          `json:"updated_paths"`
	KeptPaths    []string          `json:"kept_paths"`
	BlockedPaths []string          `json:"blocked_paths"`
	OmittedPaths []string          `json:"omitted_paths"`
}

type TemplateApplyResult struct {
	ResultFiles  map[string]string `json:"result_files"`
	CreatedPaths []string          `json:"created_paths"`
	UpdatedPaths []string          `json:"updated_paths"`
	KeptPaths    []string          `json:"kept_paths"`
	BlockedPaths []string          `json:"blocked_paths"`
	OmittedPaths []string          `json:"omitted_paths"`
	Diagnostics  []Diagnostic      `json:"diagnostics"`
}

type TemplateConvergenceResult struct {
	Converged    bool     `json:"converged"`
	PendingPaths []string `json:"pending_paths"`
}

type TemplateTreeRunResult struct {
	ExecutionPlan []TemplateExecutionPlanEntry `json:"execution_plan"`
	ApplyResult   TemplateApplyResult          `json:"apply_result"`
}

type TemplateTreeRunStatus string

const (
	TemplateTreeRunCreated TemplateTreeRunStatus = "created"
	TemplateTreeRunUpdated TemplateTreeRunStatus = "updated"
	TemplateTreeRunKept    TemplateTreeRunStatus = "kept"
	TemplateTreeRunBlocked TemplateTreeRunStatus = "blocked"
	TemplateTreeRunOmitted TemplateTreeRunStatus = "omitted"
)

type TemplateTreeRunReportEntry struct {
	TemplateSourcePath     string                  `json:"template_source_path"`
	LogicalDestinationPath string                  `json:"logical_destination_path"`
	DestinationPath        *string                 `json:"destination_path"`
	ExecutionAction        TemplateExecutionAction `json:"execution_action"`
	Status                 TemplateTreeRunStatus   `json:"status"`
}

type TemplateTreeRunReportSummary struct {
	Created int `json:"created"`
	Updated int `json:"updated"`
	Kept    int `json:"kept"`
	Blocked int `json:"blocked"`
	Omitted int `json:"omitted"`
}

type TemplateTreeRunReport struct {
	Entries []TemplateTreeRunReportEntry `json:"entries"`
	Summary TemplateTreeRunReportSummary `json:"summary"`
}

type TemplateDirectoryApplyReportEntry struct {
	TemplateSourcePath     string                  `json:"template_source_path"`
	LogicalDestinationPath string                  `json:"logical_destination_path"`
	DestinationPath        *string                 `json:"destination_path"`
	ExecutionAction        TemplateExecutionAction `json:"execution_action"`
	Status                 TemplateTreeRunStatus   `json:"status"`
	Written                bool                    `json:"written"`
}

type TemplateDirectoryApplyReportSummary struct {
	Created int `json:"created"`
	Updated int `json:"updated"`
	Kept    int `json:"kept"`
	Blocked int `json:"blocked"`
	Omitted int `json:"omitted"`
	Written int `json:"written"`
}

type TemplateDirectoryApplyReport struct {
	Entries []TemplateDirectoryApplyReportEntry `json:"entries"`
	Summary TemplateDirectoryApplyReportSummary `json:"summary"`
}

type TemplateDirectoryPlanStatus string

const (
	TemplateDirectoryPlanCreate  TemplateDirectoryPlanStatus = "create"
	TemplateDirectoryPlanUpdate  TemplateDirectoryPlanStatus = "update"
	TemplateDirectoryPlanKeep    TemplateDirectoryPlanStatus = "keep"
	TemplateDirectoryPlanBlocked TemplateDirectoryPlanStatus = "blocked"
	TemplateDirectoryPlanOmitted TemplateDirectoryPlanStatus = "omitted"
)

type TemplateDirectoryPlanReportEntry struct {
	TemplateSourcePath     string                      `json:"template_source_path"`
	LogicalDestinationPath string                      `json:"logical_destination_path"`
	DestinationPath        *string                     `json:"destination_path"`
	ExecutionAction        TemplateExecutionAction     `json:"execution_action"`
	WriteAction            string                      `json:"write_action"`
	Status                 TemplateDirectoryPlanStatus `json:"status"`
	Previewable            bool                        `json:"previewable"`
}

type TemplateDirectoryPlanReportSummary struct {
	Create  int `json:"create"`
	Update  int `json:"update"`
	Keep    int `json:"keep"`
	Blocked int `json:"blocked"`
	Omitted int `json:"omitted"`
}

type TemplateDirectoryPlanReport struct {
	Entries []TemplateDirectoryPlanReportEntry `json:"entries"`
	Summary TemplateDirectoryPlanReportSummary `json:"summary"`
}

type TemplateDirectoryRunnerReport struct {
	PlanReport  TemplateDirectoryPlanReport   `json:"plan_report"`
	Preview     *TemplatePreviewResult        `json:"preview"`
	RunReport   *TemplateTreeRunReport        `json:"run_report"`
	ApplyReport *TemplateDirectoryApplyReport `json:"apply_report"`
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
	ReplayContext            ReviewReplayContext       `json:"replay_context"`
	Decisions                []ReviewDecision          `json:"decisions"`
	ReviewedNestedExecutions []ReviewedNestedExecution `json:"reviewed_nested_executions,omitempty"`
}

const ReviewTransportVersion = 1
const StructuredEditTransportVersion = 1

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

type ReviewedNestedExecutionApplication[T any] struct {
	Diagnostics []Diagnostic                       `json:"diagnostics"`
	Results     []ReviewedNestedExecutionResult[T] `json:"results"`
}

type ConformanceManifestReviewedNestedApplication[T any] struct {
	State   ConformanceManifestReviewState     `json:"state"`
	Results []ReviewedNestedExecutionResult[T] `json:"results"`
}

type ReviewedNestedExecutionResult[T any] struct {
	Execution ReviewedNestedExecution `json:"execution"`
	Result    MergeResult[T]          `json:"result"`
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
	Report                   NamedConformanceSuiteReportEnvelope `json:"report"`
	Diagnostics              []Diagnostic                        `json:"diagnostics"`
	Requests                 []ReviewRequest                     `json:"requests"`
	AppliedDecisions         []ReviewDecision                    `json:"applied_decisions"`
	HostHints                ReviewHostHints                     `json:"host_hints"`
	ReplayContext            ReviewReplayContext                 `json:"replay_context"`
	ReviewedNestedExecutions []ReviewedNestedExecution           `json:"reviewed_nested_executions,omitempty"`
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

func NormalizeTemplateSourcePath(path string) string {
	if strings.HasSuffix(path, ".no-osc.example") {
		return strings.TrimSuffix(path, ".no-osc.example")
	}

	if strings.HasSuffix(path, ".example") {
		return strings.TrimSuffix(path, ".example")
	}

	return path
}

func ClassifyTemplateTargetPath(path string) TemplateTargetClassification {
	classify := func(fileType string, family string, dialect string) TemplateTargetClassification {
		return TemplateTargetClassification{
			DestinationPath: path,
			FileType:        fileType,
			Family:          family,
			Dialect:         dialect,
		}
	}

	normalizedPath := strings.TrimPrefix(path, "./")
	lowerPath := strings.ToLower(normalizedPath)
	base := pathBase(normalizedPath)
	lowerBase := strings.ToLower(base)

	switch normalizedPath {
	case ".git-hooks/commit-msg":
		return classify("ruby", "ruby", "ruby")
	case ".git-hooks/prepare-commit-msg":
		return classify("bash", "bash", "bash")
	}

	switch base {
	case "Gemfile", "Appraisal.root.gemfile":
		return classify("gemfile", "ruby", "ruby")
	case "Appraisals":
		return classify("appraisals", "ruby", "ruby")
	case "Rakefile", ".simplecov":
		return classify("ruby", "ruby", "ruby")
	case ".envrc":
		return classify("bash", "bash", "bash")
	case ".tool-versions":
		return classify("tool_versions", "text", "tool_versions")
	case "CITATION.cff":
		return classify("yaml", "yaml", "yaml")
	}

	switch {
	case strings.HasSuffix(lowerBase, ".gemspec"):
		return classify("gemspec", "ruby", "ruby")
	case strings.HasSuffix(lowerBase, ".gemfile"):
		return classify("gemfile", "ruby", "ruby")
	case strings.HasSuffix(lowerBase, ".rb"), strings.HasSuffix(lowerBase, ".rake"):
		return classify("ruby", "ruby", "ruby")
	case strings.HasSuffix(lowerPath, ".yml"), strings.HasSuffix(lowerPath, ".yaml"):
		return classify("yaml", "yaml", "yaml")
	case strings.HasSuffix(lowerPath, ".md"), strings.HasSuffix(lowerPath, ".markdown"):
		return classify("markdown", "markdown", "markdown")
	case strings.HasSuffix(lowerPath, ".sh"), strings.HasSuffix(lowerPath, ".bash"):
		return classify("bash", "bash", "bash")
	case lowerBase == ".env", strings.HasPrefix(lowerBase, ".env."):
		return classify("dotenv", "dotenv", "dotenv")
	case strings.HasSuffix(lowerPath, ".jsonc"):
		return classify("json", "json", "jsonc")
	case strings.HasSuffix(lowerPath, ".json"):
		return classify("json", "json", "json")
	case strings.HasSuffix(lowerPath, ".toml"):
		return classify("toml", "toml", "toml")
	case strings.HasSuffix(lowerPath, ".rbs"):
		return classify("rbs", "rbs", "rbs")
	default:
		return classify("text", "text", "text")
	}
}

func ResolveTemplateDestinationPath(path string, context *TemplateDestinationContext) *string {
	switch path {
	case ".kettle-jem.yml":
		return nil
	case ".env.local":
		resolved := ".env.local.example"
		return &resolved
	case "gem.gemspec":
		if context != nil && strings.TrimSpace(context.ProjectName) != "" {
			resolved := strings.TrimSpace(context.ProjectName) + ".gemspec"
			return &resolved
		}
	}

	resolved := path
	return &resolved
}

func DefaultTemplateTokenConfig() TemplateTokenConfig {
	return TemplateTokenConfig{
		Pre:            "{",
		Post:           "}",
		Separators:     []string{"|", ":"},
		MinSegments:    2,
		SegmentPattern: "[A-Za-z0-9_]",
	}
}

func templateTokenSeparatorAt(config TemplateTokenConfig, boundaryIndex int) string {
	if boundaryIndex < len(config.Separators) {
		return config.Separators[boundaryIndex]
	}

	return config.Separators[len(config.Separators)-1]
}

func validTemplateTokenKey(key string, config TemplateTokenConfig) bool {
	if key == "" {
		return false
	}

	segmentCharacter := regexp.MustCompile("^" + config.SegmentPattern + "$")
	index := 0
	segments := 0
	boundaryIndex := 0

	for index < len(key) {
		segmentStart := index
		for index < len(key) && segmentCharacter.MatchString(string(key[index])) {
			index++
		}

		if index == segmentStart {
			return false
		}

		segments++
		if index == len(key) {
			break
		}

		separator := templateTokenSeparatorAt(config, boundaryIndex)
		if separator == "" || !strings.HasPrefix(key[index:], separator) {
			return false
		}

		index += len(separator)
		boundaryIndex++
	}

	if segments < config.MinSegments {
		return false
	}

	return config.MaxSegments == nil || segments <= *config.MaxSegments
}

func TemplateTokenKeys(content string, config *TemplateTokenConfig) []string {
	resolvedConfig := DefaultTemplateTokenConfig()
	if config != nil {
		resolvedConfig = *config
	}
	if content == "" || !strings.Contains(content, resolvedConfig.Pre) {
		return []string{}
	}

	keys := make([]string, 0)
	seen := map[string]bool{}
	offset := 0
	for offset < len(content) {
		tokenStart := strings.Index(content[offset:], resolvedConfig.Pre)
		if tokenStart == -1 {
			break
		}
		tokenStart += offset
		contentStart := tokenStart + len(resolvedConfig.Pre)
		tokenEnd := strings.Index(content[contentStart:], resolvedConfig.Post)
		if tokenEnd == -1 {
			break
		}
		tokenEnd += contentStart

		key := content[contentStart:tokenEnd]
		if validTemplateTokenKey(key, resolvedConfig) && !seen[key] {
			seen[key] = true
			keys = append(keys, key)
		}

		offset = tokenEnd + len(resolvedConfig.Post)
	}

	return keys
}

func UnresolvedTemplateTokenKeys(content string, replacements map[string]string, config *TemplateTokenConfig) []string {
	keys := TemplateTokenKeys(content, config)
	unresolved := make([]string, 0, len(keys))
	for _, key := range keys {
		if _, ok := replacements[key]; !ok {
			unresolved = append(unresolved, key)
		}
	}

	return unresolved
}

func ResolveTemplateTokens(content string, replacements map[string]string, config *TemplateTokenConfig) string {
	resolvedConfig := DefaultTemplateTokenConfig()
	if config != nil {
		resolvedConfig = *config
	}

	resolved := content
	for _, key := range TemplateTokenKeys(content, &resolvedConfig) {
		replacement, ok := replacements[key]
		if !ok {
			continue
		}

		resolved = strings.ReplaceAll(
			resolved,
			resolvedConfig.Pre+key+resolvedConfig.Post,
			replacement,
		)
	}

	return resolved
}

func SelectTemplateStrategy(path string, defaultStrategy TemplateStrategy, overrides []TemplateStrategyOverride) TemplateStrategy {
	normalizedPath := strings.TrimPrefix(path, "./")
	for _, override := range overrides {
		if strings.TrimPrefix(override.Path, "./") == normalizedPath {
			return override.Strategy
		}
	}

	if defaultStrategy == "" {
		return TemplateStrategyMerge
	}

	return defaultStrategy
}

func PlanTemplateEntries(
	templateSourcePaths []string,
	context *TemplateDestinationContext,
	defaultStrategy TemplateStrategy,
	overrides []TemplateStrategyOverride,
) []TemplatePlanEntry {
	entries := make([]TemplatePlanEntry, 0, len(templateSourcePaths))
	for _, templateSourcePath := range templateSourcePaths {
		logicalDestinationPath := NormalizeTemplateSourcePath(templateSourcePath)
		destinationPath := ResolveTemplateDestinationPath(logicalDestinationPath, context)
		classification := ClassifyTemplateTargetPath(logicalDestinationPath)
		strategy := SelectTemplateStrategy(logicalDestinationPath, defaultStrategy, overrides)
		action := string(strategy)
		if destinationPath == nil {
			action = "omit"
		}

		entries = append(entries, TemplatePlanEntry{
			TemplateSourcePath:     templateSourcePath,
			LogicalDestinationPath: logicalDestinationPath,
			DestinationPath:        destinationPath,
			Classification:         classification,
			Strategy:               strategy,
			Action:                 action,
		})
	}

	return entries
}

func EnrichTemplatePlanEntries(entries []TemplatePlanEntry, existingDestinationPaths []string) []TemplatePlanStateEntry {
	results := make([]TemplatePlanStateEntry, 0, len(entries))
	existing := map[string]bool{}
	for _, path := range existingDestinationPaths {
		existing[path] = true
	}

	for _, entry := range entries {
		destinationExists := false
		writeAction := "omit"
		if entry.DestinationPath != nil {
			destinationExists = existing[*entry.DestinationPath]
			if entry.Strategy == TemplateStrategyKeepDestination {
				writeAction = "keep"
			} else if destinationExists {
				writeAction = "update"
			} else {
				writeAction = "create"
			}
		}

		results = append(results, TemplatePlanStateEntry{
			TemplateSourcePath:     entry.TemplateSourcePath,
			LogicalDestinationPath: entry.LogicalDestinationPath,
			DestinationPath:        entry.DestinationPath,
			Classification:         entry.Classification,
			Strategy:               entry.Strategy,
			Action:                 entry.Action,
			DestinationExists:      destinationExists,
			WriteAction:            writeAction,
		})
	}

	return results
}

func EnrichTemplatePlanEntriesWithTokenState(
	entries []TemplatePlanStateEntry,
	templateContents map[string]string,
	replacements map[string]string,
	config *TemplateTokenConfig,
) []TemplatePlanTokenStateEntry {
	results := make([]TemplatePlanTokenStateEntry, 0, len(entries))

	for _, entry := range entries {
		content := templateContents[entry.TemplateSourcePath]
		tokenKeys := TemplateTokenKeys(content, config)
		unresolvedTokenKeys := make([]string, 0, len(tokenKeys))
		for _, key := range tokenKeys {
			if _, ok := replacements[key]; !ok {
				unresolvedTokenKeys = append(unresolvedTokenKeys, key)
			}
		}

		tokenResolutionRequired := entry.DestinationPath != nil &&
			entry.Strategy != TemplateStrategyKeepDestination &&
			entry.Strategy != TemplateStrategyRawCopy
		blocked := tokenResolutionRequired && len(unresolvedTokenKeys) > 0
		var blockReason *TemplatePlanBlockReason
		if blocked {
			reason := TemplatePlanBlockReasonUnresolvedTokens
			blockReason = &reason
		}

		results = append(results, TemplatePlanTokenStateEntry{
			TemplateSourcePath:      entry.TemplateSourcePath,
			LogicalDestinationPath:  entry.LogicalDestinationPath,
			DestinationPath:         entry.DestinationPath,
			Classification:          entry.Classification,
			Strategy:                entry.Strategy,
			Action:                  entry.Action,
			DestinationExists:       entry.DestinationExists,
			WriteAction:             entry.WriteAction,
			TokenKeys:               tokenKeys,
			UnresolvedTokenKeys:     unresolvedTokenKeys,
			TokenResolutionRequired: tokenResolutionRequired,
			Blocked:                 blocked,
			BlockReason:             blockReason,
		})
	}

	return results
}

func PrepareTemplateEntries(
	entries []TemplatePlanTokenStateEntry,
	templateContents map[string]string,
	replacements map[string]string,
	config *TemplateTokenConfig,
) []TemplatePreparedEntry {
	results := make([]TemplatePreparedEntry, 0, len(entries))

	for _, entry := range entries {
		templateContent := templateContents[entry.TemplateSourcePath]
		var preparedTemplateContent *string
		preparationAction := TemplatePreparationPassThrough

		if entry.Blocked {
			preparationAction = TemplatePreparationBlocked
		} else if entry.TokenResolutionRequired {
			resolved := ResolveTemplateTokens(templateContent, replacements, config)
			preparedTemplateContent = &resolved
			preparationAction = TemplatePreparationResolveTokens
		} else {
			resolved := templateContent
			preparedTemplateContent = &resolved
		}

		results = append(results, TemplatePreparedEntry{
			TemplateSourcePath:      entry.TemplateSourcePath,
			LogicalDestinationPath:  entry.LogicalDestinationPath,
			DestinationPath:         entry.DestinationPath,
			Classification:          entry.Classification,
			Strategy:                entry.Strategy,
			Action:                  entry.Action,
			DestinationExists:       entry.DestinationExists,
			WriteAction:             entry.WriteAction,
			TokenKeys:               entry.TokenKeys,
			UnresolvedTokenKeys:     entry.UnresolvedTokenKeys,
			TokenResolutionRequired: entry.TokenResolutionRequired,
			Blocked:                 entry.Blocked,
			BlockReason:             entry.BlockReason,
			TemplateContent:         templateContent,
			PreparedTemplateContent: preparedTemplateContent,
			PreparationAction:       preparationAction,
		})
	}

	return results
}

func PlanTemplateExecution(
	entries []TemplatePreparedEntry,
	destinationContents map[string]string,
) []TemplateExecutionPlanEntry {
	results := make([]TemplateExecutionPlanEntry, 0, len(entries))

	for _, entry := range entries {
		var destinationContent *string
		if entry.DestinationPath != nil {
			if content, ok := destinationContents[*entry.DestinationPath]; ok {
				destinationContent = &content
			}
		}

		executionAction := TemplateExecutionMergePrepared
		switch {
		case entry.Blocked:
			executionAction = TemplateExecutionBlocked
		case entry.DestinationPath == nil:
			executionAction = TemplateExecutionOmit
		case entry.WriteAction == "keep":
			executionAction = TemplateExecutionKeep
		case entry.Strategy == TemplateStrategyRawCopy:
			executionAction = TemplateExecutionRawCopy
		case entry.Strategy == TemplateStrategyAcceptTemplate:
			executionAction = TemplateExecutionWritePrepared
		default:
			executionAction = TemplateExecutionMergePrepared
		}
		ready := executionAction != TemplateExecutionBlocked &&
			executionAction != TemplateExecutionOmit &&
			executionAction != TemplateExecutionKeep

		results = append(results, TemplateExecutionPlanEntry{
			TemplateSourcePath:      entry.TemplateSourcePath,
			LogicalDestinationPath:  entry.LogicalDestinationPath,
			DestinationPath:         entry.DestinationPath,
			Classification:          entry.Classification,
			Strategy:                entry.Strategy,
			Action:                  entry.Action,
			DestinationExists:       entry.DestinationExists,
			WriteAction:             entry.WriteAction,
			TokenKeys:               entry.TokenKeys,
			UnresolvedTokenKeys:     entry.UnresolvedTokenKeys,
			TokenResolutionRequired: entry.TokenResolutionRequired,
			Blocked:                 entry.Blocked,
			BlockReason:             entry.BlockReason,
			TemplateContent:         entry.TemplateContent,
			PreparedTemplateContent: entry.PreparedTemplateContent,
			PreparationAction:       entry.PreparationAction,
			ExecutionAction:         executionAction,
			Ready:                   ready,
			DestinationContent:      destinationContent,
		})
	}

	return results
}

func PlanTemplateTreeExecution(
	templateSourcePaths []string,
	templateContents map[string]string,
	existingDestinationPaths []string,
	destinationContents map[string]string,
	context *TemplateDestinationContext,
	defaultStrategy TemplateStrategy,
	overrides []TemplateStrategyOverride,
	replacements map[string]string,
	config *TemplateTokenConfig,
) []TemplateExecutionPlanEntry {
	plannedEntries := PlanTemplateEntries(templateSourcePaths, context, defaultStrategy, overrides)
	statefulEntries := EnrichTemplatePlanEntries(plannedEntries, existingDestinationPaths)
	tokenStateEntries := EnrichTemplatePlanEntriesWithTokenState(
		statefulEntries,
		templateContents,
		replacements,
		config,
	)
	preparedEntries := PrepareTemplateEntries(tokenStateEntries, templateContents, replacements, config)

	return PlanTemplateExecution(preparedEntries, destinationContents)
}

func PreviewTemplateExecution(entries []TemplateExecutionPlanEntry) TemplatePreviewResult {
	result := TemplatePreviewResult{
		ResultFiles:  map[string]string{},
		CreatedPaths: []string{},
		UpdatedPaths: []string{},
		KeptPaths:    []string{},
		BlockedPaths: []string{},
		OmittedPaths: []string{},
	}

	for _, entry := range entries {
		switch entry.ExecutionAction {
		case TemplateExecutionBlocked:
			if entry.DestinationPath != nil {
				result.BlockedPaths = append(result.BlockedPaths, *entry.DestinationPath)
			}
		case TemplateExecutionOmit:
			result.OmittedPaths = append(result.OmittedPaths, entry.LogicalDestinationPath)
		case TemplateExecutionKeep:
			if entry.DestinationPath != nil && entry.DestinationContent != nil {
				result.ResultFiles[*entry.DestinationPath] = *entry.DestinationContent
				result.KeptPaths = append(result.KeptPaths, *entry.DestinationPath)
			}
		case TemplateExecutionRawCopy, TemplateExecutionWritePrepared:
			if entry.DestinationPath != nil && entry.PreparedTemplateContent != nil {
				result.ResultFiles[*entry.DestinationPath] = *entry.PreparedTemplateContent
				if entry.DestinationExists && entry.DestinationContent != nil &&
					*entry.DestinationContent == *entry.PreparedTemplateContent {
					result.KeptPaths = append(result.KeptPaths, *entry.DestinationPath)
				} else if entry.DestinationExists {
					result.UpdatedPaths = append(result.UpdatedPaths, *entry.DestinationPath)
				} else {
					result.CreatedPaths = append(result.CreatedPaths, *entry.DestinationPath)
				}
			}
		case TemplateExecutionMergePrepared:
			if entry.DestinationPath != nil && entry.PreparedTemplateContent != nil && entry.DestinationContent == nil {
				result.ResultFiles[*entry.DestinationPath] = *entry.PreparedTemplateContent
				if entry.DestinationExists {
					result.UpdatedPaths = append(result.UpdatedPaths, *entry.DestinationPath)
				} else {
					result.CreatedPaths = append(result.CreatedPaths, *entry.DestinationPath)
				}
			}
		}
	}

	return result
}

func ApplyTemplateExecution(
	entries []TemplateExecutionPlanEntry,
	mergePreparedContent func(TemplateExecutionPlanEntry) MergeResult[string],
) TemplateApplyResult {
	result := TemplateApplyResult{
		ResultFiles:  map[string]string{},
		CreatedPaths: []string{},
		UpdatedPaths: []string{},
		KeptPaths:    []string{},
		BlockedPaths: []string{},
		OmittedPaths: []string{},
		Diagnostics:  []Diagnostic{},
	}

	for _, entry := range entries {
		switch entry.ExecutionAction {
		case TemplateExecutionBlocked:
			if entry.DestinationPath != nil {
				result.BlockedPaths = append(result.BlockedPaths, *entry.DestinationPath)
			}
		case TemplateExecutionOmit:
			result.OmittedPaths = append(result.OmittedPaths, entry.LogicalDestinationPath)
		case TemplateExecutionKeep:
			if entry.DestinationPath != nil && entry.DestinationContent != nil {
				result.ResultFiles[*entry.DestinationPath] = *entry.DestinationContent
				result.KeptPaths = append(result.KeptPaths, *entry.DestinationPath)
			}
		case TemplateExecutionRawCopy, TemplateExecutionWritePrepared:
			if entry.DestinationPath != nil && entry.PreparedTemplateContent != nil {
				recordTemplateApplyOutput(&result, entry, *entry.PreparedTemplateContent)
			}
		case TemplateExecutionMergePrepared:
			if entry.DestinationPath == nil || entry.PreparedTemplateContent == nil {
				continue
			}
			if entry.DestinationContent == nil {
				recordTemplateApplyOutput(&result, entry, *entry.PreparedTemplateContent)
				continue
			}

			mergeResult := mergePreparedContent(entry)
			result.Diagnostics = append(result.Diagnostics, mergeResult.Diagnostics...)
			if !mergeResult.OK || mergeResult.Output == nil {
				result.BlockedPaths = append(result.BlockedPaths, *entry.DestinationPath)
				continue
			}

			recordTemplateApplyOutput(&result, entry, *mergeResult.Output)
		}
	}

	return result
}

func EvaluateTemplateTreeConvergence(
	templateSourcePaths []string,
	templateContents map[string]string,
	destinationContents map[string]string,
	context *TemplateDestinationContext,
	defaultStrategy TemplateStrategy,
	overrides []TemplateStrategyOverride,
	replacements map[string]string,
	config *TemplateTokenConfig,
) TemplateConvergenceResult {
	existingDestinationPaths := mapsKeys(destinationContents)
	slices.Sort(existingDestinationPaths)
	executionPlan := PlanTemplateTreeExecution(
		templateSourcePaths,
		templateContents,
		existingDestinationPaths,
		destinationContents,
		context,
		defaultStrategy,
		overrides,
		replacements,
		config,
	)
	pendingPaths := make([]string, 0)
	for _, entry := range executionPlan {
		if entry.Blocked {
			if entry.DestinationPath != nil {
				pendingPaths = append(pendingPaths, *entry.DestinationPath)
			} else {
				pendingPaths = append(pendingPaths, entry.LogicalDestinationPath)
			}
			continue
		}
		if !entry.Ready {
			continue
		}
		if entry.DestinationContent != nil && entry.PreparedTemplateContent != nil &&
			*entry.DestinationContent == *entry.PreparedTemplateContent {
			continue
		}
		if entry.DestinationPath != nil {
			pendingPaths = append(pendingPaths, *entry.DestinationPath)
		} else {
			pendingPaths = append(pendingPaths, entry.LogicalDestinationPath)
		}
	}

	return TemplateConvergenceResult{
		Converged:    len(pendingPaths) == 0,
		PendingPaths: pendingPaths,
	}
}

func RunTemplateTreeExecution(
	templateSourcePaths []string,
	templateContents map[string]string,
	destinationContents map[string]string,
	context *TemplateDestinationContext,
	defaultStrategy TemplateStrategy,
	overrides []TemplateStrategyOverride,
	replacements map[string]string,
	mergePreparedContent func(TemplateExecutionPlanEntry) MergeResult[string],
	config *TemplateTokenConfig,
) TemplateTreeRunResult {
	existingDestinationPaths := mapsKeys(destinationContents)
	slices.Sort(existingDestinationPaths)
	executionPlan := PlanTemplateTreeExecution(
		templateSourcePaths,
		templateContents,
		existingDestinationPaths,
		destinationContents,
		context,
		defaultStrategy,
		overrides,
		replacements,
		config,
	)

	return TemplateTreeRunResult{
		ExecutionPlan: executionPlan,
		ApplyResult:   ApplyTemplateExecution(executionPlan, mergePreparedContent),
	}
}

func ReadRelativeFileTree(root string) (map[string]string, error) {
	files := map[string]string{}
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return files, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, &os.PathError{Op: "read", Path: root, Err: os.ErrInvalid}
	}

	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
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
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}

func WriteRelativeFileTree(root string, files map[string]string) error {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}

	paths := mapsKeys(files)
	slices.Sort(paths)
	for _, relativePath := range paths {
		fullPath := filepath.Join(root, filepath.FromSlash(relativePath))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(fullPath, []byte(files[relativePath]), 0o644); err != nil {
			return err
		}
	}

	return nil
}

func RunTemplateTreeExecutionFromDirectories(
	templateRoot string,
	destinationRoot string,
	context *TemplateDestinationContext,
	defaultStrategy TemplateStrategy,
	overrides []TemplateStrategyOverride,
	replacements map[string]string,
	mergePreparedContent func(TemplateExecutionPlanEntry) MergeResult[string],
	config *TemplateTokenConfig,
) (TemplateTreeRunResult, error) {
	templateContents, err := ReadRelativeFileTree(templateRoot)
	if err != nil {
		return TemplateTreeRunResult{}, err
	}
	destinationContents, err := ReadRelativeFileTree(destinationRoot)
	if err != nil {
		return TemplateTreeRunResult{}, err
	}
	templateSourcePaths := mapsKeys(templateContents)
	slices.Sort(templateSourcePaths)

	return RunTemplateTreeExecution(
		templateSourcePaths,
		templateContents,
		destinationContents,
		context,
		defaultStrategy,
		overrides,
		replacements,
		mergePreparedContent,
		config,
	), nil
}

func PlanTemplateTreeExecutionFromDirectories(
	templateRoot string,
	destinationRoot string,
	context *TemplateDestinationContext,
	defaultStrategy TemplateStrategy,
	overrides []TemplateStrategyOverride,
	replacements map[string]string,
	config *TemplateTokenConfig,
) ([]TemplateExecutionPlanEntry, error) {
	templateContents, err := ReadRelativeFileTree(templateRoot)
	if err != nil {
		return nil, err
	}
	destinationContents, err := ReadRelativeFileTree(destinationRoot)
	if err != nil {
		return nil, err
	}
	templateSourcePaths := mapsKeys(templateContents)
	slices.Sort(templateSourcePaths)

	return PlanTemplateTreeExecution(
		templateSourcePaths,
		templateContents,
		mapsKeysSorted(destinationContents),
		destinationContents,
		context,
		defaultStrategy,
		overrides,
		replacements,
		config,
	), nil
}

func ApplyTemplateTreeExecutionToDirectory(
	templateRoot string,
	destinationRoot string,
	context *TemplateDestinationContext,
	defaultStrategy TemplateStrategy,
	overrides []TemplateStrategyOverride,
	replacements map[string]string,
	mergePreparedContent func(TemplateExecutionPlanEntry) MergeResult[string],
	config *TemplateTokenConfig,
) (TemplateTreeRunResult, error) {
	runResult, err := RunTemplateTreeExecutionFromDirectories(
		templateRoot,
		destinationRoot,
		context,
		defaultStrategy,
		overrides,
		replacements,
		mergePreparedContent,
		config,
	)
	if err != nil {
		return TemplateTreeRunResult{}, err
	}

	filesToWrite := map[string]string{}
	for _, path := range runResult.ApplyResult.CreatedPaths {
		filesToWrite[path] = runResult.ApplyResult.ResultFiles[path]
	}
	for _, path := range runResult.ApplyResult.UpdatedPaths {
		filesToWrite[path] = runResult.ApplyResult.ResultFiles[path]
	}
	if err := WriteRelativeFileTree(destinationRoot, filesToWrite); err != nil {
		return TemplateTreeRunResult{}, err
	}

	return runResult, nil
}

func ReportTemplateTreeRun(result TemplateTreeRunResult) TemplateTreeRunReport {
	created := make(map[string]struct{}, len(result.ApplyResult.CreatedPaths))
	for _, path := range result.ApplyResult.CreatedPaths {
		created[path] = struct{}{}
	}
	updated := make(map[string]struct{}, len(result.ApplyResult.UpdatedPaths))
	for _, path := range result.ApplyResult.UpdatedPaths {
		updated[path] = struct{}{}
	}
	kept := make(map[string]struct{}, len(result.ApplyResult.KeptPaths))
	for _, path := range result.ApplyResult.KeptPaths {
		kept[path] = struct{}{}
	}
	blocked := make(map[string]struct{}, len(result.ApplyResult.BlockedPaths))
	for _, path := range result.ApplyResult.BlockedPaths {
		blocked[path] = struct{}{}
	}
	omitted := make(map[string]struct{}, len(result.ApplyResult.OmittedPaths))
	for _, path := range result.ApplyResult.OmittedPaths {
		omitted[path] = struct{}{}
	}

	entries := make([]TemplateTreeRunReportEntry, 0, len(result.ExecutionPlan))
	summary := TemplateTreeRunReportSummary{}
	for _, entry := range result.ExecutionPlan {
		status := TemplateTreeRunCreated
		switch {
		case entry.ExecutionAction == TemplateExecutionOmit:
			status = TemplateTreeRunOmitted
		case entry.DestinationPath != nil && containsKey(blocked, *entry.DestinationPath):
			status = TemplateTreeRunBlocked
		case entry.DestinationPath != nil && containsKey(kept, *entry.DestinationPath):
			status = TemplateTreeRunKept
		case entry.DestinationPath != nil && containsKey(updated, *entry.DestinationPath):
			status = TemplateTreeRunUpdated
		case containsKey(omitted, entry.LogicalDestinationPath):
			status = TemplateTreeRunOmitted
		}

		switch status {
		case TemplateTreeRunCreated:
			summary.Created++
		case TemplateTreeRunUpdated:
			summary.Updated++
		case TemplateTreeRunKept:
			summary.Kept++
		case TemplateTreeRunBlocked:
			summary.Blocked++
		case TemplateTreeRunOmitted:
			summary.Omitted++
		}

		entries = append(entries, TemplateTreeRunReportEntry{
			TemplateSourcePath:     entry.TemplateSourcePath,
			LogicalDestinationPath: entry.LogicalDestinationPath,
			DestinationPath:        entry.DestinationPath,
			ExecutionAction:        entry.ExecutionAction,
			Status:                 status,
		})
	}

	return TemplateTreeRunReport{
		Entries: entries,
		Summary: summary,
	}
}

func ReportTemplateDirectoryApply(result TemplateTreeRunResult) TemplateDirectoryApplyReport {
	runReport := ReportTemplateTreeRun(result)
	created := make(map[string]struct{}, len(result.ApplyResult.CreatedPaths))
	for _, path := range result.ApplyResult.CreatedPaths {
		created[path] = struct{}{}
	}
	updated := make(map[string]struct{}, len(result.ApplyResult.UpdatedPaths))
	for _, path := range result.ApplyResult.UpdatedPaths {
		updated[path] = struct{}{}
	}

	entries := make([]TemplateDirectoryApplyReportEntry, 0, len(runReport.Entries))
	summary := TemplateDirectoryApplyReportSummary{}
	for _, entry := range runReport.Entries {
		written := false
		if entry.DestinationPath != nil {
			_, written = created[*entry.DestinationPath]
			if !written {
				_, written = updated[*entry.DestinationPath]
			}
		}
		if written {
			summary.Written++
		}

		switch entry.Status {
		case TemplateTreeRunCreated:
			summary.Created++
		case TemplateTreeRunUpdated:
			summary.Updated++
		case TemplateTreeRunKept:
			summary.Kept++
		case TemplateTreeRunBlocked:
			summary.Blocked++
		case TemplateTreeRunOmitted:
			summary.Omitted++
		}

		entries = append(entries, TemplateDirectoryApplyReportEntry{
			TemplateSourcePath:     entry.TemplateSourcePath,
			LogicalDestinationPath: entry.LogicalDestinationPath,
			DestinationPath:        entry.DestinationPath,
			ExecutionAction:        entry.ExecutionAction,
			Status:                 entry.Status,
			Written:                written,
		})
	}

	return TemplateDirectoryApplyReport{
		Entries: entries,
		Summary: summary,
	}
}

func ReportTemplateDirectoryPlan(entries []TemplateExecutionPlanEntry) TemplateDirectoryPlanReport {
	reportEntries := make([]TemplateDirectoryPlanReportEntry, 0, len(entries))
	summary := TemplateDirectoryPlanReportSummary{}

	for _, entry := range entries {
		status := TemplateDirectoryPlanUpdate
		previewable := false
		switch entry.ExecutionAction {
		case TemplateExecutionBlocked:
			status = TemplateDirectoryPlanBlocked
		case TemplateExecutionOmit:
			status = TemplateDirectoryPlanOmitted
			previewable = true
		case TemplateExecutionKeep:
			status = TemplateDirectoryPlanKeep
			previewable = true
		case TemplateExecutionRawCopy, TemplateExecutionWritePrepared:
			if entry.WriteAction == "create" {
				status = TemplateDirectoryPlanCreate
			}
			previewable = true
		case TemplateExecutionMergePrepared:
			if entry.WriteAction == "create" {
				status = TemplateDirectoryPlanCreate
				previewable = true
			}
		}

		switch status {
		case TemplateDirectoryPlanCreate:
			summary.Create++
		case TemplateDirectoryPlanUpdate:
			summary.Update++
		case TemplateDirectoryPlanKeep:
			summary.Keep++
		case TemplateDirectoryPlanBlocked:
			summary.Blocked++
		case TemplateDirectoryPlanOmitted:
			summary.Omitted++
		}

		reportEntries = append(reportEntries, TemplateDirectoryPlanReportEntry{
			TemplateSourcePath:     entry.TemplateSourcePath,
			LogicalDestinationPath: entry.LogicalDestinationPath,
			DestinationPath:        entry.DestinationPath,
			ExecutionAction:        entry.ExecutionAction,
			WriteAction:            entry.WriteAction,
			Status:                 status,
			Previewable:            previewable,
		})
	}

	return TemplateDirectoryPlanReport{Entries: reportEntries, Summary: summary}
}

func ReportTemplateDirectoryRunner(entries []TemplateExecutionPlanEntry, result *TemplateTreeRunResult) TemplateDirectoryRunnerReport {
	preview := PreviewTemplateExecution(entries)
	report := TemplateDirectoryRunnerReport{
		PlanReport: ReportTemplateDirectoryPlan(entries),
		Preview:    &preview,
	}
	if result != nil {
		runReport := ReportTemplateTreeRun(*result)
		applyReport := ReportTemplateDirectoryApply(*result)
		report.RunReport = &runReport
		report.ApplyReport = &applyReport
	}
	return report
}

func pathBase(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx == -1 {
		return path
	}

	return path[idx+1:]
}

func mapsKeysSorted[V any](input map[string]V) []string {
	keys := mapsKeys(input)
	slices.Sort(keys)
	return keys
}

func recordTemplateApplyOutput(result *TemplateApplyResult, entry TemplateExecutionPlanEntry, output string) {
	if entry.DestinationPath == nil {
		return
	}

	result.ResultFiles[*entry.DestinationPath] = output
	if entry.DestinationExists && entry.DestinationContent != nil && *entry.DestinationContent == output {
		result.KeptPaths = append(result.KeptPaths, *entry.DestinationPath)
		return
	}
	if entry.DestinationExists {
		result.UpdatedPaths = append(result.UpdatedPaths, *entry.DestinationPath)
		return
	}

	result.CreatedPaths = append(result.CreatedPaths, *entry.DestinationPath)
}

func containsKey[V any](input map[string]V, key string) bool {
	_, ok := input[key]
	return ok
}

func mapsKeys[V any](input map[string]V) []string {
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}

	return keys
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

func ReviewedNestedExecutionFor(
	family string,
	reviewState DelegatedChildGroupReviewState,
	appliedChildren []AppliedDelegatedChildOutput,
) ReviewedNestedExecution {
	return ReviewedNestedExecution{
		Family:          family,
		ReviewState:     reviewState,
		AppliedChildren: appliedChildren,
	}
}

func ExecuteReviewedNestedExecution[T any](
	execution ReviewedNestedExecution,
	callbacks NestedMergeExecutionCallbacks[T],
) MergeResult[T] {
	return ExecuteReviewedNestedMerge(
		execution.ReviewState,
		execution.Family,
		execution.AppliedChildren,
		callbacks,
	)
}

func ExecuteReviewedNestedExecutions[T any](
	executions []ReviewedNestedExecution,
	callbacksForExecution func(ReviewedNestedExecution, int) NestedMergeExecutionCallbacks[T],
) []ReviewedNestedExecutionResult[T] {
	results := make([]ReviewedNestedExecutionResult[T], 0, len(executions))
	for idx, execution := range executions {
		results = append(results, ReviewedNestedExecutionResult[T]{
			Execution: execution,
			Result:    ExecuteReviewedNestedExecution(execution, callbacksForExecution(execution, idx)),
		})
	}
	return results
}

func ExecuteReviewReplayBundleReviewedNestedExecutions[T any](
	bundle ReviewReplayBundle,
	callbacksForExecution func(ReviewedNestedExecution, int) NestedMergeExecutionCallbacks[T],
) []ReviewedNestedExecutionResult[T] {
	return ExecuteReviewedNestedExecutions(bundle.ReviewedNestedExecutions, callbacksForExecution)
}

func ExecuteReviewReplayBundleEnvelopeReviewedNestedExecutions[T any](
	envelope ReviewReplayBundleEnvelope,
	callbacksForExecution func(ReviewedNestedExecution, int) NestedMergeExecutionCallbacks[T],
) ReviewedNestedExecutionApplication[T] {
	bundle, importErr := ImportReviewReplayBundleEnvelope(envelope)
	if importErr != nil {
		return ReviewedNestedExecutionApplication[T]{
			Diagnostics: []Diagnostic{{
				Severity: SeverityError,
				Category: DiagnosticCategory(importErr.Category),
				Message:  importErr.Message,
			}},
			Results: []ReviewedNestedExecutionResult[T]{},
		}
	}

	return ReviewedNestedExecutionApplication[T]{
		Diagnostics: []Diagnostic{},
		Results:     ExecuteReviewReplayBundleReviewedNestedExecutions(*bundle, callbacksForExecution),
	}
}

func ExecuteReviewStateReviewedNestedExecutions[T any](
	state ConformanceManifestReviewState,
	callbacksForExecution func(ReviewedNestedExecution, int) NestedMergeExecutionCallbacks[T],
) []ReviewedNestedExecutionResult[T] {
	return ExecuteReviewedNestedExecutions(state.ReviewedNestedExecutions, callbacksForExecution)
}

func ExecuteReviewStateEnvelopeReviewedNestedExecutions[T any](
	envelope ConformanceManifestReviewStateEnvelope,
	callbacksForExecution func(ReviewedNestedExecution, int) NestedMergeExecutionCallbacks[T],
) ReviewedNestedExecutionApplication[T] {
	state, importErr := ImportConformanceManifestReviewStateEnvelope(envelope)
	if importErr != nil {
		return ReviewedNestedExecutionApplication[T]{
			Diagnostics: []Diagnostic{{
				Severity: SeverityError,
				Category: DiagnosticCategory(importErr.Category),
				Message:  importErr.Message,
			}},
			Results: []ReviewedNestedExecutionResult[T]{},
		}
	}

	return ReviewedNestedExecutionApplication[T]{
		Diagnostics: []Diagnostic{},
		Results:     ExecuteReviewStateReviewedNestedExecutions(*state, callbacksForExecution),
	}
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
) (*ReviewReplayContext, []ReviewDecision, []ReviewedNestedExecution) {
	if options.ReviewReplayBundle != nil {
		return &options.ReviewReplayBundle.ReplayContext, options.ReviewReplayBundle.Decisions, options.ReviewReplayBundle.ReviewedNestedExecutions
	}

	return options.ReviewReplayContext, options.ReviewDecisions, nil
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

func StructuredEditApplicationEnvelopeFor(
	application StructuredEditApplication,
) StructuredEditApplicationEnvelope {
	return StructuredEditApplicationEnvelope{
		Kind:        "structured_edit_application",
		Version:     StructuredEditTransportVersion,
		Application: application,
	}
}

func ImportStructuredEditApplicationEnvelope(
	envelope StructuredEditApplicationEnvelope,
) (*StructuredEditApplication, *StructuredEditTransportImportError) {
	if envelope.Kind != "structured_edit_application" {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportKindMismatch,
			Message:  "expected structured_edit_application envelope kind.",
		}
	}

	if envelope.Version != StructuredEditTransportVersion {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportUnsupportedVersion,
			Message:  "unsupported structured_edit_application envelope version " + strconv.Itoa(envelope.Version) + ".",
		}
	}

	application := envelope.Application
	return &application, nil
}

func StructuredEditProviderExecutionRequestEnvelopeFor(
	executionRequest StructuredEditProviderExecutionRequest,
) StructuredEditProviderExecutionRequestEnvelope {
	return StructuredEditProviderExecutionRequestEnvelope{
		Kind:             "structured_edit_provider_execution_request",
		Version:          StructuredEditTransportVersion,
		ExecutionRequest: executionRequest,
	}
}

func ImportStructuredEditProviderExecutionRequestEnvelope(
	envelope StructuredEditProviderExecutionRequestEnvelope,
) (*StructuredEditProviderExecutionRequest, *StructuredEditTransportImportError) {
	if envelope.Kind != "structured_edit_provider_execution_request" {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportKindMismatch,
			Message:  "expected structured_edit_provider_execution_request envelope kind.",
		}
	}

	if envelope.Version != StructuredEditTransportVersion {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportUnsupportedVersion,
			Message:  "unsupported structured_edit_provider_execution_request envelope version " + strconv.Itoa(envelope.Version) + ".",
		}
	}

	executionRequest := envelope.ExecutionRequest
	return &executionRequest, nil
}

func StructuredEditProviderExecutionPlanEnvelopeFor(
	executionPlan StructuredEditProviderExecutionPlan,
) StructuredEditProviderExecutionPlanEnvelope {
	return StructuredEditProviderExecutionPlanEnvelope{
		Kind:          "structured_edit_provider_execution_plan",
		Version:       StructuredEditTransportVersion,
		ExecutionPlan: executionPlan,
	}
}

func ImportStructuredEditProviderExecutionPlanEnvelope(
	envelope StructuredEditProviderExecutionPlanEnvelope,
) (*StructuredEditProviderExecutionPlan, *StructuredEditTransportImportError) {
	if envelope.Kind != "structured_edit_provider_execution_plan" {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportKindMismatch,
			Message:  "expected structured_edit_provider_execution_plan envelope kind.",
		}
	}

	if envelope.Version != StructuredEditTransportVersion {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportUnsupportedVersion,
			Message:  "unsupported structured_edit_provider_execution_plan envelope version " + strconv.Itoa(envelope.Version) + ".",
		}
	}

	executionPlan := envelope.ExecutionPlan
	return &executionPlan, nil
}

func StructuredEditProviderExecutionHandoffEnvelopeFor(
	executionHandoff StructuredEditProviderExecutionHandoff,
) StructuredEditProviderExecutionHandoffEnvelope {
	return StructuredEditProviderExecutionHandoffEnvelope{
		Kind:             "structured_edit_provider_execution_handoff",
		Version:          StructuredEditTransportVersion,
		ExecutionHandoff: executionHandoff,
	}
}

func ImportStructuredEditProviderExecutionHandoffEnvelope(
	envelope StructuredEditProviderExecutionHandoffEnvelope,
) (*StructuredEditProviderExecutionHandoff, *StructuredEditTransportImportError) {
	if envelope.Kind != "structured_edit_provider_execution_handoff" {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportKindMismatch,
			Message:  "expected structured_edit_provider_execution_handoff envelope kind.",
		}
	}

	if envelope.Version != StructuredEditTransportVersion {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportUnsupportedVersion,
			Message:  "unsupported structured_edit_provider_execution_handoff envelope version " + strconv.Itoa(envelope.Version) + ".",
		}
	}

	executionHandoff := envelope.ExecutionHandoff
	return &executionHandoff, nil
}

func StructuredEditProviderExecutionInvocationEnvelopeFor(
	executionInvocation StructuredEditProviderExecutionInvocation,
) StructuredEditProviderExecutionInvocationEnvelope {
	return StructuredEditProviderExecutionInvocationEnvelope{
		Kind:                "structured_edit_provider_execution_invocation",
		Version:             StructuredEditTransportVersion,
		ExecutionInvocation: executionInvocation,
	}
}

func ImportStructuredEditProviderExecutionInvocationEnvelope(
	envelope StructuredEditProviderExecutionInvocationEnvelope,
) (*StructuredEditProviderExecutionInvocation, *StructuredEditTransportImportError) {
	if envelope.Kind != "structured_edit_provider_execution_invocation" {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportKindMismatch,
			Message:  "expected structured_edit_provider_execution_invocation envelope kind.",
		}
	}

	if envelope.Version != StructuredEditTransportVersion {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportUnsupportedVersion,
			Message:  "unsupported structured_edit_provider_execution_invocation envelope version " + strconv.Itoa(envelope.Version) + ".",
		}
	}

	executionInvocation := envelope.ExecutionInvocation
	return &executionInvocation, nil
}

func StructuredEditProviderBatchExecutionInvocationEnvelopeFor(
	batchExecutionInvocation StructuredEditProviderBatchExecutionInvocation,
) StructuredEditProviderBatchExecutionInvocationEnvelope {
	return StructuredEditProviderBatchExecutionInvocationEnvelope{
		Kind:                     "structured_edit_provider_batch_execution_invocation",
		Version:                  StructuredEditTransportVersion,
		BatchExecutionInvocation: batchExecutionInvocation,
	}
}

func ImportStructuredEditProviderBatchExecutionInvocationEnvelope(
	envelope StructuredEditProviderBatchExecutionInvocationEnvelope,
) (*StructuredEditProviderBatchExecutionInvocation, *StructuredEditTransportImportError) {
	if envelope.Kind != "structured_edit_provider_batch_execution_invocation" {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportKindMismatch,
			Message:  "expected structured_edit_provider_batch_execution_invocation envelope kind.",
		}
	}

	if envelope.Version != StructuredEditTransportVersion {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportUnsupportedVersion,
			Message:  "unsupported structured_edit_provider_batch_execution_invocation envelope version " + strconv.Itoa(envelope.Version) + ".",
		}
	}

	batchExecutionInvocation := envelope.BatchExecutionInvocation
	return &batchExecutionInvocation, nil
}

func StructuredEditProviderBatchExecutionHandoffEnvelopeFor(
	batchExecutionHandoff StructuredEditProviderBatchExecutionHandoff,
) StructuredEditProviderBatchExecutionHandoffEnvelope {
	return StructuredEditProviderBatchExecutionHandoffEnvelope{
		Kind:                  "structured_edit_provider_batch_execution_handoff",
		Version:               StructuredEditTransportVersion,
		BatchExecutionHandoff: batchExecutionHandoff,
	}
}

func ImportStructuredEditProviderBatchExecutionHandoffEnvelope(
	envelope StructuredEditProviderBatchExecutionHandoffEnvelope,
) (*StructuredEditProviderBatchExecutionHandoff, *StructuredEditTransportImportError) {
	if envelope.Kind != "structured_edit_provider_batch_execution_handoff" {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportKindMismatch,
			Message:  "expected structured_edit_provider_batch_execution_handoff envelope kind.",
		}
	}

	if envelope.Version != StructuredEditTransportVersion {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportUnsupportedVersion,
			Message:  "unsupported structured_edit_provider_batch_execution_handoff envelope version " + strconv.Itoa(envelope.Version) + ".",
		}
	}

	batchExecutionHandoff := envelope.BatchExecutionHandoff
	return &batchExecutionHandoff, nil
}

func StructuredEditProviderBatchExecutionPlanEnvelopeFor(
	batchExecutionPlan StructuredEditProviderBatchExecutionPlan,
) StructuredEditProviderBatchExecutionPlanEnvelope {
	return StructuredEditProviderBatchExecutionPlanEnvelope{
		Kind:               "structured_edit_provider_batch_execution_plan",
		Version:            StructuredEditTransportVersion,
		BatchExecutionPlan: batchExecutionPlan,
	}
}

func ImportStructuredEditProviderBatchExecutionPlanEnvelope(
	envelope StructuredEditProviderBatchExecutionPlanEnvelope,
) (*StructuredEditProviderBatchExecutionPlan, *StructuredEditTransportImportError) {
	if envelope.Kind != "structured_edit_provider_batch_execution_plan" {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportKindMismatch,
			Message:  "expected structured_edit_provider_batch_execution_plan envelope kind.",
		}
	}

	if envelope.Version != StructuredEditTransportVersion {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportUnsupportedVersion,
			Message:  "unsupported structured_edit_provider_batch_execution_plan envelope version " + strconv.Itoa(envelope.Version) + ".",
		}
	}

	batchExecutionPlan := envelope.BatchExecutionPlan
	return &batchExecutionPlan, nil
}

func StructuredEditProviderExecutionApplicationEnvelopeFor(
	application StructuredEditProviderExecutionApplication,
) StructuredEditProviderExecutionApplicationEnvelope {
	return StructuredEditProviderExecutionApplicationEnvelope{
		Kind:                         "structured_edit_provider_execution_application",
		Version:                      StructuredEditTransportVersion,
		ProviderExecutionApplication: application,
	}
}

func ImportStructuredEditProviderExecutionApplicationEnvelope(
	envelope StructuredEditProviderExecutionApplicationEnvelope,
) (*StructuredEditProviderExecutionApplication, *StructuredEditTransportImportError) {
	if envelope.Kind != "structured_edit_provider_execution_application" {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportKindMismatch,
			Message:  "expected structured_edit_provider_execution_application envelope kind.",
		}
	}

	if envelope.Version != StructuredEditTransportVersion {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportUnsupportedVersion,
			Message:  "unsupported structured_edit_provider_execution_application envelope version " + strconv.Itoa(envelope.Version) + ".",
		}
	}

	application := envelope.ProviderExecutionApplication
	return &application, nil
}

func StructuredEditProviderExecutionDispatchEnvelopeFor(
	dispatch StructuredEditProviderExecutionDispatch,
) StructuredEditProviderExecutionDispatchEnvelope {
	return StructuredEditProviderExecutionDispatchEnvelope{
		Kind:                      "structured_edit_provider_execution_dispatch",
		Version:                   StructuredEditTransportVersion,
		ProviderExecutionDispatch: dispatch,
	}
}

func ImportStructuredEditProviderExecutionDispatchEnvelope(
	envelope StructuredEditProviderExecutionDispatchEnvelope,
) (*StructuredEditProviderExecutionDispatch, *StructuredEditTransportImportError) {
	if envelope.Kind != "structured_edit_provider_execution_dispatch" {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportKindMismatch,
			Message:  "expected structured_edit_provider_execution_dispatch envelope kind.",
		}
	}

	if envelope.Version != StructuredEditTransportVersion {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportUnsupportedVersion,
			Message:  "unsupported structured_edit_provider_execution_dispatch envelope version " + strconv.Itoa(envelope.Version) + ".",
		}
	}

	dispatch := envelope.ProviderExecutionDispatch
	return &dispatch, nil
}

func StructuredEditProviderExecutionOutcomeEnvelopeFor(
	outcome StructuredEditProviderExecutionOutcome,
) StructuredEditProviderExecutionOutcomeEnvelope {
	return StructuredEditProviderExecutionOutcomeEnvelope{
		Kind:                     "structured_edit_provider_execution_outcome",
		Version:                  StructuredEditTransportVersion,
		ProviderExecutionOutcome: outcome,
	}
}

func ImportStructuredEditProviderExecutionOutcomeEnvelope(
	envelope StructuredEditProviderExecutionOutcomeEnvelope,
) (*StructuredEditProviderExecutionOutcome, *StructuredEditTransportImportError) {
	if envelope.Kind != "structured_edit_provider_execution_outcome" {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportKindMismatch,
			Message:  "expected structured_edit_provider_execution_outcome envelope kind.",
		}
	}

	if envelope.Version != StructuredEditTransportVersion {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportUnsupportedVersion,
			Message:  "unsupported structured_edit_provider_execution_outcome envelope version " + strconv.Itoa(envelope.Version) + ".",
		}
	}

	outcome := envelope.ProviderExecutionOutcome
	return &outcome, nil
}

func StructuredEditProviderExecutionRunResultEnvelopeFor(
	executionRunResult StructuredEditProviderExecutionRunResult,
) StructuredEditProviderExecutionRunResultEnvelope {
	return StructuredEditProviderExecutionRunResultEnvelope{
		Kind:               "structured_edit_provider_execution_run_result",
		Version:            StructuredEditTransportVersion,
		ExecutionRunResult: executionRunResult,
	}
}

func ImportStructuredEditProviderExecutionRunResultEnvelope(
	envelope StructuredEditProviderExecutionRunResultEnvelope,
) (*StructuredEditProviderExecutionRunResult, *StructuredEditTransportImportError) {
	if envelope.Kind != "structured_edit_provider_execution_run_result" {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportKindMismatch,
			Message:  "expected structured_edit_provider_execution_run_result envelope kind.",
		}
	}

	if envelope.Version != StructuredEditTransportVersion {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportUnsupportedVersion,
			Message:  "unsupported structured_edit_provider_execution_run_result envelope version " + strconv.Itoa(envelope.Version) + ".",
		}
	}

	executionRunResult := envelope.ExecutionRunResult
	return &executionRunResult, nil
}

func StructuredEditProviderBatchExecutionRunResultEnvelopeFor(
	batchExecutionRunResult StructuredEditProviderBatchExecutionRunResult,
) StructuredEditProviderBatchExecutionRunResultEnvelope {
	return StructuredEditProviderBatchExecutionRunResultEnvelope{
		Kind:                    "structured_edit_provider_batch_execution_run_result",
		Version:                 StructuredEditTransportVersion,
		BatchExecutionRunResult: batchExecutionRunResult,
	}
}

func ImportStructuredEditProviderBatchExecutionRunResultEnvelope(
	envelope StructuredEditProviderBatchExecutionRunResultEnvelope,
) (*StructuredEditProviderBatchExecutionRunResult, *StructuredEditTransportImportError) {
	if envelope.Kind != "structured_edit_provider_batch_execution_run_result" {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportKindMismatch,
			Message:  "expected structured_edit_provider_batch_execution_run_result envelope kind.",
		}
	}

	if envelope.Version != StructuredEditTransportVersion {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportUnsupportedVersion,
			Message:  "unsupported structured_edit_provider_batch_execution_run_result envelope version " + strconv.Itoa(envelope.Version) + ".",
		}
	}

	batchExecutionRunResult := envelope.BatchExecutionRunResult
	return &batchExecutionRunResult, nil
}

func StructuredEditProviderExecutionReceiptEnvelopeFor(
	executionReceipt StructuredEditProviderExecutionReceipt,
) StructuredEditProviderExecutionReceiptEnvelope {
	return StructuredEditProviderExecutionReceiptEnvelope{
		Kind:             "structured_edit_provider_execution_receipt",
		Version:          StructuredEditTransportVersion,
		ExecutionReceipt: executionReceipt,
	}
}

func ImportStructuredEditProviderExecutionReceiptEnvelope(
	envelope StructuredEditProviderExecutionReceiptEnvelope,
) (*StructuredEditProviderExecutionReceipt, *StructuredEditTransportImportError) {
	if envelope.Kind != "structured_edit_provider_execution_receipt" {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportKindMismatch,
			Message:  "expected structured_edit_provider_execution_receipt envelope kind.",
		}
	}

	if envelope.Version != StructuredEditTransportVersion {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportUnsupportedVersion,
			Message:  "unsupported structured_edit_provider_execution_receipt envelope version " + strconv.Itoa(envelope.Version) + ".",
		}
	}

	executionReceipt := envelope.ExecutionReceipt
	return &executionReceipt, nil
}

func StructuredEditProviderBatchExecutionReceiptEnvelopeFor(
	batchExecutionReceipt StructuredEditProviderBatchExecutionReceipt,
) StructuredEditProviderBatchExecutionReceiptEnvelope {
	return StructuredEditProviderBatchExecutionReceiptEnvelope{
		Kind:                  "structured_edit_provider_batch_execution_receipt",
		Version:               StructuredEditTransportVersion,
		BatchExecutionReceipt: batchExecutionReceipt,
	}
}

func ImportStructuredEditProviderBatchExecutionReceiptEnvelope(
	envelope StructuredEditProviderBatchExecutionReceiptEnvelope,
) (*StructuredEditProviderBatchExecutionReceipt, *StructuredEditTransportImportError) {
	if envelope.Kind != "structured_edit_provider_batch_execution_receipt" {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportKindMismatch,
			Message:  "expected structured_edit_provider_batch_execution_receipt envelope kind.",
		}
	}

	if envelope.Version != StructuredEditTransportVersion {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportUnsupportedVersion,
			Message:  "unsupported structured_edit_provider_batch_execution_receipt envelope version " + strconv.Itoa(envelope.Version) + ".",
		}
	}

	batchExecutionReceipt := envelope.BatchExecutionReceipt
	return &batchExecutionReceipt, nil
}

func StructuredEditProviderExecutionReceiptReplayRequestEnvelopeFor(
	receiptReplayRequest StructuredEditProviderExecutionReceiptReplayRequest,
) StructuredEditProviderExecutionReceiptReplayRequestEnvelope {
	return StructuredEditProviderExecutionReceiptReplayRequestEnvelope{
		Kind:                 "structured_edit_provider_execution_receipt_replay_request",
		Version:              StructuredEditTransportVersion,
		ReceiptReplayRequest: receiptReplayRequest,
	}
}

func ImportStructuredEditProviderExecutionReceiptReplayRequestEnvelope(
	envelope StructuredEditProviderExecutionReceiptReplayRequestEnvelope,
) (*StructuredEditProviderExecutionReceiptReplayRequest, *StructuredEditTransportImportError) {
	if envelope.Kind != "structured_edit_provider_execution_receipt_replay_request" {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportKindMismatch,
			Message:  "expected structured_edit_provider_execution_receipt_replay_request envelope kind.",
		}
	}

	if envelope.Version != StructuredEditTransportVersion {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportUnsupportedVersion,
			Message:  "unsupported structured_edit_provider_execution_receipt_replay_request envelope version " + strconv.Itoa(envelope.Version) + ".",
		}
	}

	receiptReplayRequest := envelope.ReceiptReplayRequest
	return &receiptReplayRequest, nil
}

func StructuredEditProviderBatchExecutionReceiptReplayRequestEnvelopeFor(
	batchReceiptReplayRequest StructuredEditProviderBatchExecutionReceiptReplayRequest,
) StructuredEditProviderBatchExecutionReceiptReplayRequestEnvelope {
	return StructuredEditProviderBatchExecutionReceiptReplayRequestEnvelope{
		Kind:                      "structured_edit_provider_batch_execution_receipt_replay_request",
		Version:                   StructuredEditTransportVersion,
		BatchReceiptReplayRequest: batchReceiptReplayRequest,
	}
}

func ImportStructuredEditProviderBatchExecutionReceiptReplayRequestEnvelope(
	envelope StructuredEditProviderBatchExecutionReceiptReplayRequestEnvelope,
) (*StructuredEditProviderBatchExecutionReceiptReplayRequest, *StructuredEditTransportImportError) {
	if envelope.Kind != "structured_edit_provider_batch_execution_receipt_replay_request" {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportKindMismatch,
			Message:  "expected structured_edit_provider_batch_execution_receipt_replay_request envelope kind.",
		}
	}

	if envelope.Version != StructuredEditTransportVersion {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportUnsupportedVersion,
			Message:  "unsupported structured_edit_provider_batch_execution_receipt_replay_request envelope version " + strconv.Itoa(envelope.Version) + ".",
		}
	}

	batchReceiptReplayRequest := envelope.BatchReceiptReplayRequest
	return &batchReceiptReplayRequest, nil
}

func StructuredEditProviderExecutionReceiptReplayApplicationEnvelopeFor(
	receiptReplayApplication StructuredEditProviderExecutionReceiptReplayApplication,
) StructuredEditProviderExecutionReceiptReplayApplicationEnvelope {
	return StructuredEditProviderExecutionReceiptReplayApplicationEnvelope{
		Kind:                     "structured_edit_provider_execution_receipt_replay_application",
		Version:                  StructuredEditTransportVersion,
		ReceiptReplayApplication: receiptReplayApplication,
	}
}

func ImportStructuredEditProviderExecutionReceiptReplayApplicationEnvelope(
	envelope StructuredEditProviderExecutionReceiptReplayApplicationEnvelope,
) (*StructuredEditProviderExecutionReceiptReplayApplication, *StructuredEditTransportImportError) {
	if envelope.Kind != "structured_edit_provider_execution_receipt_replay_application" {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportKindMismatch,
			Message:  "expected structured_edit_provider_execution_receipt_replay_application envelope kind.",
		}
	}

	if envelope.Version != StructuredEditTransportVersion {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportUnsupportedVersion,
			Message:  "unsupported structured_edit_provider_execution_receipt_replay_application envelope version " + strconv.Itoa(envelope.Version) + ".",
		}
	}

	receiptReplayApplication := envelope.ReceiptReplayApplication
	return &receiptReplayApplication, nil
}

func StructuredEditProviderBatchExecutionReceiptReplayApplicationEnvelopeFor(
	batchReceiptReplayApplication StructuredEditProviderBatchExecutionReceiptReplayApplication,
) StructuredEditProviderBatchExecutionReceiptReplayApplicationEnvelope {
	return StructuredEditProviderBatchExecutionReceiptReplayApplicationEnvelope{
		Kind:                          "structured_edit_provider_batch_execution_receipt_replay_application",
		Version:                       StructuredEditTransportVersion,
		BatchReceiptReplayApplication: batchReceiptReplayApplication,
	}
}

func ImportStructuredEditProviderBatchExecutionReceiptReplayApplicationEnvelope(
	envelope StructuredEditProviderBatchExecutionReceiptReplayApplicationEnvelope,
) (*StructuredEditProviderBatchExecutionReceiptReplayApplication, *StructuredEditTransportImportError) {
	if envelope.Kind != "structured_edit_provider_batch_execution_receipt_replay_application" {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportKindMismatch,
			Message:  "expected structured_edit_provider_batch_execution_receipt_replay_application envelope kind.",
		}
	}

	if envelope.Version != StructuredEditTransportVersion {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportUnsupportedVersion,
			Message:  "unsupported structured_edit_provider_batch_execution_receipt_replay_application envelope version " + strconv.Itoa(envelope.Version) + ".",
		}
	}

	batchReceiptReplayApplication := envelope.BatchReceiptReplayApplication
	return &batchReceiptReplayApplication, nil
}

func StructuredEditProviderBatchExecutionOutcomeEnvelopeFor(
	batchOutcome StructuredEditProviderBatchExecutionOutcome,
) StructuredEditProviderBatchExecutionOutcomeEnvelope {
	return StructuredEditProviderBatchExecutionOutcomeEnvelope{
		Kind:         "structured_edit_provider_batch_execution_outcome",
		Version:      StructuredEditTransportVersion,
		BatchOutcome: batchOutcome,
	}
}

func ImportStructuredEditProviderBatchExecutionOutcomeEnvelope(
	envelope StructuredEditProviderBatchExecutionOutcomeEnvelope,
) (*StructuredEditProviderBatchExecutionOutcome, *StructuredEditTransportImportError) {
	if envelope.Kind != "structured_edit_provider_batch_execution_outcome" {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportKindMismatch,
			Message:  "expected structured_edit_provider_batch_execution_outcome envelope kind.",
		}
	}

	if envelope.Version != StructuredEditTransportVersion {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportUnsupportedVersion,
			Message:  "unsupported structured_edit_provider_batch_execution_outcome envelope version " + strconv.Itoa(envelope.Version) + ".",
		}
	}

	batchOutcome := envelope.BatchOutcome
	return &batchOutcome, nil
}

func StructuredEditProviderExecutionProvenanceEnvelopeFor(
	provenance StructuredEditProviderExecutionProvenance,
) StructuredEditProviderExecutionProvenanceEnvelope {
	return StructuredEditProviderExecutionProvenanceEnvelope{
		Kind:       "structured_edit_provider_execution_provenance",
		Version:    StructuredEditTransportVersion,
		Provenance: provenance,
	}
}

func ImportStructuredEditProviderExecutionProvenanceEnvelope(
	envelope StructuredEditProviderExecutionProvenanceEnvelope,
) (*StructuredEditProviderExecutionProvenance, *StructuredEditTransportImportError) {
	if envelope.Kind != "structured_edit_provider_execution_provenance" {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportKindMismatch,
			Message:  "expected structured_edit_provider_execution_provenance envelope kind.",
		}
	}

	if envelope.Version != StructuredEditTransportVersion {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportUnsupportedVersion,
			Message:  "unsupported structured_edit_provider_execution_provenance envelope version " + strconv.Itoa(envelope.Version) + ".",
		}
	}

	provenance := envelope.Provenance
	return &provenance, nil
}

func StructuredEditProviderBatchExecutionProvenanceEnvelopeFor(
	batchProvenance StructuredEditProviderBatchExecutionProvenance,
) StructuredEditProviderBatchExecutionProvenanceEnvelope {
	return StructuredEditProviderBatchExecutionProvenanceEnvelope{
		Kind:            "structured_edit_provider_batch_execution_provenance",
		Version:         StructuredEditTransportVersion,
		BatchProvenance: batchProvenance,
	}
}

func ImportStructuredEditProviderBatchExecutionProvenanceEnvelope(
	envelope StructuredEditProviderBatchExecutionProvenanceEnvelope,
) (*StructuredEditProviderBatchExecutionProvenance, *StructuredEditTransportImportError) {
	if envelope.Kind != "structured_edit_provider_batch_execution_provenance" {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportKindMismatch,
			Message:  "expected structured_edit_provider_batch_execution_provenance envelope kind.",
		}
	}

	if envelope.Version != StructuredEditTransportVersion {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportUnsupportedVersion,
			Message:  "unsupported structured_edit_provider_batch_execution_provenance envelope version " + strconv.Itoa(envelope.Version) + ".",
		}
	}

	batchProvenance := envelope.BatchProvenance
	return &batchProvenance, nil
}

func StructuredEditProviderExecutionReplayBundleEnvelopeFor(
	replayBundle StructuredEditProviderExecutionReplayBundle,
) StructuredEditProviderExecutionReplayBundleEnvelope {
	return StructuredEditProviderExecutionReplayBundleEnvelope{
		Kind:         "structured_edit_provider_execution_replay_bundle",
		Version:      StructuredEditTransportVersion,
		ReplayBundle: replayBundle,
	}
}

func ImportStructuredEditProviderExecutionReplayBundleEnvelope(
	envelope StructuredEditProviderExecutionReplayBundleEnvelope,
) (*StructuredEditProviderExecutionReplayBundle, *StructuredEditTransportImportError) {
	if envelope.Kind != "structured_edit_provider_execution_replay_bundle" {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportKindMismatch,
			Message:  "expected structured_edit_provider_execution_replay_bundle envelope kind.",
		}
	}

	if envelope.Version != StructuredEditTransportVersion {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportUnsupportedVersion,
			Message:  "unsupported structured_edit_provider_execution_replay_bundle envelope version " + strconv.Itoa(envelope.Version) + ".",
		}
	}

	replayBundle := envelope.ReplayBundle
	return &replayBundle, nil
}

func StructuredEditProviderBatchExecutionReplayBundleEnvelopeFor(
	batchReplayBundle StructuredEditProviderBatchExecutionReplayBundle,
) StructuredEditProviderBatchExecutionReplayBundleEnvelope {
	return StructuredEditProviderBatchExecutionReplayBundleEnvelope{
		Kind:              "structured_edit_provider_batch_execution_replay_bundle",
		Version:           StructuredEditTransportVersion,
		BatchReplayBundle: batchReplayBundle,
	}
}

func ImportStructuredEditProviderBatchExecutionReplayBundleEnvelope(
	envelope StructuredEditProviderBatchExecutionReplayBundleEnvelope,
) (*StructuredEditProviderBatchExecutionReplayBundle, *StructuredEditTransportImportError) {
	if envelope.Kind != "structured_edit_provider_batch_execution_replay_bundle" {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportKindMismatch,
			Message:  "expected structured_edit_provider_batch_execution_replay_bundle envelope kind.",
		}
	}

	if envelope.Version != StructuredEditTransportVersion {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportUnsupportedVersion,
			Message:  "unsupported structured_edit_provider_batch_execution_replay_bundle envelope version " + strconv.Itoa(envelope.Version) + ".",
		}
	}

	batchReplayBundle := envelope.BatchReplayBundle
	return &batchReplayBundle, nil
}

func StructuredEditProviderExecutorProfileEnvelopeFor(
	executorProfile StructuredEditProviderExecutorProfile,
) StructuredEditProviderExecutorProfileEnvelope {
	return StructuredEditProviderExecutorProfileEnvelope{
		Kind:            "structured_edit_provider_executor_profile",
		Version:         StructuredEditTransportVersion,
		ExecutorProfile: executorProfile,
	}
}

func ImportStructuredEditProviderExecutorProfileEnvelope(
	envelope StructuredEditProviderExecutorProfileEnvelope,
) (*StructuredEditProviderExecutorProfile, *StructuredEditTransportImportError) {
	if envelope.Kind != "structured_edit_provider_executor_profile" {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportKindMismatch,
			Message:  "expected structured_edit_provider_executor_profile envelope kind.",
		}
	}

	if envelope.Version != StructuredEditTransportVersion {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportUnsupportedVersion,
			Message:  "unsupported structured_edit_provider_executor_profile envelope version " + strconv.Itoa(envelope.Version) + ".",
		}
	}

	executorProfile := envelope.ExecutorProfile
	return &executorProfile, nil
}

func StructuredEditProviderExecutorRegistryEnvelopeFor(
	executorRegistry StructuredEditProviderExecutorRegistry,
) StructuredEditProviderExecutorRegistryEnvelope {
	return StructuredEditProviderExecutorRegistryEnvelope{
		Kind:             "structured_edit_provider_executor_registry",
		Version:          StructuredEditTransportVersion,
		ExecutorRegistry: executorRegistry,
	}
}

func ImportStructuredEditProviderExecutorRegistryEnvelope(
	envelope StructuredEditProviderExecutorRegistryEnvelope,
) (*StructuredEditProviderExecutorRegistry, *StructuredEditTransportImportError) {
	if envelope.Kind != "structured_edit_provider_executor_registry" {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportKindMismatch,
			Message:  "expected structured_edit_provider_executor_registry envelope kind.",
		}
	}

	if envelope.Version != StructuredEditTransportVersion {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportUnsupportedVersion,
			Message:  "unsupported structured_edit_provider_executor_registry envelope version " + strconv.Itoa(envelope.Version) + ".",
		}
	}

	executorRegistry := envelope.ExecutorRegistry
	return &executorRegistry, nil
}

func StructuredEditProviderExecutorSelectionPolicyEnvelopeFor(
	selectionPolicy StructuredEditProviderExecutorSelectionPolicy,
) StructuredEditProviderExecutorSelectionPolicyEnvelope {
	return StructuredEditProviderExecutorSelectionPolicyEnvelope{
		Kind:            "structured_edit_provider_executor_selection_policy",
		Version:         StructuredEditTransportVersion,
		SelectionPolicy: selectionPolicy,
	}
}

func ImportStructuredEditProviderExecutorSelectionPolicyEnvelope(
	envelope StructuredEditProviderExecutorSelectionPolicyEnvelope,
) (*StructuredEditProviderExecutorSelectionPolicy, *StructuredEditTransportImportError) {
	if envelope.Kind != "structured_edit_provider_executor_selection_policy" {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportKindMismatch,
			Message:  "expected structured_edit_provider_executor_selection_policy envelope kind.",
		}
	}

	if envelope.Version != StructuredEditTransportVersion {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportUnsupportedVersion,
			Message:  "unsupported structured_edit_provider_executor_selection_policy envelope version " + strconv.Itoa(envelope.Version) + ".",
		}
	}

	selectionPolicy := envelope.SelectionPolicy
	return &selectionPolicy, nil
}

func StructuredEditProviderExecutorResolutionEnvelopeFor(
	executorResolution StructuredEditProviderExecutorResolution,
) StructuredEditProviderExecutorResolutionEnvelope {
	return StructuredEditProviderExecutorResolutionEnvelope{
		Kind:               "structured_edit_provider_executor_resolution",
		Version:            StructuredEditTransportVersion,
		ExecutorResolution: executorResolution,
	}
}

func ImportStructuredEditProviderExecutorResolutionEnvelope(
	envelope StructuredEditProviderExecutorResolutionEnvelope,
) (*StructuredEditProviderExecutorResolution, *StructuredEditTransportImportError) {
	if envelope.Kind != "structured_edit_provider_executor_resolution" {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportKindMismatch,
			Message:  "expected structured_edit_provider_executor_resolution envelope kind.",
		}
	}

	if envelope.Version != StructuredEditTransportVersion {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportUnsupportedVersion,
			Message:  "unsupported structured_edit_provider_executor_resolution envelope version " + strconv.Itoa(envelope.Version) + ".",
		}
	}

	executorResolution := envelope.ExecutorResolution
	return &executorResolution, nil
}

func StructuredEditExecutionReportEnvelopeFor(
	report StructuredEditExecutionReport,
) StructuredEditExecutionReportEnvelope {
	return StructuredEditExecutionReportEnvelope{
		Kind:    "structured_edit_execution_report",
		Version: StructuredEditTransportVersion,
		Report:  report,
	}
}

func ImportStructuredEditExecutionReportEnvelope(
	envelope StructuredEditExecutionReportEnvelope,
) (*StructuredEditExecutionReport, *StructuredEditTransportImportError) {
	if envelope.Kind != "structured_edit_execution_report" {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportKindMismatch,
			Message:  "expected structured_edit_execution_report envelope kind.",
		}
	}

	if envelope.Version != StructuredEditTransportVersion {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportUnsupportedVersion,
			Message:  "unsupported structured_edit_execution_report envelope version " + strconv.Itoa(envelope.Version) + ".",
		}
	}

	report := envelope.Report
	return &report, nil
}

func StructuredEditProviderBatchExecutionRequestEnvelopeFor(
	batchExecutionRequest StructuredEditProviderBatchExecutionRequest,
) StructuredEditProviderBatchExecutionRequestEnvelope {
	return StructuredEditProviderBatchExecutionRequestEnvelope{
		Kind:                  "structured_edit_provider_batch_execution_request",
		Version:               StructuredEditTransportVersion,
		BatchExecutionRequest: batchExecutionRequest,
	}
}

func ImportStructuredEditProviderBatchExecutionRequestEnvelope(
	envelope StructuredEditProviderBatchExecutionRequestEnvelope,
) (*StructuredEditProviderBatchExecutionRequest, *StructuredEditTransportImportError) {
	if envelope.Kind != "structured_edit_provider_batch_execution_request" {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportKindMismatch,
			Message:  "expected structured_edit_provider_batch_execution_request envelope kind.",
		}
	}

	if envelope.Version != StructuredEditTransportVersion {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportUnsupportedVersion,
			Message:  "unsupported structured_edit_provider_batch_execution_request envelope version " + strconv.Itoa(envelope.Version) + ".",
		}
	}

	batchExecutionRequest := envelope.BatchExecutionRequest
	return &batchExecutionRequest, nil
}

func StructuredEditProviderBatchExecutionDispatchEnvelopeFor(
	batchDispatch StructuredEditProviderBatchExecutionDispatch,
) StructuredEditProviderBatchExecutionDispatchEnvelope {
	return StructuredEditProviderBatchExecutionDispatchEnvelope{
		Kind:          "structured_edit_provider_batch_execution_dispatch",
		Version:       StructuredEditTransportVersion,
		BatchDispatch: batchDispatch,
	}
}

func ImportStructuredEditProviderBatchExecutionDispatchEnvelope(
	envelope StructuredEditProviderBatchExecutionDispatchEnvelope,
) (*StructuredEditProviderBatchExecutionDispatch, *StructuredEditTransportImportError) {
	if envelope.Kind != "structured_edit_provider_batch_execution_dispatch" {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportKindMismatch,
			Message:  "expected structured_edit_provider_batch_execution_dispatch envelope kind.",
		}
	}

	if envelope.Version != StructuredEditTransportVersion {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportUnsupportedVersion,
			Message:  "unsupported structured_edit_provider_batch_execution_dispatch envelope version " + strconv.Itoa(envelope.Version) + ".",
		}
	}

	batchDispatch := envelope.BatchDispatch
	return &batchDispatch, nil
}

func StructuredEditProviderBatchExecutionReportEnvelopeFor(
	batchReport StructuredEditProviderBatchExecutionReport,
) StructuredEditProviderBatchExecutionReportEnvelope {
	return StructuredEditProviderBatchExecutionReportEnvelope{
		Kind:        "structured_edit_provider_batch_execution_report",
		Version:     StructuredEditTransportVersion,
		BatchReport: batchReport,
	}
}

func ImportStructuredEditProviderBatchExecutionReportEnvelope(
	envelope StructuredEditProviderBatchExecutionReportEnvelope,
) (*StructuredEditProviderBatchExecutionReport, *StructuredEditTransportImportError) {
	if envelope.Kind != "structured_edit_provider_batch_execution_report" {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportKindMismatch,
			Message:  "expected structured_edit_provider_batch_execution_report envelope kind.",
		}
	}

	if envelope.Version != StructuredEditTransportVersion {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportUnsupportedVersion,
			Message:  "unsupported structured_edit_provider_batch_execution_report envelope version " + strconv.Itoa(envelope.Version) + ".",
		}
	}

	batchReport := envelope.BatchReport
	return &batchReport, nil
}

func StructuredEditBatchReportEnvelopeFor(
	batchReport StructuredEditBatchReport,
) StructuredEditBatchReportEnvelope {
	return StructuredEditBatchReportEnvelope{
		Kind:        "structured_edit_batch_report",
		Version:     StructuredEditTransportVersion,
		BatchReport: batchReport,
	}
}

func ImportStructuredEditBatchReportEnvelope(
	envelope StructuredEditBatchReportEnvelope,
) (*StructuredEditBatchReport, *StructuredEditTransportImportError) {
	if envelope.Kind != "structured_edit_batch_report" {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportKindMismatch,
			Message:  "expected structured_edit_batch_report envelope kind.",
		}
	}

	if envelope.Version != StructuredEditTransportVersion {
		return nil, &StructuredEditTransportImportError{
			Category: StructuredEditTransportUnsupportedVersion,
			Message:  "unsupported structured_edit_batch_report envelope version " + strconv.Itoa(envelope.Version) + ".",
		}
	}

	batchReport := envelope.BatchReport
	return &batchReport, nil
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
	replayInputContext, replayInputDecisions, reviewedNestedExecutions := ReviewReplayBundleInputs(options)
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
			reviewedNestedExecutions = nil
		} else if !ReviewReplayContextCompatible(replayContext, replayInputContext) {
			diagnostics = append(diagnostics, Diagnostic{
				Severity: SeverityError,
				Category: CategoryReplayRejected,
				Message:  "review replay context does not match the current conformance manifest state.",
			})
			effectiveOptions.ReviewReplayBundle = nil
			effectiveOptions.ReviewReplayContext = nil
			effectiveOptions.ReviewDecisions = nil
			reviewedNestedExecutions = nil
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
		Report:                   ReportNamedConformanceSuiteEnvelope(ReportPlannedNamedConformanceSuites(entries, execute)),
		Diagnostics:              diagnostics,
		Requests:                 requests,
		AppliedDecisions:         appliedDecisions,
		HostHints:                ConformanceReviewHostHints(options),
		ReplayContext:            replayContext,
		ReviewedNestedExecutions: reviewedNestedExecutions,
	}
}

func ReviewConformanceManifestWithReplayBundleEnvelope(
	manifest ConformanceManifest,
	options ConformanceManifestReviewOptions,
	replayBundleEnvelope ReviewReplayBundleEnvelope,
	execute func(ConformanceCaseRun) ConformanceCaseExecution,
) ConformanceManifestReviewState {
	replayBundle, importErr := ImportReviewReplayBundleEnvelope(replayBundleEnvelope)
	if importErr == nil {
		envelopeOptions := options
		envelopeOptions.ReviewReplayBundle = replayBundle
		return ReviewConformanceManifest(manifest, envelopeOptions, execute)
	}

	fallbackOptions := options
	fallbackOptions.ReviewReplayBundle = nil
	state := ReviewConformanceManifest(manifest, fallbackOptions, execute)
	state.Diagnostics = append(state.Diagnostics, Diagnostic{
		Severity: SeverityError,
		Category: DiagnosticCategory(importErr.Category),
		Message:  importErr.Message,
	})
	return state
}

func ReviewAndExecuteConformanceManifestWithReplayBundleEnvelope[T any](
	manifest ConformanceManifest,
	options ConformanceManifestReviewOptions,
	replayBundleEnvelope ReviewReplayBundleEnvelope,
	execute func(ConformanceCaseRun) ConformanceCaseExecution,
	callbacksForExecution func(ReviewedNestedExecution, int) NestedMergeExecutionCallbacks[T],
) ConformanceManifestReviewedNestedApplication[T] {
	state := ReviewConformanceManifestWithReplayBundleEnvelope(
		manifest,
		options,
		replayBundleEnvelope,
		execute,
	)

	return ConformanceManifestReviewedNestedApplication[T]{
		State:   state,
		Results: ExecuteReviewStateReviewedNestedExecutions(state, callbacksForExecution),
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
