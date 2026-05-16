package treehaver

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"
)

type AnalysisHandle interface {
	Kind() string
}

type ParserRequest struct {
	Source   string
	Language string
	Dialect  string
}

type BackendReference struct {
	ID     string
	Family string
}

type DiagnosticSeverity string

const (
	SeverityInfo    DiagnosticSeverity = "info"
	SeverityWarning DiagnosticSeverity = "warning"
	SeverityError   DiagnosticSeverity = "error"
)

type DiagnosticCategory string

const (
	CategoryParseError         DiagnosticCategory = "parse_error"
	CategoryUnsupportedFeature DiagnosticCategory = "unsupported_feature"
)

type Diagnostic struct {
	Severity DiagnosticSeverity `json:"severity"`
	Category DiagnosticCategory `json:"category"`
	Message  string             `json:"message"`
	Path     string             `json:"path,omitempty"`
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

type ParseResult[T any] struct {
	OK          bool
	Diagnostics []Diagnostic
	Analysis    *T
	Policies    []PolicyReference
}

type AdapterInfo struct {
	Backend           string
	BackendRef        *BackendReference
	SupportsDialects  bool
	SupportedPolicies []PolicyReference
}

type FeatureProfile struct {
	Backend           string
	BackendRef        *BackendReference
	SupportsDialects  bool
	SupportedPolicies []PolicyReference
}

type ParserIdentity struct {
	Name           string
	Version        string
	Implementation string
}

type LanguageVersion struct {
	Version string
	Dialect *string
}

type BackendCapability struct {
	BackendRef            BackendReference `json:"backend_ref"`
	Language              string           `json:"language"`
	ParserIdentity        ParserIdentity   `json:"parser_identity"`
	LanguageVersion       LanguageVersion  `json:"language_version"`
	ParseErrorBehavior    string           `json:"parse_error_behavior"`
	SourceSpanSupport     string           `json:"source_span_support"`
	SourceFragmentSupport string           `json:"source_fragment_support"`
	RenderStrategies      []string         `json:"render_strategies"`
	SemanticRoleSupport   string           `json:"semantic_role_support"`
	NormalizedTreeSupport bool             `json:"normalized_tree_support"`
	NativeNodeAccess      bool             `json:"native_node_access"`
	KnownNodeKinds        []string         `json:"known_node_kinds,omitempty"`
	KnownFields           []string         `json:"known_fields,omitempty"`
	GrammarInventory      string           `json:"grammar_inventory,omitempty"`
	Diagnostics           []string         `json:"diagnostics"`
}

type ParseErrorNode struct {
	Kind    string
	Span    SourceSpan
	Message string
}

type ParseErrorTolerance struct {
	BackendRef      BackendReference
	Language        string
	Behavior        string
	ToleratesErrors bool
	ErrorNodes      []ParseErrorNode
	Diagnostics     []string
}

type NativeParserProvider struct {
	ID                   string
	Family               string
	Language             string
	Operations           []string
	RetainsNativeTree    bool
	NativeTreeVisibility string
	MetadataPolicy       string
}

type NativeProviderMetadata struct {
	ProviderID           string   `json:"provider_id"`
	Family               string   `json:"family"`
	HostLanguage         string   `json:"host_language"`
	TargetLanguage       string   `json:"target_language"`
	ParserName           string   `json:"parser_name"`
	ParserVersion        string   `json:"parser_version"`
	LanguageVersion      string   `json:"language_version"`
	Dialect              string   `json:"dialect"`
	ParseErrorBehavior   string   `json:"parse_error_behavior"`
	SourceSpanSupport    string   `json:"source_span_support"`
	RenderSupport        string   `json:"render_support"`
	SemanticRoleSupport  string   `json:"semantic_role_support"`
	RetainsNativeTree    bool     `json:"retains_native_tree"`
	NativeTreeVisibility string   `json:"native_tree_visibility"`
	MetadataPolicy       string   `json:"metadata_policy"`
	Diagnostics          []string `json:"diagnostics"`
}

type NormalizedParseResult struct {
	OK                       bool
	BackendCapability        BackendCapability
	RootID                   string
	Nodes                    []NormalizedTreeNode
	ParseErrorTolerance      ParseErrorTolerance
	SourceFragmentsAvailable bool
	Diagnostics              []string
	Metadata                 map[string]map[string]string
}

type TreeHaverProfile struct {
	ProfileID            string
	Language             string
	BackendRef           BackendReference
	ProviderID           string
	NodeRoles            []NodeRole
	NormalizedNodeFields []string
	OptionalNodeFeatures []string
	UnsupportedDefaults  map[string]string
	Capability           BackendCapability
	FixtureSlices        []string
	Diagnostics          []string
}

type EditProjectionSupport struct {
	BackendRef               BackendReference `json:"backend_ref"`
	Language                 string           `json:"language"`
	SupportsEditProjection   bool             `json:"supports_edit_projection"`
	NativeEditTarget         string           `json:"native_edit_target"`
	NormalizedEditTarget     string           `json:"normalized_edit_target"`
	SupportedOperations      []string         `json:"supported_operations"`
	RequiredNodeFields       []string         `json:"required_node_fields"`
	CorrelationKeys          []string         `json:"correlation_keys"`
	PreservesSourceFragments bool             `json:"preserves_source_fragments"`
	UnsupportedReason        *string          `json:"unsupported_reason"`
	Diagnostics              []string         `json:"diagnostics"`
}

type LibraryPathValidation struct {
	Path   string
	Valid  bool
	Errors []string
}

type BackendAvailabilityStatus string

const (
	BackendAvailabilityAvailable   BackendAvailabilityStatus = "available"
	BackendAvailabilityUnavailable BackendAvailabilityStatus = "unavailable"
	BackendAvailabilityUnknown     BackendAvailabilityStatus = "unknown"
)

type BackendAvailabilityCheck struct {
	Name        string
	Status      BackendAvailabilityStatus
	Required    bool
	Diagnostics []string
}

type BackendAvailabilityReport struct {
	BackendRef  BackendReference
	Status      BackendAvailabilityStatus
	Checks      []BackendAvailabilityCheck
	Diagnostics []string
}

type ProviderDiagnostic struct {
	Severity string
	Category string
	Code     string
	Message  string
	Path     string
	Blocking bool
}

type ProviderDiagnosticsReport struct {
	ProviderID  string
	BackendRef  BackendReference
	Language    string
	Status      string
	Diagnostics []ProviderDiagnostic
}

type EditProjectionOperationRequest struct {
	Operation         string
	TargetNodeID      string
	TargetNodePath    string
	ReplacementSource string
}

type EditProjectionExecutionRequest struct {
	ProviderID string
	BackendRef BackendReference
	Language   string
	Source     string
	Operations []EditProjectionOperationRequest
}

type AppliedEditProjectionOperation struct {
	Operation        string
	TargetNodeID     string
	CorrelationKey   string
	CorrelationValue string
}

type EditProjectionExecutionResult struct {
	OK                bool
	Status            string
	Source            string
	AppliedOperations []AppliedEditProjectionOperation
	Diagnostics       []ProviderDiagnostic
}

type EditProjectionProviderOperation struct {
	Operation              string   `json:"operation"`
	Status                 string   `json:"status"`
	NodeScope              string   `json:"node_scope"`
	CorrelationKeys        []string `json:"correlation_keys"`
	FixtureSlices          []string `json:"fixture_slices"`
	FormattingPreservation string   `json:"formatting_preservation"`
	Diagnostics            []string `json:"diagnostics"`
}

type EditProjectionProviderMatrixEntry struct {
	ProviderID               string                            `json:"provider_id"`
	BackendRef               BackendReference                  `json:"backend_ref"`
	Language                 string                            `json:"language"`
	FormattingPreservation   string                            `json:"formatting_preservation"`
	PreservesSourceFragments bool                              `json:"preserves_source_fragments"`
	Operations               []EditProjectionProviderOperation `json:"operations"`
}

type EditProjectionProviderMatrix struct {
	Operations  []string                            `json:"operations"`
	Providers   []EditProjectionProviderMatrixEntry `json:"providers"`
	Diagnostics []string                            `json:"diagnostics"`
}

type OrderedSiblingEdge struct {
	ParentID          string
	NodeID            string
	PreviousSiblingID *string
	NextSiblingID     *string
}

type OrderedTreePrimitives struct {
	RootID       string
	ChildOrder   map[string][]string
	SiblingEdges []OrderedSiblingEdge
	Diagnostics  []string
}

type ParserAdapter[T AnalysisHandle] interface {
	Info() AdapterInfo
	Parse(request ParserRequest) ParseResult[T]
}

type ParserDiagnostics struct {
	Backend     string
	BackendRef  *BackendReference
	Diagnostics []Diagnostic
}

type ProcessRequest struct {
	Source   string
	Language string
}

type ProcessSpan struct {
	StartByte int
	EndByte   int
	StartRow  int
	StartCol  int
	EndRow    int
	EndCol    int
}

type ByteRange struct {
	StartByte int
	EndByte   int
}

func (byteRange ByteRange) Valid() bool {
	return byteRange.StartByte >= 0 && byteRange.EndByte >= byteRange.StartByte
}

func (byteRange ByteRange) Length() int {
	if !byteRange.Valid() {
		return 0
	}
	return byteRange.EndByte - byteRange.StartByte
}

func (byteRange ByteRange) ContainsByte(offset int) bool {
	return byteRange.Valid() && offset >= byteRange.StartByte && offset < byteRange.EndByte
}

func (byteRange ByteRange) ContainsRange(other ByteRange) bool {
	return byteRange.Valid() && other.Valid() && other.StartByte >= byteRange.StartByte && other.EndByte <= byteRange.EndByte
}

func (byteRange ByteRange) Overlaps(other ByteRange) bool {
	return byteRange.Valid() && other.Valid() && byteRange.StartByte < other.EndByte && other.StartByte < byteRange.EndByte
}

type SourcePoint struct {
	Row    int
	Column int
}

type SourceSpan struct {
	Range      ByteRange
	StartPoint SourcePoint
	EndPoint   SourcePoint
}

type NodeRole string

const (
	NodeRoleStructural NodeRole = "structural"
	NodeRoleToken      NodeRole = "token"
	NodeRoleTrivia     NodeRole = "trivia"
	NodeRoleComment    NodeRole = "comment"
	NodeRoleDelimiter  NodeRole = "delimiter"
	NodeRoleSeparator  NodeRole = "separator"
	NodeRoleVirtual    NodeRole = "virtual"
	NodeRoleError      NodeRole = "error"
	NodeRoleOpaque     NodeRole = "opaque"
)

type NormalizedTreeNode struct {
	ID                  string
	Kind                string
	Role                NodeRole
	ParentID            *string
	ChildIDs            []string
	Span                SourceSpan
	FieldName           *string
	Named               bool
	Anonymous           bool
	HasSourceText       bool
	SourceFragment      string
	BackendKind         string
	SemanticRoles       []string
	BackendRoles        []string
	UnsupportedFeatures []string
	Metadata            map[string]map[string]string
}

type SourceFragment struct {
	Text        string
	Span        SourceSpan
	Available   bool
	Strategy    string
	ByteLength  int
	Diagnostics []string
}

func NodeRoles() []NodeRole {
	return []NodeRole{
		NodeRoleStructural,
		NodeRoleToken,
		NodeRoleTrivia,
		NodeRoleComment,
		NodeRoleDelimiter,
		NodeRoleSeparator,
		NodeRoleVirtual,
		NodeRoleError,
		NodeRoleOpaque,
	}
}

type ByteEditSpan struct {
	StartByte   int
	OldEndByte  int
	NewEndByte  int
	StartPoint  SourcePoint
	OldEndPoint SourcePoint
	NewEndPoint SourcePoint
}

func (edit ByteEditSpan) OldRange() ByteRange {
	return ByteRange{StartByte: edit.StartByte, EndByte: edit.OldEndByte}
}

func (edit ByteEditSpan) NewRange() ByteRange {
	return ByteRange{StartByte: edit.StartByte, EndByte: edit.NewEndByte}
}

func (edit ByteEditSpan) ByteDelta() int {
	return edit.NewEndByte - edit.OldEndByte
}

type BinaryScalarValue struct {
	Kind        string
	Value       any
	Symbol      string
	RawValue    any
	Encoding    string
	Format      string
	Description string
}

type BinaryRenderPolicy struct {
	SchemaPath  string
	ByteRange   *ByteRange
	Operation   string
	Disposition string
	Reason      string
}

type BinaryDiagnostic struct {
	Severity   string
	Category   string
	Message    string
	SchemaPath string
	ByteRange  *ByteRange
}

type BinaryNestedDispatch struct {
	SchemaPath string
	Family     string
	Status     string
}

type BinaryPayloadRegion struct {
	Kind        string
	SchemaPath  string
	ByteRange   ByteRange
	ExpectedHex string
}

type BinaryRawPayload struct {
	Encoding   string
	Value      string
	ByteLength int
	Regions    []BinaryPayloadRegion
}

type BinaryMergeReport struct {
	Format             string
	Schema             string
	MatchedSchemaPaths []string
	PreservedRanges    []ByteRange
	RewrittenNodes     []string
	ChecksumUpdates    []string
	NestedDispatches   []BinaryNestedDispatch
	Diagnostics        []BinaryDiagnostic
}

type ZipArchiveInfo struct {
	Format                string
	Schema                string
	EntryCount            int
	CentralDirectoryRange ByteRange
}

type ZipArchiveEntry struct {
	Path                  string
	NormalizedPath        string
	Directory             bool
	Compression           string
	CompressedSize        int
	UncompressedSize      int
	CRC32                 string
	LocalHeaderRange      ByteRange
	DataRange             ByteRange
	CentralDirectoryRange ByteRange
}

type ZipMemberDecision struct {
	NormalizedPath string
	Operation      string
	Disposition    string
	NestedFamily   string
	Reason         string
}

type ZipUnsafeEntry struct {
	Path           string
	NormalizedPath string
	Category       string
	Reason         string
}

type ZipFamilyReport struct {
	Archive         ZipArchiveInfo
	Entries         []ZipArchiveEntry
	MemberDecisions []ZipMemberDecision
	UnsafeEntries   []ZipUnsafeEntry
	MergeReport     BinaryMergeReport
}

func SliceByteRange(source string, byteRange ByteRange) (string, error) {
	sourceBytes := []byte(source)
	if !byteRange.Valid() || byteRange.EndByte > len(sourceBytes) {
		return "", fmt.Errorf("invalid byte range [%d, %d) for source length %d", byteRange.StartByte, byteRange.EndByte, len(sourceBytes))
	}

	return string(sourceBytes[byteRange.StartByte:byteRange.EndByte]), nil
}

func ExtractSourceFragment(source string, span SourceSpan, strategy string) SourceFragment {
	text, err := SliceByteRange(source, span.Range)
	if err != nil {
		return SourceFragment{
			Span:        span,
			Available:   false,
			Strategy:    strategy,
			Diagnostics: []string{err.Error()},
		}
	}

	return SourceFragment{
		Text:        text,
		Span:        span,
		Available:   true,
		Strategy:    strategy,
		ByteLength:  len([]byte(text)),
		Diagnostics: []string{},
	}
}

func ByteOffsetForPoint(source string, point SourcePoint) (int, error) {
	if point.Row < 0 || point.Column < 0 {
		return 0, fmt.Errorf("invalid source point (%d, %d)", point.Row, point.Column)
	}

	row := 0
	column := 0
	for offset, value := range []byte(source) {
		if row == point.Row && column == point.Column {
			return offset, nil
		}
		if value == '\n' {
			row++
			column = 0
		} else {
			column++
		}
	}
	if row == point.Row && column == point.Column {
		return len([]byte(source)), nil
	}

	return 0, fmt.Errorf("source point (%d, %d) is outside source", point.Row, point.Column)
}

type ProcessStructureItem struct {
	Kind string
	Name string
	Span ProcessSpan
}

type ProcessImportInfo struct {
	Source string
	Items  []string
	Span   ProcessSpan
}

type ProcessDiagnostic struct {
	Message  string
	Severity string
}

type LanguagePackAnalysis struct {
	Language   string
	Dialect    string
	RootType   string
	HasError   bool
	BackendRef BackendReference
}

type LanguagePackProcessAnalysis struct {
	Language    string
	Structure   []ProcessStructureItem
	Imports     []ProcessImportInfo
	Diagnostics []ProcessDiagnostic
	BackendRef  BackendReference
}

type KaitaiByteSpan struct {
	StartByte int
	EndByte   int
}

type KaitaiTreeNode struct {
	Kind       string
	SchemaPath string
	Span       KaitaiByteSpan
	Fields     map[string]any
	Children   []KaitaiTreeNode
}

type KaitaiTreeAnalysis struct {
	Schema           string
	SourceByteLength int
	Root             KaitaiTreeNode
	BackendRef       BackendReference
	Diagnostics      []BinaryDiagnostic
}

func (LanguagePackAnalysis) Kind() string {
	return "tree-sitter"
}

func (LanguagePackProcessAnalysis) Kind() string {
	return "tree-sitter-process"
}

func (KaitaiTreeAnalysis) Kind() string {
	return "kaitai-tree"
}

var KreuzbergLanguagePackBackend = BackendReference{
	ID:     "kreuzberg-language-pack",
	Family: "tree-sitter",
}

var PigeonBackend = BackendReference{
	ID:     "pigeon",
	Family: "peg",
}

var KaitaiStructBackend = BackendReference{
	ID:     "kaitai-struct",
	Family: "kaitai",
}

var (
	backendRegistryMu sync.RWMutex
	backendRegistry   = map[string]BackendReference{
		KreuzbergLanguagePackBackend.ID: KreuzbergLanguagePackBackend,
		PigeonBackend.ID:                PigeonBackend,
		KaitaiStructBackend.ID:          KaitaiStructBackend,
	}
)

func RegisterBackend(backend BackendReference) {
	backendRegistryMu.Lock()
	defer backendRegistryMu.Unlock()

	backendRegistry[backend.ID] = backend
}

func BackendReferenceByID(id string) *BackendReference {
	backendRegistryMu.RLock()
	defer backendRegistryMu.RUnlock()

	backend, ok := backendRegistry[id]
	if !ok {
		return nil
	}
	backendCopy := backend
	return &backendCopy
}

func RegisteredBackends() []BackendReference {
	backendRegistryMu.RLock()
	defer backendRegistryMu.RUnlock()

	backends := make([]BackendReference, 0, len(backendRegistry))
	for _, backend := range backendRegistry {
		backends = append(backends, backend)
	}
	slices.SortFunc(backends, func(left, right BackendReference) int {
		return strings.Compare(left.ID, right.ID)
	})
	return backends
}

var (
	validLibraryFilenamePattern  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
	validLanguageNamePattern     = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	validSymbolNamePattern       = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	versionedSharedObjectPattern = regexp.MustCompile(`\.so\.\d+$`)
)

const MaxLibraryPathLength = 4096

func ValidateLibraryPath(libraryPath string) LibraryPathValidation {
	errors := LibraryPathErrors(libraryPath)
	return LibraryPathValidation{Path: libraryPath, Valid: len(errors) == 0, Errors: errors}
}

func LibraryPathErrors(libraryPath string) []string {
	errors := []string{}
	if libraryPath == "" {
		return []string{"path_empty"}
	}
	if len(libraryPath) > MaxLibraryPathLength {
		errors = append(errors, "path_too_long")
	}
	if strings.Contains(libraryPath, "\x00") {
		errors = append(errors, "path_contains_null_byte")
	}
	if !strings.HasPrefix(libraryPath, "/") && !windowsAbsolutePath(libraryPath) {
		errors = append(errors, "path_not_absolute")
	}
	segments := strings.FieldsFunc(libraryPath, func(r rune) bool { return r == '/' || r == '\\' })
	for _, segment := range segments {
		if segment == ".." {
			errors = append(errors, "path_contains_parent_traversal")
			break
		}
	}
	for _, segment := range segments {
		if segment == "." {
			errors = append(errors, "path_contains_current_directory_traversal")
			break
		}
	}
	if !hasAllowedLibraryExtension(libraryPath) {
		errors = append(errors, "path_extension_not_allowed")
	}
	if !validLibraryFilenamePattern.MatchString(libraryFilename(libraryPath)) {
		errors = append(errors, "filename_contains_invalid_characters")
	}
	return errors
}

func SafeLanguageName(name string) bool {
	return len(name) <= 64 && validLanguageNamePattern.MatchString(name)
}

func SanitizeLanguageName(name string) *string {
	var sanitized strings.Builder
	for _, char := range strings.ToLower(name) {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '_' {
			sanitized.WriteRune(char)
		}
	}
	result := sanitized.String()
	if result == "" || result[0] < 'a' || result[0] > 'z' {
		return nil
	}
	return &result
}

func SafeSymbolName(symbol string) bool {
	return len(symbol) <= 256 && validSymbolNamePattern.MatchString(symbol)
}

func SafeBackendName(name string) bool {
	return name == "auto" || BackendReferenceByID(name) != nil
}

func BuildBackendAvailabilityReport(backendRef BackendReference, checks []BackendAvailabilityCheck) BackendAvailabilityReport {
	if len(checks) == 0 {
		return BackendAvailabilityReport{
			BackendRef:  backendRef,
			Status:      BackendAvailabilityUnknown,
			Checks:      []BackendAvailabilityCheck{},
			Diagnostics: []string{"backend availability unknown: no checks supplied"},
		}
	}

	diagnostics := []string{}
	status := BackendAvailabilityAvailable
	for _, check := range checks {
		if check.Required && check.Status != BackendAvailabilityAvailable {
			status = BackendAvailabilityUnavailable
			diagnostics = append(diagnostics, "backend unavailable: required check "+check.Name+" is "+string(check.Status))
		}
	}
	return BackendAvailabilityReport{BackendRef: backendRef, Status: status, Checks: checks, Diagnostics: diagnostics}
}

func BuildProviderDiagnosticsReport(providerID string, backendRef BackendReference, language string, diagnostics []ProviderDiagnostic) ProviderDiagnosticsReport {
	status := "clean"
	for _, diagnostic := range diagnostics {
		if diagnostic.Blocking {
			status = "blocked"
			break
		}
		if diagnostic.Severity == "warning" {
			status = "warning"
		}
	}
	return ProviderDiagnosticsReport{
		ProviderID:  providerID,
		BackendRef:  backendRef,
		Language:    language,
		Status:      status,
		Diagnostics: diagnostics,
	}
}

func BuildEditProjectionExecutionResult(source string, applied []AppliedEditProjectionOperation, diagnostics []ProviderDiagnostic) EditProjectionExecutionResult {
	if applied == nil {
		applied = []AppliedEditProjectionOperation{}
	}
	if diagnostics == nil {
		diagnostics = []ProviderDiagnostic{}
	}
	blocking := false
	for _, diagnostic := range diagnostics {
		if diagnostic.Blocking {
			blocking = true
			break
		}
	}
	if blocking {
		return EditProjectionExecutionResult{
			OK:                false,
			Status:            "rejected",
			Source:            source,
			AppliedOperations: []AppliedEditProjectionOperation{},
			Diagnostics:       diagnostics,
		}
	}
	return EditProjectionExecutionResult{
		OK:                true,
		Status:            "applied",
		Source:            source,
		AppliedOperations: applied,
		Diagnostics:       diagnostics,
	}
}

func BuildEditProjectionProviderMatrix(operations []string, providers []EditProjectionProviderMatrixEntry, diagnostics []string) EditProjectionProviderMatrix {
	if operations == nil {
		operations = []string{}
	}
	if providers == nil {
		providers = []EditProjectionProviderMatrixEntry{}
	}
	if diagnostics == nil {
		diagnostics = []string{}
	}
	return EditProjectionProviderMatrix{
		Operations:  operations,
		Providers:   providers,
		Diagnostics: diagnostics,
	}
}

func windowsAbsolutePath(libraryPath string) bool {
	if len(libraryPath) < 3 {
		return false
	}
	letter := libraryPath[0]
	return ((letter >= 'A' && letter <= 'Z') || (letter >= 'a' && letter <= 'z')) &&
		libraryPath[1] == ':' &&
		(libraryPath[2] == '/' || libraryPath[2] == '\\')
}

func hasAllowedLibraryExtension(libraryPath string) bool {
	return strings.HasSuffix(libraryPath, ".so") ||
		strings.HasSuffix(libraryPath, ".dylib") ||
		strings.HasSuffix(libraryPath, ".dll") ||
		versionedSharedObjectPattern.MatchString(libraryPath)
}

func libraryFilename(libraryPath string) string {
	segments := strings.FieldsFunc(libraryPath, func(r rune) bool { return r == '/' || r == '\\' })
	if len(segments) == 0 {
		return ""
	}
	return segments[len(segments)-1]
}

func LanguagePackAdapterInfo() AdapterInfo {
	return AdapterInfo{
		Backend:          KreuzbergLanguagePackBackend.ID,
		BackendRef:       &KreuzbergLanguagePackBackend,
		SupportsDialects: false,
	}
}

func PigeonAdapterInfo() AdapterInfo {
	return AdapterInfo{
		Backend:          PigeonBackend.ID,
		BackendRef:       &PigeonBackend,
		SupportsDialects: false,
	}
}

func PigeonFeatureProfile() FeatureProfile {
	return FeatureProfile{
		Backend:          PigeonBackend.ID,
		BackendRef:       &PigeonBackend,
		SupportsDialects: false,
	}
}

func KaitaiAdapterInfo() AdapterInfo {
	return AdapterInfo{
		Backend:          KaitaiStructBackend.ID,
		BackendRef:       &KaitaiStructBackend,
		SupportsDialects: false,
	}
}

func KaitaiFeatureProfile() FeatureProfile {
	return FeatureProfile{
		Backend:          KaitaiStructBackend.ID,
		BackendRef:       &KaitaiStructBackend,
		SupportsDialects: false,
	}
}
