package goccygoyamlmerge

import (
	"fmt"

	"github.com/goccy/go-yaml"
	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/treehaver"
	"github.com/structuredmerge/structuredmerge-go/yamlmerge"
)

const BackendGoccyGoYAML = "goccy-go-yaml"

func init() {
	treehaver.RegisterBackend(treehaver.BackendReference{ID: BackendGoccyGoYAML, Family: "native"})
}

func unsupportedFeature(message string) astmerge.Diagnostic {
	return astmerge.Diagnostic{
		Severity: astmerge.SeverityError,
		Category: astmerge.CategoryUnsupportedFeature,
		Message:  message,
	}
}

func parseError(message string) astmerge.Diagnostic {
	return astmerge.Diagnostic{
		Severity: astmerge.SeverityError,
		Category: astmerge.CategoryParseError,
		Message:  message,
	}
}

func YAMLFeatureProfileInfo() yamlmerge.YAMLFeatureProfile {
	return yamlmerge.YAMLFeatureProfileInfo()
}

func AvailableYAMLBackends() []string {
	return []string{BackendGoccyGoYAML}
}

func YAMLBackendFeatureProfileInfo() yamlmerge.YAMLBackendFeatureProfile {
	return yamlmerge.YAMLBackendFeatureProfile{
		Family:            "yaml",
		SupportedDialects: []yamlmerge.YAMLDialect{yamlmerge.DialectYAML},
		SupportedPolicies: yamlmerge.YAMLFeatureProfileInfo().SupportedPolicies,
		Backend:           BackendGoccyGoYAML,
	}
}

func YAMLPlanContext() astmerge.ConformanceFamilyPlanContext {
	return astmerge.ConformanceFamilyPlanContext{
		FamilyProfile: astmerge.FamilyFeatureProfile{
			Family:            "yaml",
			SupportedDialects: []string{"yaml"},
			SupportedPolicies: yamlmerge.YAMLFeatureProfileInfo().SupportedPolicies,
		},
		FeatureProfile: &astmerge.ConformanceFeatureProfileView{
			Backend:           BackendGoccyGoYAML,
			SupportsDialects:  true,
			SupportedPolicies: yamlmerge.YAMLFeatureProfileInfo().SupportedPolicies,
		},
	}
}

func ParseYAML(source string, dialect yamlmerge.YAMLDialect, backend ...string) astmerge.ParseResult[yamlmerge.YAMLAnalysis] {
	requested := BackendGoccyGoYAML
	if len(backend) > 0 && backend[0] != "" {
		requested = backend[0]
	}
	if requested != BackendGoccyGoYAML {
		return astmerge.ParseResult[yamlmerge.YAMLAnalysis]{
			OK:          false,
			Diagnostics: []astmerge.Diagnostic{unsupportedFeature(fmt.Sprintf("Unsupported YAML backend %s.", requested))},
		}
	}
	if dialect != yamlmerge.DialectYAML {
		return astmerge.ParseResult[yamlmerge.YAMLAnalysis]{
			OK:          false,
			Diagnostics: []astmerge.Diagnostic{unsupportedFeature(fmt.Sprintf("Unsupported YAML dialect %s.", dialect))},
		}
	}

	var parsed any
	if err := yaml.Unmarshal([]byte(source), &parsed); err != nil {
		return astmerge.ParseResult[yamlmerge.YAMLAnalysis]{
			OK:          false,
			Diagnostics: []astmerge.Diagnostic{parseError(err.Error())},
		}
	}

	return yamlmerge.AnalyzeYAMLParsedDocument(parsed, dialect)
}

func MatchYAMLOwners(template, destination yamlmerge.YAMLAnalysis) yamlmerge.YAMLOwnerMatchResult {
	return yamlmerge.MatchYAMLOwners(template, destination)
}

func MergeYAML(templateSource string, destinationSource string, dialect yamlmerge.YAMLDialect, backend ...string) astmerge.MergeResult[string] {
	requested := BackendGoccyGoYAML
	if len(backend) > 0 && backend[0] != "" {
		requested = backend[0]
	}
	if requested != BackendGoccyGoYAML {
		return astmerge.MergeResult[string]{
			OK:          false,
			Diagnostics: []astmerge.Diagnostic{unsupportedFeature(fmt.Sprintf("Unsupported YAML backend %s.", requested))},
		}
	}

	return yamlmerge.MergeYAMLWithParser(templateSource, destinationSource, dialect, func(source string, parseDialect yamlmerge.YAMLDialect) astmerge.ParseResult[yamlmerge.YAMLAnalysis] {
		return ParseYAML(source, parseDialect)
	})
}
