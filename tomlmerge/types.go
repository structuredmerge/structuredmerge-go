package tomlmerge

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/tomlmerge/internal/pigeontoml"
	"github.com/structuredmerge/structuredmerge-go/treehaver"
)

type TOMLDialect string
type TOMLBackend string

const (
	DialectTOML TOMLDialect = "toml"

	BackendTreeSitter TOMLBackend = "kreuzberg-language-pack"
)

type TOMLRootKind string

const (
	RootTable TOMLRootKind = "table"
)

type TOMLOwnerKind string

const (
	OwnerTable     TOMLOwnerKind = "table"
	OwnerKeyValue  TOMLOwnerKind = "key_value"
	OwnerArrayItem TOMLOwnerKind = "array_item"
)

type TOMLOwner struct {
	Path      string
	OwnerKind TOMLOwnerKind
	MatchKey  string
}

type TOMLOwnerMatch struct {
	TemplatePath    string
	DestinationPath string
}

type TOMLOwnerMatchResult struct {
	Matched              []TOMLOwnerMatch
	UnmatchedTemplate    []string
	UnmatchedDestination []string
}

type TOMLAnalysis struct {
	Dialect          TOMLDialect
	NormalizedSource string
	RootKind         TOMLRootKind
	Owners           []TOMLOwner
}

func (TOMLAnalysis) Kind() string {
	return "toml"
}

type TOMLMergeResolution struct {
	Output string
}

type TOMLFeatureProfile struct {
	Family            string
	SupportedDialects []TOMLDialect
	SupportedPolicies []astmerge.PolicyReference
}

type TOMLBackendFeatureProfile struct {
	Backend           string
	BackendRef        *treehaver.BackendReference
	SupportsDialects  bool
	SupportedPolicies []astmerge.PolicyReference
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

func unsupportedFeature(message string) astmerge.Diagnostic {
	return astmerge.Diagnostic{
		Severity: astmerge.SeverityError,
		Category: astmerge.CategoryUnsupportedFeature,
		Message:  message,
	}
}

func destinationWinsArrayPolicy() astmerge.PolicyReference {
	return astmerge.PolicyReference{
		Surface: astmerge.PolicySurfaceArray,
		Name:    "destination_wins_array",
	}
}

func TOMLFeatureProfileInfo() TOMLFeatureProfile {
	shared := astmerge.FamilyFeatureProfile{
		Family:            "toml",
		SupportedDialects: []string{"toml"},
		SupportedPolicies: []astmerge.PolicyReference{destinationWinsArrayPolicy()},
	}

	return TOMLFeatureProfile{
		Family:            shared.Family,
		SupportedDialects: []TOMLDialect{DialectTOML},
		SupportedPolicies: shared.SupportedPolicies,
	}
}

func AvailableTOMLBackends() []TOMLBackend {
	return []TOMLBackend{BackendTreeSitter}
}

func TOMLBackendFeatureProfileInfo(backend TOMLBackend) TOMLBackendFeatureProfile {
	resolved := resolveBackend(backend)
	if resolved != BackendTreeSitter {
		return TOMLBackendFeatureProfile{
			Backend:           string(resolved),
			BackendRef:        nil,
			SupportsDialects:  false,
			SupportedPolicies: []astmerge.PolicyReference{destinationWinsArrayPolicy()},
		}
	}

	return TOMLBackendFeatureProfile{
		Backend:           treehaver.KreuzbergLanguagePackBackend.ID,
		BackendRef:        &treehaver.KreuzbergLanguagePackBackend,
		SupportsDialects:  false,
		SupportedPolicies: []astmerge.PolicyReference{destinationWinsArrayPolicy()},
	}
}

func TOMLPlanContext(backend TOMLBackend) astmerge.ConformanceFamilyPlanContext {
	backendProfile := TOMLBackendFeatureProfileInfo(backend)
	return astmerge.ConformanceFamilyPlanContext{
		FamilyProfile: astmerge.FamilyFeatureProfile{
			Family:            TOMLFeatureProfileInfo().Family,
			SupportedDialects: []string{string(DialectTOML)},
			SupportedPolicies: TOMLFeatureProfileInfo().SupportedPolicies,
		},
		FeatureProfile: &astmerge.ConformanceFeatureProfileView{
			Backend:           backendProfile.Backend,
			SupportsDialects:  backendProfile.SupportsDialects,
			SupportedPolicies: backendProfile.SupportedPolicies,
		},
	}
}

func displayPath(path string) string {
	if path == "" {
		return "/"
	}
	return path
}

func isScalar(value any) bool {
	switch value.(type) {
	case string, int64, float64, bool:
		return true
	default:
		return false
	}
}

func validateTOMLNode(value any, path string) *astmerge.Diagnostic {
	switch node := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(node))
		for key := range node {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		for _, key := range keys {
			nextPath := path + "/" + key
			if diagnostic := validateTOMLNode(node[key], nextPath); diagnostic != nil {
				return diagnostic
			}
		}
		return nil
	case []any:
		for _, item := range node {
			if !isScalar(item) {
				diagnostic := unsupportedFeature(
					fmt.Sprintf(
						"Unsupported TOML array value at %s. Only scalar arrays are supported.",
						displayPath(path),
					),
				)
				return &diagnostic
			}
		}
		return nil
	default:
		if isScalar(node) {
			return nil
		}
		diagnostic := unsupportedFeature(
			fmt.Sprintf(
				"Unsupported TOML value at %s. Only tables, scalar values, and scalar arrays are supported.",
				displayPath(path),
			),
		)
		return &diagnostic
	}
}

