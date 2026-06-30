package analysis

import (
	"testing"

	"github.com/BurntSushi/toml"
)

// TestGenericStandardsPopulation verifies that generic pillars are populated
// from TOML without requiring Go struct definitions for new rule categories.
func TestGenericStandardsPopulation(t *testing.T) {
	tomlData := `
[meta]
version = "1.0"

[security]
severity = "blocker"

[security.secrets]
rule = "security.secrets"
checks = ["pattern", "entropy"]

[custom_pillar]
severity = "advisory"

[custom_pillar.custom_rule_category]
rule = "custom_pillar.custom_rule_category"
severity = "major"
remediation = "This is a custom rule discovered from TOML"
custom_field = "custom value"
`

	var std Standards
	if _, err := toml.Decode(tomlData, &std); err != nil {
		t.Fatalf("Failed to decode TOML: %v", err)
	}

	// Verify pillars were populated
	if std.Pillars == nil {
		t.Fatal("Pillars map not populated")
	}

	// Check built-in pillar (security) is in the generic map
	if security, ok := std.Pillars["security"]; ok {
		if _, hasSecrets := security["secrets"]; !hasSecrets {
			t.Error("security.secrets category not found")
		}
	} else {
		t.Error("security pillar not found in generic pillars")
	}

	// Check custom pillar was discovered without Go code changes
	if customPillar, ok := std.Pillars["custom_pillar"]; ok {
		if customRule, hasCustomRule := customPillar["custom_rule_category"]; hasCustomRule {
			rule := customRule.GetString("rule")
			if rule != "custom_pillar.custom_rule_category" {
				t.Errorf("Expected rule ID, got '%s'", rule)
			}
			remediation := customRule.GetString("remediation")
			if remediation == "" {
				t.Error("remediation field not found")
			}
		} else {
			t.Error("custom_rule_category not found")
		}
	} else {
		t.Fatal("custom_pillar not found - TOML-driven rules NOT working!")
	}

	// Verify rules were registered
	if _, exists := RuleRegistry["custom_pillar.custom_rule_category"]; !exists {
		t.Error("custom rule not registered in RuleRegistry")
	}
}

// TestGenericRuleCategoryFieldAccess verifies helper methods work correctly
func TestGenericRuleCategoryFieldAccess(t *testing.T) {
	category := GenericRuleCategory{
		"rule":     "test.rule",
		"severity": "blocker",
		"checks":   []interface{}{"check1", "check2"},
		"max_value": int64(10),
		"enabled":  true,
	}

	if rule := category.GetString("rule"); rule != "test.rule" {
		t.Errorf("GetString failed: expected 'test.rule', got '%s'", rule)
	}

	checks := category.GetStringSlice("checks")
	if len(checks) != 2 || checks[0] != "check1" {
		t.Errorf("GetStringSlice failed: got %v", checks)
	}

	maxVal := category.GetInt("max_value")
	if maxVal != 10 {
		t.Errorf("GetInt failed: expected 10, got %d", maxVal)
	}

	if enabled := category.GetBool("enabled"); !enabled {
		t.Error("GetBool failed: expected true")
	}
}
