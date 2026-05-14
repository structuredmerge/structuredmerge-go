//go:build tspack || tspack_dev

package treehaver

import (
	"fmt"
	"slices"
	"strings"

	tspack "github.com/kreuzberg-dev/tree-sitter-language-pack/packages/go"
)

func ensureLanguageAvailable(language string) error {
	if boolValue(tspack.HasLanguage(language)) {
		return nil
	}

	if _, err := tspack.Download([]string{language}); err != nil {
		return err
	}

	if boolValue(tspack.HasLanguage(language)) {
		return nil
	}

	return fmt.Errorf("tree-sitter-language-pack language %q is not available after download", language)
}

func ParseWithLanguagePack(request ParserRequest) ParseResult[LanguagePackAnalysis] {
	if err := ensureLanguageAvailable(request.Language); err != nil {
		return ParseResult[LanguagePackAnalysis]{
			OK: false,
			Diagnostics: []Diagnostic{
				{
					Severity: SeverityError,
					Category: CategoryUnsupportedFeature,
					Message:  err.Error(),
				},
			},
		}
	}

	result, err := tspack.Process(request.Source, tspack.ProcessConfig{
		Language:    request.Language,
		Diagnostics: true,
	})
	if err != nil {
		return ParseResult[LanguagePackAnalysis]{
			OK: false,
			Diagnostics: []Diagnostic{
				{
					Severity: SeverityError,
					Category: CategoryParseError,
					Message:  err.Error(),
				},
			},
		}
	}

	hasError := result.Metrics.ErrorCount > 0 || len(result.Diagnostics) > 0
	if hasError {
		return ParseResult[LanguagePackAnalysis]{
			OK: false,
			Diagnostics: []Diagnostic{
				{
					Severity: SeverityError,
					Category: CategoryParseError,
					Message:  "tree-sitter-language-pack reported syntax errors for " + request.Language + ".",
				},
			},
		}
	}

	analysis := LanguagePackAnalysis{
		Language:   request.Language,
		Dialect:    request.Dialect,
		RootType:   result.Language,
		HasError:   false,
		BackendRef: KreuzbergLanguagePackBackend,
	}
	return ParseResult[LanguagePackAnalysis]{
		OK:          true,
		Diagnostics: []Diagnostic{},
		Analysis:    &analysis,
	}
}

func ProcessWithLanguagePack(request ProcessRequest) ParseResult[LanguagePackProcessAnalysis] {
	if err := ensureLanguageAvailable(request.Language); err != nil {
		return ParseResult[LanguagePackProcessAnalysis]{
			OK: false,
			Diagnostics: []Diagnostic{
				{
					Severity: SeverityError,
					Category: CategoryUnsupportedFeature,
					Message:  err.Error(),
				},
			},
		}
	}

	result, err := tspack.Process(request.Source, tspack.ProcessConfig{
		Language:    request.Language,
		Structure:   boolPtr(true),
		Imports:     boolPtr(true),
		Diagnostics: true,
	})
	if err != nil {
		return ParseResult[LanguagePackProcessAnalysis]{
			OK: false,
			Diagnostics: []Diagnostic{
				{
					Severity: SeverityError,
					Category: CategoryUnsupportedFeature,
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
			Kind: strings.ToLower(string(item.Kind)),
			Name: derefStringPtr(item.Name),
			Span: processSpanFromLanguagePack(item.Span),
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
			Span:   processSpanFromLanguagePack(item.Span),
		})
	}
	for _, item := range result.Diagnostics {
		analysis.Diagnostics = append(analysis.Diagnostics, ProcessDiagnostic{
			Message:  item.Message,
			Severity: string(item.Severity),
		})
	}

	return ParseResult[LanguagePackProcessAnalysis]{
		OK:          true,
		Diagnostics: []Diagnostic{},
		Analysis:    &analysis,
	}
}

func boolPtr(value bool) *bool {
	return &value
}

func boolValue(value *bool) bool {
	return value != nil && *value
}

func derefStringPtr(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func processSpanFromLanguagePack(span tspack.Span) ProcessSpan {
	return ProcessSpan{
		StartByte: int(span.StartByte),
		EndByte:   int(span.EndByte),
		StartRow:  int(span.StartLine),
		StartCol:  int(span.StartColumn),
		EndRow:    int(span.EndLine),
		EndCol:    int(span.EndColumn),
	}
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
		Span:   processSpanFromLanguagePack(item.Span),
	}
}
