package jsonmerge

import "testing"

func TestParseJSONCAcceptsComments(t *testing.T) {
	source := "{\n  // package status\n  \"enabled\": true,\n  /* package name */\n  \"name\": \"structuredmerge\"\n}\n"

	result := ParseJSON(source, DialectJSONC)
	if !result.OK {
		t.Fatalf("expected parse success, got diagnostics: %+v", result.Diagnostics)
	}

	if result.Analysis == nil || !result.Analysis.AllowsComments {
		t.Fatalf("expected JSONC analysis with comments enabled")
	}
}

func TestParseJSONRejectsTrailingComma(t *testing.T) {
	source := "{\n  \"enabled\": true,\n  \"items\": [1, 2,],\n}\n"

	result := ParseJSON(source, DialectJSONC)
	if result.OK {
		t.Fatalf("expected parse failure")
	}

	if len(result.Diagnostics) == 0 || result.Diagnostics[0].Category != "parse_error" {
		t.Fatalf("unexpected diagnostics: %+v", result.Diagnostics)
	}
}
