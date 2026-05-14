package treehaver

import (
	"fmt"
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
	BackendRef            BackendReference
	Language              string
	ParserIdentity        ParserIdentity
	LanguageVersion       LanguageVersion
	ParseErrorBehavior    string
	SourceSpanSupport     string
	SourceFragmentSupport string
	RenderStrategies      []string
	SemanticRoleSupport   string
	NormalizedTreeSupport bool
	NativeNodeAccess      bool
	Diagnostics           []string
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
