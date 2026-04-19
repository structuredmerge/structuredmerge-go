package jsonmerge

import (
	"encoding/json"
	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/treehaver"
	"slices"
	"strconv"
	"strings"
)

type JSONDialect string

const (
	DialectJSON  JSONDialect = "json"
	DialectJSONC JSONDialect = "jsonc"
)

type JSONRootKind string

const (
	RootObject JSONRootKind = "object"
	RootArray  JSONRootKind = "array"
	RootScalar JSONRootKind = "scalar"
)

type JSONOwnerKind string

const (
	OwnerMember  JSONOwnerKind = "member"
	OwnerElement JSONOwnerKind = "element"
)

type JSONOwner struct {
	Path      string
	OwnerKind JSONOwnerKind
	MatchKey  string
}

type JSONOwnerMatch struct {
	TemplatePath    string
	DestinationPath string
}

type JSONOwnerMatchResult struct {
	Matched              []JSONOwnerMatch
	UnmatchedTemplate    []string
	UnmatchedDestination []string
}

type JSONAnalysis struct {
	Dialect          JSONDialect
	AllowsComments   bool
	NormalizedSource string
	RootKind         JSONRootKind
	Owners           []JSONOwner
}

func (JSONAnalysis) Kind() string {
	return "json"
}

type JSONParserAdapter interface {
	treehaver.ParserAdapter[JSONAnalysis]
}

type JSONAnalyzer interface {
	Parse(source string, dialect JSONDialect) astmerge.ParseResult[JSONAnalysis]
}

type JSONStructureAnalyzer interface {
	Analyze(source string, dialect JSONDialect) astmerge.ParseResult[JSONAnalysis]
}

type JSONOwnerMatcher interface {
	MatchOwners(template JSONAnalysis, destination JSONAnalysis) JSONOwnerMatchResult
}

type JSONMergeResolution struct {
	Output string
}

type JSONFeatureProfile struct {
	Family            string
	SupportedDialects []JSONDialect
	SupportedPolicies []astmerge.PolicyReference
}

type JSONMerger interface {
	Merge(template JSONAnalysis, destination JSONAnalysis) astmerge.MergeResult[string]
}

func parseError(message string) astmerge.Diagnostic {
	return astmerge.Diagnostic{
		Severity: astmerge.SeverityError,
		Category: astmerge.CategoryParseError,
		Message:  message,
	}
}

func destinationParseError(message string) astmerge.Diagnostic {
	return astmerge.Diagnostic{
		Severity: astmerge.SeverityError,
		Category: astmerge.CategoryDestinationParseError,
		Message:  message,
	}
}

func fallbackApplied(message string) astmerge.Diagnostic {
	return astmerge.Diagnostic{
		Severity: astmerge.SeverityWarning,
		Category: astmerge.CategoryFallbackApplied,
		Message:  message,
	}
}

func destinationWinsArrayPolicy() astmerge.PolicyReference {
	return astmerge.PolicyReference{
		Surface: astmerge.PolicySurfaceArray,
		Name:    "destination_wins_array",
	}
}

func trailingCommaFallbackPolicy() astmerge.PolicyReference {
	return astmerge.PolicyReference{
		Surface: astmerge.PolicySurfaceFallback,
		Name:    "trailing_comma_destination_fallback",
	}
}

func JSONFeatureProfileInfo() JSONFeatureProfile {
	shared := astmerge.FamilyFeatureProfile{
		Family:            "json",
		SupportedDialects: []string{"json", "jsonc"},
		SupportedPolicies: []astmerge.PolicyReference{
			destinationWinsArrayPolicy(),
			trailingCommaFallbackPolicy(),
		},
	}

	dialects := make([]JSONDialect, 0, len(shared.SupportedDialects))
	for _, dialect := range shared.SupportedDialects {
		switch dialect {
		case "jsonc":
			dialects = append(dialects, DialectJSONC)
		default:
			dialects = append(dialects, DialectJSON)
		}
	}

	return JSONFeatureProfile{
		Family:            shared.Family,
		SupportedDialects: dialects,
		SupportedPolicies: shared.SupportedPolicies,
	}
}

func ParseJSONWithLanguagePack(source string, dialect JSONDialect) astmerge.ParseResult[JSONAnalysis] {
	if dialect != DialectJSON {
		return astmerge.ParseResult[JSONAnalysis]{
			OK: false,
			Diagnostics: []astmerge.Diagnostic{
				{
					Severity: astmerge.SeverityError,
					Category: astmerge.CategoryUnsupportedFeature,
					Message:  "tree-sitter-language-pack json parsing currently supports only the json dialect.",
				},
			},
		}
	}

	backendResult := treehaver.ParseWithLanguagePack(JSONParseRequest(source, dialect))
	if !backendResult.OK {
		return astmerge.ParseResult[JSONAnalysis]{
			OK:          false,
			Diagnostics: backendResult.Diagnostics,
		}
	}

	return ParseJSON(source, dialect)
}

