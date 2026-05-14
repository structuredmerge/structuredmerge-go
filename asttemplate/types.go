package asttemplate

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/markdownmerge"
	"github.com/structuredmerge/structuredmerge-go/rubymerge"
	"github.com/structuredmerge/structuredmerge-go/tomlmerge"
)

type DirectorySessionMode string

const (
	DirectorySessionModePlan    DirectorySessionMode = "plan"
	DirectorySessionModeApply   DirectorySessionMode = "apply"
	DirectorySessionModeReapply DirectorySessionMode = "reapply"
)

var ReadmeFamilyLanguageOrder = []string{"go", "ruby", "rust", "typescript"}

var readmeFamilyLanguageLabels = map[string]string{
	"go":         "Go",
	"ruby":       "Ruby",
	"rust":       "Rust",
	"typescript": "TypeScript",
}

type DirectorySessionReport struct {
	Mode         DirectorySessionMode                   `json:"mode"`
	RunnerReport astmerge.TemplateDirectoryRunnerReport `json:"runner_report"`
}

type FamilyMergeAdapter func(astmerge.TemplateExecutionPlanEntry) astmerge.MergeResult[string]

type FamilyMergeAdapterRegistry map[string]FamilyMergeAdapter

func ReadmeFamilyLanguageAliases(selfLanguage string, languageOrder []string) map[string]string {
	order := languageOrder
	if len(order) == 0 {
		order = ReadmeFamilyLanguageOrder
	}

	aliases := map[string]string{
		"SELF_LANG":    readmeFamilyLabel(selfLanguage),
		"SELF_LANG_ID": selfLanguage,
	}
	aliasIndex := 1
	for _, language := range order {
		if language == selfLanguage {
			continue
		}
		prefix := "IMP_LANG" + strconv.Itoa(aliasIndex)
		aliases[prefix] = readmeFamilyLabel(language)
		aliases[prefix+"_ID"] = language
		aliasIndex++
	}

	return aliases
}

func ReadmeFamilyTokenValues(family map[string]any) map[string]string {
	selfEntry := mapValue(family["self"])
	selfID := stringValue(selfEntry["id"])
	implementations := map[string]map[string]any{}
	if rawImplementations, ok := family["implementations"].([]any); ok {
		for _, raw := range rawImplementations {
			entry := mapValue(raw)
			implementations[stringValue(entry["id"])] = entry
		}
	}

	tokens := ReadmeFamilyLanguageAliases(selfID, ReadmeFamilyLanguageOrder)
	tokens["FAMILY_SECTION_HEADING"] = stringValue(family["section_heading"])
	tokens["FAMILY_DESCRIPTION"] = stringValue(family["description"])
	tokens["SELF_PACKAGE_ROOT"] = stringValue(selfEntry["package_root"])
	tokens["SELF_PACKAGE_MANAGER"] = stringValue(selfEntry["package_manager"])

	aliasIndex := 1
	for _, language := range ReadmeFamilyLanguageOrder {
		if language == selfID {
			continue
		}
		entry := implementations[language]
		prefix := "IMP_LANG" + strconv.Itoa(aliasIndex)
		tokens[prefix+"_PACKAGE_ROOT"] = stringValue(entry["package_root"])
		tokens[prefix+"_PACKAGE_MANAGER"] = stringValue(entry["package_manager"])
		aliasIndex++
	}

	return tokens
}

func RenderReadmeFamilySection(templatePartial string, family map[string]any, config *astmerge.TemplateTokenConfig) string {
	tokenValues := ReadmeFamilyTokenValues(family)
	replacements := make(map[string]string, len(tokenValues))
	for key, value := range tokenValues {
		replacements["SM|"+key] = value
	}

	return astmerge.ResolveTemplateTokens(templatePartial, replacements, config)
}

func readmeFamilyLabel(language string) string {
	if label, ok := readmeFamilyLanguageLabels[language]; ok {
		return label
	}
	return language
}

func mapValue(value any) map[string]any {
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return map[string]any{}
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	if typed, ok := value.(string); ok {
		return typed
	}
	return ""
}

type DirectoryRegistrySessionReport struct {
	Mode            DirectorySessionMode                   `json:"mode"`
	AdapterFamilies []string                               `json:"adapter_families"`
	Diagnostics     []astmerge.Diagnostic                  `json:"diagnostics"`
	RunnerReport    astmerge.TemplateDirectoryRunnerReport `json:"runner_report"`
}

type AdapterCapabilityReport struct {
	RequiredFamilies []string `json:"required_families"`
	AdapterFamilies  []string `json:"adapter_families"`
	MissingFamilies  []string `json:"missing_families"`
	Ready            bool     `json:"ready"`
}

type SessionEnvelopeReport struct {
	SessionReport       any                     `json:"session_report"`
	AdapterCapabilities AdapterCapabilityReport `json:"adapter_capabilities"`
}

type SessionStatusReport struct {
	Mode              DirectorySessionMode `json:"mode"`
	Ready             bool                 `json:"ready"`
	MissingFamilies   []string             `json:"missing_families"`
	BlockedPaths      []string             `json:"blocked_paths"`
	PlannedWriteCount int                  `json:"planned_write_count"`
	WrittenCount      int                  `json:"written_count"`
}

const SessionStatusTransportVersion = 1

type SessionStatusTransportImportErrorCategory string

const (
	SessionStatusTransportKindMismatch       SessionStatusTransportImportErrorCategory = "kind_mismatch"
	SessionStatusTransportUnsupportedVersion SessionStatusTransportImportErrorCategory = "unsupported_version"
)

type SessionStatusTransportImportError struct {
	Category SessionStatusTransportImportErrorCategory `json:"category"`
	Message  string                                    `json:"message"`
}

type SessionStatusEnvelope struct {
	Kind    string              `json:"kind"`
	Version int                 `json:"version"`
	Status  SessionStatusReport `json:"status"`
}

func SessionStatusEnvelopeFor(status SessionStatusReport) SessionStatusEnvelope {
	return SessionStatusEnvelope{
		Kind:    "template_directory_session_status",
		Version: SessionStatusTransportVersion,
		Status:  status,
	}
}

func ImportSessionStatusEnvelope(
	envelope SessionStatusEnvelope,
) (*SessionStatusReport, *SessionStatusTransportImportError) {
	if envelope.Kind != "template_directory_session_status" {
		return nil, &SessionStatusTransportImportError{
			Category: SessionStatusTransportKindMismatch,
			Message:  "expected template_directory_session_status envelope kind.",
		}
	}

	if envelope.Version != SessionStatusTransportVersion {
		return nil, &SessionStatusTransportImportError{
			Category: SessionStatusTransportUnsupportedVersion,
			Message:  fmt.Sprintf("unsupported template_directory_session_status envelope version %d.", envelope.Version),
		}
	}

	status := envelope.Status
	return &status, nil
}

type SessionDiagnostic struct {
	Severity astmerge.DiagnosticSeverity `json:"severity"`
	Category astmerge.DiagnosticCategory `json:"category"`
	Reason   string                      `json:"reason"`
	Path     string                      `json:"path,omitempty"`
	Family   string                      `json:"family,omitempty"`
	Message  string                      `json:"message"`
}

type SessionDiagnosticsReport struct {
	Mode        DirectorySessionMode `json:"mode"`
	Ready       bool                 `json:"ready"`
	Diagnostics []SessionDiagnostic  `json:"diagnostics"`
}

const SessionDiagnosticsTransportVersion = 1

type SessionDiagnosticsTransportImportErrorCategory string

const (
	SessionDiagnosticsTransportKindMismatch       SessionDiagnosticsTransportImportErrorCategory = "kind_mismatch"
	SessionDiagnosticsTransportUnsupportedVersion SessionDiagnosticsTransportImportErrorCategory = "unsupported_version"
)

type SessionDiagnosticsTransportImportError struct {
	Category SessionDiagnosticsTransportImportErrorCategory `json:"category"`
	Message  string                                         `json:"message"`
}

type SessionDiagnosticsEnvelope struct {
	Kind        string                   `json:"kind"`
	Version     int                      `json:"version"`
	Diagnostics SessionDiagnosticsReport `json:"diagnostics"`
}

func SessionDiagnosticsEnvelopeFor(diagnostics SessionDiagnosticsReport) SessionDiagnosticsEnvelope {
	return SessionDiagnosticsEnvelope{
		Kind:        "template_directory_session_diagnostics",
		Version:     SessionDiagnosticsTransportVersion,
		Diagnostics: diagnostics,
	}
}

func ImportSessionDiagnosticsEnvelope(
	envelope SessionDiagnosticsEnvelope,
) (*SessionDiagnosticsReport, *SessionDiagnosticsTransportImportError) {
	if envelope.Kind != "template_directory_session_diagnostics" {
		return nil, &SessionDiagnosticsTransportImportError{
			Category: SessionDiagnosticsTransportKindMismatch,
			Message:  "expected template_directory_session_diagnostics envelope kind.",
		}
	}

	if envelope.Version != SessionDiagnosticsTransportVersion {
		return nil, &SessionDiagnosticsTransportImportError{
			Category: SessionDiagnosticsTransportUnsupportedVersion,
			Message:  fmt.Sprintf("unsupported template_directory_session_diagnostics envelope version %d.", envelope.Version),
		}
	}

	diagnostics := envelope.Diagnostics
	return &diagnostics, nil
}

type SessionOutcomeReport struct {
	SessionReport any                      `json:"session_report"`
	Status        SessionStatusReport      `json:"status"`
	Diagnostics   SessionDiagnosticsReport `json:"diagnostics"`
}

const SessionOutcomeTransportVersion = 1

type SessionOutcomeTransportImportErrorCategory string

const (
	SessionOutcomeTransportKindMismatch       SessionOutcomeTransportImportErrorCategory = "kind_mismatch"
	SessionOutcomeTransportUnsupportedVersion SessionOutcomeTransportImportErrorCategory = "unsupported_version"
)

type SessionOutcomeTransportImportError struct {
	Category SessionOutcomeTransportImportErrorCategory `json:"category"`
	Message  string                                     `json:"message"`
}

type SessionOutcomeEnvelope struct {
	Kind    string               `json:"kind"`
	Version int                  `json:"version"`
	Outcome SessionOutcomeReport `json:"outcome"`
}

