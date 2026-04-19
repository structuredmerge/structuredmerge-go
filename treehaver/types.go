package treehaver

import "github.com/structuredmerge/structuredmerge-go/astmerge"

type AnalysisHandle interface {
	Kind() string
}

type ParserRequest struct {
	Source   string
	Language string
	Dialect  string
}

type AdapterInfo struct {
	Backend           string
	SupportsDialects  bool
	SupportedPolicies []astmerge.PolicyReference
}

type FeatureProfile struct {
	Backend           string
	SupportsDialects  bool
	SupportedPolicies []astmerge.PolicyReference
}

type ParserAdapter[T AnalysisHandle] interface {
	Info() AdapterInfo
	Parse(request ParserRequest) astmerge.ParseResult[T]
}

type ParserDiagnostics struct {
	Backend     string
	Diagnostics []astmerge.Diagnostic
}
