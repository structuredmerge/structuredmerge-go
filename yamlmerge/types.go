package yamlmerge

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/treehaver"
	yamlv3 "gopkg.in/yaml.v3"
)

type YAMLDialect string

const (
	DialectYAML YAMLDialect = "yaml"
)

type YAMLBackend string

const (
	BackendKreuzberg YAMLBackend = "kreuzberg-language-pack"
)

type YAMLRootKind string

const (
	RootMapping YAMLRootKind = "mapping"
)

type YAMLOwnerKind string

const (
	OwnerMapping      YAMLOwnerKind = "mapping"
	OwnerKeyValue     YAMLOwnerKind = "key_value"
	OwnerSequenceItem YAMLOwnerKind = "sequence_item"
)

type orderedYAMLMapping struct {
	keys   []string
	values map[string]any
}

type YAMLOwner struct {
	Path      string
	OwnerKind YAMLOwnerKind
	MatchKey  string
}

type YAMLOwnerMatch struct {
	TemplatePath    string
	DestinationPath string
}

type YAMLOwnerMatchResult struct {
	Matched              []YAMLOwnerMatch
	UnmatchedTemplate    []string
	UnmatchedDestination []string
}

type YAMLAnalysis struct {
	Dialect          YAMLDialect
	NormalizedSource string
	RootKind         YAMLRootKind
	Owners           []YAMLOwner
}

func (YAMLAnalysis) Kind() string {
	return "yaml"
}

type YAMLFeatureProfile struct {
	Family            string
	SupportedDialects []YAMLDialect
	SupportedPolicies []astmerge.PolicyReference
}

type YAMLBackendFeatureProfile struct {
	Family            string
	SupportedDialects []YAMLDialect
	SupportedPolicies []astmerge.PolicyReference
	Backend           string
	BackendRef        *treehaver.BackendReference
}

