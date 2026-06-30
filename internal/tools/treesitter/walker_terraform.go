package treesitter

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

// checkTerraformConventions analyzes Terraform HCL files for IaC best practices.
// Since tree-sitter doesn't include a Terraform grammar, we use regex patterns on
// raw HCL content. This covers the most common violations without full AST parsing.
func (w *fileWalker) checkTerraformConventions(lines []string) {
	if w.lang != analysis.LangTerraform {
		return
	}
	w.checkTFHardcodedValues(lines)
	w.checkTFProviderVersionPinning(lines)
	w.checkTFVariableDefaultsSensitive(lines)
	w.checkTFStateFileExposure(lines)
	w.checkTFResourceNaming(lines)
	w.checkTFCountVsForEach(lines)
	w.checkTFModuleCoupling(lines)
	w.checkTFDataSourceSafety(lines)
	w.checkTFPublicResourceExposure(lines)
	w.checkTFEncryptionDisabled(lines)
	w.checkTFLoggingDisabled(lines)
	w.checkTFBackupMissing(lines)
}

// checkTFHardcodedValues detects hardcoded resource names, regions, and environment-specific values.
// Pattern: resource "type" "name" with literal region, vpc_id, subnet_id, etc.
func (w *fileWalker) checkTFHardcodedValues(lines []string) {
	hardcodedPatterns := []*regexp.Regexp{
		regexp.MustCompile(`aws_region\s*=\s*"(?:us|eu|ap|ca|sa|me|af)-(?:east|west|central)-\d+"`),
		regexp.MustCompile(`region\s*=\s*"(?:us|eu|ap|ca|sa|me|af)-(?:east|west|central)-\d+"`),
		regexp.MustCompile(`availability_zone\s*=\s*"[a-z]{2}-[a-z]+-\d[a-z]?"`),
		regexp.MustCompile(`vpc_id\s*=\s*"vpc-[a-f0-9]{17}"`),
		regexp.MustCompile(`subnet_id\s*=\s*"subnet-[a-f0-9]{17}"`),
	}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		for _, pattern := range hardcodedPatterns {
			if pattern.MatchString(line) {
				w.emitFinding(analysis.Finding{
					Rule:        "terraform_conventions.hardcoded_values",
					Pillar:      "terraform_conventions",
					Severity:    analysis.SeverityBlocker,
					Line:        i + 1,
					Message:     "Hardcoded AWS region, AZ, VPC, or subnet ID detected — use variables for environment portability",
					Remediation: "Extract to a variable: region = var.aws_region or use terraform.tfvars for defaults",
				})
			}
		}
	}
}

// checkTFProviderVersionPinning detects unpinned provider versions.
// Pattern: provider "X" without version constraint, or "~> latest"
func (w *fileWalker) checkTFProviderVersionPinning(lines []string) {
	var inProviderBlock bool
	var providerName string
	var providerStart int
	var providerHasVersion bool

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.Contains(trimmed, "provider") && strings.Contains(trimmed, "{") {
			inProviderBlock = true
			providerName = extractProviderName(trimmed)
			providerStart = i + 1
			providerHasVersion = false
			continue
		}
		if inProviderBlock {
			if strings.Contains(trimmed, "}") {
				inProviderBlock = false
				if !providerHasVersion && providerName != "" {
					w.emitFinding(analysis.Finding{
						Rule:        "terraform_conventions.provider_version_pinning",
						Pillar:      "terraform_conventions",
						Severity:    analysis.SeverityMajor,
						Line:        providerStart,
						Message:     fmt.Sprintf("Provider %q has no version constraint — use required_version for reproducibility", providerName),
						Remediation: fmt.Sprintf("Set required_version in provider block: required_version = \"~> X.Y.Z\""),
					})
				}
			}
			if strings.Contains(trimmed, "required_version") || strings.Contains(trimmed, "version") {
				providerHasVersion = true
			}
		}
	}
}

