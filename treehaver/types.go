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

func (LanguagePackAnalysis) Kind() string {
	return "tree-sitter"
}

func (LanguagePackProcessAnalysis) Kind() string {
	return "tree-sitter-process"
}

var KreuzbergLanguagePackBackend = BackendReference{
	ID:     "kreuzberg-language-pack",
	Family: "tree-sitter",
}

var PigeonBackend = BackendReference{
	ID:     "pigeon",
	Family: "peg",
}

func BackendReferenceByID(id string) *BackendReference {
	switch id {
	case KreuzbergLanguagePackBackend.ID:
		return &KreuzbergLanguagePackBackend
	case PigeonBackend.ID:
		return &PigeonBackend
	default:
		return nil
	}
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
