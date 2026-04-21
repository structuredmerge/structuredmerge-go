package tomlmerge

import "testing"

func TestTOMLFeatureProfileInfo(t *testing.T) {
	profile := TOMLFeatureProfileInfo()
	if profile.Family != "toml" {
		t.Fatalf("unexpected family: %+v", profile)
	}
	if len(profile.SupportedDialects) != 1 || profile.SupportedDialects[0] != DialectTOML {
		t.Fatalf("unexpected dialects: %+v", profile.SupportedDialects)
	}
	if len(profile.SupportedPolicies) != 1 || profile.SupportedPolicies[0].Name != "destination_wins_array" {
		t.Fatalf("unexpected policies: %+v", profile.SupportedPolicies)
	}

	treeSitterProfile := TOMLBackendFeatureProfileInfo(BackendTreeSitter)
	if treeSitterProfile.Backend != "kreuzberg-language-pack" || treeSitterProfile.BackendRef == nil {
		t.Fatalf("unexpected tree-sitter backend profile: %+v", treeSitterProfile)
	}
}

func TestParseTOMLAndMatchOwners(t *testing.T) {
	template := ParseTOML("title = \"Structured Merge\"\ntags = [\"merge\", \"toml\"]\n\n[package]\nname = \"structuredmerge\"\nversion = \"0.1.0\"\n", DialectTOML)
	destination := ParseTOML("title = \"Structured Merge\"\ntags = [\"merge\"]\nextra = 1\n\n[package]\nname = \"structuredmerge\"\nversion = \"0.2.0\"\n", DialectTOML)
	if !template.OK || template.Analysis == nil || !destination.OK || destination.Analysis == nil {
		t.Fatalf("expected parse success: %+v %+v", template, destination)
	}

	result := MatchTOMLOwners(*template.Analysis, *destination.Analysis)
	if len(result.Matched) != 6 {
		t.Fatalf("unexpected matched owners: %+v", result)
	}
	if len(result.UnmatchedTemplate) != 1 || result.UnmatchedTemplate[0] != "/tags/1" {
		t.Fatalf("unexpected unmatched template: %+v", result.UnmatchedTemplate)
	}
	if len(result.UnmatchedDestination) != 1 || result.UnmatchedDestination[0] != "/extra" {
		t.Fatalf("unexpected unmatched destination: %+v", result.UnmatchedDestination)
	}
}

func TestMergeTOML(t *testing.T) {
	result := MergeTOML(
		"title = \"Structured Merge\"\n\n[package]\nname = \"structuredmerge\"\ntags = [\"template\"]\nversion = \"0.1.0\"\n\n[package.meta]\nenabled = false\n",
		"[package]\ntags = [\"destination\"]\nversion = \"0.2.0\"\n\n[package.meta]\nauthors = [\"pb\"]\nrelease = true\n",
		DialectTOML,
	)
	if !result.OK || result.Output == nil {
		t.Fatalf("expected merge success: %+v", result)
	}
	expected := "title = \"Structured Merge\"\n\n[package]\nname = \"structuredmerge\"\ntags = [\"destination\"]\nversion = \"0.2.0\"\n\n[package.meta]\nauthors = [\"pb\"]\nenabled = false\nrelease = true\n"
	if *result.Output != expected {
		t.Fatalf("unexpected output:\n%s", *result.Output)
	}
}