func SessionOutcomeEnvelopeFor(outcome SessionOutcomeReport) SessionOutcomeEnvelope {
	return SessionOutcomeEnvelope{
		Kind:    "template_directory_session_outcome",
		Version: SessionOutcomeTransportVersion,
		Outcome: outcome,
	}
}

func ImportSessionOutcomeEnvelope(
	envelope SessionOutcomeEnvelope,
) (*SessionOutcomeReport, *SessionOutcomeTransportImportError) {
	if envelope.Kind != "template_directory_session_outcome" {
		return nil, &SessionOutcomeTransportImportError{
			Category: SessionOutcomeTransportKindMismatch,
			Message:  "expected template_directory_session_outcome envelope kind.",
		}
	}

	if envelope.Version != SessionOutcomeTransportVersion {
		return nil, &SessionOutcomeTransportImportError{
			Category: SessionOutcomeTransportUnsupportedVersion,
			Message:  fmt.Sprintf("unsupported template_directory_session_outcome envelope version %d.", envelope.Version),
		}
	}

	outcome := envelope.Outcome
	return &outcome, nil
}

type SessionRequestReport struct {
	RequestKind     string                   `json:"request_kind"`
	ProfileName     string                   `json:"profile_name,omitempty"`
	Mode            DirectorySessionMode     `json:"mode"`
	Ready           bool                     `json:"ready"`
	Diagnostics     []SessionDiagnostic      `json:"diagnostics"`
	ResolvedOptions *DirectorySessionOptions `json:"resolved_options"`
}

const SessionRequestTransportVersion = 1

type SessionRequestTransportImportErrorCategory string

const (
	SessionRequestTransportKindMismatch       SessionRequestTransportImportErrorCategory = "kind_mismatch"
	SessionRequestTransportUnsupportedVersion SessionRequestTransportImportErrorCategory = "unsupported_version"
)

type SessionRequestTransportImportError struct {
	Category SessionRequestTransportImportErrorCategory `json:"category"`
	Message  string                                     `json:"message"`
}

type SessionRequestEnvelope struct {
	Kind    string               `json:"kind"`
	Version int                  `json:"version"`
	Request SessionRequestReport `json:"request"`
}

func SessionRequestEnvelopeFor(request SessionRequestReport) SessionRequestEnvelope {
	return SessionRequestEnvelope{
		Kind:    "template_directory_session_request",
		Version: SessionRequestTransportVersion,
		Request: request,
	}
}

func ImportSessionRequestEnvelope(
	envelope SessionRequestEnvelope,
) (*SessionRequestReport, *SessionRequestTransportImportError) {
	if envelope.Kind != "template_directory_session_request" {
		return nil, &SessionRequestTransportImportError{
			Category: SessionRequestTransportKindMismatch,
			Message:  "expected template_directory_session_request envelope kind.",
		}
	}

	if envelope.Version != SessionRequestTransportVersion {
		return nil, &SessionRequestTransportImportError{
			Category: SessionRequestTransportUnsupportedVersion,
			Message:  fmt.Sprintf("unsupported template_directory_session_request envelope version %d.", envelope.Version),
		}
	}

	request := envelope.Request
	return &request, nil
}

type SessionRunnerRequest struct {
	RequestKind string         `json:"request_kind"`
	ProfileName string         `json:"profile_name,omitempty"`
	Options     map[string]any `json:"options,omitempty"`
	Overrides   map[string]any `json:"overrides,omitempty"`
}

const SessionRunnerRequestTransportVersion = 1

type SessionRunnerRequestTransportImportErrorCategory string

const (
	SessionRunnerRequestTransportKindMismatch       SessionRunnerRequestTransportImportErrorCategory = "kind_mismatch"
	SessionRunnerRequestTransportUnsupportedVersion SessionRunnerRequestTransportImportErrorCategory = "unsupported_version"
)

type SessionRunnerRequestTransportImportError struct {
	Category SessionRunnerRequestTransportImportErrorCategory `json:"category"`
	Message  string                                           `json:"message"`
}

type SessionRunnerRequestEnvelope struct {
	Kind    string               `json:"kind"`
	Version int                  `json:"version"`
	Request SessionRunnerRequest `json:"request"`
}

func SessionRunnerRequestEnvelopeFor(request SessionRunnerRequest) SessionRunnerRequestEnvelope {
	return SessionRunnerRequestEnvelope{
		Kind:    "template_directory_session_runner_request",
		Version: SessionRunnerRequestTransportVersion,
		Request: request,
	}
}

func ImportSessionRunnerRequestEnvelope(
	envelope SessionRunnerRequestEnvelope,
) (*SessionRunnerRequest, *SessionRunnerRequestTransportImportError) {
	if envelope.Kind != "template_directory_session_runner_request" {
		return nil, &SessionRunnerRequestTransportImportError{
			Category: SessionRunnerRequestTransportKindMismatch,
			Message:  "expected template_directory_session_runner_request envelope kind.",
		}
	}

	if envelope.Version != SessionRunnerRequestTransportVersion {
		return nil, &SessionRunnerRequestTransportImportError{
			Category: SessionRunnerRequestTransportUnsupportedVersion,
			Message:  fmt.Sprintf("unsupported template_directory_session_runner_request envelope version %d.", envelope.Version),
		}
	}

	request := envelope.Request
	return &request, nil
}

type SessionRunnerInput struct {
	RequestKind     string                               `json:"request_kind"`
	ProfileName     string                               `json:"profile_name,omitempty"`
	Mode            DirectorySessionMode                 `json:"mode"`
	TemplateRoot    string                               `json:"template_root"`
	DestinationRoot string                               `json:"destination_root"`
	Context         *astmerge.TemplateDestinationContext `json:"context"`
	DefaultStrategy astmerge.TemplateStrategy            `json:"default_strategy"`
	Overrides       []astmerge.TemplateStrategyOverride  `json:"overrides"`
	Replacements    map[string]string                    `json:"replacements"`
	AllowedFamilies []string                             `json:"allowed_families"`
}

type SessionRunnerPayload struct {
	RequestKind        string                               `json:"request_kind,omitempty"`
	DefaultProfileName string                               `json:"default_profile_name,omitempty"`
	ProfileName        string                               `json:"profile_name,omitempty"`
	Mode               DirectorySessionMode                 `json:"mode"`
	TemplateRoot       string                               `json:"template_root"`
	DestinationRoot    string                               `json:"destination_root"`
	Context            *astmerge.TemplateDestinationContext `json:"context"`
	DefaultStrategy    astmerge.TemplateStrategy            `json:"default_strategy"`
	Overrides          []astmerge.TemplateStrategyOverride  `json:"overrides"`
	Replacements       map[string]string                    `json:"replacements"`
	AllowedFamilies    []string                             `json:"allowed_families"`
}

const SessionRunnerPayloadTransportVersion = 1

type SessionRunnerPayloadTransportImportErrorCategory string

const (
	SessionRunnerPayloadTransportKindMismatch       SessionRunnerPayloadTransportImportErrorCategory = "kind_mismatch"
	SessionRunnerPayloadTransportUnsupportedVersion SessionRunnerPayloadTransportImportErrorCategory = "unsupported_version"
)

type SessionRunnerPayloadTransportImportError struct {
	Category SessionRunnerPayloadTransportImportErrorCategory `json:"category"`
	Message  string                                           `json:"message"`
}

type SessionRunnerPayloadEnvelope struct {
	Kind    string               `json:"kind"`
	Version int                  `json:"version"`
	Payload SessionRunnerPayload `json:"payload"`
}

func SessionRunnerPayloadEnvelopeFor(payload SessionRunnerPayload) SessionRunnerPayloadEnvelope {
	return SessionRunnerPayloadEnvelope{
		Kind:    "template_directory_session_runner_payload",
		Version: SessionRunnerPayloadTransportVersion,
		Payload: payload,
	}
}

func ImportSessionRunnerPayloadEnvelope(
	envelope SessionRunnerPayloadEnvelope,
) (*SessionRunnerPayload, *SessionRunnerPayloadTransportImportError) {
	if envelope.Kind != "template_directory_session_runner_payload" {
		return nil, &SessionRunnerPayloadTransportImportError{
			Category: SessionRunnerPayloadTransportKindMismatch,
			Message:  "expected template_directory_session_runner_payload envelope kind.",
		}
	}

	if envelope.Version != SessionRunnerPayloadTransportVersion {
		return nil, &SessionRunnerPayloadTransportImportError{
			Category: SessionRunnerPayloadTransportUnsupportedVersion,
			Message:  fmt.Sprintf("unsupported template_directory_session_runner_payload envelope version %d.", envelope.Version),
		}
	}

	payload := envelope.Payload
	return &payload, nil
}

type SessionEntrypoint struct {
	Payload *SessionRunnerPayload `json:"payload,omitempty"`
	Request *SessionRunnerRequest `json:"request,omitempty"`
}

const SessionEntrypointTransportVersion = 1

type SessionEntrypointTransportImportErrorCategory string

const (
	SessionEntrypointTransportKindMismatch       SessionEntrypointTransportImportErrorCategory = "kind_mismatch"
	SessionEntrypointTransportUnsupportedVersion SessionEntrypointTransportImportErrorCategory = "unsupported_version"
)

type SessionEntrypointTransportImportError struct {
	Category SessionEntrypointTransportImportErrorCategory `json:"category"`
	Message  string                                        `json:"message"`
}

type SessionEntrypointEnvelope struct {
	Kind       string            `json:"kind"`
	Version    int               `json:"version"`
	Entrypoint SessionEntrypoint `json:"entrypoint"`
}

func SessionEntrypointEnvelopeFor(entrypoint SessionEntrypoint) SessionEntrypointEnvelope {
	return SessionEntrypointEnvelope{
		Kind:       "template_directory_session_entrypoint",
		Version:    SessionEntrypointTransportVersion,
		Entrypoint: entrypoint,
	}
}

