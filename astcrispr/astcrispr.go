// Package astcrispr provides a thin structural edit tool layer over astmerge.
package astcrispr

import (
	"reflect"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
)

const PackageName = "ast-crispr"

func AstMergeContractAnchor() string {
	return reflect.TypeOf(astmerge.StructuredEditCrisprExampleParityReport{}).Name()
}

func BoundaryReport() map[string]any {
	return map[string]any{
		"package":               PackageName,
		"layer":                 "structural_edit_tool",
		"status":                "active_thin_package",
		"base_contract_package": "ast-merge",
		"relationship": map[string]any{
			"ast_merge": []any{
				"owns portable structured-edit envelope contracts",
				"owns transport, report, replay, review, and provider handoff vocabulary",
				"remains the substrate for provider-neutral fixtures",
			},
			"ast_crispr": []any{
				"owns ergonomic structural-edit selectors, profiles, and operation helpers",
				"wraps ast-merge contracts instead of forking them",
				"may grow compatibility helpers for old ast-crispr concepts after fixture-backed review",
			},
			"provider_packages": []any{
				"own parser-specific execution and metadata projection",
				"may expose provider adapters consumed by ast-crispr",
				"keep raw parser details behind normalized tree metadata or semantic sidecars",
			},
			"ast_template": []any{
				"orchestrates template and directory workflows",
				"invokes structural edits through ast-merge or ast-crispr registries/envelopes",
				"does not own parser-specific selectors",
			},
		},
		"implementations": []any{
			map[string]any{
				"language":     "go",
				"package_name": "astcrispr",
				"import":       "github.com/structuredmerge/structuredmerge-go/astcrispr",
			},
			map[string]any{
				"language":     "ruby",
				"package_name": "ast-crispr",
				"require":      "ast/crispr",
			},
			map[string]any{
				"language":     "rust",
				"package_name": "ast-crispr",
				"crate":        "ast_crispr",
			},
			map[string]any{
				"language":     "typescript",
				"package_name": "@structuredmerge/ast-crispr",
				"import":       "@structuredmerge/ast-crispr",
			},
		},
		"initial_exports": []any{
			"package identity",
			"boundary report",
			"ast-merge structured-edit contract anchor",
		},
		"future_exports": []any{
			"limit helpers",
			"match profile helpers",
			"selection profile helpers",
			"destination profile helpers",
			"operation profile helpers",
			"replace/delete/insert/move helpers",
			"batch operation helpers",
		},
		"metadata": map[string]any{
			"source":   "legacy_crispr_reference",
			"decision": "Keep ast-merge as the base contract layer and revive ast-crispr as a separate thin package in every implementation.",
		},
	}
}
