package treehaver

import (
	"fmt"
	"slices"
	"strings"
	"sync"

	tspack "github.com/kreuzberg-dev/tree-sitter-language-pack/packages/go"
	"github.com/structuredmerge/structuredmerge-go/astmerge"
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

type AdapterInfo struct {
	Backend           string
	BackendRef        *BackendReference
	SupportsDialects  bool
	SupportedPolicies []astmerge.PolicyReference
}

type FeatureProfile struct {
	Backend           string
	BackendRef        *BackendReference
	SupportsDialects  bool
	SupportedPolicies []astmerge.PolicyReference
}

type ParserAdapter[T AnalysisHandle] interface {
	Info() AdapterInfo
	Parse(request ParserRequest) astmerge.ParseResult[T]
}

type ParserDiagnostics struct {
	Backend     string
	BackendRef  *BackendReference
	Diagnostics []astmerge.Diagnostic
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

var (
	languagePackRegistryOnce sync.Once
	languagePackRegistry     *tspack.Registry
	languagePackRegistryErr  error
)

func languagePackRegistryInstance() (*tspack.Registry, error) {
	languagePackRegistryOnce.Do(func() {
		languagePackRegistry, languagePackRegistryErr = tspack.NewRegistry()
	})
	return languagePackRegistry, languagePackRegistryErr
}

func ensureLanguageAvailable(registry *tspack.Registry, language string) error {
	if registry.HasLanguage(language) {
		return nil
	}

	if _, err := tspack.Download([]string{language}); err != nil {
		return err
	}

	if registry.HasLanguage(language) {
		return nil
	}

	return fmt.Errorf("tree-sitter-language-pack language %q is not available after download", language)
}

func ParseWithLanguagePack(request ParserRequest) astmerge.ParseResult[LanguagePackAnalysis] {
	registry, err := languagePackRegistryInstance()
	if err != nil {
		return astmerge.ParseResult[LanguagePackAnalysis]{
			OK: false,
			Diagnostics: []astmerge.Diagnostic{
				{
					Severity: astmerge.SeverityError,
					Category: astmerge.CategoryUnsupportedFeature,
					Message:  err.Error(),
				},
			},
		}
	}

	if err := ensureLanguageAvailable(registry, request.Language); err != nil {
		return astmerge.ParseResult[LanguagePackAnalysis]{
			OK: false,
			Diagnostics: []astmerge.Diagnostic{
				{
					Severity: astmerge.SeverityError,
					Category: astmerge.CategoryUnsupportedFeature,
					Message:  err.Error(),
				},
			},
		}
	}

	tree, err := registry.ParseString(request.Language, request.Source)
	if err != nil {
		return astmerge.ParseResult[LanguagePackAnalysis]{
			OK: false,
			Diagnostics: []astmerge.Diagnostic{
				{
					Severity: astmerge.SeverityError,
					Category: astmerge.CategoryParseError,
					Message:  err.Error(),
				},
			},
		}
	}
	defer tree.Close()

	hasError, err := tree.HasErrorNodes()
	if err != nil {
		return astmerge.ParseResult[LanguagePackAnalysis]{
			OK: false,
			Diagnostics: []astmerge.Diagnostic{
				{
					Severity: astmerge.SeverityError,
					Category: astmerge.CategoryUnsupportedFeature,
					Message:  err.Error(),
				},
			},
		}
	}
	if hasError {
		return astmerge.ParseResult[LanguagePackAnalysis]{
			OK: false,
			Diagnostics: []astmerge.Diagnostic{
				{
					Severity: astmerge.SeverityError,
					Category: astmerge.CategoryParseError,
					Message:  "tree-sitter-language-pack reported syntax errors for " + request.Language + ".",
				},
			},
		}
	}

	rootType, err := tree.RootNodeType()
	if err != nil {
		return astmerge.ParseResult[LanguagePackAnalysis]{
			OK: false,
			Diagnostics: []astmerge.Diagnostic{
				{
					Severity: astmerge.SeverityError,
					Category: astmerge.CategoryUnsupportedFeature,
					Message:  err.Error(),
				},
			},
		}
	}

	analysis := LanguagePackAnalysis{
		Language:   request.Language,
		Dialect:    request.Dialect,
		RootType:   rootType,
		HasError:   false,
		BackendRef: KreuzbergLanguagePackBackend,
	}
	return astmerge.ParseResult[LanguagePackAnalysis]{
		OK:          true,
		Diagnostics: []astmerge.Diagnostic{},
		Analysis:    &analysis,
	}
}

func ProcessWithLanguagePack(request ProcessRequest) astmerge.ParseResult[LanguagePackProcessAnalysis] {
	registry, err := languagePackRegistryInstance()
	if err != nil {
		return astmerge.ParseResult[LanguagePackProcessAnalysis]{
			OK: false,
			Diagnostics: []astmerge.Diagnostic{
				{
					Severity: astmerge.SeverityError,
					Category: astmerge.CategoryUnsupportedFeature,
					Message:  err.Error(),
				},
			},
		}
	}

	if err := ensureLanguageAvailable(registry, request.Language); err != nil {
		return astmerge.ParseResult[LanguagePackProcessAnalysis]{
			OK: false,
			Diagnostics: []astmerge.Diagnostic{
				{
					Severity: astmerge.SeverityError,
					Category: astmerge.CategoryUnsupportedFeature,
					Message:  err.Error(),
				},
			},
		}
	}

	result, err := registry.Process(request.Source, tspack.ProcessConfig{
		Language:    request.Language,
		Structure:   true,
		Imports:     true,
		Diagnostics: true,
	})
	if err != nil {
		return astmerge.ParseResult[LanguagePackProcessAnalysis]{
			OK: false,
			Diagnostics: []astmerge.Diagnostic{
				{
					Severity: astmerge.SeverityError,
					Category: astmerge.CategoryUnsupportedFeature,
					Message:  err.Error(),
				},
			},
		}
	}

	analysis := LanguagePackProcessAnalysis{
		Language:    request.Language,
		Structure:   make([]ProcessStructureItem, 0, len(result.Structure)),
		Imports:     make([]ProcessImportInfo, 0, len(result.Imports)),
		Diagnostics: make([]ProcessDiagnostic, 0, len(result.Diagnostics)),
		BackendRef:  KreuzbergLanguagePackBackend,
	}
	for _, item := range result.Structure {
		analysis.Structure = append(analysis.Structure, ProcessStructureItem{
			Kind: strings.ToLower(derefString(item.Kind)),
			Name: derefStringPtr(item.Name),
			Span: ProcessSpan{
				StartByte: item.Span.StartByte,
				EndByte:   item.Span.EndByte,
				StartRow:  item.Span.StartLine,
				StartCol:  item.Span.StartColumn,
				EndRow:    item.Span.EndLine,
				EndCol:    item.Span.EndColumn,
			},
		})
	}
	for _, item := range result.Imports {
		if request.Language == "typescript" {
			analysis.Imports = append(analysis.Imports, normalizeTypeScriptImport(item))
			continue
		}
		analysis.Imports = append(analysis.Imports, ProcessImportInfo{
			Source: item.Source,
			Items:  slices.Clone(item.Items),
			Span: ProcessSpan{
				StartByte: item.Span.StartByte,
				EndByte:   item.Span.EndByte,
				StartRow:  item.Span.StartLine,
				StartCol:  item.Span.StartColumn,
				EndRow:    item.Span.EndLine,
				EndCol:    item.Span.EndColumn,
			},
		})
	}
	for _, item := range result.Diagnostics {
		analysis.Diagnostics = append(analysis.Diagnostics, ProcessDiagnostic{
			Message:  item.Message,
			Severity: item.Severity,
		})
	}

	return astmerge.ParseResult[LanguagePackProcessAnalysis]{
		OK:          true,
		Diagnostics: []astmerge.Diagnostic{},
		Analysis:    &analysis,
	}
}

func derefString(value string) string {
	return value
}

func derefStringPtr(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func normalizeTypeScriptImport(item tspack.ImportInfo) ProcessImportInfo {
	source := item.Source
	if strings.Contains(source, "from") {
		if quoteParts := strings.Split(source, "'"); len(quoteParts) >= 2 {
			source = quoteParts[1]
		} else if quoteParts := strings.Split(source, "\""); len(quoteParts) >= 2 {
			source = quoteParts[1]
		}
	}

	items := make([]string, 0)
	if start := strings.Index(item.Source, "{"); start >= 0 {
		if end := strings.Index(item.Source[start+1:], "}"); end >= 0 {
			rawItems := item.Source[start+1 : start+1+end]
			for _, part := range strings.Split(rawItems, ",") {
				part = strings.TrimSpace(strings.ReplaceAll(part, "type", ""))
				if part != "" {
					items = append(items, part)
				}
			}
		}
	}

	return ProcessImportInfo{
		Source: source,
		Items:  items,
		Span: ProcessSpan{
			StartByte: item.Span.StartByte,
			EndByte:   item.Span.EndByte,
			StartRow:  item.Span.StartLine,
			StartCol:  item.Span.StartColumn,
			EndRow:    item.Span.EndLine,
			EndCol:    item.Span.EndColumn,
		},
	}
}
