package treehaver

import (
	"fmt"
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

type LanguagePackAnalysis struct {
	Language   string
	Dialect    string
	RootType   string
	HasError   bool
	BackendRef BackendReference
}

func (LanguagePackAnalysis) Kind() string {
	return "tree-sitter"
}

var KreuzbergLanguagePackBackend = BackendReference{
	ID:     "kreuzberg-language-pack",
	Family: "tree-sitter",
}

func LanguagePackAdapterInfo() AdapterInfo {
	return AdapterInfo{
		Backend:          KreuzbergLanguagePackBackend.ID,
		BackendRef:       &KreuzbergLanguagePackBackend,
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