func ImportSessionEntrypointEnvelope(
	envelope SessionEntrypointEnvelope,
) (*SessionEntrypoint, *SessionEntrypointTransportImportError) {
	if envelope.Kind != "template_directory_session_entrypoint" {
		return nil, &SessionEntrypointTransportImportError{
			Category: SessionEntrypointTransportKindMismatch,
			Message:  "expected template_directory_session_entrypoint envelope kind.",
		}
	}

	if envelope.Version != SessionEntrypointTransportVersion {
		return nil, &SessionEntrypointTransportImportError{
			Category: SessionEntrypointTransportUnsupportedVersion,
			Message:  fmt.Sprintf("unsupported template_directory_session_entrypoint envelope version %d.", envelope.Version),
		}
	}

	entrypoint := envelope.Entrypoint
	return &entrypoint, nil
}

type SessionEntrypointReport struct {
	SourceKind    string               `json:"source_kind"`
	RunnerRequest SessionRunnerRequest `json:"runner_request"`
}

type SessionResolutionReport struct {
	SourceKind     string               `json:"source_kind"`
	RunnerRequest  SessionRunnerRequest `json:"runner_request"`
	SessionRequest SessionRequestReport `json:"session_request"`
}

type SessionInspectionReport struct {
	EntrypointReport    SessionEntrypointReport  `json:"entrypoint_report"`
	SessionResolution   SessionResolutionReport  `json:"session_resolution"`
	AdapterCapabilities AdapterCapabilityReport  `json:"adapter_capabilities"`
	Status              SessionStatusReport      `json:"status"`
	Diagnostics         SessionDiagnosticsReport `json:"diagnostics"`
}

const SessionInspectionTransportVersion = 1

type SessionInspectionTransportImportErrorCategory string

const (
	SessionInspectionTransportKindMismatch       SessionInspectionTransportImportErrorCategory = "kind_mismatch"
	SessionInspectionTransportUnsupportedVersion SessionInspectionTransportImportErrorCategory = "unsupported_version"
)

type SessionInspectionTransportImportError struct {
	Category SessionInspectionTransportImportErrorCategory `json:"category"`
	Message  string                                        `json:"message"`
}

type SessionInspectionEnvelope struct {
	Kind       string                  `json:"kind"`
	Version    int                     `json:"version"`
	Inspection SessionInspectionReport `json:"inspection"`
}

func SessionInspectionEnvelopeFor(inspection SessionInspectionReport) SessionInspectionEnvelope {
	return SessionInspectionEnvelope{
		Kind:       "template_directory_session_inspection",
		Version:    SessionInspectionTransportVersion,
		Inspection: inspection,
	}
}

func ImportSessionInspectionEnvelope(
	envelope SessionInspectionEnvelope,
) (*SessionInspectionReport, *SessionInspectionTransportImportError) {
	if envelope.Kind != "template_directory_session_inspection" {
		return nil, &SessionInspectionTransportImportError{
			Category: SessionInspectionTransportKindMismatch,
			Message:  "expected template_directory_session_inspection envelope kind.",
		}
	}

	if envelope.Version != SessionInspectionTransportVersion {
		return nil, &SessionInspectionTransportImportError{
			Category: SessionInspectionTransportUnsupportedVersion,
			Message:  fmt.Sprintf("unsupported template_directory_session_inspection envelope version %d.", envelope.Version),
		}
	}

	inspection := envelope.Inspection
	return &inspection, nil
}

type SessionDispatchReport struct {
	Operation  string                   `json:"operation"`
	Inspection *SessionInspectionReport `json:"inspection"`
	Outcome    *SessionOutcomeReport    `json:"outcome"`
}

type SessionCommand struct {
	Operation string                `json:"operation"`
	Payload   *SessionRunnerPayload `json:"payload,omitempty"`
	Request   *SessionRunnerRequest `json:"request,omitempty"`
}

const SessionCommandTransportVersion = 1

type SessionCommandTransportImportErrorCategory string

const (
	SessionCommandTransportKindMismatch       SessionCommandTransportImportErrorCategory = "kind_mismatch"
	SessionCommandTransportUnsupportedVersion SessionCommandTransportImportErrorCategory = "unsupported_version"
)

type SessionCommandTransportImportError struct {
	Category SessionCommandTransportImportErrorCategory `json:"category"`
	Message  string                                     `json:"message"`
}

type SessionCommandEnvelope struct {
	Kind    string         `json:"kind"`
	Version int            `json:"version"`
	Command SessionCommand `json:"command"`
}

func SessionCommandEnvelopeFor(command SessionCommand) SessionCommandEnvelope {
	return SessionCommandEnvelope{
		Kind:    "template_directory_session_command",
		Version: SessionCommandTransportVersion,
		Command: command,
	}
}

func ImportSessionCommandEnvelope(
	envelope SessionCommandEnvelope,
) (*SessionCommand, *SessionCommandTransportImportError) {
	if envelope.Kind != "template_directory_session_command" {
		return nil, &SessionCommandTransportImportError{
			Category: SessionCommandTransportKindMismatch,
			Message:  "expected template_directory_session_command envelope kind.",
		}
	}

	if envelope.Version != SessionCommandTransportVersion {
		return nil, &SessionCommandTransportImportError{
			Category: SessionCommandTransportUnsupportedVersion,
			Message:  fmt.Sprintf("unsupported template_directory_session_command envelope version %d.", envelope.Version),
		}
	}

	command := envelope.Command
	return &command, nil
}

type SessionCommandPayload struct {
	Operation          string                               `json:"operation"`
	RequestKind        string                               `json:"request_kind,omitempty"`
	DefaultProfileName string                               `json:"default_profile_name,omitempty"`
	ProfileName        string                               `json:"profile_name,omitempty"`
	Mode               DirectorySessionMode                 `json:"mode"`
	TemplateRoot       string                               `json:"template_root"`
	DestinationRoot    string                               `json:"destination_root"`
	Context            *astmerge.TemplateDestinationContext `json:"context"`
	DefaultStrategy    astmerge.TemplateStrategy            `json:"default_strategy"`
	Overrides          []astmerge.TemplateStrategyOverride  `json:"overrides"`
	Replacements       map[string]string                    `json:"replacements"`
	AllowedFamilies    []string                             `json:"allowed_families"`
}

const SessionCommandPayloadTransportVersion = 1

type SessionCommandPayloadTransportImportErrorCategory string

const (
	SessionCommandPayloadTransportKindMismatch       SessionCommandPayloadTransportImportErrorCategory = "kind_mismatch"
	SessionCommandPayloadTransportUnsupportedVersion SessionCommandPayloadTransportImportErrorCategory = "unsupported_version"
)

type SessionCommandPayloadTransportImportError struct {
	Category SessionCommandPayloadTransportImportErrorCategory `json:"category"`
	Message  string                                            `json:"message"`
}

type SessionCommandPayloadEnvelope struct {
	Kind    string                `json:"kind"`
	Version int                   `json:"version"`
	Payload SessionCommandPayload `json:"payload"`
}

func SessionCommandPayloadEnvelopeFor(payload SessionCommandPayload) SessionCommandPayloadEnvelope {
	return SessionCommandPayloadEnvelope{
		Kind:    "template_directory_session_command_payload",
		Version: SessionCommandPayloadTransportVersion,
		Payload: payload,
	}
}

func ImportSessionCommandPayloadEnvelope(
	envelope SessionCommandPayloadEnvelope,
) (*SessionCommandPayload, *SessionCommandPayloadTransportImportError) {
	if envelope.Kind != "template_directory_session_command_payload" {
		return nil, &SessionCommandPayloadTransportImportError{
			Category: SessionCommandPayloadTransportKindMismatch,
			Message:  "expected template_directory_session_command_payload envelope kind.",
		}
	}

	if envelope.Version != SessionCommandPayloadTransportVersion {
		return nil, &SessionCommandPayloadTransportImportError{
			Category: SessionCommandPayloadTransportUnsupportedVersion,
			Message:  fmt.Sprintf("unsupported template_directory_session_command_payload envelope version %d.", envelope.Version),
		}
	}

	payload := envelope.Payload
	return &payload, nil
}

type SessionInvocation struct {
	Operation          string                               `json:"operation"`
	Payload            *SessionRunnerPayload                `json:"payload,omitempty"`
	Request            *SessionRunnerRequest                `json:"request,omitempty"`
	RequestKind        string                               `json:"request_kind,omitempty"`
	DefaultProfileName string                               `json:"default_profile_name,omitempty"`
	ProfileName        string                               `json:"profile_name,omitempty"`
	Mode               DirectorySessionMode                 `json:"mode,omitempty"`
	TemplateRoot       string                               `json:"template_root,omitempty"`
	DestinationRoot    string                               `json:"destination_root,omitempty"`
	Context            *astmerge.TemplateDestinationContext `json:"context,omitempty"`
	DefaultStrategy    astmerge.TemplateStrategy            `json:"default_strategy,omitempty"`
	Overrides          []astmerge.TemplateStrategyOverride  `json:"overrides,omitempty"`
	Replacements       map[string]string                    `json:"replacements,omitempty"`
	AllowedFamilies    []string                             `json:"allowed_families"`
}

const SessionInvocationTransportVersion = 1

type SessionInvocationTransportImportErrorCategory string

const (
	SessionInvocationTransportKindMismatch       SessionInvocationTransportImportErrorCategory = "kind_mismatch"
	SessionInvocationTransportUnsupportedVersion SessionInvocationTransportImportErrorCategory = "unsupported_version"
)

type SessionInvocationTransportImportError struct {
	Category SessionInvocationTransportImportErrorCategory `json:"category"`
	Message  string                                        `json:"message"`
}

type SessionInvocationEnvelope struct {
	Kind       string            `json:"kind"`
	Version    int               `json:"version"`
	Invocation SessionInvocation `json:"invocation"`
}

func SessionInvocationEnvelopeFor(invocation SessionInvocation) SessionInvocationEnvelope {
	return SessionInvocationEnvelope{
		Kind:       "template_directory_session_invocation",
		Version:    SessionInvocationTransportVersion,
		Invocation: invocation,
	}
}

func ImportSessionInvocationEnvelope(
	envelope SessionInvocationEnvelope,
) (*SessionInvocation, *SessionInvocationTransportImportError) {
	if envelope.Kind != "template_directory_session_invocation" {
		return nil, &SessionInvocationTransportImportError{
			Category: SessionInvocationTransportKindMismatch,
			Message:  "expected template_directory_session_invocation envelope kind.",
		}
	}

	if envelope.Version != SessionInvocationTransportVersion {
		return nil, &SessionInvocationTransportImportError{
			Category: SessionInvocationTransportUnsupportedVersion,
			Message:  fmt.Sprintf("unsupported template_directory_session_invocation envelope version %d.", envelope.Version),
		}
	}

	invocation := envelope.Invocation
	return &invocation, nil
}

