package astmerge

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const PortableBenchmarkSchemaMajor = 1

var portableBenchmarkSchemaPattern = regexp.MustCompile(`^structuredmerge\.benchmark(?:\.[a-z_]+)?/v([0-9]+)(?:\.[0-9]+)?$`)
var portableBenchmarkSHA256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

var portableBenchmarkEnumKeys = []string{
	"kinds", "operations", "partitions", "profiles", "outcomes", "oracle_classes",
	"equivalence_classes", "transformations", "false_auto_merge_severities",
	"preservation_requirements", "unsupported_policies", "budget_exceed_actions",
	"llm_hard_gate_forbidden_profiles", "quality_dimensions",
}

// PortableBenchmarkValidationCategory is a stable, machine-readable validation failure class.
type PortableBenchmarkValidationCategory string

const (
	PortableBenchmarkInvalidJSON        PortableBenchmarkValidationCategory = "invalid_json"
	PortableBenchmarkInvalidStructure   PortableBenchmarkValidationCategory = "invalid_structure"
	PortableBenchmarkInvalidSchema      PortableBenchmarkValidationCategory = "invalid_schema"
	PortableBenchmarkInvalidEnum        PortableBenchmarkValidationCategory = "invalid_enum"
	PortableBenchmarkInvalidReference   PortableBenchmarkValidationCategory = "invalid_reference"
	PortableBenchmarkInvalidDigest      PortableBenchmarkValidationCategory = "invalid_digest"
	PortableBenchmarkInvalidEligibility PortableBenchmarkValidationCategory = "invalid_eligibility"
	PortableBenchmarkInvalidSafety      PortableBenchmarkValidationCategory = "invalid_safety"
	PortableBenchmarkProhibitedLLMGate  PortableBenchmarkValidationCategory = "prohibited_llm_gate"
	PortableBenchmarkProhibitedScore    PortableBenchmarkValidationCategory = "prohibited_scalar_score"
)

// PortableBenchmarkValidationError is returned for every contract conformance failure.
type PortableBenchmarkValidationError struct {
	Category PortableBenchmarkValidationCategory
	Message  string
}

func (e *PortableBenchmarkValidationError) Error() string {
	return fmt.Sprintf("portable benchmark contract %s: %s", e.Category, e.Message)
}

// PortableBenchmarkContract is the typed, inert representation of a benchmark contract.
// Payload bytes and adapter identities are data only; this package never executes them.
type PortableBenchmarkContract struct {
	SchemaVersion         string                           `json:"schema_version"`
	FixtureKind           string                           `json:"fixture_kind"`
	Contract              PortableBenchmarkEnums           `json:"contract"`
	CaseSchema            PortableBenchmarkCaseSchema      `json:"case_schema"`
	RunManifestSchema     PortableBenchmarkRunSchema       `json:"run_manifest_schema"`
	CaseResultSchema      PortableBenchmarkResultSchema    `json:"case_result_schema"`
	AggregateReportSchema PortableBenchmarkAggregateSchema `json:"aggregate_report_schema"`
	Cases                 []PortableBenchmarkCase          `json:"cases"`
	RunManifest           PortableBenchmarkRunManifest     `json:"run_manifest"`
	CaseResults           []PortableBenchmarkCaseResult    `json:"case_results"`
	AggregateReport       PortableBenchmarkAggregateReport `json:"aggregate_report"`
}

type PortableBenchmarkEnums struct {
	Kinds                        []string `json:"kinds"`
	Operations                   []string `json:"operations"`
	Partitions                   []string `json:"partitions"`
	Profiles                     []string `json:"profiles"`
	Outcomes                     []string `json:"outcomes"`
	OracleClasses                []string `json:"oracle_classes"`
	EquivalenceClasses           []string `json:"equivalence_classes"`
	Transformations              []string `json:"transformations"`
	FalseAutoMergeSeverities     []string `json:"false_auto_merge_severities"`
	PreservationRequirements     []string `json:"preservation_requirements"`
	UnsupportedPolicies          []string `json:"unsupported_policies"`
	BudgetExceedActions          []string `json:"budget_exceed_actions"`
	LLMHardGateForbiddenProfiles []string `json:"llm_hard_gate_forbidden_profiles"`
	QualityDimensions            []string `json:"quality_dimensions"`
	ScalarScoreAllowed           *bool    `json:"scalar_score_allowed"`
	FalseAutoMergeCompensable    *bool    `json:"false_auto_merge_compensable"`
}

type PortableBenchmarkCaseSchema struct {
	Required                []string            `json:"required"`
	OperationRequiredFields map[string][]string `json:"operation_required_fields"`
	InputPolicy             struct {
		InlineMaxBytes int `json:"inline_max_bytes"`
	} `json:"input_policy"`
}

type PortableBenchmarkRunSchema struct {
	Required                []string `json:"required"`
	AdapterIdentityRequired []string `json:"adapter_identity_required"`
	FastSelectionRequired   []string `json:"fast_selection_required"`
}

type PortableBenchmarkResultSchema struct {
	Required []string `json:"required"`
}

type PortableBenchmarkAggregateSchema struct {
	RequiredStrata     []string `json:"required_strata"`
	ScalarScoreAllowed *bool    `json:"scalar_score_allowed"`
}

type PortableBenchmarkInlineInput struct {
	Mode   string `json:"mode"`
	Bytes  string `json:"bytes"`
	SHA256 string `json:"sha256"`
}

type PortableBenchmarkOracle struct {
	Class         string `json:"class"`
	ScoreEligible *bool  `json:"score_eligible"`
}

type PortableBenchmarkEquivalence struct {
	Class string `json:"class"`
}

type PortableBenchmarkTransformation struct {
	Type string `json:"type"`
}

