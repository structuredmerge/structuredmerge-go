package yamlmerge

import "testing"

func TestYAMLFeatureProfileInfo(t *testing.T) {
	profile := YAMLFeatureProfileInfo()
	if profile.Family != "yaml" {
		t.Fatalf("unexpected family: %+v", profile)
	}
	if len(profile.SupportedDialects) != 1 || profile.SupportedDialects[0] != DialectYAML {
		t.Fatalf("unexpected dialects: %+v", profile.SupportedDialects)
	}
	if len(profile.SupportedPolicies) != 1 || profile.SupportedPolicies[0].Name != "destination_wins_array" {
		t.Fatalf("unexpected policies: %+v", profile.SupportedPolicies)
	}

	backends := AvailableYAMLBackends()
	if len(backends) != 1 || backends[0] != BackendKreuzberg {
		t.Fatalf("unexpected backends: %+v", backends)
	}

	kreuzbergProfile := YAMLBackendFeatureProfileInfo(BackendKreuzberg)
	if kreuzbergProfile.Backend != "kreuzberg-language-pack" {
		t.Fatalf("unexpected kreuzberg backend profile: %+v", kreuzbergProfile)
	}
}