type DirectorySessionOptions struct {
	Mode            DirectorySessionMode                 `json:"mode"`
	TemplateRoot    string                               `json:"template_root"`
	DestinationRoot string                               `json:"destination_root"`
	Context         *astmerge.TemplateDestinationContext `json:"context"`
	DefaultStrategy astmerge.TemplateStrategy            `json:"default_strategy"`
	Overrides       []astmerge.TemplateStrategyOverride  `json:"overrides"`
	Replacements    map[string]string                    `json:"replacements"`
	AllowedFamilies []string                             `json:"allowed_families"`
	Config          *astmerge.TemplateTokenConfig        `json:"config,omitempty"`
}

type DirectorySessionProfile struct {
	Mode            DirectorySessionMode                 `json:"mode"`
	Context         *astmerge.TemplateDestinationContext `json:"context"`
	DefaultStrategy astmerge.TemplateStrategy            `json:"default_strategy"`
	Overrides       []astmerge.TemplateStrategyOverride  `json:"overrides"`
	Replacements    map[string]string                    `json:"replacements"`
	AllowedFamilies []string                             `json:"allowed_families"`
	Config          *astmerge.TemplateTokenConfig        `json:"config,omitempty"`
}

func ReportTemplateDirectorySession(mode DirectorySessionMode, entries []astmerge.TemplateExecutionPlanEntry, result *astmerge.TemplateTreeRunResult) DirectorySessionReport {
	return DirectorySessionReport{
		Mode:         mode,
		RunnerReport: astmerge.ReportTemplateDirectoryRunner(entries, result),
	}
}

func PlanTemplateDirectorySessionFromDirectories(
	templateRoot string,
	destinationRoot string,
	context *astmerge.TemplateDestinationContext,
	defaultStrategy astmerge.TemplateStrategy,
	overrides []astmerge.TemplateStrategyOverride,
	replacements map[string]string,
	config *astmerge.TemplateTokenConfig,
) (DirectorySessionReport, error) {
	plan, err := astmerge.PlanTemplateTreeExecutionFromDirectories(
		templateRoot,
		destinationRoot,
		context,
		defaultStrategy,
		overrides,
		replacements,
		config,
	)
	if err != nil {
		return DirectorySessionReport{}, err
	}
	return ReportTemplateDirectorySession(DirectorySessionModePlan, plan, nil), nil
}

func ApplyTemplateDirectorySessionToDirectory(
	templateRoot string,
	destinationRoot string,
	context *astmerge.TemplateDestinationContext,
	defaultStrategy astmerge.TemplateStrategy,
	overrides []astmerge.TemplateStrategyOverride,
	replacements map[string]string,
	mergePreparedContent func(astmerge.TemplateExecutionPlanEntry) astmerge.MergeResult[string],
	config *astmerge.TemplateTokenConfig,
) (DirectorySessionReport, error) {
	result, err := astmerge.ApplyTemplateTreeExecutionToDirectory(
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
		return DirectorySessionReport{}, err
	}
	return ReportTemplateDirectorySession(DirectorySessionModeApply, result.ExecutionPlan, &result), nil
}

func ReapplyTemplateDirectorySessionToDirectory(
	templateRoot string,
	destinationRoot string,
	context *astmerge.TemplateDestinationContext,
	defaultStrategy astmerge.TemplateStrategy,
	overrides []astmerge.TemplateStrategyOverride,
	replacements map[string]string,
	mergePreparedContent func(astmerge.TemplateExecutionPlanEntry) astmerge.MergeResult[string],
	config *astmerge.TemplateTokenConfig,
) (DirectorySessionReport, error) {
	result, err := astmerge.ApplyTemplateTreeExecutionToDirectory(
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
		return DirectorySessionReport{}, err
	}
	return ReportTemplateDirectorySession(DirectorySessionModeReapply, result.ExecutionPlan, &result), nil
}

func MergePreparedContentFromRegistry(registry FamilyMergeAdapterRegistry, entry astmerge.TemplateExecutionPlanEntry) astmerge.MergeResult[string] {
	family := entry.Classification.Family
	adapter, ok := registry[family]
	if !ok {
		return astmerge.MergeResult[string]{
			OK: false,
			Diagnostics: []astmerge.Diagnostic{{
				Severity: astmerge.SeverityError,
				Category: astmerge.CategoryConfigurationError,
				Message:  "missing family adapter for " + family,
			}},
		}
	}
	return adapter(entry)
}

func RegisteredAdapterFamilies(registry FamilyMergeAdapterRegistry) []string {
	families := make([]string, 0, len(registry))
	for family := range registry {
		families = append(families, family)
	}
	slices.Sort(families)
	return families
}

func ReportTemplateDirectoryRegistrySession(mode DirectorySessionMode, entries []astmerge.TemplateExecutionPlanEntry, result *astmerge.TemplateTreeRunResult, registry FamilyMergeAdapterRegistry) DirectoryRegistrySessionReport {
	diagnostics := []astmerge.Diagnostic{}
	if result != nil {
		diagnostics = append(diagnostics, result.ApplyResult.Diagnostics...)
	}
	return DirectoryRegistrySessionReport{
		Mode:            mode,
		AdapterFamilies: RegisteredAdapterFamilies(registry),
		Diagnostics:     diagnostics,
		RunnerReport:    astmerge.ReportTemplateDirectoryRunner(entries, result),
	}
}

func ApplyTemplateDirectorySessionWithRegistryToDirectory(
	templateRoot string,
	destinationRoot string,
	context *astmerge.TemplateDestinationContext,
	defaultStrategy astmerge.TemplateStrategy,
	overrides []astmerge.TemplateStrategyOverride,
	replacements map[string]string,
	registry FamilyMergeAdapterRegistry,
	config *astmerge.TemplateTokenConfig,
) (DirectoryRegistrySessionReport, error) {
	result, err := astmerge.ApplyTemplateTreeExecutionToDirectory(
		templateRoot,
		destinationRoot,
		context,
		defaultStrategy,
		overrides,
		replacements,
		func(entry astmerge.TemplateExecutionPlanEntry) astmerge.MergeResult[string] {
			return MergePreparedContentFromRegistry(registry, entry)
		},
		config,
	)
	if err != nil {
		return DirectoryRegistrySessionReport{}, err
	}
	return ReportTemplateDirectoryRegistrySession(DirectorySessionModeApply, result.ExecutionPlan, &result, registry), nil
}

func DefaultFamilyMergeAdapterRegistry(allowedFamilies ...string) FamilyMergeAdapterRegistry {
	allowed := map[string]struct{}{}
	for _, family := range allowedFamilies {
		allowed[family] = struct{}{}
	}
	include := func(family string) bool {
		if len(allowed) == 0 {
			return true
		}
		_, ok := allowed[family]
		return ok
	}

	registry := FamilyMergeAdapterRegistry{}
	if include("markdown") {
		registry["markdown"] = func(entry astmerge.TemplateExecutionPlanEntry) astmerge.MergeResult[string] {
			template := ""
			if entry.PreparedTemplateContent != nil {
				template = *entry.PreparedTemplateContent
			}
			destination := ""
			if entry.DestinationContent != nil {
				destination = *entry.DestinationContent
			}
			return markdownmerge.MergeMarkdown(template, destination, "markdown")
		}
	}
	if include("toml") {
		registry["toml"] = func(entry astmerge.TemplateExecutionPlanEntry) astmerge.MergeResult[string] {
			template := ""
			if entry.PreparedTemplateContent != nil {
				template = *entry.PreparedTemplateContent
			}
			destination := ""
			if entry.DestinationContent != nil {
				destination = *entry.DestinationContent
			}
			return tomlmerge.MergeTOML(template, destination, "toml")
		}
	}
	if include("ruby") {
		registry["ruby"] = func(entry astmerge.TemplateExecutionPlanEntry) astmerge.MergeResult[string] {
			template := ""
			if entry.PreparedTemplateContent != nil {
				template = *entry.PreparedTemplateContent
			}
			destination := ""
			if entry.DestinationContent != nil {
				destination = *entry.DestinationContent
			}
			return rubymerge.MergeRuby(template, destination, "ruby")
		}
	}
	return registry
}

func ApplyTemplateDirectorySessionWithDefaultRegistryToDirectory(
	templateRoot string,
	destinationRoot string,
	context *astmerge.TemplateDestinationContext,
	defaultStrategy astmerge.TemplateStrategy,
	overrides []astmerge.TemplateStrategyOverride,
	replacements map[string]string,
	allowedFamilies []string,
	config *astmerge.TemplateTokenConfig,
) (DirectoryRegistrySessionReport, error) {
	return ApplyTemplateDirectorySessionWithRegistryToDirectory(
		templateRoot,
		destinationRoot,
		context,
		defaultStrategy,
		overrides,
		replacements,
		DefaultFamilyMergeAdapterRegistry(allowedFamilies...),
		config,
	)
}

func RequiredFamilies(entries []astmerge.TemplateExecutionPlanEntry) []string {
	families := map[string]struct{}{}
	for _, entry := range entries {
		if entry.ExecutionAction != astmerge.TemplateExecutionMergePrepared {
			continue
		}
		families[entry.Classification.Family] = struct{}{}
	}
	required := make([]string, 0, len(families))
	for family := range families {
		required = append(required, family)
	}
	slices.Sort(required)
	return required
}

func ReportAdapterCapabilities(entries []astmerge.TemplateExecutionPlanEntry, registry FamilyMergeAdapterRegistry) AdapterCapabilityReport {
	required := RequiredFamilies(entries)
	available := RegisteredAdapterFamilies(registry)
	availableSet := map[string]struct{}{}
	for _, family := range available {
		availableSet[family] = struct{}{}
	}
	missing := []string{}
	for _, family := range required {
		if _, ok := availableSet[family]; !ok {
			missing = append(missing, family)
		}
	}
	return AdapterCapabilityReport{
		RequiredFamilies: required,
		AdapterFamilies:  available,
		MissingFamilies:  missing,
		Ready:            len(missing) == 0,
	}
}