func detectTrailingComma(source string) bool {
	inString := false
	inLineComment := false
	inBlockComment := false
	escaped := false

	for index := 0; index < len(source); index++ {
		char := source[index]
		var next byte
		if index+1 < len(source) {
			next = source[index+1]
		}

		if inLineComment {
			if char == '\n' {
				inLineComment = false
			}
			continue
		}

		if inBlockComment {
			if char == '*' && next == '/' {
				inBlockComment = false
				index++
			}
			continue
		}

		if inString {
			if escaped {
				escaped = false
				continue
			}
			if char == '\\' {
				escaped = true
				continue
			}
			if char == '"' {
				inString = false
			}
			continue
		}

		if char == '"' {
			inString = true
			continue
		}

		if char == '/' && next == '/' {
			inLineComment = true
			index++
			continue
		}

		if char == '/' && next == '*' {
			inBlockComment = true
			index++
			continue
		}

		if char == ',' {
			lookahead := index + 1
			for lookahead < len(source) {
				switch source[lookahead] {
				case ' ', '\n', '\r', '\t':
					lookahead++
				default:
					goto done
				}
			}
		done:
			if lookahead < len(source) && (source[lookahead] == ']' || source[lookahead] == '}') {
				return true
			}
		}
	}

	return false
}

func stripJSONComments(source string) string {
	result := make([]byte, 0, len(source))
	inString := false
	inLineComment := false
	inBlockComment := false
	escaped := false

	for index := 0; index < len(source); index++ {
		char := source[index]
		var next byte
		if index+1 < len(source) {
			next = source[index+1]
		}

		if inLineComment {
			if char == '\n' {
				inLineComment = false
				result = append(result, '\n')
			}
			continue
		}

		if inBlockComment {
			if char == '*' && next == '/' {
				inBlockComment = false
				index++
			}
			continue
		}

		if inString {
			result = append(result, char)
			if escaped {
				escaped = false
				continue
			}
			if char == '\\' {
				escaped = true
				continue
			}
			if char == '"' {
				inString = false
			}
			continue
		}

		if char == '"' {
			inString = true
			result = append(result, char)
			continue
		}

		if char == '/' && next == '/' {
			inLineComment = true
			index++
			continue
		}

		if char == '/' && next == '*' {
			inBlockComment = true
			index++
			continue
		}

		result = append(result, char)
	}

	return string(result)
}

func stripTrailingCommas(source string) string {
	result := make([]byte, 0, len(source))
	inString := false
	inLineComment := false
	inBlockComment := false
	escaped := false

	for index := 0; index < len(source); index++ {
		char := source[index]
		var next byte
		if index+1 < len(source) {
			next = source[index+1]
		}

		if inLineComment {
			result = append(result, char)
			if char == '\n' {
				inLineComment = false
			}
			continue
		}

		if inBlockComment {
			result = append(result, char)
			if char == '*' && next == '/' {
				result = append(result, next)
				inBlockComment = false
				index++
			}
			continue
		}

		if inString {
			result = append(result, char)
			if escaped {
				escaped = false
				continue
			}
			if char == '\\' {
				escaped = true
				continue
			}
			if char == '"' {
				inString = false
			}
			continue
		}

		if char == '"' {
			inString = true
			result = append(result, char)
			continue
		}

		if char == '/' && next == '/' {
			inLineComment = true
			result = append(result, char, next)
			index++
			continue
		}

		if char == '/' && next == '*' {
			inBlockComment = true
			result = append(result, char, next)
			index++
			continue
		}

		if char == ',' {
			lookahead := index + 1
			for lookahead < len(source) {
				switch source[lookahead] {
				case ' ', '\n', '\r', '\t':
					lookahead++
				default:
					goto done
				}
			}
		done:
			if lookahead < len(source) && (source[lookahead] == ']' || source[lookahead] == '}') {
				continue
			}
		}

		result = append(result, char)
	}

	return string(result)
}

func escapePointerSegment(segment string) string {
	segment = strings.ReplaceAll(segment, "~", "~0")
	segment = strings.ReplaceAll(segment, "/", "~1")
	return segment
}

