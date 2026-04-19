package treehaver

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/structuredmerge/structuredmerge-go/astmerge"
)

func readParserFixture(t *testing.T, parts ...string) map[string]any {
	t.Helper()

	pathParts := append([]string{"..", "..", "fixtures"}, parts...)
	path := filepath.Join(pathParts...)
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	var fixture map[string]any
	if err := json.Unmarshal(source, &fixture); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}

	return fixture
}

func TestSharedFixtureParserRequest(t *testing.T) {
	fixture := readParserFixture(t, "diagnostics", "slice-06-parser-adapters", "parser-request.json")
	requestFixture := fixture["request"].(map[string]any)
	infoFixture := fixture["adapter_info"].(map[string]any)

	request := ParserRequest{
		Source:   requestFixture["source"].(string),
		Language: requestFixture["language"].(string),
		Dialect:  requestFixture["dialect"].(string),
	}
	info := AdapterInfo{
		Backend:           infoFixture["backend"].(string),
		BackendRef:        nil,
		SupportsDialects:  infoFixture["supports_dialects"].(bool),
		SupportedPolicies: []astmerge.PolicyReference{},
	}

	if request.Source != requestFixture["source"].(string) ||
		request.Language != requestFixture["language"].(string) ||
		request.Dialect != requestFixture["dialect"].(string) {
		t.Fatalf("unexpected parser request: %+v", request)
	}
	if info.Backend != infoFixture["backend"].(string) ||
		info.SupportsDialects != infoFixture["supports_dialects"].(bool) {
		t.Fatalf("unexpected adapter info: %+v", info)
	}
}

func TestSharedFixtureAdapterPolicySupport(t *testing.T) {
	fixture := readParserFixture(t, "diagnostics", "slice-19-adapter-policy-support", "adapter-info.json")
	infoFixture := fixture["adapter_info"].(map[string]any)

	info := AdapterInfo{
		Backend:          infoFixture["backend"].(string),
		BackendRef:       nil,
		SupportsDialects: infoFixture["supports_dialects"].(bool),
		SupportedPolicies: []astmerge.PolicyReference{
			{
				Surface: astmerge.PolicySurfaceArray,
				Name:    "destination_wins_array",
			},
			{
				Surface: astmerge.PolicySurfaceFallback,
				Name:    "trailing_comma_destination_fallback",
			},
		},
	}

	if info.Backend != infoFixture["backend"].(string) ||
		info.SupportsDialects != infoFixture["supports_dialects"].(bool) {
		t.Fatalf("unexpected adapter info: %+v", info)
	}
	expectedPolicies := infoFixture["supported_policies"].([]any)
	if len(info.SupportedPolicies) != len(expectedPolicies) {
		t.Fatalf("unexpected supported policies: %+v", info.SupportedPolicies)
	}
	for index, policy := range info.SupportedPolicies {
		expected := expectedPolicies[index].(map[string]any)
		if string(policy.Surface) != expected["surface"].(string) || policy.Name != expected["name"].(string) {
			t.Fatalf("unexpected supported policy at %d: %+v", index, policy)
		}
	}
}

func TestSharedFixtureAdapterFeatureProfile(t *testing.T) {
	fixture := readParserFixture(t, "diagnostics", "slice-20-adapter-feature-profile", "feature-profile.json")
	profileFixture := fixture["feature_profile"].(map[string]any)

	profile := FeatureProfile{
		Backend:          profileFixture["backend"].(string),
		BackendRef:       nil,
		SupportsDialects: profileFixture["supports_dialects"].(bool),
		SupportedPolicies: []astmerge.PolicyReference{
			{
				Surface: astmerge.PolicySurfaceArray,
				Name:    "destination_wins_array",
			},
			{
				Surface: astmerge.PolicySurfaceFallback,
				Name:    "trailing_comma_destination_fallback",
			},
		},
	}

	if profile.Backend != profileFixture["backend"].(string) ||
		profile.SupportsDialects != profileFixture["supports_dialects"].(bool) {
		t.Fatalf("unexpected feature profile: %+v", profile)
	}
	expectedPolicies := profileFixture["supported_policies"].([]any)
	if len(profile.SupportedPolicies) != len(expectedPolicies) {
		t.Fatalf("unexpected feature-profile policies: %+v", profile.SupportedPolicies)
	}
	for index, policy := range profile.SupportedPolicies {
		expected := expectedPolicies[index].(map[string]any)
		if string(policy.Surface) != expected["surface"].(string) || policy.Name != expected["name"].(string) {
			t.Fatalf("unexpected feature-profile policy at %d: %+v", index, policy)
		}
	}
}

func TestSharedFixtureBackendRegistry(t *testing.T) {
	fixture := readParserFixture(t, "diagnostics", "slice-25-backend-registry", "backend-identities.json")

	backends := []BackendReference{
		{ID: "native", Family: "builtin"},
		{ID: "tree-sitter", Family: "tree-sitter"},
	}
	profile := FeatureProfile{
		Backend:           "tree-sitter",
		BackendRef:        &backends[1],
		SupportsDialects:  true,
		SupportedPolicies: []astmerge.PolicyReference{},
	}

	expectedBackends := fixture["backends"].([]any)
	if len(backends) != len(expectedBackends) {
		t.Fatalf("unexpected backends: %+v", backends)
	}
	for index, backend := range backends {
		expected := expectedBackends[index].(map[string]any)
		if backend.ID != expected["id"].(string) || backend.Family != expected["family"].(string) {
			t.Fatalf("unexpected backend at %d: %+v", index, backend)
		}
	}
	if profile.BackendRef == nil || profile.BackendRef.ID != "tree-sitter" || profile.BackendRef.Family != "tree-sitter" {
		t.Fatalf("unexpected backend reference on feature profile: %+v", profile)
	}
}