// checkTFVariableDefaultsSensitive detects sensitive data in variable defaults.
// Pattern: variable with name or default containing secret-like strings (password, key, secret, token)
func (w *fileWalker) checkTFVariableDefaultsSensitive(lines []string) {
	secretKeywords := []string{"password", "secret", "api_key", "token", "private_key"}

	var inVariableBlock bool
	var variableBlockName string
	var hasDefault bool

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Detect variable block opening
		if strings.HasPrefix(trimmed, "variable") && strings.Contains(trimmed, "{") {
			inVariableBlock = true
			hasDefault = false
			// Extract variable name for context
			parts := strings.Split(trimmed, "\"")
			if len(parts) >= 2 {
				variableBlockName = parts[1]
			}
			continue
		}

		// Check if we're in a variable block
		if inVariableBlock {
			if strings.Contains(trimmed, "}") {
				inVariableBlock = false
				continue
			}

			// Track if we see a default value
			if strings.Contains(trimmed, "default") && strings.Contains(trimmed, "=") {
				hasDefault = true
			}

			// Check for sensitive keywords in variable name or in default value
			for _, keyword := range secretKeywords {
				// Check if variable name contains sensitive keyword
				if strings.Contains(strings.ToLower(variableBlockName), keyword) {
					// If it's a sensitive-named variable with a default, flag it
					if hasDefault || (strings.Contains(trimmed, "default") && strings.Contains(line, "=")) {
						w.emitFinding(analysis.Finding{
							Rule:        "terraform_conventions.variable_defaults_sensitive",
							Pillar:      "terraform_conventions",
							Severity:    analysis.SeverityBlocker,
							Line:        i + 1,
							Message:     fmt.Sprintf("Variable %q: sensitive data (password, key, secret, token) should not have a default value", variableBlockName),
							Remediation: "Remove default value; require the variable to be passed at runtime via terraform.tfvars or -var flags",
						})
						return
					}
				}

				// Check if default line contains sensitive value
				if strings.Contains(trimmed, "default") {
					lowerLine := strings.ToLower(line)
					if strings.Contains(lowerLine, keyword) && strings.Contains(line, "=") {
						w.emitFinding(analysis.Finding{
							Rule:        "terraform_conventions.variable_defaults_sensitive",
							Pillar:      "terraform_conventions",
							Severity:    analysis.SeverityBlocker,
							Line:        i + 1,
							Message:     fmt.Sprintf("Variable %q: sensitive data (password, key, secret, token) should not have a default value", variableBlockName),
							Remediation: "Remove default value; require the variable to be passed at runtime via terraform.tfvars or -var flags",
						})
						return
					}
				}
			}
		}
	}
}

// checkTFStateFileExposure detects Terraform state files in .gitignore absence.
// Pattern: terraform.tfstate or *.tfstate in the current directory implies risk
func (w *fileWalker) checkTFStateFileExposure(lines []string) {
	// Note: This check is best done by scanning .gitignore in the root,
	// but for simplicity here we flag *.tfstate patterns in HCL as a risk indicator.
	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), "tfstate") && !strings.HasPrefix(strings.TrimSpace(line), "#") {
			// Only flag if it looks like a hardcoded path or filename reference
			if strings.Contains(line, "terraform.tfstate") || strings.Contains(line, "backend") {
				w.emitFinding(analysis.Finding{
					Rule:        "terraform_conventions.state_file_exposure",
					Pillar:      "terraform_conventions",
					Severity:    analysis.SeverityBlocker,
					Line:        i + 1,
					Message:     "Ensure *.tfstate files are never committed to version control — add terraform.tfstate* to .gitignore",
					Remediation: "Add to .gitignore: terraform.tfstate, terraform.tfstate.*, .terraform/",
				})
			}
		}
	}
}

// checkTFResourceNaming detects inconsistent resource naming patterns.
// Pattern: resource names should follow snake_case convention
func (w *fileWalker) checkTFResourceNaming(lines []string) {
	resourcePattern := regexp.MustCompile(`^\s*resource\s+"([^"]+)"\s+"([^"]+)"\s*{`)

	for i, line := range lines {
		match := resourcePattern.FindStringSubmatch(line)
		if match == nil {
			continue
		}
		resourceName := match[2]
		// Check for inconsistent naming (mixing camelCase, UPPERCASE, etc)
		if !isValidTFResourceName(resourceName) {
			w.emitFinding(analysis.Finding{
				Rule:        "terraform_conventions.resource_naming",
				Pillar:      "terraform_conventions",
				Severity:    analysis.SeverityAdvisory,
				Line:        i + 1,
				Message:     fmt.Sprintf("Resource name %q should use snake_case (e.g., my_resource_name)", resourceName),
				Remediation: "Rename to follow snake_case convention for consistency across the codebase",
			})
		}
	}
}