func analyzeValue(value any, path string) (JSONRootKind, []JSONOwner) {
	switch typed := value.(type) {
	case map[string]any:
		owners := make([]JSONOwner, 0)
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		for _, key := range keys {
			childPath := path + "/" + escapePointerSegment(key)
			owners = append(owners, JSONOwner{
				Path:      childPath,
				OwnerKind: OwnerMember,
				MatchKey:  key,
			})
			_, childOwners := analyzeValue(typed[key], childPath)
			owners = append(owners, childOwners...)
		}
		return RootObject, owners
	case []any:
		owners := make([]JSONOwner, 0, len(typed))
		for index, child := range typed {
			childPath := path + "/" + strconv.Itoa(index)
			owners = append(owners, JSONOwner{
				Path:      childPath,
				OwnerKind: OwnerElement,
			})
			_, childOwners := analyzeValue(child, childPath)
			owners = append(owners, childOwners...)
		}
		return RootArray, owners
	default:
		return RootScalar, []JSONOwner{}
	}
}

func ParseJSON(source string, dialect JSONDialect) astmerge.ParseResult[JSONAnalysis] {
	if detectTrailingComma(source) {
		return astmerge.ParseResult[JSONAnalysis]{
			OK:          false,
			Diagnostics: []astmerge.Diagnostic{parseError("Trailing commas are not supported.")},
			Policies:    []astmerge.PolicyReference{},
		}
	}

	stripped := stripJSONComments(source)
	normalizedSource := source
	if dialect == DialectJSON {
		if stripped != source {
			return astmerge.ParseResult[JSONAnalysis]{
				OK:          false,
				Diagnostics: []astmerge.Diagnostic{parseError("Comments are not supported in strict JSON.")},
				Policies:    []astmerge.PolicyReference{},
			}
		}
	} else {
		normalizedSource = stripped
	}

	var decoded any
	if err := json.Unmarshal([]byte(normalizedSource), &decoded); err != nil {
		return astmerge.ParseResult[JSONAnalysis]{
			OK:          false,
			Diagnostics: []astmerge.Diagnostic{parseError("JSON parse failed.")},
			Policies:    []astmerge.PolicyReference{},
		}
	}
	rootKind, owners := analyzeValue(decoded, "")

	analysis := JSONAnalysis{
		Dialect:          dialect,
		AllowsComments:   dialect == DialectJSONC,
		NormalizedSource: normalizedSource,
		RootKind:         rootKind,
		Owners:           owners,
	}

	return astmerge.ParseResult[JSONAnalysis]{
		OK:          true,
		Diagnostics: []astmerge.Diagnostic{},
		Analysis:    &analysis,
		Policies:    []astmerge.PolicyReference{},
	}
}

func JSONParseRequest(source string, dialect JSONDialect) treehaver.ParserRequest {
	return treehaver.ParserRequest{
		Source:   source,
		Language: "json",
		Dialect:  string(dialect),
	}
}

func MatchJSONOwners(template JSONAnalysis, destination JSONAnalysis) JSONOwnerMatchResult {
	destinationPaths := map[string]struct{}{}
	templatePaths := map[string]struct{}{}
	for _, owner := range destination.Owners {
		destinationPaths[owner.Path] = struct{}{}
	}
	for _, owner := range template.Owners {
		templatePaths[owner.Path] = struct{}{}
	}

	matched := make([]JSONOwnerMatch, 0)
	unmatchedTemplate := make([]string, 0)
	unmatchedDestination := make([]string, 0)

	for _, owner := range template.Owners {
		if _, ok := destinationPaths[owner.Path]; ok {
			matched = append(matched, JSONOwnerMatch{
				TemplatePath:    owner.Path,
				DestinationPath: owner.Path,
			})
			continue
		}
		unmatchedTemplate = append(unmatchedTemplate, owner.Path)
	}

	for _, owner := range destination.Owners {
		if _, ok := templatePaths[owner.Path]; !ok {
			unmatchedDestination = append(unmatchedDestination, owner.Path)
		}
	}

	return JSONOwnerMatchResult{
		Matched:              matched,
		UnmatchedTemplate:    unmatchedTemplate,
		UnmatchedDestination: unmatchedDestination,
	}
}

func parseNormalizedJSON(source string, dialect JSONDialect, diagnosticFactory func(string) astmerge.Diagnostic) (any, astmerge.Diagnostic, bool) {
	result := ParseJSON(source, dialect)
	if !result.OK || result.Analysis == nil {
		if len(result.Diagnostics) > 0 {
			return nil, diagnosticFactory(result.Diagnostics[0].Message), false
		}
		return nil, diagnosticFactory("JSON parse failed."), false
	}

	var decoded any
	if err := json.Unmarshal([]byte(result.Analysis.NormalizedSource), &decoded); err != nil {
		return nil, diagnosticFactory("JSON parse failed."), false
	}

	return decoded, astmerge.Diagnostic{}, true
}

