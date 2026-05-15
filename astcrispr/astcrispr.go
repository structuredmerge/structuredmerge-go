// Package astcrispr provides a thin structural edit tool layer over astmerge.
package astcrispr

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
)

const PackageName = "ast-crispr"

type Error struct {
	Code    string
	Message string
	Details map[string]any
}

func (e Error) Error() string {
	return e.Message
}

type limitConstraint struct {
	description string
	predicate   func(int) bool
}

type Limit struct {
	constraints []limitConstraint
}

type MatchProfile struct {
	StartBoundary string
	EndBoundary   string
	PayloadKind   string
}

type profileDescriptor struct {
	family      string
	description string
}

var knownStartBoundaries = map[string]profileDescriptor{
	"owner_start":          {family: "structural_owner", description: "Span starts at the structural owner's boundary"},
	"comment_region_start": {family: "comment_anchor", description: "Span starts at an owning comment-region boundary"},
}

var knownEndBoundaries = map[string]profileDescriptor{
	"owner_end":                   {family: "structural_owner", description: "Span ends at the structural owner's boundary"},
	"owner_end_plus_trailing_gap": {family: "gap_extension", description: "Span extends past the owner boundary to include trailing blank-line gap"},
}

var knownPayloadKinds = map[string]profileDescriptor{
	"structural_owner_body": {family: "owner_body", description: "Span represents a structural owner's body"},
	"comment_owned_body":    {family: "comment_owned", description: "Span represents a structural owner body selected through an owning comment marker"},
	"section_branch":        {family: "section_branch", description: "Span represents a heading-owned section branch payload"},
}

func NewLimit(spec any) (Limit, error) {
	if spec == nil {
		spec = map[string]any{"exactly": float64(1)}
	}
	constraints, err := normalizeLimit(spec)
	if err != nil {
		return Limit{}, err
	}
	return Limit{constraints: constraints}, nil
}

func NewMatchProfile(startBoundary, endBoundary, payloadKind string) MatchProfile {
	if startBoundary == "" {
		startBoundary = "owner_start"
	}
	if endBoundary == "" {
		endBoundary = "owner_end"
	}
	if payloadKind == "" {
		payloadKind = "structural_owner_body"
	}
	return MatchProfile{StartBoundary: startBoundary, EndBoundary: endBoundary, PayloadKind: payloadKind}
}

func (profile MatchProfile) Report() map[string]any {
	startDescriptor, knownStart := knownStartBoundaries[profile.StartBoundary]
	endDescriptor, knownEnd := knownEndBoundaries[profile.EndBoundary]
	payloadDescriptor, knownPayload := knownPayloadKinds[profile.PayloadKind]
	startFamily := startDescriptor.family
	if startFamily == "" {
		startFamily = "unknown"
	}
	endFamily := endDescriptor.family
	if endFamily == "" {
		endFamily = "unknown"
	}
	payloadFamily := payloadDescriptor.family
	if payloadFamily == "" {
		payloadFamily = "unknown"
	}
	return map[string]any{
		"start_boundary":        profile.StartBoundary,
		"start_boundary_family": startFamily,
		"known_start_boundary":  knownStart,
		"end_boundary":          profile.EndBoundary,
		"end_boundary_family":   endFamily,
		"known_end_boundary":    knownEnd,
		"payload_kind":          profile.PayloadKind,
		"payload_family":        payloadFamily,
		"known_payload_kind":    knownPayload,
		"comment_anchored":      startFamily == "comment_anchor" || payloadFamily == "comment_owned",
		"trailing_gap_extended": endFamily == "gap_extension",
	}
}

func (limit Limit) Allows(count int) bool {
	for _, constraint := range limit.constraints {
		if !constraint.predicate(count) {
			return false
		}
	}
	return true
}

func (limit Limit) Describe() string {
	descriptions := make([]string, 0, len(limit.constraints))
	for _, constraint := range limit.constraints {
		descriptions = append(descriptions, constraint.description)
	}
	return strings.Join(descriptions, " and ")
}

func normalizeLimit(spec any) ([]limitConstraint, error) {
	switch value := spec.(type) {
	case Limit:
		return value.constraints, nil
	case map[string]any:
		return normalizeLimitMap(value)
	case []any:
		var constraints []limitConstraint
		for _, entry := range value {
			normalized, err := normalizeLimit(entry)
			if err != nil {
				return nil, err
			}
			constraints = append(constraints, normalized...)
		}
		return constraints, nil
	case string:
		constraint, err := limitConstraintForOperator(value)
		if err != nil {
			return nil, err
		}
		return []limitConstraint{constraint}, nil
	default:
		return nil, Error{Code: "ast_crispr_limit_unsupported", Message: "Unsupported ast-crispr limit specification", Details: map[string]any{"spec": fmt.Sprintf("%v", spec)}}
	}
}

func normalizeLimitMap(spec map[string]any) ([]limitConstraint, error) {
	var constraints []limitConstraint
	if raw, ok := spec["exactly"]; ok {
		value, err := intLimitValue(raw)
		if err != nil {
			return nil, err
		}
		constraints = append(constraints, limitConstraint{description: fmt.Sprintf("== %d", value), predicate: func(count int) bool { return count == value }})
	}
	if raw, ok := spec["at_most"]; ok {
		value, err := intLimitValue(raw)
		if err != nil {
			return nil, err
		}
		constraints = append(constraints, limitConstraint{description: fmt.Sprintf("<= %d", value), predicate: func(count int) bool { return count <= value }})
	}
	if raw, ok := spec["at_least"]; ok {
		value, err := intLimitValue(raw)
		if err != nil {
			return nil, err
		}
		constraints = append(constraints, limitConstraint{description: fmt.Sprintf(">= %d", value), predicate: func(count int) bool { return count >= value }})
	}
	if raw, ok := spec["none_or_one"]; ok {
		if value, ok := raw.(bool); ok && value {
			constraints = append(constraints, limitConstraint{description: "<= 1", predicate: func(count int) bool { return count <= 1 }})
		}
	}
	if len(constraints) == 0 {
		return nil, Error{Code: "ast_crispr_limit_empty", Message: "ast-crispr limit must define at least one constraint", Details: map[string]any{"spec": spec}}
	}
	return constraints, nil
}

var limitExpressionPattern = regexp.MustCompile(`^(==|!=|<=|>=|<|>)\s*(\d+)$`)

func limitConstraintForOperator(spec string) (limitConstraint, error) {
	match := limitExpressionPattern.FindStringSubmatch(strings.TrimSpace(spec))
	if match == nil {
		return limitConstraint{}, Error{Code: "ast_crispr_limit_invalid_expression", Message: "Invalid ast-crispr limit expression", Details: map[string]any{"spec": spec}}
	}
	operator := match[1]
	value, err := strconv.Atoi(match[2])
	if err != nil {
		return limitConstraint{}, err
	}
	return limitConstraint{description: fmt.Sprintf("%s %d", operator, value), predicate: func(count int) bool {
		switch operator {
		case "==":
			return count == value
		case "!=":
			return count != value
		case "<=":
			return count <= value
		case ">=":
			return count >= value
		case "<":
			return count < value
		case ">":
			return count > value
		default:
			return false
		}
	}}, nil
}

func intLimitValue(raw any) (int, error) {
	switch value := raw.(type) {
	case int:
		return value, nil
	case float64:
		return int(value), nil
	default:
		return 0, Error{Code: "ast_crispr_limit_unsupported", Message: "Unsupported ast-crispr limit value", Details: map[string]any{"value": raw}}
	}
}

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
			"limit helpers",
			"match profile helpers",
		},
		"future_exports": []any{
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