type PortableBenchmarkHistoryReference struct {
	Slice       int    `json:"slice"`
	CorpusID    string `json:"corpus_id"`
	Manifest    string `json:"manifest"`
	CaseID      string `json:"case_id"`
	MergeCommit string `json:"merge_commit"`
}

type PortableBenchmarkCase struct {
	SchemaVersion          string                                  `json:"schema_version"`
	Kind                   string                                  `json:"kind"`
	ID                     string                                  `json:"id"`
	Operation              string                                  `json:"operation"`
	Capabilities           []string                                `json:"capabilities"`
	Partition              string                                  `json:"partition"`
	Inputs                 map[string]PortableBenchmarkInlineInput `json:"inputs"`
	Oracle                 PortableBenchmarkOracle                 `json:"oracle"`
	AcceptableEquivalence  []PortableBenchmarkEquivalence          `json:"acceptable_equivalence"`
	PreservationPolicy     map[string]string                       `json:"preservation_policy"`
	FalseAutoMergeSeverity string                                  `json:"false_auto_merge_severity"`
	HistoryReference       *PortableBenchmarkHistoryReference      `json:"history_reference"`
	ParentCaseID           string                                  `json:"parent_case_id"`
	Transformations        []PortableBenchmarkTransformation       `json:"transformations"`
}

type PortableBenchmarkAdapter struct {
	ID                  string `json:"id"`
	SourceSHA           string `json:"source_sha"`
	ArtifactSHA256      string `json:"artifact_sha256"`
	Version             string `json:"version"`
	ConfigurationSHA256 string `json:"configuration_sha256"`
}

type PortableBenchmarkSelection struct {
	SelectedCaseIDs []string                              `json:"selected_case_ids"`
	ExcludedCaseIDs []string                              `json:"excluded_case_ids"`
	Explanation     PortableBenchmarkSelectionExplanation `json:"explanation"`
}

type PortableBenchmarkSelectionExplanation struct {
	ChangedPaths         []string `json:"changed_paths"`
	InferredCapabilities []string `json:"inferred_capabilities"`
	DirectCases          []struct {
		CaseIDs []string `json:"case_ids"`
	} `json:"direct_cases"`
	Sentinels []struct {
		CaseID string `json:"case_id"`
	} `json:"sentinels"`
	NeighborSamples []struct {
		Population      []string `json:"population"`
		SelectedCaseIDs []string `json:"selected_case_ids"`
	} `json:"neighbor_samples"`
	BudgetExceed struct {
		Action                string `json:"action"`
		RegenerateExplanation *bool  `json:"regenerate_explanation"`
		SilentExtension       *bool  `json:"silent_extension"`
	} `json:"budget_exceed"`
}

type PortableBenchmarkRunManifest struct {
	SchemaVersion     string                     `json:"schema_version"`
	Kind              string                     `json:"kind"`
	ID                string                     `json:"id"`
	Profile           string                     `json:"profile"`
	CandidateAdapter  PortableBenchmarkAdapter   `json:"candidate_adapter"`
	BaseAdapter       *PortableBenchmarkAdapter  `json:"base_adapter"`
	Selection         PortableBenchmarkSelection `json:"selection"`
	UnsupportedPolicy string                     `json:"unsupported_policy"`
}

type PortableBenchmarkRawRecord struct {
	Inline *string `json:"inline"`
	Bytes  *int    `json:"bytes"`
	SHA256 string  `json:"sha256"`
}

type PortableBenchmarkSafety struct {
	FalseAutoMerge bool   `json:"false_auto_merge"`
	Severity       string `json:"severity"`
	Compensable    *bool  `json:"compensable"`
}

type PortableBenchmarkCaseResult struct {
	ID            string                                `json:"id"`
	RunID         string                                `json:"run_id"`
	CaseID        string                                `json:"case_id"`
	AdapterID     string                                `json:"adapter_id"`
	Outcome       string                                `json:"outcome"`
	ScoreEligible *bool                                 `json:"score_eligible"`
	Raw           map[string]PortableBenchmarkRawRecord `json:"raw"`
	Dimensions    struct {
		Safety PortableBenchmarkSafety `json:"safety"`
	} `json:"dimensions"`
}

type PortableBenchmarkPairedDelta struct {
	CaseID            string `json:"case_id"`
	BaseResultID      string `json:"base_result_id"`
	CandidateResultID string `json:"candidate_result_id"`
}

type PortableBenchmarkAggregateReport struct {
	SchemaVersion string         `json:"schema_version"`
	Kind          string         `json:"kind"`
	ID            string         `json:"id"`
	RunID         string         `json:"run_id"`
	RawResultIDs  []string       `json:"raw_result_ids"`
	OutcomeCounts map[string]int `json:"outcome_counts"`
	Dimensions    struct {
		Effectiveness struct {
			Eligible int `json:"eligible"`
		} `json:"effectiveness"`
		Safety struct {
			Eligible int `json:"eligible"`
		} `json:"safety"`
		Preservation struct {
			Eligible int `json:"eligible"`
		} `json:"preservation"`
		Coverage struct {
			UnsupportedIsQualityFailure *bool `json:"unsupported_is_quality_failure"`
		} `json:"coverage"`
	} `json:"dimensions"`
	Stratification map[string]json.RawMessage     `json:"stratification"`
	PairedDeltas   []PortableBenchmarkPairedDelta `json:"paired_deltas"`
	Gates          struct {
		Safety struct {
			Status                      string   `json:"status"`
			EligibleFalseAutoMergeCount int      `json:"eligible_false_auto_merge_count"`
			NonCompensable              bool     `json:"non_compensable"`
			OffsetsAllowed              []string `json:"offsets_allowed"`
		} `json:"safety"`
	} `json:"gates"`
	ScalarScore json.RawMessage `json:"scalar_score"`
}