func ReportAdapterCapabilitiesFromDirectories(
	templateRoot string,
	destinationRoot string,
	context *astmerge.TemplateDestinationContext,
	defaultStrategy astmerge.TemplateStrategy,
	overrides []astmerge.TemplateStrategyOverride,
	replacements map[string]string,
	registry FamilyMergeAdapterRegistry,
	config *astmerge.TemplateTokenConfig,
) (AdapterCapabilityReport, error) {
	plan, err := astmerge.PlanTemplateTreeExecutionFromDirectories(
		templateRoot,
		destinationRoot,
		context,
		defaultStrategy,
		overrides,
		replacements,
		config,
	)
	if err != nil {
		return AdapterCapabilityReport{}, err
	}
	return ReportAdapterCapabilities(plan, registry), nil
}

func ReportDefaultAdapterCapabilitiesFromDirectories(
	templateRoot string,
	destinationRoot string,
	context *astmerge.TemplateDestinationContext,
	defaultStrategy astmerge.TemplateStrategy,
	overrides []astmerge.TemplateStrategyOverride,
	replacements map[string]string,
	allowedFamilies []string,
	config *astmerge.TemplateTokenConfig,
) (AdapterCapabilityReport, error) {
	return ReportAdapterCapabilitiesFromDirectories(
		templateRoot,
		destinationRoot,
		context,
		defaultStrategy,
		overrides,
		replacements,
		DefaultFamilyMergeAdapterRegistry(allowedFamilies...),
		config,
	)
}

func ReportTemplateDirectorySessionEnvelope(
	sessionReport any,
	adapterCapabilities AdapterCapabilityReport,
) SessionEnvelopeReport {
	return SessionEnvelopeReport{
		SessionReport:       sessionReport,
		AdapterCapabilities: adapterCapabilities,
	}
}

func PlanTemplateDirectorySessionEnvelopeFromDirectories(
	templateRoot string,
	destinationRoot string,
	context *astmerge.TemplateDestinationContext,
	defaultStrategy astmerge.TemplateStrategy,
	overrides []astmerge.TemplateStrategyOverride,
	replacements map[string]string,
	allowedFamilies []string,
	config *astmerge.TemplateTokenConfig,
) (SessionEnvelopeReport, error) {
	sessionReport, err := PlanTemplateDirectorySessionFromDirectories(
		templateRoot,
		destinationRoot,
		context,
		defaultStrategy,
		overrides,
		replacements,
		config,
	)
	if err != nil {
		return SessionEnvelopeReport{}, err
	}
	capabilities, err := ReportDefaultAdapterCapabilitiesFromDirectories(
		templateRoot,
		destinationRoot,
		context,
		defaultStrategy,
		overrides,
		replacements,
		allowedFamilies,
		config,
	)
	if err != nil {
		return SessionEnvelopeReport{}, err
	}
	return ReportTemplateDirectorySessionEnvelope(sessionReport, capabilities), nil
}

func ApplyTemplateDirectorySessionEnvelopeWithDefaultRegistryToDirectory(
	templateRoot string,
	destinationRoot string,
	context *astmerge.TemplateDestinationContext,
	defaultStrategy astmerge.TemplateStrategy,
	overrides []astmerge.TemplateStrategyOverride,
	replacements map[string]string,
	allowedFamilies []string,
	config *astmerge.TemplateTokenConfig,
) (SessionEnvelopeReport, error) {
	sessionReport, err := ApplyTemplateDirectorySessionWithDefaultRegistryToDirectory(
		templateRoot,
		destinationRoot,
		context,
		defaultStrategy,
		overrides,
		replacements,
		allowedFamilies,
		config,
	)
	if err != nil {
		return SessionEnvelopeReport{}, err
	}
	capabilities, err := ReportDefaultAdapterCapabilitiesFromDirectories(
		templateRoot,
		destinationRoot,
		context,
		defaultStrategy,
		overrides,
		replacements,
		allowedFamilies,
		config,
	)
	if err != nil {
		return SessionEnvelopeReport{}, err
	}
	return ReportTemplateDirectorySessionEnvelope(sessionReport, capabilities), nil
}

func ReportTemplateDirectorySessionStatus(envelope SessionEnvelopeReport) SessionStatusReport {
	mode, runnerReport := sessionEnvelopeModeAndRunner(envelope.SessionReport)
	blockedPathSet := map[string]struct{}{}
	for _, entry := range runnerReport.PlanReport.Entries {
		if entry.Status == astmerge.TemplateDirectoryPlanBlocked && entry.DestinationPath != nil {
			blockedPathSet[*entry.DestinationPath] = struct{}{}
		}
	}
	if runnerReport.ApplyReport != nil {
		for _, entry := range runnerReport.ApplyReport.Entries {
			if entry.Status == astmerge.TemplateTreeRunBlocked && entry.DestinationPath != nil {
				blockedPathSet[*entry.DestinationPath] = struct{}{}
			}
		}
	}
	blockedPaths := make([]string, 0, len(blockedPathSet))
	for path := range blockedPathSet {
		blockedPaths = append(blockedPaths, path)
	}
	slices.Sort(blockedPaths)
	plannedWriteCount := runnerReport.PlanReport.Summary.Create + runnerReport.PlanReport.Summary.Update
	writtenCount := 0
	if runnerReport.ApplyReport != nil {
		writtenCount = runnerReport.ApplyReport.Summary.Written
	}
	missingFamilies := append([]string{}, envelope.AdapterCapabilities.MissingFamilies...)
	slices.Sort(missingFamilies)
	return SessionStatusReport{
		Mode:              mode,
		Ready:             envelope.AdapterCapabilities.Ready && len(blockedPaths) == 0,
		MissingFamilies:   missingFamilies,
		BlockedPaths:      blockedPaths,
		PlannedWriteCount: plannedWriteCount,
		WrittenCount:      writtenCount,
	}
}

func sessionEnvelopeModeAndRunner(sessionReport any) (DirectorySessionMode, astmerge.TemplateDirectoryRunnerReport) {
	switch report := sessionReport.(type) {
	case DirectorySessionReport:
		return report.Mode, report.RunnerReport
	case *DirectorySessionReport:
		return report.Mode, report.RunnerReport
	case DirectoryRegistrySessionReport:
		return report.Mode, report.RunnerReport
	case *DirectoryRegistrySessionReport:
		return report.Mode, report.RunnerReport
	default:
		return DirectorySessionModePlan, astmerge.TemplateDirectoryRunnerReport{}
	}
}

func ReportTemplateDirectorySessionDiagnostics(
	mode DirectorySessionMode,
	entries []astmerge.TemplateExecutionPlanEntry,
	result *astmerge.TemplateTreeRunResult,
	capabilities AdapterCapabilityReport,
) SessionDiagnosticsReport {
	diagnostics := []SessionDiagnostic{}
	missingFamilies := map[string]struct{}{}
	for _, family := range capabilities.MissingFamilies {
		missingFamilies[family] = struct{}{}
	}
	blockedByPath := map[string]struct{}{}
	if result != nil {
		for _, path := range result.ApplyResult.BlockedPaths {
			blockedByPath[path] = struct{}{}
		}
	}
	for _, entry := range entries {
		path := entry.LogicalDestinationPath
		if entry.DestinationPath != nil {
			path = *entry.DestinationPath
		}
		if entry.Blocked && entry.BlockReason != nil && *entry.BlockReason == astmerge.TemplatePlanBlockReasonUnresolvedTokens {
			diagnostics = append(diagnostics, SessionDiagnostic{
				Severity: astmerge.SeverityError,
				Category: astmerge.CategoryConfigurationError,
				Reason:   "unresolved_tokens",
				Path:     path,
				Message:  "unresolved template tokens block " + path,
			})
		}
		if _, ok := missingFamilies[entry.Classification.Family]; ok &&
			entry.ExecutionAction == astmerge.TemplateExecutionMergePrepared {
			if result == nil || len(blockedByPath) == 0 {
				diagnostics = append(diagnostics, SessionDiagnostic{
					Severity: astmerge.SeverityError,
					Category: astmerge.CategoryConfigurationError,
					Reason:   "missing_family_adapter",
					Path:     path,
					Family:   entry.Classification.Family,
					Message:  "missing family adapter for " + entry.Classification.Family + " blocks " + path,
				})
				continue
			}
			if _, blocked := blockedByPath[path]; blocked {
				diagnostics = append(diagnostics, SessionDiagnostic{
					Severity: astmerge.SeverityError,
					Category: astmerge.CategoryConfigurationError,
					Reason:   "missing_family_adapter",
					Path:     path,
					Family:   entry.Classification.Family,
					Message:  "missing family adapter for " + entry.Classification.Family + " blocks " + path,
				})
			}
		}
	}
	slices.SortFunc(diagnostics, func(a, b SessionDiagnostic) int {
		if a.Path != b.Path {
			return strings.Compare(a.Path, b.Path)
		}
		if a.Reason != b.Reason {
			return strings.Compare(a.Reason, b.Reason)
		}
		return strings.Compare(a.Family, b.Family)
	})
	return SessionDiagnosticsReport{
		Mode:        mode,
		Ready:       len(diagnostics) == 0,
		Diagnostics: diagnostics,
	}
}

func PlanTemplateDirectorySessionDiagnosticsFromDirectories(
	templateRoot string,
	destinationRoot string,
	context *astmerge.TemplateDestinationContext,
	defaultStrategy astmerge.TemplateStrategy,
	overrides []astmerge.TemplateStrategyOverride,
	replacements map[string]string,
	allowedFamilies []string,
	config *astmerge.TemplateTokenConfig,
) (SessionDiagnosticsReport, error) {
	entries, err := astmerge.PlanTemplateTreeExecutionFromDirectories(
		templateRoot,
		destinationRoot,
		context,
		defaultStrategy,
		overrides,
		replacements,
		config,
	)
	if err != nil {
		return SessionDiagnosticsReport{}, err
	}
	capabilities := ReportAdapterCapabilities(entries, DefaultFamilyMergeAdapterRegistry(allowedFamilies...))
	return ReportTemplateDirectorySessionDiagnostics(DirectorySessionModePlan, entries, nil, capabilities), nil
}

