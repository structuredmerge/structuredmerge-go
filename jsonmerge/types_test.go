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

func TestAnalyzeJSONStructure(t *testing.T) {
	source := "{\n  \"name\": \"structuredmerge\",\n  \"tags\": [\"merge\", \"ast\"],\n  \"meta\": {\"enabled\": true}\n}\n"

	result := ParseJSON(source, DialectJSON)
	if !result.OK || result.Analysis == nil {
		t.Fatalf("expected parse success")
	}

	analysis := result.Analysis
	if analysis.RootKind != RootObject {
		t.Fatalf("unexpected root kind: %s", analysis.RootKind)
	}

	expected := []JSONOwner{
		{Path: "/meta", OwnerKind: OwnerMember, MatchKey: "meta"},
		{Path: "/meta/enabled", OwnerKind: OwnerMember, MatchKey: "enabled"},
		{Path: "/name", OwnerKind: OwnerMember, MatchKey: "name"},
		{Path: "/tags", OwnerKind: OwnerMember, MatchKey: "tags"},
		{Path: "/tags/0", OwnerKind: OwnerElement},
		{Path: "/tags/1", OwnerKind: OwnerElement},
	}

	if len(analysis.Owners) != len(expected) {
		t.Fatalf("unexpected owners: %+v", analysis.Owners)
	}

	for index := range expected {
		if analysis.Owners[index] != expected[index] {
			t.Fatalf("unexpected owner at %d: %+v", index, analysis.Owners[index])
		}
	}
}

func TestMatchJSONOwners(t *testing.T) {
	template := ParseJSON("{\n  \"name\": \"structuredmerge\",\n  \"tags\": [\"merge\", \"ast\"],\n  \"meta\": {\"enabled\": true}\n}\n", DialectJSON)
	destination := ParseJSON("{\n  \"name\": \"structuredmerge\",\n  \"tags\": [\"merge\"],\n  \"meta\": {\"enabled\": true},\n  \"extra\": 1\n}\n", DialectJSON)

	if template.Analysis == nil || destination.Analysis == nil {
		t.Fatalf("expected parse success for both documents")
	}

	result := MatchJSONOwners(*template.Analysis, *destination.Analysis)

	expectedMatched := []JSONOwnerMatch{
		{TemplatePath: "/meta", DestinationPath: "/meta"},
		{TemplatePath: "/meta/enabled", DestinationPath: "/meta/enabled"},
		{TemplatePath: "/name", DestinationPath: "/name"},
		{TemplatePath: "/tags", DestinationPath: "/tags"},
		{TemplatePath: "/tags/0", DestinationPath: "/tags/0"},
	}

	if len(result.Matched) != len(expectedMatched) {
		t.Fatalf("unexpected matched owners: %+v", result.Matched)
	}

	for index := range expectedMatched {
		if result.Matched[index] != expectedMatched[index] {
			t.Fatalf("unexpected matched owner at %d: %+v", index, result.Matched[index])
		}
	}

	if len(result.UnmatchedTemplate) != 1 || result.UnmatchedTemplate[0] != "/tags/1" {
		t.Fatalf("unexpected unmatched template owners: %+v", result.UnmatchedTemplate)
	}

	if len(result.UnmatchedDestination) != 1 || result.UnmatchedDestination[0] != "/extra" {
		t.Fatalf("unexpected unmatched destination owners: %+v", result.UnmatchedDestination)
	}
}
