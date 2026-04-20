package markdownmerge

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/treehaver"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
)

type MarkdownDialect string

const (
	DialectMarkdown MarkdownDialect = "markdown"
)

type MarkdownBackend string

const (
	BackendGoldmark  MarkdownBackend = "goldmark"
	BackendKreuzberg MarkdownBackend = "kreuzberg-language-pack"
)

type MarkdownRootKind string

const (
	RootDocument MarkdownRootKind = "document"
)

type MarkdownOwnerKind string

const (
	OwnerHeading   MarkdownOwnerKind = "heading"
	OwnerCodeFence MarkdownOwnerKind = "code_fence"
)

type MarkdownOwner struct {
	Path       string
	OwnerKind  MarkdownOwnerKind
	MatchKey   string
	Level      int
	InfoString string
}

type MarkdownOwnerMatch struct {
	TemplatePath    string
	DestinationPath string
}

type MarkdownOwnerMatchResult struct {
	Matched              []MarkdownOwnerMatch
	UnmatchedTemplate    []string
	UnmatchedDestination []string
}

type MarkdownAnalysis struct {
	Dialect          MarkdownDialect
	NormalizedSource string
	RootKind         MarkdownRootKind
	Owners           []MarkdownOwner
}

func (MarkdownAnalysis) Kind() string {
	return "markdown"
}

type MarkdownFeatureProfile struct {
	Family            string
	SupportedDialects []MarkdownDialect
	SupportedPolicies []astmerge.PolicyReference
}

type MarkdownBackendFeatureProfile struct {
	Family            string
	SupportedDialects []MarkdownDialect
	SupportedPolicies []astmerge.PolicyReference
	Backend           string
}

func unsupportedFeature(message string) astmerge.Diagnostic {
	return astmerge.Diagnostic{
		Severity: astmerge.SeverityError,
		Category: astmerge.CategoryUnsupportedFeature,
		Message:  message,
	}
}

func NormalizeMarkdownSource(source string) string {
	source = strings.ReplaceAll(source, "\r\n", "\n")
	return strings.ReplaceAll(source, "\r", "\n")
}