func parseError(message string) astmerge.Diagnostic {
	return astmerge.Diagnostic{
		Severity: astmerge.SeverityError,
		Category: astmerge.CategoryParseError,
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

func YAMLFeatureProfileInfo() YAMLFeatureProfile {
	shared := astmerge.FamilyFeatureProfile{
		Family:            "yaml",
		SupportedDialects: []string{"yaml"},
		SupportedPolicies: []astmerge.PolicyReference{destinationWinsArrayPolicy()},
	}

	return YAMLFeatureProfile{
		Family:            shared.Family,
		SupportedDialects: []YAMLDialect{DialectYAML},
		SupportedPolicies: shared.SupportedPolicies,
	}
}

func AvailableYAMLBackends() []YAMLBackend {
	return []YAMLBackend{BackendKreuzberg}
}

func YAMLBackendFeatureProfileInfo(backend YAMLBackend) YAMLBackendFeatureProfile {
	return YAMLBackendFeatureProfile{
		Family:            YAMLFeatureProfileInfo().Family,
		SupportedDialects: YAMLFeatureProfileInfo().SupportedDialects,
		SupportedPolicies: YAMLFeatureProfileInfo().SupportedPolicies,
		Backend:           string(backend),
		BackendRef:        treehaver.BackendReferenceByID(string(backend)),
	}
}

func YAMLPlanContext() astmerge.ConformanceFamilyPlanContext {
	return YAMLPlanContextWithBackend(BackendKreuzberg)
}

func YAMLPlanContextWithBackend(backend YAMLBackend) astmerge.ConformanceFamilyPlanContext {
	backendProfile := YAMLBackendFeatureProfileInfo(backend)
	return astmerge.ConformanceFamilyPlanContext{
		FamilyProfile: astmerge.FamilyFeatureProfile{
			Family:            backendProfile.Family,
			SupportedDialects: []string{string(DialectYAML)},
			SupportedPolicies: backendProfile.SupportedPolicies,
		},
		FeatureProfile: &astmerge.ConformanceFeatureProfileView{
			Backend:           backendProfile.Backend,
			SupportsDialects:  backend != BackendKreuzberg,
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
	case string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
		return true
	default:
		return false
	}
}

func normalizeYAMLValue(value any) any {
	switch node := value.(type) {
	case map[string]any:
		mapping := make(map[string]any, len(node))
		for key, child := range node {
			mapping[key] = normalizeYAMLValue(child)
		}
		return mapping
	case map[any]any:
		mapping := make(map[string]any, len(node))
		for key, child := range node {
			mapping[fmt.Sprint(key)] = normalizeYAMLValue(child)
		}
		return mapping
	case []any:
		items := make([]any, len(node))
		for index, child := range node {
			items[index] = normalizeYAMLValue(child)
		}
		return items
	case int:
		return node
	case int64:
		return node
	case float64:
		return node
	case bool:
		return node
	case string:
		return node
	default:
		return node
	}
}

func validateYAMLNode(value any, path string) *astmerge.Diagnostic {
	switch node := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(node))
		for key := range node {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		for _, key := range keys {
			nextPath := path + "/" + key
			if diagnostic := validateYAMLNode(node[key], nextPath); diagnostic != nil {
				return diagnostic
			}
		}
		return nil
	case []any:
		for _, item := range node {
			if !isScalar(item) {
				diagnostic := unsupportedFeature(
					fmt.Sprintf(
						"Unsupported YAML sequence value at %s. Only scalar sequences are supported.",
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
				"Unsupported YAML value at %s. Only mappings, scalar values, and scalar sequences are supported.",
				displayPath(path),
			),
		)
		return &diagnostic
	}
}

func renderYAMLScalar(value any) string {
	switch node := value.(type) {
	case string:
		if node != "" && strings.IndexFunc(node, func(r rune) bool {
			switch {
			case r == '_', r == '.', r == '-':
				return false
			case r >= 'a' && r <= 'z':
				return false
			case r >= 'A' && r <= 'Z':
				return false
			case r >= '0' && r <= '9':
				return false
			default:
				return true
			}
		}) == -1 {
			return node
		}
		return strconv.Quote(node)
	case bool:
		if node {
			return "true"
		}
		return "false"
	case int:
		return strconv.Itoa(node)
	case int8:
		return strconv.FormatInt(int64(node), 10)
	case int16:
		return strconv.FormatInt(int64(node), 10)
	case int32:
		return strconv.FormatInt(int64(node), 10)
	case int64:
		return strconv.FormatInt(node, 10)
	case uint:
		return strconv.FormatUint(uint64(node), 10)
	case uint8:
		return strconv.FormatUint(uint64(node), 10)
	case uint16:
		return strconv.FormatUint(uint64(node), 10)
	case uint32:
		return strconv.FormatUint(uint64(node), 10)
	case uint64:
		return strconv.FormatUint(node, 10)
	case float32:
		return strconv.FormatFloat(float64(node), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(node, 'f', -1, 64)
	default:
		return strconv.Quote(fmt.Sprint(node))
	}
}

func renderYAMLNode(key string, value any, indent int) []string {
	prefix := strings.Repeat(" ", indent)

	switch node := value.(type) {
	case []any:
		lines := []string{prefix + key + ":"}
		for _, item := range node {
			lines = append(lines, strings.Repeat(" ", indent+2)+"- "+renderYAMLScalar(item))
		}
		return lines
	case map[string]any:
		return append([]string{prefix + key + ":"}, renderYAMLMapping(node, indent+2)...)
	case orderedYAMLMapping:
		return append([]string{prefix + key + ":"}, renderOrderedYAMLMapping(node, indent+2)...)
	default:
		return []string{prefix + key + ": " + renderYAMLScalar(node)}
	}
}

func renderYAMLMapping(mapping map[string]any, indent int) []string {
	keys := make([]string, 0, len(mapping))
	for key := range mapping {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	lines := make([]string, 0)
	for _, key := range keys {
		lines = append(lines, renderYAMLNode(key, mapping[key], indent)...)
	}

	return lines
}

func canonicalYAML(mapping map[string]any) string {
	return strings.Join(renderYAMLMapping(mapping, 0), "\n") + "\n"
}

func renderOrderedYAMLMapping(mapping orderedYAMLMapping, indent int) []string {
	lines := make([]string, 0)
	for _, key := range mapping.keys {
		lines = append(lines, renderYAMLNode(key, mapping.values[key], indent)...)
	}

	return lines
}

func canonicalOrderedYAML(mapping orderedYAMLMapping) string {
	return strings.Join(renderOrderedYAMLMapping(mapping, 0), "\n") + "\n"
}

func collectYAMLOwners(mapping map[string]any, prefix string) []YAMLOwner {
	keys := make([]string, 0, len(mapping))
	for key := range mapping {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	owners := make([]YAMLOwner, 0)
	for _, key := range keys {
		path := prefix + "/" + key
		switch node := mapping[key].(type) {
		case map[string]any:
			owners = append(owners, YAMLOwner{
				Path:      path,
				OwnerKind: OwnerMapping,
				MatchKey:  key,
			})
			owners = append(owners, collectYAMLOwners(node, path)...)
		case []any:
			owners = append(owners, YAMLOwner{
				Path:      path,
				OwnerKind: OwnerKeyValue,
				MatchKey:  key,
			})
			for index := range node {
				owners = append(owners, YAMLOwner{
					Path:      fmt.Sprintf("%s/%d", path, index),
					OwnerKind: OwnerSequenceItem,
				})
			}
		default:
			owners = append(owners, YAMLOwner{
				Path:      path,
				OwnerKind: OwnerKeyValue,
				MatchKey:  key,
			})
		}
	}

	return owners
}

func parseYAMLMapping(source string, backend YAMLBackend) (map[string]any, error) {
	var parsed any
	if backend != BackendKreuzberg {
		return nil, fmt.Errorf("unsupported YAML backend %s", backend)
	}
	if err := yamlv3.Unmarshal([]byte(source), &parsed); err != nil {
		return nil, err
	}

	normalized, ok := normalizeYAMLValue(parsed).(map[string]any)
	if !ok {
		return nil, fmt.Errorf("YAML documents must parse to a mapping root")
	}
	return normalized, nil
}

func parseOrderedYAMLMapping(source string, backend YAMLBackend) (orderedYAMLMapping, error) {
	if backend != BackendKreuzberg {
		return orderedYAMLMapping{}, fmt.Errorf("unsupported YAML backend %s", backend)
	}

	var document yamlv3.Node
	if err := yamlv3.Unmarshal([]byte(source), &document); err != nil {
		return orderedYAMLMapping{}, err
	}
	if len(document.Content) == 0 {
		return orderedYAMLMapping{}, fmt.Errorf("YAML documents must parse to a mapping root")
	}

	normalized, ok := normalizeOrderedYAMLNode(document.Content[0]).(orderedYAMLMapping)
	if !ok {
		return orderedYAMLMapping{}, fmt.Errorf("YAML documents must parse to a mapping root")
	}
	return normalized, nil
}

func normalizeOrderedYAMLNode(node *yamlv3.Node) any {
	switch node.Kind {
	case yamlv3.MappingNode:
		mapping := orderedYAMLMapping{values: map[string]any{}}
		for index := 0; index+1 < len(node.Content); index += 2 {
			key := node.Content[index].Value
			mapping.keys = append(mapping.keys, key)
			mapping.values[key] = normalizeOrderedYAMLNode(node.Content[index+1])
		}
		return mapping
	case yamlv3.SequenceNode:
		items := make([]any, 0, len(node.Content))
		for _, child := range node.Content {
			items = append(items, normalizeOrderedYAMLNode(child))
		}
		return items
	case yamlv3.ScalarNode:
		var decoded any
		if err := node.Decode(&decoded); err == nil {
			return decoded
		}
		return node.Value
	default:
		return nil
	}
}

func ParseYAML(source string, dialect YAMLDialect) astmerge.ParseResult[YAMLAnalysis] {
	return ParseYAMLWithBackend(source, dialect, BackendKreuzberg)
}

func ParseYAMLWithBackend(source string, dialect YAMLDialect, backend YAMLBackend) astmerge.ParseResult[YAMLAnalysis] {
	if dialect != DialectYAML {
		return astmerge.ParseResult[YAMLAnalysis]{
			OK:          false,
			Diagnostics: []astmerge.Diagnostic{unsupportedFeature("Unsupported YAML dialect.")},
		}
	}

	if backend != BackendKreuzberg {
		return astmerge.ParseResult[YAMLAnalysis]{
			OK:          false,
			Diagnostics: []astmerge.Diagnostic{unsupportedFeature(fmt.Sprintf("Unsupported YAML backend %s.", backend))},
		}
	}

	backendResult := treehaver.ParseWithLanguagePack(treehaver.ParserRequest{
		Source:   source,
		Language: "yaml",
		Dialect:  "yaml",
	})
	if !backendResult.OK {
		return astmerge.ParseResult[YAMLAnalysis]{
			OK:          false,
			Diagnostics: backendResult.Diagnostics,
		}
	}

	parsed, err := parseYAMLMapping(source, backend)
	if err != nil {
		return astmerge.ParseResult[YAMLAnalysis]{
			OK:          false,
			Diagnostics: []astmerge.Diagnostic{parseError(err.Error())},
		}
	}

	return AnalyzeYAMLParsedDocument(parsed, dialect)
}

func AnalyzeYAMLParsedDocument(parsed any, dialect YAMLDialect) astmerge.ParseResult[YAMLAnalysis] {
	if dialect != DialectYAML {
		return astmerge.ParseResult[YAMLAnalysis]{
			OK:          false,
			Diagnostics: []astmerge.Diagnostic{unsupportedFeature("Unsupported YAML dialect.")},
		}
	}

	normalized, ok := normalizeYAMLValue(parsed).(map[string]any)
	if !ok {
		return astmerge.ParseResult[YAMLAnalysis]{
			OK:          false,
			Diagnostics: []astmerge.Diagnostic{parseError("YAML documents must parse to a mapping root")},
		}
	}

	if diagnostic := validateYAMLNode(normalized, ""); diagnostic != nil {
		return astmerge.ParseResult[YAMLAnalysis]{
			OK:          false,
			Diagnostics: []astmerge.Diagnostic{*diagnostic},
		}
	}

	analysis := YAMLAnalysis{
		Dialect:          DialectYAML,
		NormalizedSource: canonicalYAML(normalized),
		RootKind:         RootMapping,
		Owners:           collectYAMLOwners(normalized, ""),
	}

	return astmerge.ParseResult[YAMLAnalysis]{
		OK:          true,
		Diagnostics: []astmerge.Diagnostic{},
		Analysis:    &analysis,
	}
}

func ParseYAMLWithParser(source string, dialect YAMLDialect, parser func(string, YAMLDialect) astmerge.ParseResult[YAMLAnalysis]) astmerge.ParseResult[YAMLAnalysis] {
	return parser(source, dialect)
}

func MergeYAMLWithParser(templateSource string, destinationSource string, dialect YAMLDialect, parser func(string, YAMLDialect) astmerge.ParseResult[YAMLAnalysis]) astmerge.MergeResult[string] {
	template := ParseYAMLWithParser(templateSource, dialect, parser)
	if !template.OK || template.Analysis == nil {
		return astmerge.MergeResult[string]{
			OK:          false,
			Diagnostics: template.Diagnostics,
		}
	}

	destination := ParseYAMLWithParser(destinationSource, dialect, parser)
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

	templateMapping, err := parseOrderedYAMLMapping(templateSource, BackendKreuzberg)
	if err != nil {
		return astmerge.MergeResult[string]{
			OK:          false,
			Diagnostics: []astmerge.Diagnostic{parseError(err.Error())},
		}
	}
	destinationMapping, err := parseOrderedYAMLMapping(destinationSource, BackendKreuzberg)
	if err != nil {
		return astmerge.MergeResult[string]{
			OK: false,
			Diagnostics: []astmerge.Diagnostic{{
				Severity: astmerge.SeverityError,
				Category: astmerge.CategoryDestinationParseError,
				Message:  err.Error(),
			}},
		}
	}

	output := canonicalOrderedYAML(mergeYAMLMappings(templateMapping, destinationMapping))
	return astmerge.MergeResult[string]{
		OK:          true,
		Diagnostics: []astmerge.Diagnostic{},
		Output:      &output,
		Policies:    []astmerge.PolicyReference{destinationWinsArrayPolicy()},
	}
}

func MatchYAMLOwners(template YAMLAnalysis, destination YAMLAnalysis) YAMLOwnerMatchResult {
	destinationOwners := make(map[string]struct{}, len(destination.Owners))
	for _, owner := range destination.Owners {
		destinationOwners[owner.Path] = struct{}{}
	}

	templateOwners := make(map[string]struct{}, len(template.Owners))
	for _, owner := range template.Owners {
		templateOwners[owner.Path] = struct{}{}
	}

	matched := make([]YAMLOwnerMatch, 0)
	unmatchedTemplate := make([]string, 0)
	for _, owner := range template.Owners {
		if _, ok := destinationOwners[owner.Path]; ok {
			matched = append(matched, YAMLOwnerMatch{
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

	return YAMLOwnerMatchResult{
		Matched:              matched,
		UnmatchedTemplate:    unmatchedTemplate,
		UnmatchedDestination: unmatchedDestination,
	}
}

func mergeYAMLMappings(template orderedYAMLMapping, destination orderedYAMLMapping) orderedYAMLMapping {
	merged := orderedYAMLMapping{values: map[string]any{}}

	for _, key := range template.keys {
		templateValue := template.values[key]
		destinationValue, inDestination := destination.values[key]
		if !inDestination {
			merged.values[key] = templateValue
			merged.keys = append(merged.keys, key)
			continue
		}

		templateMapping, templateIsMapping := templateValue.(orderedYAMLMapping)
		destinationMapping, destinationIsMapping := destinationValue.(orderedYAMLMapping)
		if templateIsMapping && destinationIsMapping {
			merged.values[key] = mergeYAMLMappings(templateMapping, destinationMapping)
		} else {
			merged.values[key] = destinationValue
		}
		merged.keys = append(merged.keys, key)
	}

	for _, key := range destination.keys {
		if _, inTemplate := template.values[key]; !inTemplate {
			merged.keys = append(merged.keys, key)
			merged.values[key] = destination.values[key]
		}
	}

	return merged
}

func MergeYAML(templateSource string, destinationSource string, dialect YAMLDialect) astmerge.MergeResult[string] {
	return MergeYAMLWithBackend(templateSource, destinationSource, dialect, BackendKreuzberg)
}

func MergeYAMLWithBackend(templateSource string, destinationSource string, dialect YAMLDialect, backend YAMLBackend) astmerge.MergeResult[string] {
	if backend != BackendKreuzberg {
		return astmerge.MergeResult[string]{
			OK:          false,
			Diagnostics: []astmerge.Diagnostic{unsupportedFeature(fmt.Sprintf("Unsupported YAML backend %s.", backend))},
		}
	}

	return MergeYAMLWithParser(templateSource, destinationSource, dialect, func(source string, parseDialect YAMLDialect) astmerge.ParseResult[YAMLAnalysis] {
		return ParseYAMLWithBackend(source, parseDialect, backend)
	})
}
