package goldmarkmerge

import (
	"fmt"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/markdownmerge"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
)

const BackendGoldmark = "goldmark"

func unsupportedFeature(message string) astmerge.Diagnostic {
	return astmerge.Diagnostic{
		Severity: astmerge.SeverityError,
		Category: astmerge.CategoryUnsupportedFeature,
		Message:  message,
	}
}

func MarkdownFeatureProfileInfo() markdownmerge.MarkdownFeatureProfile {
	return markdownmerge.MarkdownFeatureProfileInfo()
}

func AvailableMarkdownBackends() []string {
	return []string{BackendGoldmark}
}

func MarkdownBackendFeatureProfileInfo() markdownmerge.MarkdownBackendFeatureProfile {
	return markdownmerge.MarkdownBackendFeatureProfile{
		Family:            "markdown",
		SupportedDialects: []markdownmerge.MarkdownDialect{markdownmerge.DialectMarkdown},
		SupportedPolicies: []astmerge.PolicyReference{},
		Backend:           BackendGoldmark,
	}
}

func MarkdownPlanContext() astmerge.ConformanceFamilyPlanContext {
	return astmerge.ConformanceFamilyPlanContext{
		FamilyProfile: astmerge.FamilyFeatureProfile{
			Family:            "markdown",
			SupportedDialects: []string{"markdown"},
			SupportedPolicies: []astmerge.PolicyReference{},
		},
		FeatureProfile: &astmerge.ConformanceFeatureProfileView{
			Backend:           BackendGoldmark,
			SupportsDialects:  true,
			SupportedPolicies: []astmerge.PolicyReference{},
		},
	}
}

func ParseMarkdown(source string, dialect markdownmerge.MarkdownDialect, backend ...string) astmerge.ParseResult[markdownmerge.MarkdownAnalysis] {
	requested := BackendGoldmark
	if len(backend) > 0 && backend[0] != "" {
		requested = backend[0]
	}
	if requested != BackendGoldmark {
		return astmerge.ParseResult[markdownmerge.MarkdownAnalysis]{
			OK:          false,
			Diagnostics: []astmerge.Diagnostic{unsupportedFeature(fmt.Sprintf("Unsupported Markdown backend %s.", requested))},
		}
	}
	if dialect != markdownmerge.DialectMarkdown {
		return astmerge.ParseResult[markdownmerge.MarkdownAnalysis]{
			OK:          false,
			Diagnostics: []astmerge.Diagnostic{unsupportedFeature(fmt.Sprintf("Unsupported Markdown dialect %s.", dialect))},
		}
	}

	parser := goldmark.DefaultParser()
	_ = parser.Parse(text.NewReader([]byte(source)))
	normalized := markdownmerge.NormalizeMarkdownSource(source)

	return astmerge.ParseResult[markdownmerge.MarkdownAnalysis]{
		OK: true,
		Analysis: &markdownmerge.MarkdownAnalysis{
			Dialect:          dialect,
			NormalizedSource: normalized,
			RootKind:         markdownmerge.RootDocument,
			Owners:           markdownmerge.CollectMarkdownOwners(normalized),
		},
		Diagnostics: []astmerge.Diagnostic{},
		Policies:    []astmerge.PolicyReference{},
	}
}

func MatchMarkdownOwners(template, destination markdownmerge.MarkdownAnalysis) markdownmerge.MarkdownOwnerMatchResult {
	return markdownmerge.MatchMarkdownOwners(template, destination)
}