// checkTFCountVsForEach detects use of count for dynamic resources (prefer for_each).
// Pattern: count.index used where for_each would be clearer
func (w *fileWalker) checkTFCountVsForEach(lines []string) {
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.Contains(line, "count") && strings.Contains(line, "count.index") {
			w.emitFinding(analysis.Finding{
				Rule:        "terraform_conventions.count_vs_for_each",
				Pillar:      "terraform_conventions",
				Severity:    analysis.SeverityAdvisory,
				Line:        i + 1,
				Message:     "Use for_each instead of count for dynamic resources — for_each is more explicit and resilient to list reordering",
				Remediation: "Replace count with for_each: for_each = toset(var.names) instead of count = length(var.names)",
			})
		}
	}
}

// checkTFModuleCoupling detects modules with hard dependencies on other modules.
// Pattern: module source referencing another module by relative path, or cross-module variable dependencies
func (w *fileWalker) checkTFModuleCoupling(lines []string) {
	var inModuleBlock bool

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Detect module block opening
		if strings.HasPrefix(trimmed, "module") && strings.Contains(trimmed, "{") {
			inModuleBlock = true
			continue
		}

		// Check if we're in a module block
		if inModuleBlock {
			if strings.Contains(trimmed, "}") {
				inModuleBlock = false
				continue
			}
			// Look for source with relative paths or module references
			if strings.Contains(trimmed, "source") {
				if strings.Contains(line, "../") || strings.Contains(line, "module.") {
					w.emitFinding(analysis.Finding{
						Rule:        "terraform_conventions.module_coupling",
						Pillar:      "terraform_conventions",
						Severity:    analysis.SeverityAdvisory,
						Line:        i + 1,
						Message:     "Module has hard dependency on another module path or output — consider loose coupling via variables",
						Remediation: "Use input variables to decouple modules; avoid relative paths (use registry or remote sources)",
					})
				}
			}
		}
	}
}

// checkTFDataSourceSafety detects unsafe data source queries.
// Pattern: data source without filter constraints, or data "aws_ami" without most_recent or specific filters
func (w *fileWalker) checkTFDataSourceSafety(lines []string) {
	dataSourcePattern := regexp.MustCompile(`^\s*data\s+"([^"]+)"\s+"([^"]+)"\s*{`)
	var inDataBlock bool
	var dataBlockStart int
	var dataType string
	var hasFilters bool

	for i, line := range lines {
		match := dataSourcePattern.FindStringSubmatch(line)
		if match != nil {
			inDataBlock = true
			dataBlockStart = i + 1
			dataType = match[1]
			hasFilters = false
			continue
		}
		if inDataBlock {
			if strings.TrimSpace(line) == "}" {
				inDataBlock = false
				if !hasFilters && isUnsafeDataSource(dataType) {
					w.emitFinding(analysis.Finding{
						Rule:        "terraform_conventions.data_source_safety",
						Pillar:      "terraform_conventions",
						Severity:    analysis.SeverityMajor,
						Line:        dataBlockStart,
						Message:     fmt.Sprintf("Data source %q lacks safety filters — may return unexpected results", dataType),
						Remediation: "Add filter block: filter { name = \"state\"; values = [\"available\"] } to constrain results",
					})
				}
			}
			if strings.Contains(line, "filter") {
				hasFilters = true
			}
		}
	}
}

// checkTFPublicResourceExposure detects publicly accessible resources without auth.
// Pattern: publicly_accessible = true or ingress rule 0.0.0.0/0 without auth
func (w *fileWalker) checkTFPublicResourceExposure(lines []string) {
	publicPatterns := []*regexp.Regexp{
		regexp.MustCompile(`publicly_accessible\s*=\s*true`),
		regexp.MustCompile(`cidr_blocks\s*=\s*\["0\.0\.0\.0/0"\]`),
		regexp.MustCompile(`ipv6_cidr_blocks\s*=\s*\["::/0"\]`),
	}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		for _, pattern := range publicPatterns {
			if pattern.MatchString(line) {
				w.emitFinding(analysis.Finding{
					Rule:        "terraform_conventions.public_resource_exposure",
					Pillar:      "terraform_conventions",
					Severity:    analysis.SeverityBlocker,
					Line:        i + 1,
					Message:     "Resource is publicly accessible without authentication — restrict CIDR or enable auth",
					Remediation: "Set publicly_accessible = false or limit cidr_blocks to internal networks; add authentication layer",
				})
			}
		}
	}
}

