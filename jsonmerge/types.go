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
		}
	}

	stripped := stripJSONComments(source)
	normalizedSource := source
	if dialect == DialectJSON {
		if stripped != source {
			return astmerge.ParseResult[JSONAnalysis]{
				OK:          false,
				Diagnostics: []astmerge.Diagnostic{parseError("Comments are not supported in strict JSON.")},
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
	}
}

func JSONParseRequest(source string, dialect JSONDialect) treehaver.ParserRequest {
	return treehaver.ParserRequest{
		Source:   source,
		Language: "json",
		Dialect:  string(dialect),
	}
}
