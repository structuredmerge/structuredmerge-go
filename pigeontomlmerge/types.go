package pigeontomlmerge

import (
	"fmt"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/tomlmerge"
)

const BackendPigeon = "pigeon"

func unsupportedFeature(message string) astmerge.Diagnostic {
	return astmerge.Diagnostic{
		Severity: astmerge.SeverityError,
		Category: astmerge.CategoryUnsupportedFeature,
		Message:  message,
	}
}

func TOMLFeatureProfileInfo() tomlmerge.TOMLFeatureProfile {
	return tomlmerge.TOMLFeatureProfileInfo()
}

func AvailableTOMLBackends() []string {
	return []string{BackendPigeon}
}

func TOMLBackendFeatureProfileInfo() tomlmerge.TOMLFeatureProfile {
	return tomlmerge.TOMLFeatureProfile{
		Family:            "toml",
		SupportedDialects: []tomlmerge.TOMLDialect{tomlmerge.DialectTOML},
		SupportedPolicies: []astmerge.PolicyReference{{Surface: astmerge.PolicySurfaceArray, Name: "destination_wins_array"}},
	}
}

func TOMLPlanContext() astmerge.ConformanceFamilyPlanContext {
	return astmerge.ConformanceFamilyPlanContext{
		FamilyProfile: astmerge.FamilyFeatureProfile{
			Family:            "toml",
			SupportedDialects: []string{"toml"},
			SupportedPolicies: []astmerge.PolicyReference{{Surface: astmerge.PolicySurfaceArray, Name: "destination_wins_array"}},
		},
		FeatureProfile: &astmerge.ConformanceFeatureProfileView{
			Backend:           BackendPigeon,
			SupportsDialects:  false,
			SupportedPolicies: []astmerge.PolicyReference{{Surface: astmerge.PolicySurfaceArray, Name: "destination_wins_array"}},
		},
	}
}

func ParseTOML(source string, dialect tomlmerge.TOMLDialect, backend ...string) astmerge.ParseResult[tomlmerge.TOMLAnalysis] {
	requested := BackendPigeon
	if len(backend) > 0 && backend[0] != "" {
		requested = backend[0]
	}
	if requested != BackendPigeon {
		return astmerge.ParseResult[tomlmerge.TOMLAnalysis]{
			OK:          false,
			Diagnostics: []astmerge.Diagnostic{unsupportedFeature(fmt.Sprintf("Unsupported TOML backend %s.", requested))},
		}
	}

	if diagnostic := tomlmerge.ValidatePigeonSyntax(source); diagnostic != nil {
		return astmerge.ParseResult[tomlmerge.TOMLAnalysis]{
			OK:          false,
			Diagnostics: []astmerge.Diagnostic{*diagnostic},
		}
	}

	return tomlmerge.AnalyzeTOMLSource(source, dialect)
}

func MatchTOMLOwners(template, destination tomlmerge.TOMLAnalysis) tomlmerge.TOMLOwnerMatchResult {
	return tomlmerge.MatchTOMLOwners(template, destination)
}

func MergeTOML(templateSource string, destinationSource string, dialect tomlmerge.TOMLDialect, backend ...string) astmerge.MergeResult[string] {
	requested := BackendPigeon
	if len(backend) > 0 && backend[0] != "" {
		requested = backend[0]
	}
	if requested != BackendPigeon {
		return astmerge.MergeResult[string]{
			OK:          false,
			Diagnostics: []astmerge.Diagnostic{unsupportedFeature(fmt.Sprintf("Unsupported TOML backend %s.", requested))},
		}
	}

	return tomlmerge.MergeTOMLWithParser(templateSource, destinationSource, dialect, func(source string, parseDialect tomlmerge.TOMLDialect) astmerge.ParseResult[tomlmerge.TOMLAnalysis] {
		return ParseTOML(source, parseDialect)
	})
}