func slugify(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	var builder strings.Builder
	lastDash := false
	for _, character := range value {
		if (character >= 'a' && character <= 'z') || (character >= '0' && character <= '9') {
			builder.WriteRune(character)
			lastDash = false
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	result := strings.Trim(builder.String(), "-")
	if result == "" {
		return "section"
	}
	return result
}

var headingPattern = regexp.MustCompile(`^(#{1,6})\s+(.+?)\s*#*\s*$`)
var codeFencePattern = regexp.MustCompile("^\\s*([`]{3,}|[~]{3,})\\s*(.*?)\\s*$")

func CollectMarkdownOwners(source string) []MarkdownOwner {
	lines := strings.Split(NormalizeMarkdownSource(source), "\n")
	owners := make([]MarkdownOwner, 0)
	headingIndex := 0
	codeFenceIndex := 0

	for index := 0; index < len(lines); index++ {
		line := lines[index]
		if heading := headingPattern.FindStringSubmatch(line); heading != nil {
			level := len(heading[1])
			title := strings.TrimSpace(heading[2])
			owners = append(owners, MarkdownOwner{
				Path:      fmt.Sprintf("/heading/%d", headingIndex),
				OwnerKind: OwnerHeading,
				MatchKey:  fmt.Sprintf("h%d:%s", level, slugify(title)),
				Level:     level,
			})
			headingIndex++
			continue
		}

		fence := codeFencePattern.FindStringSubmatch(line)
		if fence == nil {
			continue
		}

		marker := fence[1]
		infoString := ""
		if rest := strings.TrimSpace(fence[2]); rest != "" {
			infoString = strings.Fields(rest)[0]
		}

		matchKey := "fence:plain"
		if infoString != "" {
			matchKey = "fence:" + infoString
		}

		owners = append(owners, MarkdownOwner{
			Path:       fmt.Sprintf("/code_fence/%d", codeFenceIndex),
			OwnerKind:  OwnerCodeFence,
			MatchKey:   matchKey,
			InfoString: infoString,
		})
		codeFenceIndex++

		markerChar := marker[:1]
		markerLength := len(marker)
		for cursor := index + 1; cursor < len(lines); cursor++ {
			trimmed := strings.TrimSpace(lines[cursor])
			if len(trimmed) >= markerLength &&
				strings.Trim(trimmed, markerChar) == "" &&
				strings.HasPrefix(trimmed, strings.Repeat(markerChar, markerLength)) {
				index = cursor
				break
			}
			if cursor == len(lines)-1 {
				index = cursor
			}
		}
	}

	return owners
}

func validateNativeMarkdown(source string) {
	parser := goldmark.DefaultParser()
	_ = parser.Parse(text.NewReader([]byte(source)))
}

func MarkdownFeatureProfileInfo() MarkdownFeatureProfile {
	return MarkdownFeatureProfile{
		Family:            "markdown",
		SupportedDialects: []MarkdownDialect{DialectMarkdown},
		SupportedPolicies: []astmerge.PolicyReference{},
	}
}

func AvailableMarkdownBackends() []MarkdownBackend {
	return []MarkdownBackend{BackendGoldmark, BackendKreuzberg}
}

func MarkdownBackendFeatureProfileInfo(backend MarkdownBackend) MarkdownBackendFeatureProfile {
	return MarkdownBackendFeatureProfile{
		Family:            "markdown",
		SupportedDialects: []MarkdownDialect{DialectMarkdown},
		SupportedPolicies: []astmerge.PolicyReference{},
		Backend:           string(backend),
	}
}

func MarkdownPlanContext() astmerge.ConformanceFamilyPlanContext {
	return MarkdownPlanContextWithBackend(BackendGoldmark)
}

func MarkdownPlanContextWithBackend(backend MarkdownBackend) astmerge.ConformanceFamilyPlanContext {
	return astmerge.ConformanceFamilyPlanContext{
		FamilyProfile: astmerge.FamilyFeatureProfile{
			Family:            "markdown",
			SupportedDialects: []string{"markdown"},
			SupportedPolicies: []astmerge.PolicyReference{},
		},
		FeatureProfile: &astmerge.ConformanceFeatureProfileView{
			Backend:           string(backend),
			SupportsDialects:  backend != BackendKreuzberg,
			SupportedPolicies: []astmerge.PolicyReference{},
		},
	}
}

func ParseMarkdown(source string, dialect MarkdownDialect) astmerge.ParseResult[MarkdownAnalysis] {
	return ParseMarkdownWithBackend(source, dialect, BackendGoldmark)
}

func ParseMarkdownWithBackend(source string, dialect MarkdownDialect, backend MarkdownBackend) astmerge.ParseResult[MarkdownAnalysis] {
	if dialect != DialectMarkdown {
		return astmerge.ParseResult[MarkdownAnalysis]{
			OK:          false,
			Diagnostics: []astmerge.Diagnostic{unsupportedFeature(fmt.Sprintf("Unsupported Markdown dialect %s.", dialect))},
		}
	}

	switch backend {
	case BackendGoldmark:
		validateNativeMarkdown(source)
	case BackendKreuzberg:
		result := treehaver.ParseWithLanguagePack(treehaver.ParserRequest{
			Source:   source,
			Language: "markdown",
			Dialect:  "markdown",
		})
		if !result.OK {
			return astmerge.ParseResult[MarkdownAnalysis]{
				OK:          false,
				Diagnostics: result.Diagnostics,
			}
		}
	default:
		return astmerge.ParseResult[MarkdownAnalysis]{
			OK:          false,
			Diagnostics: []astmerge.Diagnostic{unsupportedFeature(fmt.Sprintf("Unsupported Markdown backend %s.", backend))},
		}
	}

	normalized := NormalizeMarkdownSource(source)
	return astmerge.ParseResult[MarkdownAnalysis]{
		OK:          true,
		Diagnostics: []astmerge.Diagnostic{},
		Analysis: &MarkdownAnalysis{
			Dialect:          dialect,
			NormalizedSource: normalized,
			RootKind:         RootDocument,
			Owners:           CollectMarkdownOwners(normalized),
		},
		Policies: []astmerge.PolicyReference{},
	}
}

func MatchMarkdownOwners(template MarkdownAnalysis, destination MarkdownAnalysis) MarkdownOwnerMatchResult {
	destinationPaths := make(map[string]bool, len(destination.Owners))
	templatePaths := make(map[string]bool, len(template.Owners))
	for _, owner := range destination.Owners {
		destinationPaths[owner.Path] = true
	}
	for _, owner := range template.Owners {
		templatePaths[owner.Path] = true
	}

	result := MarkdownOwnerMatchResult{
		Matched:              make([]MarkdownOwnerMatch, 0),
		UnmatchedTemplate:    make([]string, 0),
		UnmatchedDestination: make([]string, 0),
	}

	for _, owner := range template.Owners {
		if destinationPaths[owner.Path] {
			result.Matched = append(result.Matched, MarkdownOwnerMatch{
				TemplatePath:    owner.Path,
				DestinationPath: owner.Path,
			})
		} else {
			result.UnmatchedTemplate = append(result.UnmatchedTemplate, owner.Path)
		}
	}

	for _, owner := range destination.Owners {
		if !templatePaths[owner.Path] {
			result.UnmatchedDestination = append(result.UnmatchedDestination, owner.Path)
		}
	}

	slices.Sort(result.UnmatchedTemplate)
	slices.Sort(result.UnmatchedDestination)
	return result
}