func mergeValues(template any, destination any) any {
	switch templateTyped := template.(type) {
	case map[string]any:
		destinationTyped, ok := destination.(map[string]any)
		if !ok {
			return destination
		}

		keys := make([]string, 0)
		keySet := map[string]struct{}{}
		for key := range templateTyped {
			keySet[key] = struct{}{}
		}
		for key := range destinationTyped {
			keySet[key] = struct{}{}
		}
		for key := range keySet {
			keys = append(keys, key)
		}
		slices.Sort(keys)

		merged := map[string]any{}
		for _, key := range keys {
			templateValue, hasTemplate := templateTyped[key]
			destinationValue, hasDestination := destinationTyped[key]
			switch {
			case hasTemplate && hasDestination:
				merged[key] = mergeValues(templateValue, destinationValue)
			case hasDestination:
				merged[key] = destinationValue
			case hasTemplate:
				merged[key] = templateValue
			}
		}
		return merged
	case []any:
		if destinationArray, ok := destination.([]any); ok {
			return destinationArray
		}
		return destination
	default:
		return destination
	}
}

func canonicalJSON(value any) string {
	switch typed := value.(type) {
	case nil:
		return "null"
	case bool:
		if typed {
			return "true"
		}
		return "false"
	case float64, string, []any, map[string]any:
		switch inner := typed.(type) {
		case float64:
			bytes, _ := json.Marshal(inner)
			return string(bytes)
		case string:
			bytes, _ := json.Marshal(inner)
			return string(bytes)
		case []any:
			parts := make([]string, 0, len(inner))
			for _, item := range inner {
				parts = append(parts, canonicalJSON(item))
			}
			return "[" + strings.Join(parts, ",") + "]"
		case map[string]any:
			keys := make([]string, 0, len(inner))
			for key := range inner {
				keys = append(keys, key)
			}
			slices.Sort(keys)
			parts := make([]string, 0, len(keys))
			for _, key := range keys {
				keyBytes, _ := json.Marshal(key)
				parts = append(parts, string(keyBytes)+":"+canonicalJSON(inner[key]))
			}
			return "{" + strings.Join(parts, ",") + "}"
		default:
			return ""
		}
	default:
		bytes, _ := json.Marshal(typed)
		return string(bytes)
	}
}

func MergeJSON(templateSource string, destinationSource string, dialect JSONDialect) astmerge.MergeResult[string] {
	template, diagnostic, ok := parseNormalizedJSON(templateSource, dialect, parseError)
	if !ok {
		return astmerge.MergeResult[string]{
			OK:          false,
			Diagnostics: []astmerge.Diagnostic{diagnostic},
			Policies:    []astmerge.PolicyReference{},
		}
	}

	diagnostics := []astmerge.Diagnostic{}
	policies := []astmerge.PolicyReference{destinationWinsArrayPolicy()}
	destination, diagnostic, ok := parseNormalizedJSON(destinationSource, dialect, destinationParseError)
	if !ok {
		if diagnostic.Category == astmerge.CategoryDestinationParseError && detectTrailingComma(destinationSource) {
			sanitizedDestination := stripTrailingCommas(destinationSource)
			if sanitizedDestination == destinationSource {
				return astmerge.MergeResult[string]{
					OK:          false,
					Diagnostics: []astmerge.Diagnostic{diagnostic},
					Policies:    []astmerge.PolicyReference{},
				}
			}
			destination, diagnostic, ok = parseNormalizedJSON(sanitizedDestination, dialect, destinationParseError)
			if !ok {
				return astmerge.MergeResult[string]{
					OK:          false,
					Diagnostics: []astmerge.Diagnostic{diagnostic},
					Policies:    []astmerge.PolicyReference{},
				}
			}
			diagnostics = append(diagnostics, fallbackApplied("Applied destination trailing-comma fallback during merge."))
			policies = append(policies, trailingCommaFallbackPolicy())
		} else {
			return astmerge.MergeResult[string]{
				OK:          false,
				Diagnostics: []astmerge.Diagnostic{diagnostic},
				Policies:    []astmerge.PolicyReference{},
			}
		}
	}

	merged := mergeValues(template, destination)
	output := canonicalJSON(merged)
	return astmerge.MergeResult[string]{
		OK:          true,
		Diagnostics: diagnostics,
		Output:      &output,
		Policies:    policies,
	}
}