func ApplyTemplateDirectorySessionDiagnosticsWithDefaultRegistryToDirectory(
	templateRoot string,
	destinationRoot string,
	context *astmerge.TemplateDestinationContext,
	defaultStrategy astmerge.TemplateStrategy,
	overrides []astmerge.TemplateStrategyOverride,
	replacements map[string]string,
	allowedFamilies []string,
	config *astmerge.TemplateTokenConfig,
) (SessionDiagnosticsReport, error) {
	registry := DefaultFamilyMergeAdapterRegistry(allowedFamilies...)
	result, err := astmerge.ApplyTemplateTreeExecutionToDirectory(
		templateRoot,
		destinationRoot,
		context,
		defaultStrategy,
		overrides,
		replacements,
		func(entry astmerge.TemplateExecutionPlanEntry) astmerge.MergeResult[string] {
			return MergePreparedContentFromRegistry(registry, entry)
		},
		config,
	)
	if err != nil {
		return SessionDiagnosticsReport{}, err
	}
	capabilities := ReportAdapterCapabilities(result.ExecutionPlan, registry)
	return ReportTemplateDirectorySessionDiagnostics(DirectorySessionModeApply, result.ExecutionPlan, &result, capabilities), nil
}

func ReportTemplateDirectorySessionOutcome(
	sessionReport any,
	status SessionStatusReport,
	diagnostics SessionDiagnosticsReport,
) SessionOutcomeReport {
	return SessionOutcomeReport{
		SessionReport: sessionReport,
		Status:        status,
		Diagnostics:   diagnostics,
	}
}

func PlanTemplateDirectorySessionOutcomeFromDirectories(
	templateRoot string,
	destinationRoot string,
	context *astmerge.TemplateDestinationContext,
	defaultStrategy astmerge.TemplateStrategy,
	overrides []astmerge.TemplateStrategyOverride,
	replacements map[string]string,
	allowedFamilies []string,
	config *astmerge.TemplateTokenConfig,
) (SessionOutcomeReport, error) {
	sessionReport, err := PlanTemplateDirectorySessionFromDirectories(
		templateRoot,
		destinationRoot,
		context,
		defaultStrategy,
		overrides,
		replacements,
		config,
	)
	if err != nil {
		return SessionOutcomeReport{}, err
	}
	envelope, err := PlanTemplateDirectorySessionEnvelopeFromDirectories(
		templateRoot,
		destinationRoot,
		context,
		defaultStrategy,
		overrides,
		replacements,
		allowedFamilies,
		config,
	)
	if err != nil {
		return SessionOutcomeReport{}, err
	}
	diagnostics, err := PlanTemplateDirectorySessionDiagnosticsFromDirectories(
		templateRoot,
		destinationRoot,
		context,
		defaultStrategy,
		overrides,
		replacements,
		allowedFamilies,
		config,
	)
	if err != nil {
		return SessionOutcomeReport{}, err
	}
	return ReportTemplateDirectorySessionOutcome(
		sessionReport,
		ReportTemplateDirectorySessionStatus(envelope),
		diagnostics,
	), nil
}

func ApplyTemplateDirectorySessionOutcomeWithDefaultRegistryToDirectory(
	templateRoot string,
	destinationRoot string,
	context *astmerge.TemplateDestinationContext,
	defaultStrategy astmerge.TemplateStrategy,
	overrides []astmerge.TemplateStrategyOverride,
	replacements map[string]string,
	allowedFamilies []string,
	config *astmerge.TemplateTokenConfig,
) (SessionOutcomeReport, error) {
	registry := DefaultFamilyMergeAdapterRegistry(allowedFamilies...)
	result, err := astmerge.ApplyTemplateTreeExecutionToDirectory(
		templateRoot,
		destinationRoot,
		context,
		defaultStrategy,
		overrides,
		replacements,
		func(entry astmerge.TemplateExecutionPlanEntry) astmerge.MergeResult[string] {
			return MergePreparedContentFromRegistry(registry, entry)
		},
		config,
	)
	if err != nil {
		return SessionOutcomeReport{}, err
	}
	sessionReport := ReportTemplateDirectoryRegistrySession(DirectorySessionModeApply, result.ExecutionPlan, &result, registry)
	capabilities := ReportAdapterCapabilities(result.ExecutionPlan, registry)
	status := ReportTemplateDirectorySessionStatus(ReportTemplateDirectorySessionEnvelope(sessionReport, capabilities))
	diagnostics := ReportTemplateDirectorySessionDiagnostics(DirectorySessionModeApply, result.ExecutionPlan, &result, capabilities)
	return ReportTemplateDirectorySessionOutcome(
		sessionReport,
		status,
		diagnostics,
	), nil
}

func ReapplyTemplateDirectorySessionOutcomeWithDefaultRegistryToDirectory(
	templateRoot string,
	destinationRoot string,
	context *astmerge.TemplateDestinationContext,
	defaultStrategy astmerge.TemplateStrategy,
	overrides []astmerge.TemplateStrategyOverride,
	replacements map[string]string,
	allowedFamilies []string,
	config *astmerge.TemplateTokenConfig,
) (SessionOutcomeReport, error) {
	registry := DefaultFamilyMergeAdapterRegistry(allowedFamilies...)
	result, err := astmerge.ApplyTemplateTreeExecutionToDirectory(
		templateRoot,
		destinationRoot,
		context,
		defaultStrategy,
		overrides,
		replacements,
		func(entry astmerge.TemplateExecutionPlanEntry) astmerge.MergeResult[string] {
			return MergePreparedContentFromRegistry(registry, entry)
		},
		config,
	)
	if err != nil {
		return SessionOutcomeReport{}, err
	}
	sessionReport := ReportTemplateDirectoryRegistrySession(DirectorySessionModeReapply, result.ExecutionPlan, &result, registry)
	capabilities := ReportAdapterCapabilities(result.ExecutionPlan, registry)
	status := ReportTemplateDirectorySessionStatus(ReportTemplateDirectorySessionEnvelope(sessionReport, capabilities))
	diagnostics := ReportTemplateDirectorySessionDiagnostics(DirectorySessionModeReapply, result.ExecutionPlan, &result, capabilities)
	return ReportTemplateDirectorySessionOutcome(
		sessionReport,
		status,
		diagnostics,
	), nil
}

func RunTemplateDirectorySessionWithDefaultRegistryToDirectory(
	mode DirectorySessionMode,
	templateRoot string,
	destinationRoot string,
	context *astmerge.TemplateDestinationContext,
	defaultStrategy astmerge.TemplateStrategy,
	overrides []astmerge.TemplateStrategyOverride,
	replacements map[string]string,
	allowedFamilies []string,
	config *astmerge.TemplateTokenConfig,
) (SessionOutcomeReport, error) {
	switch mode {
	case DirectorySessionModePlan:
		return PlanTemplateDirectorySessionOutcomeFromDirectories(
			templateRoot,
			destinationRoot,
			context,
			defaultStrategy,
			overrides,
			replacements,
			allowedFamilies,
			config,
		)
	case DirectorySessionModeApply:
		return ApplyTemplateDirectorySessionOutcomeWithDefaultRegistryToDirectory(
			templateRoot,
			destinationRoot,
			context,
			defaultStrategy,
			overrides,
			replacements,
			allowedFamilies,
			config,
		)
	case DirectorySessionModeReapply:
		return ReapplyTemplateDirectorySessionOutcomeWithDefaultRegistryToDirectory(
			templateRoot,
			destinationRoot,
			context,
			defaultStrategy,
			overrides,
			replacements,
			allowedFamilies,
			config,
		)
	default:
		return SessionOutcomeReport{}, nil
	}
}

func RunTemplateDirectorySessionWithOptions(options DirectorySessionOptions) (SessionOutcomeReport, error) {
	request := ReportTemplateDirectorySessionOptionsRequest(options)
	if !request.Ready {
		return reportTemplateDirectorySessionConfigurationOutcome(request.Mode, SessionDiagnosticsReport{
			Mode:        request.Mode,
			Ready:       request.Ready,
			Diagnostics: request.Diagnostics,
		}), nil
	}
	resolved := *request.ResolvedOptions
	return RunTemplateDirectorySessionWithDefaultRegistryToDirectory(
		resolved.Mode,
		resolved.TemplateRoot,
		resolved.DestinationRoot,
		resolved.Context,
		resolved.DefaultStrategy,
		resolved.Overrides,
		resolved.Replacements,
		resolved.AllowedFamilies,
		resolved.Config,
	)
}

func normalizeSessionMode(mode DirectorySessionMode) DirectorySessionMode {
	switch mode {
	case DirectorySessionModeApply, DirectorySessionModeReapply:
		return mode
	default:
		return DirectorySessionModePlan
	}
}

func ReportTemplateDirectorySessionOptionsConfiguration(options DirectorySessionOptions) SessionDiagnosticsReport {
	diagnostics := []SessionDiagnostic{}
	if options.DestinationRoot == "" {
		diagnostics = append(diagnostics, SessionDiagnostic{
			Severity: astmerge.SeverityError,
			Category: astmerge.CategoryConfigurationError,
			Reason:   "missing_destination_root",
			Message:  "missing destination_root for template session",
		})
	}
	if options.TemplateRoot == "" {
		diagnostics = append(diagnostics, SessionDiagnostic{
			Severity: astmerge.SeverityError,
			Category: astmerge.CategoryConfigurationError,
			Reason:   "missing_template_root",
			Message:  "missing template_root for template session",
		})
	}
	slices.SortFunc(diagnostics, func(a, b SessionDiagnostic) int {
		return strings.Compare(a.Reason, b.Reason)
	})
	return SessionDiagnosticsReport{
		Mode:        normalizeSessionMode(options.Mode),
		Ready:       len(diagnostics) == 0,
		Diagnostics: diagnostics,
	}
}

