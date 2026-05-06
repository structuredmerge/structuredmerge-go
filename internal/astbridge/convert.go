package astbridge

import (
	"github.com/structuredmerge/structuredmerge-go/astmerge"
	"github.com/structuredmerge/structuredmerge-go/treehaver"
)

func DiagnosticFromTreeHaver(diagnostic treehaver.Diagnostic) astmerge.Diagnostic {
	category := astmerge.CategoryUnsupportedFeature
	if diagnostic.Category == treehaver.CategoryParseError {
		category = astmerge.CategoryParseError
	}

	return astmerge.Diagnostic{
		Severity: astmerge.DiagnosticSeverity(diagnostic.Severity),
		Category: category,
		Message:  diagnostic.Message,
		Path:     diagnostic.Path,
	}
}

func DiagnosticsFromTreeHaver(diagnostics []treehaver.Diagnostic) []astmerge.Diagnostic {
	converted := make([]astmerge.Diagnostic, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		converted = append(converted, DiagnosticFromTreeHaver(diagnostic))
	}
	return converted
}

func PolicyReferenceFromTreeHaver(policy treehaver.PolicyReference) astmerge.PolicyReference {
	return astmerge.PolicyReference{
		Surface: astmerge.PolicySurface(policy.Surface),
		Name:    policy.Name,
	}
}

func PolicyReferencesFromTreeHaver(policies []treehaver.PolicyReference) []astmerge.PolicyReference {
	converted := make([]astmerge.PolicyReference, 0, len(policies))
	for _, policy := range policies {
		converted = append(converted, PolicyReferenceFromTreeHaver(policy))
	}
	return converted
}