func renderTOMLScalar(value any) string {
	switch node := value.(type) {
	case string:
		return strconv.Quote(node)
	case bool:
		if node {
			return "true"
		}
		return "false"
	case int64:
		return strconv.FormatInt(node, 10)
	case float64:
		return strconv.FormatFloat(node, 'f', -1, 64)
	default:
		return strconv.Quote(fmt.Sprint(node))
	}
}

func renderTOMLValue(value any) string {
	switch node := value.(type) {
	case []any:
		parts := make([]string, 0, len(node))
		for _, item := range node {
			parts = append(parts, renderTOMLScalar(item))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	default:
		return renderTOMLScalar(node)
	}
}

func renderTOMLTable(table map[string]any, path []string) []string {
	keys := make([]string, 0, len(table))
	for key := range table {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	valueKeys := make([]string, 0, len(keys))
	tableKeys := make([]string, 0, len(keys))
	for _, key := range keys {
		if _, ok := table[key].(map[string]any); ok {
			tableKeys = append(tableKeys, key)
		} else {
			valueKeys = append(valueKeys, key)
		}
	}

	lines := make([]string, 0, len(keys)+1)
	if len(path) > 0 {
		lines = append(lines, "["+strings.Join(path, ".")+"]")
	}

	for _, key := range valueKeys {
		lines = append(lines, key+" = "+renderTOMLValue(table[key]))
	}

	for _, key := range tableKeys {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, renderTOMLTable(table[key].(map[string]any), append(slices.Clone(path), key))...)
	}

	return lines
}

func canonicalTOML(table map[string]any) string {
	return strings.Join(renderTOMLTable(table, nil), "\n") + "\n"
}

func collectTOMLOwners(table map[string]any, prefix string) []TOMLOwner {
	keys := make([]string, 0, len(table))
	for key := range table {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	owners := make([]TOMLOwner, 0)
	for _, key := range keys {
		path := prefix + "/" + key
		switch node := table[key].(type) {
		case map[string]any:
			owners = append(owners, TOMLOwner{
				Path:      path,
				OwnerKind: OwnerTable,
				MatchKey:  key,
			})
			owners = append(owners, collectTOMLOwners(node, path)...)
		case []any:
			owners = append(owners, TOMLOwner{
				Path:      path,
				OwnerKind: OwnerKeyValue,
				MatchKey:  key,
			})
			for index := range node {
				owners = append(owners, TOMLOwner{
					Path:      fmt.Sprintf("%s/%d", path, index),
					OwnerKind: OwnerArrayItem,
				})
			}
		default:
			owners = append(owners, TOMLOwner{
				Path:      path,
				OwnerKind: OwnerKeyValue,
				MatchKey:  key,
			})
		}
	}

	return owners
}

func parseTOMLTable(source string) (map[string]any, error) {
	var parsed map[string]any
	if err := toml.Unmarshal([]byte(source), &parsed); err != nil {
		return nil, err
	}
	return parsed, nil
}

func resolveBackend(backend TOMLBackend) TOMLBackend {
	if backend == "" {
		return BackendTreeSitter
	}
	return backend
}

func AnalyzeTOMLSource(source string, dialect TOMLDialect) astmerge.ParseResult[TOMLAnalysis] {
	if dialect != DialectTOML {
		return astmerge.ParseResult[TOMLAnalysis]{
			OK:          false,
			Diagnostics: []astmerge.Diagnostic{unsupportedFeature("Unsupported TOML dialect.")},
		}
	}

	parsed, err := parseTOMLTable(source)
	if err != nil {
		return astmerge.ParseResult[TOMLAnalysis]{
			OK:          false,
			Diagnostics: []astmerge.Diagnostic{parseError(err.Error())},
		}
	}

	if diagnostic := validateTOMLNode(parsed, ""); diagnostic != nil {
		return astmerge.ParseResult[TOMLAnalysis]{
			OK:          false,
			Diagnostics: []astmerge.Diagnostic{*diagnostic},
		}
	}

	analysis := TOMLAnalysis{
		Dialect:          DialectTOML,
		NormalizedSource: canonicalTOML(parsed),
		RootKind:         RootTable,
		Owners:           collectTOMLOwners(parsed, ""),
	}

	return astmerge.ParseResult[TOMLAnalysis]{
		OK:          true,
		Diagnostics: []astmerge.Diagnostic{},
		Analysis:    &analysis,
	}
}

func ValidatePigeonSyntax(source string) *astmerge.Diagnostic {
	if _, err := pigeontoml.Parse("", []byte(source)); err != nil {
		diagnostic := parseError(err.Error())
		return &diagnostic
	}
	return nil
}

func ParseTOML(source string, dialect TOMLDialect) astmerge.ParseResult[TOMLAnalysis] {
	return ParseTOMLWithBackend(source, dialect, BackendTreeSitter)
}

func ParseTOMLWithBackend(source string, dialect TOMLDialect, backend TOMLBackend) astmerge.ParseResult[TOMLAnalysis] {
	resolved := resolveBackend(backend)
	if resolved != BackendTreeSitter {
		return astmerge.ParseResult[TOMLAnalysis]{
			OK:          false,
			Diagnostics: []astmerge.Diagnostic{unsupportedFeature(fmt.Sprintf("Unsupported TOML backend %s.", resolved))},
		}
	}

	syntax := treehaver.ParseWithLanguagePack(treehaver.ParserRequest{
		Source:   source,
		Language: "toml",
		Dialect:  string(dialect),
	})
	if !syntax.OK {
		return astmerge.ParseResult[TOMLAnalysis]{
			OK:          false,
			Diagnostics: astmerge.DiagnosticsFromTreeHaver(syntax.Diagnostics),
		}
	}

	return AnalyzeTOMLSource(source, dialect)
}

func MatchTOMLOwners(template TOMLAnalysis, destination TOMLAnalysis) TOMLOwnerMatchResult {
	destinationOwners := make(map[string]struct{}, len(destination.Owners))
	for _, owner := range destination.Owners {
		destinationOwners[owner.Path] = struct{}{}
	}

	templateOwners := make(map[string]struct{}, len(template.Owners))
	for _, owner := range template.Owners {
		templateOwners[owner.Path] = struct{}{}
	}

	matched := make([]TOMLOwnerMatch, 0)
	unmatchedTemplate := make([]string, 0)
	for _, owner := range template.Owners {
		if _, ok := destinationOwners[owner.Path]; ok {
			matched = append(matched, TOMLOwnerMatch{
				TemplatePath:    owner.Path,
				DestinationPath: owner.Path,
			})
		} else {
			unmatchedTemplate = append(unmatchedTemplate, owner.Path)
		}
	}

	unmatchedDestination := make([]string, 0)
	for _, owner := range destination.Owners {
		if _, ok := templateOwners[owner.Path]; !ok {
			unmatchedDestination = append(unmatchedDestination, owner.Path)
		}
	}

	return TOMLOwnerMatchResult{
		Matched:              matched,
		UnmatchedTemplate:    unmatchedTemplate,
		UnmatchedDestination: unmatchedDestination,
	}
}

func mergeTOMLTables(template map[string]any, destination map[string]any) map[string]any {
	merged := make(map[string]any, len(template)+len(destination))
	keys := make([]string, 0, len(template)+len(destination))
	for key := range template {
		keys = append(keys, key)
	}
	for key := range destination {
		if !slices.Contains(keys, key) {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)

	for _, key := range keys {
		templateValue, inTemplate := template[key]
		destinationValue, inDestination := destination[key]

		switch {
		case !inTemplate && inDestination:
			merged[key] = destinationValue
		case inTemplate && !inDestination:
			merged[key] = templateValue
		default:
			templateTable, templateIsTable := templateValue.(map[string]any)
			destinationTable, destinationIsTable := destinationValue.(map[string]any)
			if templateIsTable && destinationIsTable {
				merged[key] = mergeTOMLTables(templateTable, destinationTable)
			} else {
				merged[key] = destinationValue
			}
		}
	}

	return merged
}

func MergeTOMLWithParser(
	templateSource string,
	destinationSource string,
	dialect TOMLDialect,
	parser func(source string, dialect TOMLDialect) astmerge.ParseResult[TOMLAnalysis],
) astmerge.MergeResult[string] {
	template := parser(templateSource, dialect)
	if !template.OK || template.Analysis == nil {
		return astmerge.MergeResult[string]{
			OK:          false,
			Diagnostics: template.Diagnostics,
		}
	}

	destination := parser(destinationSource, dialect)
	if !destination.OK || destination.Analysis == nil {
		diagnostics := make([]astmerge.Diagnostic, 0, len(destination.Diagnostics))
		for _, diagnostic := range destination.Diagnostics {
			if diagnostic.Category == astmerge.CategoryParseError {
				diagnostic.Category = astmerge.CategoryDestinationParseError
			}
			diagnostics = append(diagnostics, diagnostic)
		}
		return astmerge.MergeResult[string]{
			OK:          false,
			Diagnostics: diagnostics,
		}
	}

	templateTable, err := parseTOMLTable(template.Analysis.NormalizedSource)
	if err != nil {
		return astmerge.MergeResult[string]{
			OK:          false,
			Diagnostics: []astmerge.Diagnostic{parseError(err.Error())},
		}
	}
	destinationTable, err := parseTOMLTable(destination.Analysis.NormalizedSource)
	if err != nil {
		return astmerge.MergeResult[string]{
			OK:          false,
			Diagnostics: []astmerge.Diagnostic{destinationParseError(err.Error())},
		}
	}

	output := canonicalTOML(mergeTOMLTables(templateTable, destinationTable))
	return astmerge.MergeResult[string]{
		OK:          true,
		Diagnostics: []astmerge.Diagnostic{},
		Output:      &output,
		Policies:    []astmerge.PolicyReference{destinationWinsArrayPolicy()},
	}
}

func MergeTOML(templateSource string, destinationSource string, dialect TOMLDialect) astmerge.MergeResult[string] {
	return MergeTOMLWithBackend(templateSource, destinationSource, dialect, BackendTreeSitter)
}

func MergeTOMLWithBackend(templateSource string, destinationSource string, dialect TOMLDialect, backend TOMLBackend) astmerge.MergeResult[string] {
	resolved := resolveBackend(backend)
	if resolved != BackendTreeSitter {
		return astmerge.MergeResult[string]{
			OK:          false,
			Diagnostics: []astmerge.Diagnostic{unsupportedFeature(fmt.Sprintf("Unsupported TOML backend %s.", resolved))},
		}
	}

	return MergeTOMLWithParser(templateSource, destinationSource, dialect, func(source string, parseDialect TOMLDialect) astmerge.ParseResult[TOMLAnalysis] {
		return ParseTOMLWithBackend(source, parseDialect, resolved)
	})
}
