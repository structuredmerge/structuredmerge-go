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
	if len(backends) != 3 || backends[0] != BackendYAMLV3 || backends[1] != BackendGoccyGoYAML || backends[2] != BackendKreuzberg {
		t.Fatalf("unexpected backends: %+v", backends)
	}

	yamlV3Profile := YAMLBackendFeatureProfileInfo(BackendYAMLV3)
	if yamlV3Profile.Backend != "yaml-v3" {
		t.Fatalf("unexpected yaml-v3 backend profile: %+v", yamlV3Profile)
	}

	goccyProfile := YAMLBackendFeatureProfileInfo(BackendGoccyGoYAML)
	if goccyProfile.Backend != "goccy-go-yaml" {
		t.Fatalf("unexpected goccy backend profile: %+v", goccyProfile)
	}

	kreuzbergProfile := YAMLBackendFeatureProfileInfo(BackendKreuzberg)
	if kreuzbergProfile.Backend != "kreuzberg-language-pack" {
		t.Fatalf("unexpected kreuzberg backend profile: %+v", kreuzbergProfile)
	}
}