func ReportTemplateDirectorySessionProfileConfiguration(
	profiles map[string]DirectorySessionProfile,
	profileName string,
	overrides DirectorySessionOptions,
) SessionDiagnosticsReport {
	report := ReportTemplateDirectorySessionOptionsConfiguration(overrides)
	if profile, ok := profiles[profileName]; ok {
		report.Mode = normalizeSessionMode(firstNonEmptySessionMode(overrides.Mode, profile.Mode))
	} else {
		report.Diagnostics = append(report.Diagnostics, SessionDiagnostic{
			Severity: astmerge.SeverityError,
			Category: astmerge.CategoryConfigurationError,
			Reason:   "missing_profile",
			Message:  "unknown template session profile: " + profileName,
		})
		report.Mode = normalizeSessionMode(overrides.Mode)
	}
	slices.SortFunc(report.Diagnostics, func(a, b SessionDiagnostic) int {
		return strings.Compare(a.Reason, b.Reason)
	})
	report.Ready = len(report.Diagnostics) == 0
	return report
}

func ReportTemplateDirectorySessionOptionsRequest(options DirectorySessionOptions) SessionRequestReport {
	configuration := ReportTemplateDirectorySessionOptionsConfiguration(options)
	var resolved *DirectorySessionOptions
	if configuration.Ready {
		copy := options
		resolved = &copy
	}
	return SessionRequestReport{
		RequestKind:     "options",
		Mode:            configuration.Mode,
		Ready:           configuration.Ready,
		Diagnostics:     configuration.Diagnostics,
		ResolvedOptions: resolved,
	}
}

func ReportTemplateDirectorySessionProfileRequest(
	profiles map[string]DirectorySessionProfile,
	profileName string,
	overrides DirectorySessionOptions,
) SessionRequestReport {
	configuration := ReportTemplateDirectorySessionProfileConfiguration(profiles, profileName, overrides)
	var resolved *DirectorySessionOptions
	if configuration.Ready {
		if options, ok := ResolveTemplateDirectorySessionOptions(profiles, profileName, overrides); ok {
			copy := options
			resolved = &copy
		}
	}
	return SessionRequestReport{
		RequestKind:     "profile",
		ProfileName:     profileName,
		Mode:            configuration.Mode,
		Ready:           configuration.Ready,
		Diagnostics:     configuration.Diagnostics,
		ResolvedOptions: resolved,
	}
}

func reportTemplateDirectorySessionConfigurationOutcome(
	mode DirectorySessionMode,
	diagnostics SessionDiagnosticsReport,
) SessionOutcomeReport {
	return ReportTemplateDirectorySessionOutcome(
		ReportTemplateDirectorySession(mode, nil, nil),
		SessionStatusReport{
			Mode:              mode,
			Ready:             false,
			MissingFamilies:   []string{},
			BlockedPaths:      []string{},
			PlannedWriteCount: 0,
			WrittenCount:      0,
		},
		diagnostics,
	)
}

func RunTemplateDirectorySessionRequest(request SessionRequestReport) (SessionOutcomeReport, error) {
	if !request.Ready {
		return reportTemplateDirectorySessionConfigurationOutcome(request.Mode, SessionDiagnosticsReport{
			Mode:        request.Mode,
			Ready:       request.Ready,
			Diagnostics: request.Diagnostics,
		}), nil
	}
	if request.ResolvedOptions == nil {
		return reportTemplateDirectorySessionConfigurationOutcome(request.Mode, SessionDiagnosticsReport{
			Mode:  request.Mode,
			Ready: false,
			Diagnostics: []SessionDiagnostic{{
				Severity: astmerge.SeverityError,
				Category: astmerge.CategoryConfigurationError,
				Reason:   "missing_resolved_options",
				Message:  "ready template session request is missing resolved_options",
			}},
		}), nil
	}
	return RunTemplateDirectorySessionWithOptions(*request.ResolvedOptions)
}

func RunTemplateDirectorySessionRunnerRequest(
	request SessionRunnerRequest,
	profiles map[string]DirectorySessionProfile,
) (SessionOutcomeReport, error) {
	if request.RequestKind == "profile" {
		overrides := DirectorySessionOptions{}
		if request.Overrides != nil {
			overrides = decodeSessionRunnerOptions(request.Overrides, true)
		}
		return RunTemplateDirectorySessionRequest(
			ReportTemplateDirectorySessionProfileRequest(profiles, request.ProfileName, overrides),
		)
	}
	options := DirectorySessionOptions{}
	if request.Options != nil {
		options = decodeSessionRunnerOptions(request.Options, false)
	}
	return RunTemplateDirectorySessionRequest(ReportTemplateDirectorySessionOptionsRequest(options))
}

func ReportTemplateDirectorySessionRunnerInput(input SessionRunnerInput) SessionRunnerRequest {
	if input.RequestKind == "profile" {
		return SessionRunnerRequest{
			RequestKind: input.RequestKind,
			ProfileName: input.ProfileName,
			Overrides:   reportSessionRunnerInputOverrides(input),
		}
	}
	return SessionRunnerRequest{
		RequestKind: input.RequestKind,
		Options:     reportSessionRunnerInputOptions(input),
	}
}

func ReportTemplateDirectorySessionRunnerPayload(payload SessionRunnerPayload) SessionRunnerInput {
	requestKind := payload.RequestKind
	if requestKind == "" {
		if payload.ProfileName != "" || payload.DefaultProfileName != "" {
			requestKind = "profile"
		} else {
			requestKind = "options"
		}
	}
	profileName := payload.ProfileName
	if profileName == "" {
		profileName = payload.DefaultProfileName
	}
	context := payload.Context
	if context == nil {
		context = &astmerge.TemplateDestinationContext{}
	}
	defaultStrategy := payload.DefaultStrategy
	if defaultStrategy == "" {
		defaultStrategy = astmerge.TemplateStrategyMerge
	}
	overrides := cloneStrategyOverrides(payload.Overrides)
	if overrides == nil {
		overrides = []astmerge.TemplateStrategyOverride{}
	}
	replacements := cloneStringMap(payload.Replacements)
	if replacements == nil {
		replacements = map[string]string{}
	}
	return SessionRunnerInput{
		RequestKind:     requestKind,
		ProfileName:     profileName,
		Mode:            payload.Mode,
		TemplateRoot:    payload.TemplateRoot,
		DestinationRoot: payload.DestinationRoot,
		Context:         context,
		DefaultStrategy: defaultStrategy,
		Overrides:       overrides,
		Replacements:    replacements,
		AllowedFamilies: cloneStringSlice(payload.AllowedFamilies),
	}
}

func RunTemplateDirectorySessionRunnerPayload(
	payload SessionRunnerPayload,
	profiles map[string]DirectorySessionProfile,
) (SessionOutcomeReport, error) {
	return RunTemplateDirectorySessionRunnerRequest(
		ReportTemplateDirectorySessionRunnerInput(
			ReportTemplateDirectorySessionRunnerPayload(payload),
		),
		profiles,
	)
}

func RunTemplateDirectorySessionEntrypoint(
	entrypoint SessionEntrypoint,
	profiles map[string]DirectorySessionProfile,
) (SessionOutcomeReport, error) {
	if entrypoint.Payload != nil {
		return RunTemplateDirectorySessionRunnerPayload(*entrypoint.Payload, profiles)
	}
	if entrypoint.Request != nil {
		return RunTemplateDirectorySessionRunnerRequest(*entrypoint.Request, profiles)
	}
	return SessionOutcomeReport{}, nil
}

func ReportTemplateDirectorySessionEntrypoint(entrypoint SessionEntrypoint) SessionEntrypointReport {
	if entrypoint.Payload != nil {
		return SessionEntrypointReport{
			SourceKind: "payload",
			RunnerRequest: ReportTemplateDirectorySessionRunnerInput(
				ReportTemplateDirectorySessionRunnerPayload(*entrypoint.Payload),
			),
		}
	}
	if entrypoint.Request != nil {
		return SessionEntrypointReport{
			SourceKind:    "request",
			RunnerRequest: *entrypoint.Request,
		}
	}
	return SessionEntrypointReport{
		SourceKind:    "",
		RunnerRequest: SessionRunnerRequest{},
	}
}

func ReportTemplateDirectorySessionResolution(
	entrypoint SessionEntrypoint,
	profiles map[string]DirectorySessionProfile,
) SessionResolutionReport {
	entrypointReport := ReportTemplateDirectorySessionEntrypoint(entrypoint)
	return SessionResolutionReport{
		SourceKind:     entrypointReport.SourceKind,
		RunnerRequest:  entrypointReport.RunnerRequest,
		SessionRequest: reportSessionRequestFromRunnerRequest(entrypointReport.RunnerRequest, profiles),
	}
}

func reportSessionRequestFromRunnerRequest(
	request SessionRunnerRequest,
	profiles map[string]DirectorySessionProfile,
) SessionRequestReport {
	if request.RequestKind == "profile" {
		overrides := DirectorySessionOptions{}
		if request.Overrides != nil {
			overrides = decodeSessionRunnerOptions(request.Overrides, true)
		}
		return ReportTemplateDirectorySessionProfileRequest(profiles, request.ProfileName, overrides)
	}
	options := DirectorySessionOptions{}
	if request.Options != nil {
		options = decodeSessionRunnerOptions(request.Options, false)
	}
	return ReportTemplateDirectorySessionOptionsRequest(options)
}

