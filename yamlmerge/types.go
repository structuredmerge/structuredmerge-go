package yamlmerge

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/structuredmerge/structuredmerge-go/astmerge"
	yamlv3 "gopkg.in/yaml.v3"
)

type YAMLDialect string

const (
	DialectYAML YAMLDialect = "yaml"
)

type YAMLBackend string

const (
	BackendYAMLV3      YAMLBackend = "yaml-v3"
	BackendGoccyGoYAML YAMLBackend = "goccy-go-yaml"
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
	return []YAMLBackend{BackendYAMLV3, BackendGoccyGoYAML}
}

func YAMLBackendFeatureProfileInfo(backend YAMLBackend) YAMLBackendFeatureProfile {
	return YAMLBackendFeatureProfile{
		Family:            YAMLFeatureProfileInfo().Family,
		SupportedDialects: YAMLFeatureProfileInfo().SupportedDialects,
		SupportedPolicies: YAMLFeatureProfileInfo().SupportedPolicies,
		Backend:           string(backend),
	}
}

func YAMLPlanContext() astmerge.ConformanceFamilyPlanContext {
	return YAMLPlanContextWithBackend(BackendYAMLV3)
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
			SupportsDialects:  true,
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
	switch backend {
	case BackendGoccyGoYAML:
		if err := yaml.Unmarshal([]byte(source), &parsed); err != nil {
			return nil, err
		}
	default:
		if err := yamlv3.Unmarshal([]byte(source), &parsed); err != nil {
			return nil, err
		}
	}

	normalized, ok := normalizeYAMLValue(parsed).(map[string]any)
	if !ok {
		return nil, fmt.Errorf("YAML documents must parse to a mapping root")
	}
	return normalized, nil
}

func ParseYAML(source string, dialect YAMLDialect) astmerge.ParseResult[YAMLAnalysis] {
	return ParseYAMLWithBackend(source, dialect, BackendYAMLV3)
}

func ParseYAMLWithBackend(source string, dialect YAMLDialect, backend YAMLBackend) astmerge.ParseResult[YAMLAnalysis] {
	if dialect != DialectYAML {
		return astmerge.ParseResult[YAMLAnalysis]{
			OK:          false,
			Diagnostics: []astmerge.Diagnostic{unsupportedFeature("Unsupported YAML dialect.")},
		}
	}

	parsed, err := parseYAMLMapping(source, backend)
	if err != nil {
		return astmerge.ParseResult[YAMLAnalysis]{
			OK:          false,
			Diagnostics: []astmerge.Diagnostic{parseError(err.Error())},
		}
	}

	if diagnostic := validateYAMLNode(parsed, ""); diagnostic != nil {
		return astmerge.ParseResult[YAMLAnalysis]{
			OK:          false,
			Diagnostics: []astmerge.Diagnostic{*diagnostic},
		}
	}

	analysis := YAMLAnalysis{
		Dialect:          DialectYAML,
		NormalizedSource: canonicalYAML(parsed),
		RootKind:         RootMapping,
		Owners:           collectYAMLOwners(parsed, ""),
	}

	return astmerge.ParseResult[YAMLAnalysis]{
		OK:          true,
		Diagnostics: []astmerge.Diagnostic{},
		Analysis:    &analysis,
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

func mergeYAMLMappings(template map[string]any, destination map[string]any) map[string]any {
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
			templateMapping, templateIsMapping := templateValue.(map[string]any)
			destinationMapping, destinationIsMapping := destinationValue.(map[string]any)
			if templateIsMapping && destinationIsMapping {
				merged[key] = mergeYAMLMappings(templateMapping, destinationMapping)
			} else {
				merged[key] = destinationValue
			}
		}
	}

	return merged
}

func MergeYAML(templateSource string, destinationSource string, dialect YAMLDialect) astmerge.MergeResult[string] {
	return MergeYAMLWithBackend(templateSource, destinationSource, dialect, BackendYAMLV3)
}

func MergeYAMLWithBackend(templateSource string, destinationSource string, dialect YAMLDialect, backend YAMLBackend) astmerge.MergeResult[string] {
	template := ParseYAMLWithBackend(templateSource, dialect, backend)
	if !template.OK || template.Analysis == nil {
		return astmerge.MergeResult[string]{
			OK:          false,
			Diagnostics: template.Diagnostics,
		}
	}

	destination := ParseYAMLWithBackend(destinationSource, dialect, backend)
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

	templateMapping, err := parseYAMLMapping(template.Analysis.NormalizedSource, backend)
	if err != nil {
		return astmerge.MergeResult[string]{
			OK:          false,
			Diagnostics: []astmerge.Diagnostic{parseError(err.Error())},
		}
	}
	destinationMapping, err := parseYAMLMapping(destination.Analysis.NormalizedSource, backend)
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

	output := canonicalYAML(mergeYAMLMappings(templateMapping, destinationMapping))
	return astmerge.MergeResult[string]{
		OK:          true,
		Diagnostics: []astmerge.Diagnostic{},
		Output:      &output,
		Policies:    []astmerge.PolicyReference{destinationWinsArrayPolicy()},
	}
}
