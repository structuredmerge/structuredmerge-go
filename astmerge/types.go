package astmerge

type DiagnosticSeverity string

const (
	SeverityInfo    DiagnosticSeverity = "info"
	SeverityWarning DiagnosticSeverity = "warning"
	SeverityError   DiagnosticSeverity = "error"
)

type DiagnosticCategory string

const (
	CategoryParseError            DiagnosticCategory = "parse_error"
	CategoryDestinationParseError DiagnosticCategory = "destination_parse_error"
	CategoryUnsupportedFeature    DiagnosticCategory = "unsupported_feature"
	CategoryFallbackApplied       DiagnosticCategory = "fallback_applied"
	CategoryAmbiguity             DiagnosticCategory = "ambiguity"
)

type Diagnostic struct {
	Severity DiagnosticSeverity
	Category DiagnosticCategory
	Message  string
	Path     string
}

type ParseResult[T any] struct {
	OK          bool
	Diagnostics []Diagnostic
	Analysis    *T
	Policies    []PolicyReference
}

type MergeResult[T any] struct {
	OK          bool
	Diagnostics []Diagnostic
	Output      *T
	Policies    []PolicyReference
}

type PolicySurface string

const (
	PolicySurfaceFallback PolicySurface = "fallback"
	PolicySurfaceArray    PolicySurface = "array"
)

type PolicyReference struct {
	Surface PolicySurface
	Name    string
}

type FamilyFeatureProfile struct {
	Family            string
	SupportedDialects []string
	SupportedPolicies []PolicyReference
}

type ConformanceOutcome string

const (
	ConformancePassed  ConformanceOutcome = "passed"
	ConformanceFailed  ConformanceOutcome = "failed"
	ConformanceSkipped ConformanceOutcome = "skipped"
)

type ConformanceCaseRef struct {
	Family string
	Role   string
	Case   string
}

type ConformanceCaseResult struct {
	Ref      ConformanceCaseRef
	Outcome  ConformanceOutcome
	Messages []string
}