type PortableBenchmarkSummaryCounts struct {
	Adapters                          int `json:"adapters"`
	Cases                             int `json:"cases"`
	Results                           int `json:"results"`
	SelectedCases                     int `json:"selected_cases"`
	ScoreEligibleResults              int `json:"score_eligible_results"`
	QualityDenominatorExcludedResults int `json:"quality_denominator_excluded_results"`
	EligibleFalseAutoMerges           int `json:"eligible_false_auto_merges"`
}

type PortableBenchmarkFalseAutoMergeSummary struct {
	ID       string `json:"id"`
	Severity string `json:"severity"`
}

type PortableBenchmarkSafetyGateSummary struct {
	Status                      string `json:"status"`
	EligibleFalseAutoMergeCount int    `json:"eligible_false_auto_merge_count"`
	NonCompensable              bool   `json:"non_compensable"`
}

// PortableBenchmarkCanonicalSummary is the deterministic interpretation used by benchmark tooling.
type PortableBenchmarkCanonicalSummary struct {
	Schema                              string                                   `json:"schema"`
	Counts                              PortableBenchmarkSummaryCounts           `json:"counts"`
	CaseIDsByOperation                  map[string][]string                      `json:"case_ids_by_operation"`
	CaseIDsByPartition                  map[string][]string                      `json:"case_ids_by_partition"`
	ScoreEligibleResultIDs              []string                                 `json:"score_eligible_result_ids"`
	QualityDenominatorExcludedResultIDs []string                                 `json:"quality_denominator_excluded_result_ids"`
	FalseAutoMerges                     []PortableBenchmarkFalseAutoMergeSummary `json:"false_auto_merges"`
	SafetyGate                          PortableBenchmarkSafetyGateSummary       `json:"safety_gate"`
	SelectionReasonCategories           []string                                 `json:"selection_reason_categories"`
	ContractDigest                      string                                   `json:"contract_digest"`
}

// PortableBenchmarkConsumer parses, validates, and interprets contract data without executing it.
type PortableBenchmarkConsumer struct {
	Document       PortableBenchmarkContract
	ContractDigest string
	raw            map[string]any
}

// ParsePortableBenchmarkContract consumes exact JSON bytes and validates the contract.
func ParsePortableBenchmarkContract(source []byte) (*PortableBenchmarkConsumer, error) {
	var decoded any
	if err := json.Unmarshal(source, &decoded); err != nil {
		return nil, benchmarkError(PortableBenchmarkInvalidJSON, "invalid benchmark contract JSON: %v", err)
	}
	raw, ok := decoded.(map[string]any)
	if !ok {
		return nil, benchmarkError(PortableBenchmarkInvalidStructure, "contract must be an object")
	}

	var document PortableBenchmarkContract
	if err := json.Unmarshal(source, &document); err != nil {
		return nil, benchmarkError(PortableBenchmarkInvalidStructure, "contract has an invalid field type: %v", err)
	}
	sum := sha256.Sum256(source)
	consumer := &PortableBenchmarkConsumer{Document: document, ContractDigest: hex.EncodeToString(sum[:]), raw: raw}
	if err := consumer.Validate(); err != nil {
		return nil, err
	}
	return consumer, nil
}

// LoadPortableBenchmarkContract reads exact file bytes and delegates to ParsePortableBenchmarkContract.
func LoadPortableBenchmarkContract(path string) (*PortableBenchmarkConsumer, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParsePortableBenchmarkContract(source)
}

func benchmarkError(category PortableBenchmarkValidationCategory, format string, args ...any) error {
	return &PortableBenchmarkValidationError{Category: category, Message: fmt.Sprintf(format, args...)}
}

func (c *PortableBenchmarkConsumer) Validate() error {
	if c == nil || c.raw == nil {
		return benchmarkError(PortableBenchmarkInvalidStructure, "contract must be an object")
	}
	if err := c.validateSchemas(); err != nil {
		return err
	}
	if err := c.validateEnums(); err != nil {
		return err
	}
	if err := c.validateCases(); err != nil {
		return err
	}
	if err := c.validateRun(); err != nil {
		return err
	}
	if err := c.validateResults(); err != nil {
		return err
	}
	return c.validateReport()
}

func (c *PortableBenchmarkConsumer) validateSchemas() error {
	values := []string{c.Document.SchemaVersion}
	for _, item := range c.Document.Cases {
		values = append(values, item.SchemaVersion)
	}
	values = append(values, c.Document.RunManifest.SchemaVersion, c.Document.AggregateReport.SchemaVersion)
	for i, value := range values {
		match := portableBenchmarkSchemaPattern.FindStringSubmatch(value)
		if match == nil {
			return benchmarkError(PortableBenchmarkInvalidSchema, "schema version %d is invalid: %q", i, value)
		}
		major, err := strconv.Atoi(match[1])
		if err != nil || major != PortableBenchmarkSchemaMajor {
			return benchmarkError(PortableBenchmarkInvalidSchema, "unsupported benchmark schema major v%s", match[1])
		}
	}
	return nil
}

func (c *PortableBenchmarkConsumer) enum(key string) []string {
	switch key {
	case "kinds":
		return c.Document.Contract.Kinds
	case "operations":
		return c.Document.Contract.Operations
	case "partitions":
		return c.Document.Contract.Partitions
	case "profiles":
		return c.Document.Contract.Profiles
	case "outcomes":
		return c.Document.Contract.Outcomes
	case "oracle_classes":
		return c.Document.Contract.OracleClasses
	case "equivalence_classes":
		return c.Document.Contract.EquivalenceClasses
	case "transformations":
		return c.Document.Contract.Transformations
	case "false_auto_merge_severities":
		return c.Document.Contract.FalseAutoMergeSeverities
	case "preservation_requirements":
		return c.Document.Contract.PreservationRequirements
	case "unsupported_policies":
		return c.Document.Contract.UnsupportedPolicies
	case "budget_exceed_actions":
		return c.Document.Contract.BudgetExceedActions
	case "llm_hard_gate_forbidden_profiles":
		return c.Document.Contract.LLMHardGateForbiddenProfiles
	case "quality_dimensions":
		return c.Document.Contract.QualityDimensions
	default:
		return nil
	}
}

