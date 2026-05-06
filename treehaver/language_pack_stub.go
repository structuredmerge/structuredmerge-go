//go:build !tspack

package treehaver

func ParseWithLanguagePack(request ParserRequest) ParseResult[LanguagePackAnalysis] {
	return languagePackUnavailable[LanguagePackAnalysis](request.Language)
}

func ProcessWithLanguagePack(request ProcessRequest) ParseResult[LanguagePackProcessAnalysis] {
	return languagePackUnavailable[LanguagePackProcessAnalysis](request.Language)
}

func languagePackUnavailable[T any](language string) ParseResult[T] {
	return ParseResult[T]{
		OK: false,
		Diagnostics: []Diagnostic{
			{
				Severity: SeverityError,
				Category: CategoryUnsupportedFeature,
				Message:  "tree-sitter-language-pack backend requires the tspack build tag for " + language + ".",
			},
		},
	}
}