// checkTFEncryptionDisabled detects storage resources without encryption.
// Pattern: encrypted = false, server_side_encryption_configuration absent, etc.
func (w *fileWalker) checkTFEncryptionDisabled(lines []string) {
	encryptionPatterns := []*regexp.Regexp{
		regexp.MustCompile(`encrypted\s*=\s*false`),
		regexp.MustCompile(`enable_encryption\s*=\s*false`),
		regexp.MustCompile(`kms_key_id\s*=\s*""`),
	}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		for _, pattern := range encryptionPatterns {
			if pattern.MatchString(line) {
				w.emitFinding(analysis.Finding{
					Rule:        "terraform_conventions.encryption_disabled",
					Pillar:      "terraform_conventions",
					Severity:    analysis.SeverityBlocker,
					Line:        i + 1,
					Message:     "Storage resource encryption is disabled — enable encryption at rest",
					Remediation: "Set encrypted = true and provide kms_key_id (AWS) or similar for encryption key management",
				})
			}
		}
	}
}

// checkTFLoggingDisabled detects resources without logging.
// Pattern: enable_logging = false, access_logging block absent, etc.
func (w *fileWalker) checkTFLoggingDisabled(lines []string) {
	loggingPatterns := []*regexp.Regexp{
		regexp.MustCompile(`enable_logging\s*=\s*false`),
		regexp.MustCompile(`logging_enabled\s*=\s*false`),
	}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		for _, pattern := range loggingPatterns {
			if pattern.MatchString(line) {
				w.emitFinding(analysis.Finding{
					Rule:        "terraform_conventions.logging_disabled",
					Pillar:      "terraform_conventions",
					Severity:    analysis.SeverityMajor,
					Line:        i + 1,
					Message:     "Logging is disabled — enable audit logs for compliance and troubleshooting",
					Remediation: "Set enable_logging = true; configure log destination (CloudWatch, S3, etc.)",
				})
			}
		}
	}
}

// checkTFBackupMissing detects resources without backup strategy.
// Pattern: backup_retention_days absent or = 0, skip_final_snapshot = true, etc.
func (w *fileWalker) checkTFBackupMissing(lines []string) {
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.Contains(line, "skip_final_snapshot") && strings.Contains(line, "true") {
			w.emitFinding(analysis.Finding{
				Rule:        "terraform_conventions.backup_missing",
				Pillar:      "terraform_conventions",
				Severity:    analysis.SeverityMajor,
				Line:        i + 1,
				Message:     "Resource deletion will skip final snapshot — enable backups and final snapshots for disaster recovery",
				Remediation: "Set skip_final_snapshot = false; enable backup_retention_days; use automated backup policies",
			})
		}
		if strings.Contains(line, "backup_retention_days") && strings.Contains(line, "= 0") {
			w.emitFinding(analysis.Finding{
				Rule:        "terraform_conventions.backup_missing",
				Pillar:      "terraform_conventions",
				Severity:    analysis.SeverityMajor,
				Line:        i + 1,
				Message:     "Backup retention is disabled (0 days) — enable automatic backups",
				Remediation: "Set backup_retention_days to at least 7 (or your org's retention policy)",
			})
		}
	}
}

// Helper functions

func extractProviderName(line string) string {
	// Extract provider name from: provider "aws" { or provider "google" {
	re := regexp.MustCompile(`provider\s+"([^"]+)"`)
	match := re.FindStringSubmatch(line)
	if match != nil {
		return match[1]
	}
	return ""
}

func isValidTFResourceName(name string) bool {
	// Terraform resource names should be lowercase with underscores only
	for _, ch := range name {
		if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '_') {
			return false
		}
	}
	return true
}

func isUnsafeDataSource(dataType string) bool {
	// Data sources that require safety filters
	unsafeTypes := map[string]bool{
		"aws_ami":               true,
		"aws_images":            true,
		"aws_rds_database":      true,
		"azurerm_storage_blob":  true,
		"google_compute_image":  true,
		"aws_ec2_instance":      true,
	}
	return unsafeTypes[dataType]
}