func ReportTemplateDirectorySessionInspection(
	entrypoint SessionEntrypoint,
	profiles map[string]DirectorySessionProfile,
) (SessionInspectionReport, error) {
	entrypointReport := ReportTemplateDirectorySessionEntrypoint(entrypoint)
	sessionResolution := ReportTemplateDirectorySessionResolution(entrypoint, profiles)
	if !sessionResolution.SessionRequest.Ready || sessionResolution.SessionRequest.ResolvedOptions == nil {
		return SessionInspectionReport{
			EntrypointReport:  entrypointReport,
			SessionResolution: sessionResolution,
			AdapterCapabilities: AdapterCapabilityReport{
				RequiredFamilies: []string{},
				AdapterFamilies:  []string{},
				MissingFamilies:  []string{},
				Ready:            false,
			},
			Status: SessionStatusReport{
				Mode:              sessionResolution.SessionRequest.Mode,
				Ready:             false,
				MissingFamilies:   []string{},
				BlockedPaths:      []string{},
				PlannedWriteCount: 0,
				WrittenCount:      0,
			},
			Diagnostics: SessionDiagnosticsReport{
				Mode:        sessionResolution.SessionRequest.Mode,
				Ready:       false,
				Diagnostics: sessionResolution.SessionRequest.Diagnostics,
			},
		}, nil
	}

	resolved := *sessionResolution.SessionRequest.ResolvedOptions
	capabilities, err := ReportDefaultAdapterCapabilitiesFromDirectories(
		resolved.TemplateRoot,
		resolved.DestinationRoot,
		resolved.Context,
		resolved.DefaultStrategy,
		resolved.Overrides,
		resolved.Replacements,
		resolved.AllowedFamilies,
		resolved.Config,
	)
	if err != nil {
		return SessionInspectionReport{}, err
	}
	sessionReport, err := PlanTemplateDirectorySessionFromDirectories(
		resolved.TemplateRoot,
		resolved.DestinationRoot,
		resolved.Context,
		resolved.DefaultStrategy,
		resolved.Overrides,
		resolved.Replacements,
		resolved.Config,
	)
	if err != nil {
		return SessionInspectionReport{}, err
	}
	status := ReportTemplateDirectorySessionStatus(
		ReportTemplateDirectorySessionEnvelope(sessionReport, capabilities),
	)
	diagnostics, err := PlanTemplateDirectorySessionDiagnosticsFromDirectories(
		resolved.TemplateRoot,
		resolved.DestinationRoot,
		resolved.Context,
		resolved.DefaultStrategy,
		resolved.Overrides,
		resolved.Replacements,
		resolved.AllowedFamilies,
		resolved.Config,
	)
	if err != nil {
		return SessionInspectionReport{}, err
	}
	return SessionInspectionReport{
		EntrypointReport:    entrypointReport,
		SessionResolution:   sessionResolution,
		AdapterCapabilities: capabilities,
		Status:              status,
		Diagnostics:         diagnostics,
	}, nil
}

func RunTemplateDirectorySessionDispatch(
	operation string,
	entrypoint SessionEntrypoint,
	profiles map[string]DirectorySessionProfile,
) (SessionDispatchReport, error) {
	switch operation {
	case "inspect":
		inspection, err := ReportTemplateDirectorySessionInspection(entrypoint, profiles)
		if err != nil {
			return SessionDispatchReport{}, err
		}
		return SessionDispatchReport{
			Operation:  operation,
			Inspection: &inspection,
			Outcome:    nil,
		}, nil
	case "run":
		outcome, err := RunTemplateDirectorySessionEntrypoint(entrypoint, profiles)
		if err != nil {
			return SessionDispatchReport{}, err
		}
		return SessionDispatchReport{
			Operation:  operation,
			Inspection: nil,
			Outcome:    &outcome,
		}, nil
	default:
		return SessionDispatchReport{}, fmt.Errorf("unsupported template directory session operation: %s", operation)
	}
}

func RunTemplateDirectorySessionCommand(
	command SessionCommand,
	profiles map[string]DirectorySessionProfile,
) (SessionDispatchReport, error) {
	return RunTemplateDirectorySessionDispatch(command.Operation, SessionEntrypoint{
		Payload: command.Payload,
		Request: command.Request,
	}, profiles)
}

func RunTemplateDirectorySessionCommandPayload(
	payload SessionCommandPayload,
	profiles map[string]DirectorySessionProfile,
) (SessionDispatchReport, error) {
	return RunTemplateDirectorySessionCommand(SessionCommand{
		Operation: payload.Operation,
		Payload: &SessionRunnerPayload{
			RequestKind:        payload.RequestKind,
			DefaultProfileName: payload.DefaultProfileName,
			ProfileName:        payload.ProfileName,
			Mode:               payload.Mode,
			TemplateRoot:       payload.TemplateRoot,
			DestinationRoot:    payload.DestinationRoot,
			Context:            payload.Context,
			DefaultStrategy:    payload.DefaultStrategy,
			Overrides:          payload.Overrides,
			Replacements:       payload.Replacements,
			AllowedFamilies:    payload.AllowedFamilies,
		},
	}, profiles)
}

func RunTemplateDirectorySession(
	invocation SessionInvocation,
	profiles map[string]DirectorySessionProfile,
) (SessionDispatchReport, error) {
	if invocation.Payload != nil || invocation.Request != nil {
		return RunTemplateDirectorySessionCommand(SessionCommand{
			Operation: invocation.Operation,
			Payload:   invocation.Payload,
			Request:   invocation.Request,
		}, profiles)
	}

	return RunTemplateDirectorySessionCommandPayload(SessionCommandPayload{
		Operation:          invocation.Operation,
		RequestKind:        invocation.RequestKind,
		DefaultProfileName: invocation.DefaultProfileName,
		ProfileName:        invocation.ProfileName,
		Mode:               invocation.Mode,
		TemplateRoot:       invocation.TemplateRoot,
		DestinationRoot:    invocation.DestinationRoot,
		Context:            invocation.Context,
		DefaultStrategy:    invocation.DefaultStrategy,
		Overrides:          invocation.Overrides,
		Replacements:       invocation.Replacements,
		AllowedFamilies:    invocation.AllowedFamilies,
	}, profiles)
}

func reportSessionRunnerInputOptions(input SessionRunnerInput) map[string]any {
	return map[string]any{
		"mode":             input.Mode,
		"template_root":    input.TemplateRoot,
		"destination_root": input.DestinationRoot,
		"context":          input.Context,
		"default_strategy": defaultTemplateStrategy(input.DefaultStrategy),
		"overrides":        cloneStrategyOverrides(input.Overrides),
		"replacements":     cloneStringMap(input.Replacements),
		"allowed_families": cloneStringSlice(input.AllowedFamilies),
	}
}

func reportSessionRunnerInputOverrides(input SessionRunnerInput) map[string]any {
	overrides := map[string]any{
		"mode":             input.Mode,
		"template_root":    input.TemplateRoot,
		"destination_root": input.DestinationRoot,
	}
	if input.Context != nil && input.Context.ProjectName != "" {
		overrides["context"] = input.Context
	}
	if input.DefaultStrategy != "" && input.DefaultStrategy != astmerge.TemplateStrategyMerge {
		overrides["default_strategy"] = input.DefaultStrategy
	}
	if len(input.Overrides) > 0 {
		overrides["overrides"] = cloneStrategyOverrides(input.Overrides)
	}
	if len(input.Replacements) > 0 {
		overrides["replacements"] = cloneStringMap(input.Replacements)
	}
	if input.AllowedFamilies != nil {
		overrides["allowed_families"] = cloneStringSlice(input.AllowedFamilies)
	}
	return overrides
}

func decodeSessionRunnerOptions(raw map[string]any, sparse bool) DirectorySessionOptions {
	options := DirectorySessionOptions{}
	if !sparse {
		options.DefaultStrategy = astmerge.TemplateStrategyMerge
	}
	if raw == nil {
		return options
	}
	payload, err := json.Marshal(raw)
	if err == nil {
		_ = json.Unmarshal(payload, &options)
	}
	if !sparse && options.Context == nil {
		options.Context = &astmerge.TemplateDestinationContext{}
	}
	if !sparse && options.Overrides == nil {
		options.Overrides = []astmerge.TemplateStrategyOverride{}
	}
	if !sparse && options.Replacements == nil {
		options.Replacements = map[string]string{}
	}
	if !sparse && options.DefaultStrategy == "" {
		options.DefaultStrategy = astmerge.TemplateStrategyMerge
	}
	return options
}

func defaultTemplateStrategy(strategy astmerge.TemplateStrategy) astmerge.TemplateStrategy {
	if strategy == "" {
		return astmerge.TemplateStrategyMerge
	}
	return strategy
}

func ResolveTemplateDirectorySessionOptions(
	profiles map[string]DirectorySessionProfile,
	profileName string,
	overrides DirectorySessionOptions,
) (DirectorySessionOptions, bool) {
	profile, ok := profiles[profileName]
	if !ok {
		return DirectorySessionOptions{}, false
	}
	options := DirectorySessionOptions{
		Mode:            profile.Mode,
		TemplateRoot:    overrides.TemplateRoot,
		DestinationRoot: overrides.DestinationRoot,
		Context:         profile.Context,
		DefaultStrategy: profile.DefaultStrategy,
		Overrides:       cloneStrategyOverrides(profile.Overrides),
		Replacements:    cloneStringMap(profile.Replacements),
		AllowedFamilies: cloneStringSlice(profile.AllowedFamilies),
		Config:          profile.Config,
	}
	if overrides.Mode != "" {
		options.Mode = overrides.Mode
	}
	if overrides.Context != nil {
		options.Context = overrides.Context
	}
	if overrides.DefaultStrategy != "" {
		options.DefaultStrategy = overrides.DefaultStrategy
	}
	if overrides.Overrides != nil {
		options.Overrides = overrides.Overrides
	}
	if overrides.Replacements != nil {
		options.Replacements = overrides.Replacements
	}
	if overrides.AllowedFamilies != nil {
		options.AllowedFamilies = overrides.AllowedFamilies
	}
	if overrides.Config != nil {
		options.Config = overrides.Config
	}
	return options, true
}

func RunTemplateDirectorySessionWithProfile(
	profiles map[string]DirectorySessionProfile,
	profileName string,
	overrides DirectorySessionOptions,
) (SessionOutcomeReport, error) {
	request := ReportTemplateDirectorySessionProfileRequest(profiles, profileName, overrides)
	if !request.Ready {
		return reportTemplateDirectorySessionConfigurationOutcome(request.Mode, SessionDiagnosticsReport{
			Mode:        request.Mode,
			Ready:       request.Ready,
			Diagnostics: request.Diagnostics,
		}), nil
	}
	return RunTemplateDirectorySessionWithOptions(*request.ResolvedOptions)
}

func firstNonEmptySessionMode(values ...DirectorySessionMode) DirectorySessionMode {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return DirectorySessionModePlan
}

func cloneStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func cloneStringSlice(values []string) []string {
	if values == nil {
		return nil
	}
	cloned := make([]string, len(values))
	copy(cloned, values)
	return cloned
}

func cloneStrategyOverrides(values []astmerge.TemplateStrategyOverride) []astmerge.TemplateStrategyOverride {
	if values == nil {
		return nil
	}
	cloned := make([]astmerge.TemplateStrategyOverride, len(values))
	copy(cloned, values)
	return cloned
}
