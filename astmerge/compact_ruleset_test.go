package astmerge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseCompactRulesetFixtures(t *testing.T) {
	rulesetRoot := filepath.Join("..", "..", "fixtures", "rulesets")
	count := 0
	if err := filepath.Walk(rulesetRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".smrules" {
			return nil
		}

		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		result := ParseCompactRuleset(string(source))
		if !result.OK {
			t.Fatalf("expected %s to parse, diagnostics: %#v", path, result.Diagnostics)
		}
		if result.Analysis == nil || len(result.Analysis.Directives) == 0 {
			t.Fatalf("expected %s to produce directives", path)
		}
		count++
		return nil
	}); err != nil {
		t.Fatalf("walk ruleset fixtures: %v", err)
	}
	if count == 0 {
		t.Fatal("expected compact ruleset fixtures")
	}
}

func TestParseCompactRulesetEdges(t *testing.T) {
	cases := map[string]string{
		"missing-required":  "format json\nowners line_bound_statements\nmatch stable_path\nread native_read_portable_write\n",
		"repeated-format":   "format json\nformat yaml\nowners line_bound_statements\nmatch stable_path\nread native_read_portable_write\nattach layout_only\n",
		"unknown-read":      "format json\nowners line_bound_statements\nmatch stable_path\nread imaginary\nattach layout_only\n",
		"unknown-directive": "format json\nowners line_bound_statements\nmatch stable_path\nread native_read_portable_write\nattach layout_only\nmystery value\n",
	}
	for name, source := range cases {
		t.Run(name, func(t *testing.T) {
			result := ParseCompactRuleset(source)
			if result.OK {
				t.Fatalf("expected %s to fail", name)
			}
			if len(result.Diagnostics) == 0 {
				t.Fatalf("expected %s to produce diagnostics", name)
			}
		})
	}
}

func TestCompactRulesetFeatureProfileFixture(t *testing.T) {
	fixtureSource, err := os.ReadFile(filepath.Join("..", "..", "fixtures", "diagnostics", "slice-781-compact-ruleset-profile", "module-profile.json"))
	if err != nil {
		t.Fatalf("read profile fixture: %v", err)
	}
	var fixture struct {
		RulesetPath []string              `json:"ruleset_path"`
		Profile     CompactRulesetProfile `json:"profile"`
	}
	if err := json.Unmarshal(fixtureSource, &fixture); err != nil {
		t.Fatalf("parse profile fixture: %v", err)
	}

	rulesetSource, err := os.ReadFile(filepath.Join(append([]string{"..", "..", "fixtures"}, fixture.RulesetPath...)...))
	if err != nil {
		t.Fatalf("read ruleset fixture: %v", err)
	}
	result := ParseCompactRuleset(string(rulesetSource))
	if !result.OK || result.Analysis == nil {
		t.Fatalf("expected profile ruleset to parse: %#v", result.Diagnostics)
	}

	actual := CompactRulesetFeatureProfile(*result.Analysis)
	if !reflect.DeepEqual(actual, fixture.Profile) {
		t.Fatalf("unexpected compact ruleset profile:\nactual:   %#v\nexpected: %#v", actual, fixture.Profile)
	}
}