func (c *PortableBenchmarkConsumer) validateEnums() error {
	rawContract, err := rawObject(c.raw, "contract", "contract")
	if err != nil {
		return err
	}
	for _, key := range portableBenchmarkEnumKeys {
		if _, err := rawStringArray(rawContract, key, "contract enums"); err != nil {
			return err
		}
		values := c.enum(key)
		if hasDuplicate(values) {
			return benchmarkError(PortableBenchmarkInvalidEnum, "%s contains duplicate values", key)
		}
		for _, value := range values {
			if value == "" {
				return benchmarkError(PortableBenchmarkInvalidEnum, "%s must contain only non-empty strings", key)
			}
		}
	}
	if c.Document.Contract.ScalarScoreAllowed == nil || *c.Document.Contract.ScalarScoreAllowed {
		return benchmarkError(PortableBenchmarkProhibitedScore, "scalar scores must be prohibited")
	}
	if c.Document.Contract.FalseAutoMergeCompensable == nil || *c.Document.Contract.FalseAutoMergeCompensable {
		return benchmarkError(PortableBenchmarkInvalidSafety, "false auto-merges must be non-compensable")
	}
	return nil
}

func (c *PortableBenchmarkConsumer) membership(value, enumKey, label string) error {
	if !contains(c.enum(enumKey), value) {
		return benchmarkError(PortableBenchmarkInvalidEnum, "%s is not a declared %s member: %q", label, enumKey, value)
	}
	return nil
}

func (c *PortableBenchmarkConsumer) validateCases() error {
	if err := uniqueIDs(caseIDs(c.Document.Cases), "case"); err != nil {
		return err
	}
	rawCases, err := rawArray(c.raw, "cases", "contract")
	if err != nil {
		return err
	}
	if len(rawCases) != len(c.Document.Cases) {
		return benchmarkError(PortableBenchmarkInvalidStructure, "cases must be an array")
	}
	for i, item := range c.Document.Cases {
		label := "case " + item.ID
		if err := requiredFields(rawCases[i], c.Document.CaseSchema.Required, label); err != nil {
			return err
		}
		for _, check := range []struct{ value, key, label string }{
			{item.Kind, "kinds", label + " kind"}, {item.Operation, "operations", label + " operation"},
			{item.Partition, "partitions", label + " partition"}, {item.Oracle.Class, "oracle_classes", label + " oracle"},
			{item.FalseAutoMergeSeverity, "false_auto_merge_severities", label + " severity"},
		} {
			if err := c.membership(check.value, check.key, check.label); err != nil {
				return err
			}
		}
		if item.Kind != "benchmark_case" {
			return benchmarkError(PortableBenchmarkInvalidStructure, "%s kind must be benchmark_case", label)
		}
		if item.Oracle.ScoreEligible == nil {
			return benchmarkError(PortableBenchmarkInvalidStructure, "%s oracle score eligibility must be boolean", label)
		}
		if !sort.StringsAreSorted(item.Capabilities) || hasDuplicate(item.Capabilities) {
			return benchmarkError(PortableBenchmarkInvalidStructure, "%s capabilities must be unique and sorted", label)
		}
		for _, eq := range item.AcceptableEquivalence {
			if err := c.membership(eq.Class, "equivalence_classes", label+" equivalence"); err != nil {
				return err
			}
		}
		for _, requirement := range item.PreservationPolicy {
			if err := c.membership(requirement, "preservation_requirements", label+" preservation"); err != nil {
				return err
			}
		}
		fields, ok := c.Document.CaseSchema.OperationRequiredFields[item.Operation]
		if !ok {
			return benchmarkError(PortableBenchmarkInvalidStructure, "%s operation requirements are missing", label)
		}
		if err := requiredFields(rawCases[i], fields, label+" "+item.Operation); err != nil {
			return err
		}
		if err := c.validateInlineHashes(rawCases[i], label, c.Document.CaseSchema.InputPolicy.InlineMaxBytes); err != nil {
			return err
		}
		if item.Operation == "history_replay" {
			if err := validateHistory(item, rawCases[i]); err != nil {
				return err
			}
		}
		if item.Operation == "metamorphic" {
			for _, transform := range item.Transformations {
				if err := c.membership(transform.Type, "transformations", label+" transformation"); err != nil {
					return err
				}
			}
		}
	}
	ids := caseIDs(c.Document.Cases)
	for _, item := range c.Document.Cases {
		if item.Operation == "metamorphic" && !contains(ids, item.ParentCaseID) {
			return benchmarkError(PortableBenchmarkInvalidReference, "case %s parent_case_id is dangling: %q", item.ID, item.ParentCaseID)
		}
	}
	return nil
}

