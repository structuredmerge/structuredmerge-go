package treehaver

import "github.com/structuredmerge/structuredmerge-go/astmerge"

type AnalysisHandle interface {
	Kind() string
}

type ParserAdapter[T AnalysisHandle] interface {
	Parse(source string) astmerge.ParseResult[T]
}

type ParserDiagnostics struct {
	Diagnostics []astmerge.Diagnostic
}