func (c *PortableBenchmarkConsumer) validateInlineHashes(value any, label string, limit int) error {
	switch current := value.(type) {
	case map[string]any:
		if current["mode"] == "inline" {
			bytes, ok := current["bytes"].(string)
			if !ok {
				return benchmarkError(PortableBenchmarkInvalidStructure, "%s inline bytes must be a string", label)
			}
			if len([]byte(bytes)) > limit {
				return benchmarkError(PortableBenchmarkInvalidStructure, "%s inline bytes exceed %d", label, limit)
			}
			digest, _ := current["sha256"].(string)
			if err := exactDigest(digest, bytes, label+" inline input"); err != nil {
				return err
			}
		}
		for _, child := range current {
			if err := c.validateInlineHashes(child, label, limit); err != nil {
				return err
			}
		}
	case []any:
		for _, child := range current {
			if err := c.validateInlineHashes(child, label, limit); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateHistory(item PortableBenchmarkCase, raw map[string]any) error {
	label := "case " + item.ID + " history reference"
	history, ok := raw["history_reference"].(map[string]any)
	if !ok {
		return benchmarkError(PortableBenchmarkInvalidStructure, "%s must be an object", label)
	}
	if err := requiredFields(history, []string{"slice", "corpus_id", "manifest", "case_id", "merge_commit"}, label); err != nil {
		return err
	}
	if item.HistoryReference == nil || item.HistoryReference.Slice != 1021 {
		return benchmarkError(PortableBenchmarkInvalidReference, "case %s history reference must target Slice 1021", item.ID)
	}
	if item.HistoryReference.Manifest != "diagnostics/slice-1021-reviewed-git-history-corpus/manifest.json" {
		return benchmarkError(PortableBenchmarkInvalidReference, "case %s history manifest must target Slice 1021", item.ID)
	}
	if _, exists := raw["inputs"]; exists {
		return benchmarkError(PortableBenchmarkInvalidStructure, "case %s history reference must not inline source bytes", item.ID)
	}
	return nil
}

func (c *PortableBenchmarkConsumer) adapters() []PortableBenchmarkAdapter {
	result := []PortableBenchmarkAdapter{c.Document.RunManifest.CandidateAdapter}
	if c.Document.RunManifest.BaseAdapter != nil {
		result = append(result, *c.Document.RunManifest.BaseAdapter)
	}
	return result
}

func (c *PortableBenchmarkConsumer) validateRun() error {
	runRaw, err := rawObject(c.raw, "run_manifest", "contract")
	if err != nil {
		return err
	}
	run := c.Document.RunManifest
	if err := requiredFields(runRaw, c.Document.RunManifestSchema.Required, "run manifest"); err != nil {
		return err
	}
	if run.Kind != "run_manifest" {
		return benchmarkError(PortableBenchmarkInvalidStructure, "run manifest kind must be run_manifest")
	}
	if err := c.membership(run.Profile, "profiles", "run profile"); err != nil {
		return err
	}
	if err := c.membership(run.UnsupportedPolicy, "unsupported_policies", "unsupported policy"); err != nil {
		return err
	}
	for _, adapter := range c.adapters() {
		if adapter.ID == "" {
			return benchmarkError(PortableBenchmarkInvalidStructure, "adapter IDs must be non-empty strings")
		}
		values := map[string]string{"source_sha": adapter.SourceSHA, "artifact_sha256": adapter.ArtifactSHA256, "version": adapter.Version, "configuration_sha256": adapter.ConfigurationSHA256}
		for _, field := range c.Document.RunManifestSchema.AdapterIdentityRequired {
			if values[field] == "" {
				return benchmarkError(PortableBenchmarkInvalidStructure, "adapter %s is missing required field %s", adapter.ID, field)
			}
		}
	}
	adapterIDs := []string{}
	for _, adapter := range c.adapters() {
		adapterIDs = append(adapterIDs, adapter.ID)
	}
	if err := uniqueIDs(adapterIDs, "adapter"); err != nil {
		return err
	}
	selected, excluded := run.Selection.SelectedCaseIDs, run.Selection.ExcludedCaseIDs
	if hasDuplicate(selected) {
		return benchmarkError(PortableBenchmarkInvalidReference, "selected case IDs contain duplicates")
	}
	if hasDuplicate(excluded) {
		return benchmarkError(PortableBenchmarkInvalidReference, "excluded case IDs contain duplicates")
	}
	for _, id := range selected {
		if contains(excluded, id) {
			return benchmarkError(PortableBenchmarkInvalidReference, "selected and excluded case IDs overlap")
		}
	}
	ids := caseIDs(c.Document.Cases)
	for _, id := range append(append([]string{}, selected...), excluded...) {
		if !contains(ids, id) {
			return benchmarkError(PortableBenchmarkInvalidReference, "run selection case_id is dangling: %q", id)
		}
	}
	if err := c.validateSelection(runRaw, ids); err != nil {
		return err
	}
	if contains(c.Document.Contract.LLMHardGateForbiddenProfiles, run.Profile) {
		for _, item := range c.Document.Cases {
			if contains(selected, item.ID) && item.Oracle.Class == "llm" && item.Oracle.ScoreEligible != nil && *item.Oracle.ScoreEligible {
				return benchmarkError(PortableBenchmarkProhibitedLLMGate, "LLM oracle cannot participate in %s hard gates", run.Profile)
			}
		}
	}
	return nil
}

func (c *PortableBenchmarkConsumer) validateSelection(runRaw map[string]any, caseIDs []string) error {
	selection, ok := runRaw["selection"].(map[string]any)
	if !ok {
		return benchmarkError(PortableBenchmarkInvalidStructure, "run manifest.selection must be an object")
	}
	explanation, ok := selection["explanation"].(map[string]any)
	if !ok {
		return benchmarkError(PortableBenchmarkInvalidStructure, "run selection.explanation must be an object")
	}
	if err := requiredFields(explanation, c.Document.RunManifestSchema.FastSelectionRequired, "selection explanation"); err != nil {
		return err
	}
	typed := c.Document.RunManifest.Selection.Explanation
	for _, direct := range typed.DirectCases {
		for _, id := range direct.CaseIDs {
			if !contains(caseIDs, id) {
				return benchmarkError(PortableBenchmarkInvalidReference, "direct selection case_id is dangling: %q", id)
			}
		}
	}
	for _, sentinel := range typed.Sentinels {
		if !contains(caseIDs, sentinel.CaseID) {
			return benchmarkError(PortableBenchmarkInvalidReference, "sentinel selection case_id is dangling: %q", sentinel.CaseID)
		}
	}
	for _, sample := range typed.NeighborSamples {
		for _, id := range append(append([]string{}, sample.Population...), sample.SelectedCaseIDs...) {
			if !contains(caseIDs, id) {
				return benchmarkError(PortableBenchmarkInvalidReference, "neighbor selection case_id is dangling: %q", id)
			}
		}
	}
	budget := typed.BudgetExceed
	if err := c.membership(budget.Action, "budget_exceed_actions", "budget exceed action"); err != nil {
		return err
	}
	if budget.RegenerateExplanation == nil || !*budget.RegenerateExplanation {
		return benchmarkError(PortableBenchmarkInvalidStructure, "budget exceed must regenerate the explanation")
	}
	if budget.SilentExtension == nil || *budget.SilentExtension {
		return benchmarkError(PortableBenchmarkInvalidStructure, "silent budget extension is prohibited")
	}
	return nil
}

func (c *PortableBenchmarkConsumer) validateResults() error {
	if err := uniqueIDs(resultIDs(c.Document.CaseResults), "result"); err != nil {
		return err
	}
	rawResults, err := rawArray(c.raw, "case_results", "contract")
	if err != nil {
		return err
	}
	cases := map[string]PortableBenchmarkCase{}
	for _, item := range c.Document.Cases {
		cases[item.ID] = item
	}
	adapterIDs := []string{}
	for _, adapter := range c.adapters() {
		adapterIDs = append(adapterIDs, adapter.ID)
	}
	for i, result := range c.Document.CaseResults {
		label := "result " + result.ID
		if err := requiredFields(rawResults[i], c.Document.CaseResultSchema.Required, label); err != nil {
			return err
		}
		if result.RunID != c.Document.RunManifest.ID {
			return benchmarkError(PortableBenchmarkInvalidReference, "%s run_id is dangling: %q", label, result.RunID)
		}
		item, ok := cases[result.CaseID]
		if !ok {
			return benchmarkError(PortableBenchmarkInvalidReference, "%s case_id is dangling: %q", label, result.CaseID)
		}
		if !contains(adapterIDs, result.AdapterID) {
			return benchmarkError(PortableBenchmarkInvalidReference, "%s adapter_id is dangling: %q", label, result.AdapterID)
		}
		if err := c.membership(result.Outcome, "outcomes", label+" outcome"); err != nil {
			return err
		}
		if result.ScoreEligible == nil {
			return benchmarkError(PortableBenchmarkInvalidStructure, "%s score eligibility must be boolean", label)
		}
		expected := item.Oracle.ScoreEligible != nil && *item.Oracle.ScoreEligible
		if qualityExcluded(result.Outcome) {
			expected = false
		}
		if *result.ScoreEligible != expected {
			return benchmarkError(PortableBenchmarkInvalidEligibility, "%s score eligibility conflicts with case/outcome", label)
		}
		if err := validateRawInlineRecords(rawResults[i], label); err != nil {
			return err
		}
		dimensions, ok := rawResults[i]["dimensions"].(map[string]any)
		if !ok {
			return benchmarkError(PortableBenchmarkInvalidStructure, "%s.dimensions must be an object", label)
		}
		if _, ok := dimensions["safety"].(map[string]any); !ok {
			return benchmarkError(PortableBenchmarkInvalidStructure, "%s safety must be an object", label)
		}
		safety := result.Dimensions.Safety
		if result.Outcome == "false_auto_merge" {
			if !safety.FalseAutoMerge {
				return benchmarkError(PortableBenchmarkInvalidSafety, "%s must identify a false auto-merge", label)
			}
			if safety.Compensable == nil || *safety.Compensable {
				return benchmarkError(PortableBenchmarkInvalidSafety, "%s false auto-merge cannot be compensable", label)
			}
			if err := c.membership(safety.Severity, "false_auto_merge_severities", label+" severity"); err != nil {
				return err
			}
		} else if safety.FalseAutoMerge {
			return benchmarkError(PortableBenchmarkInvalidSafety, "%s safety classification conflicts with outcome", label)
		}
	}
	return nil
}

func validateRawInlineRecords(result map[string]any, label string) error {
	raw, ok := result["raw"].(map[string]any)
	if !ok {
		return benchmarkError(PortableBenchmarkInvalidStructure, "%s.raw must be an object", label)
	}
	for name, value := range raw {
		record, ok := value.(map[string]any)
		if !ok {
			continue
		}
		inlineValue, hasInline := record["inline"]
		if !hasInline {
			continue
		}
		inline, ok := inlineValue.(string)
		if !ok {
			return benchmarkError(PortableBenchmarkInvalidStructure, "%s raw %s inline content must be a string", label, name)
		}
		byteLength, ok := record["bytes"].(float64)
		if !ok || byteLength != float64(len([]byte(inline))) {
			return benchmarkError(PortableBenchmarkInvalidDigest, "%s raw %s byte length is not exact", label, name)
		}
		digest, _ := record["sha256"].(string)
		if err := exactDigest(digest, inline, label+" raw "+name); err != nil {
			return err
		}
	}
	return nil
}

func (c *PortableBenchmarkConsumer) validateReport() error {
	report := c.Document.AggregateReport
	if report.Kind != "aggregate_report" {
		return benchmarkError(PortableBenchmarkInvalidStructure, "aggregate report kind must be aggregate_report")
	}
	if report.RunID != c.Document.RunManifest.ID {
		return benchmarkError(PortableBenchmarkInvalidReference, "aggregate report run_id is dangling: %q", report.RunID)
	}
	ids := resultIDs(c.Document.CaseResults)
	if hasDuplicate(report.RawResultIDs) {
		return benchmarkError(PortableBenchmarkInvalidReference, "aggregate raw result IDs contain duplicates")
	}
	if !sameStrings(report.RawResultIDs, ids) {
		return benchmarkError(PortableBenchmarkInvalidReference, "aggregate raw result IDs must exactly cover results")
	}
	for _, stratum := range c.Document.AggregateReportSchema.RequiredStrata {
		if _, ok := report.Stratification[stratum]; !ok {
			return benchmarkError(PortableBenchmarkInvalidStructure, "report stratification is missing required field %s", stratum)
		}
	}
	actual := map[string]int{}
	for _, outcome := range c.Document.Contract.Outcomes {
		actual[outcome] = 0
	}
	eligible := 0
	for _, result := range c.Document.CaseResults {
		actual[result.Outcome]++
		if result.ScoreEligible != nil && *result.ScoreEligible {
			eligible++
		}
	}
	if !sameCounts(report.OutcomeCounts, actual) {
		return benchmarkError(PortableBenchmarkInvalidStructure, "aggregate outcome counts do not match raw results")
	}
	for name, count := range map[string]int{"effectiveness": report.Dimensions.Effectiveness.Eligible, "safety": report.Dimensions.Safety.Eligible, "preservation": report.Dimensions.Preservation.Eligible} {
		if count != eligible {
			return benchmarkError(PortableBenchmarkInvalidEligibility, "aggregate %s denominator is incorrect", name)
		}
	}
	if c.Document.RunManifest.UnsupportedPolicy == "coverage_only" &&
		(report.Dimensions.Coverage.UnsupportedIsQualityFailure == nil || *report.Dimensions.Coverage.UnsupportedIsQualityFailure) {
		return benchmarkError(PortableBenchmarkInvalidEligibility, "coverage-only unsupported results cannot be quality failures")
	}
	byID := map[string]PortableBenchmarkCaseResult{}
	for _, result := range c.Document.CaseResults {
		byID[result.ID] = result
	}
	for _, pair := range report.PairedDeltas {
		base, baseOK := byID[pair.BaseResultID]
		candidate, candidateOK := byID[pair.CandidateResultID]
		if !baseOK {
			return benchmarkError(PortableBenchmarkInvalidReference, "paired delta %s has dangling base result", pair.CaseID)
		}
		if !candidateOK {
			return benchmarkError(PortableBenchmarkInvalidReference, "paired delta %s has dangling candidate result", pair.CaseID)
		}
		if base.CaseID != pair.CaseID || candidate.CaseID != pair.CaseID {
			return benchmarkError(PortableBenchmarkInvalidReference, "paired delta %s mixes cases", pair.CaseID)
		}
		if c.Document.RunManifest.BaseAdapter == nil || base.AdapterID != c.Document.RunManifest.BaseAdapter.ID {
			return benchmarkError(PortableBenchmarkInvalidReference, "paired delta %s base adapter is incorrect", pair.CaseID)
		}
		if candidate.AdapterID != c.Document.RunManifest.CandidateAdapter.ID {
			return benchmarkError(PortableBenchmarkInvalidReference, "paired delta %s candidate adapter is incorrect", pair.CaseID)
		}
	}
	if c.Document.AggregateReportSchema.ScalarScoreAllowed == nil || *c.Document.AggregateReportSchema.ScalarScoreAllowed {
		return benchmarkError(PortableBenchmarkProhibitedScore, "report schema must prohibit scalar scores")
	}
	if len(report.ScalarScore) != 0 && string(report.ScalarScore) != "null" {
		return benchmarkError(PortableBenchmarkProhibitedScore, "aggregate scalar score is prohibited")
	}
	fams := c.eligibleFalseAutoMerges()
	gate := report.Gates.Safety
	status := "pass"
	if len(fams) != 0 {
		status = "fail"
	}
	if gate.Status != status {
		return benchmarkError(PortableBenchmarkInvalidSafety, "safety gate status does not reflect eligible false auto-merges")
	}
	if gate.EligibleFalseAutoMergeCount != len(fams) {
		return benchmarkError(PortableBenchmarkInvalidSafety, "safety gate false-auto-merge count is incorrect")
	}
	if !gate.NonCompensable {
		return benchmarkError(PortableBenchmarkInvalidSafety, "safety gate must be non-compensable")
	}
	if gate.OffsetsAllowed == nil || len(gate.OffsetsAllowed) != 0 {
		return benchmarkError(PortableBenchmarkInvalidSafety, "safety gate must prohibit offsets")
	}
	return nil
}

// CanonicalSummary validates and returns the complete deterministic benchmark summary.
func (c *PortableBenchmarkConsumer) CanonicalSummary() (PortableBenchmarkCanonicalSummary, error) {
	if err := c.Validate(); err != nil {
		return PortableBenchmarkCanonicalSummary{}, err
	}
	eligibleIDs, excludedIDs := []string{}, []string{}
	for _, result := range c.Document.CaseResults {
		if result.ScoreEligible != nil && *result.ScoreEligible {
			eligibleIDs = append(eligibleIDs, result.ID)
		}
		if qualityExcluded(result.Outcome) {
			excludedIDs = append(excludedIDs, result.ID)
		}
	}
	sort.Strings(eligibleIDs)
	sort.Strings(excludedIDs)
	fams := c.eligibleFalseAutoMerges()
	status := "pass"
	if len(fams) != 0 {
		status = "fail"
	}
	selectionReasonCategories, err := c.selectionReasonCategories()
	if err != nil {
		return PortableBenchmarkCanonicalSummary{}, err
	}
	return PortableBenchmarkCanonicalSummary{
		Schema: c.Document.SchemaVersion,
		Counts: PortableBenchmarkSummaryCounts{
			Adapters: len(c.adapters()), Cases: len(c.Document.Cases), Results: len(c.Document.CaseResults),
			SelectedCases: len(c.Document.RunManifest.Selection.SelectedCaseIDs), ScoreEligibleResults: len(eligibleIDs),
			QualityDenominatorExcludedResults: len(excludedIDs), EligibleFalseAutoMerges: len(fams),
		},
		CaseIDsByOperation:     c.groupedCases(c.Document.Contract.Operations, func(item PortableBenchmarkCase) string { return item.Operation }),
		CaseIDsByPartition:     c.groupedCases(c.Document.Contract.Partitions, func(item PortableBenchmarkCase) string { return item.Partition }),
		ScoreEligibleResultIDs: eligibleIDs, QualityDenominatorExcludedResultIDs: excludedIDs, FalseAutoMerges: fams,
		SafetyGate:                PortableBenchmarkSafetyGateSummary{Status: status, EligibleFalseAutoMergeCount: len(fams), NonCompensable: true},
		SelectionReasonCategories: selectionReasonCategories,
		ContractDigest:            c.ContractDigest,
	}, nil
}

func (c *PortableBenchmarkConsumer) selectionReasonCategories() ([]string, error) {
	run, err := rawObject(c.raw, "run_manifest", "contract")
	if err != nil {
		return nil, err
	}
	selection, ok := run["selection"].(map[string]any)
	if !ok {
		return nil, benchmarkError(PortableBenchmarkInvalidStructure, "run manifest.selection must be an object")
	}
	explanation, ok := selection["explanation"].(map[string]any)
	if !ok {
		return nil, benchmarkError(PortableBenchmarkInvalidStructure, "run selection.explanation must be an object")
	}
	categories := make([]string, 0, len(explanation))
	for category := range explanation {
		categories = append(categories, category)
	}
	sort.Strings(categories)
	return categories, nil
}

func (c *PortableBenchmarkConsumer) groupedCases(values []string, field func(PortableBenchmarkCase) string) map[string][]string {
	result := make(map[string][]string, len(values))
	for _, value := range values {
		result[value] = []string{}
		for _, item := range c.Document.Cases {
			if field(item) == value {
				result[value] = append(result[value], item.ID)
			}
		}
		sort.Strings(result[value])
	}
	return result
}

func (c *PortableBenchmarkConsumer) eligibleFalseAutoMerges() []PortableBenchmarkFalseAutoMergeSummary {
	result := []PortableBenchmarkFalseAutoMergeSummary{}
	for _, item := range c.Document.CaseResults {
		if item.Outcome == "false_auto_merge" && item.ScoreEligible != nil && *item.ScoreEligible {
			result = append(result, PortableBenchmarkFalseAutoMergeSummary{ID: item.ID, Severity: item.Dimensions.Safety.Severity})
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func exactDigest(digest, content, label string) error {
	if !portableBenchmarkSHA256Pattern.MatchString(digest) {
		return benchmarkError(PortableBenchmarkInvalidDigest, "%s SHA-256 is malformed", label)
	}
	sum := sha256.Sum256([]byte(content))
	if digest != hex.EncodeToString(sum[:]) {
		return benchmarkError(PortableBenchmarkInvalidDigest, "%s SHA-256 does not match exact bytes", label)
	}
	return nil
}

func requiredFields(object map[string]any, fields []string, label string) error {
	for _, field := range fields {
		var value any = object
		for _, part := range strings.Split(field, ".") {
			current, ok := value.(map[string]any)
			if !ok {
				value = nil
				break
			}
			value = current[part]
		}
		if value == nil {
			return benchmarkError(PortableBenchmarkInvalidStructure, "%s is missing required field %s", label, field)
		}
	}
	return nil
}

func rawObject(object map[string]any, key, label string) (map[string]any, error) {
	value, ok := object[key].(map[string]any)
	if !ok {
		return nil, benchmarkError(PortableBenchmarkInvalidStructure, "%s.%s must be an object", label, key)
	}
	return value, nil
}

func rawArray(object map[string]any, key, label string) ([]map[string]any, error) {
	value, ok := object[key].([]any)
	if !ok {
		return nil, benchmarkError(PortableBenchmarkInvalidStructure, "%s.%s must be an array", label, key)
	}
	result := make([]map[string]any, len(value))
	for i, item := range value {
		entry, ok := item.(map[string]any)
		if !ok {
			return nil, benchmarkError(PortableBenchmarkInvalidStructure, "%s.%s must contain objects", label, key)
		}
		result[i] = entry
	}
	return result, nil
}

func rawStringArray(object map[string]any, key, label string) ([]string, error) {
	value, ok := object[key].([]any)
	if !ok {
		return nil, benchmarkError(PortableBenchmarkInvalidStructure, "%s.%s must be an array", label, key)
	}
	result := make([]string, len(value))
	for i, item := range value {
		entry, ok := item.(string)
		if !ok {
			return nil, benchmarkError(PortableBenchmarkInvalidEnum, "%s must contain only non-empty strings", key)
		}
		result[i] = entry
	}
	return result, nil
}

func uniqueIDs(ids []string, label string) error {
	for _, id := range ids {
		if id == "" {
			return benchmarkError(PortableBenchmarkInvalidStructure, "%s IDs must be non-empty strings", label)
		}
	}
	if hasDuplicate(ids) {
		return benchmarkError(PortableBenchmarkInvalidReference, "duplicate %s ID", label)
	}
	return nil
}

func hasDuplicate(values []string) bool {
	seen := map[string]struct{}{}
	for _, value := range values {
		if _, ok := seen[value]; ok {
			return true
		}
		seen[value] = struct{}{}
	}
	return false
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func caseIDs(values []PortableBenchmarkCase) []string {
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = value.ID
	}
	return result
}

func resultIDs(values []PortableBenchmarkCaseResult) []string {
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = value.ID
	}
	return result
}

func qualityExcluded(outcome string) bool {
	return outcome == "unsupported" || outcome == "excluded_ambiguous"
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	lcopy, rcopy := append([]string{}, left...), append([]string{}, right...)
	sort.Strings(lcopy)
	sort.Strings(rcopy)
	for i := range lcopy {
		if lcopy[i] != rcopy[i] {
			return false
		}
	}
	return true
}

func sameCounts(left, right map[string]int) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}
