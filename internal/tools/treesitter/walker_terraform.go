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

type tfBlockState struct {
	inBlock   bool
	name      string
	startLine int
	hasKey    bool
}

func (s *tfBlockState) reset(name string, start int) {
	s.inBlock = true
	s.name = name
	s.startLine = start
	s.hasKey = false
}

// checkTFProviderVersionPinning detects unpinned provider versions.
func (w *fileWalker) checkTFProviderVersionPinning(lines []string) {
	var st tfBlockState
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		if matched, name := matchProviderOpen(trimmed); matched {
			st.reset(name, i+1)
			continue
		}
		if !st.inBlock {
			continue
		}
		if strings.Contains(trimmed, "}") {
			st.inBlock = false
			if !st.hasKey && st.name != "" {
				w.emitProviderVersionFinding(st.name, st.startLine)
			}
			continue
		}
		if strings.Contains(trimmed, "required_version") || strings.Contains(trimmed, "version") {
			st.hasKey = true
		}
	}
}

func matchProviderOpen(trimmed string) (bool, string) {
	if strings.Contains(trimmed, "provider") && strings.Contains(trimmed, "{") {
		return true, extractProviderName(trimmed)
	}
	return false, ""
}

func (w *fileWalker) emitProviderVersionFinding(name string, line int) {
	w.emitFinding(analysis.Finding{
		Rule:        "terraform_conventions.provider_version_pinning",
		Pillar:      "terraform_conventions",
		Severity:    analysis.SeverityMajor,
		Line:        line,
		Message:     fmt.Sprintf("Provider %q has no version constraint — use required_version for reproducibility", name),
		Remediation: fmt.Sprintf("Set required_version in provider block: required_version = \"~> X.Y.Z\""),
	})
}

var tfSecretKeywords = []string{"password", "secret", "api_key", "token", "private_key"}

type tfVarState struct {
	inBlock bool
	name    string
	hasDef  bool
}

func (s *tfVarState) open(trimmed string) {
	s.inBlock = true
	s.hasDef = false
	parts := strings.Split(trimmed, "\"")
	if len(parts) >= 2 {
		s.name = parts[1]
	}
}

func (s *tfVarState) close() { s.inBlock = false }

// checkTFVariableDefaultsSensitive detects sensitive data in variable defaults.
func (w *fileWalker) checkTFVariableDefaultsSensitive(lines []string) {
	var st tfVarState
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "variable") && strings.Contains(trimmed, "{") {
			st.open(trimmed)
			continue
		}
		if !st.inBlock {
			continue
		}
		if strings.Contains(trimmed, "}") {
			st.close()
			continue
		}
		if strings.Contains(trimmed, "default") && strings.Contains(trimmed, "=") {
			st.hasDef = true
		}
		if w.emitIfSecretInVar(line, trimmed, st, i) {
			return
		}
	}
}

func (w *fileWalker) emitIfSecretInVar(line, trimmed string, st tfVarState, idx int) bool {
	if hasSecretKeyword(strings.ToLower(st.name)) && (st.hasDef || hasDefaultAssignment(trimmed)) {
		w.emitSecretDefaultFinding(st.name, idx)
		return true
	}
	if strings.Contains(trimmed, "default") && hasSecretKeyword(strings.ToLower(line)) && strings.Contains(line, "=") {
		w.emitSecretDefaultFinding(st.name, idx)
		return true
	}
	return false
}

func hasSecretKeyword(s string) bool {
	for _, kw := range tfSecretKeywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}

func hasDefaultAssignment(trimmed string) bool {
	return strings.Contains(trimmed, "default") && strings.Contains(trimmed, "=")
}

func (w *fileWalker) emitSecretDefaultFinding(name string, line int) {
	w.emitFinding(analysis.Finding{
		Rule:        "terraform_conventions.variable_defaults_sensitive",
		Pillar:      "terraform_conventions",
		Severity:    analysis.SeverityBlocker,
		Line:        line + 1,
		Message:     fmt.Sprintf("Variable %q: sensitive data (password, key, secret, token) should not have a default value", name),
		Remediation: "Remove default value; require the variable to be passed at runtime via terraform.tfvars or -var flags",
	})
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
func (w *fileWalker) checkTFModuleCoupling(lines []string) {
	var inBlock bool
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "module") && strings.Contains(trimmed, "{") {
			inBlock = true
			continue
		}
		if !inBlock {
			continue
		}
		if strings.Contains(trimmed, "}") {
			inBlock = false
			continue
		}
		if strings.Contains(trimmed, "source") && (strings.Contains(line, "../") || strings.Contains(line, "module.")) {
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

type tfDataSourceState struct {
	inBlock   bool
	dataType  string
	startLine int
	hasFilter bool
}

// checkTFDataSourceSafety detects unsafe data source queries.
func (w *fileWalker) checkTFDataSourceSafety(lines []string) {
	var reDataSource = regexp.MustCompile(`^\s*data\s+"([^"]+)"\s+"([^"]+)"\s*{`)
	var st tfDataSourceState
	for i, line := range lines {
		if m := reDataSource.FindStringSubmatch(line); m != nil {
			st = tfDataSourceState{inBlock: true, dataType: m[1], startLine: i + 1}
			continue
		}
		if !st.inBlock {
			continue
		}
		if strings.TrimSpace(line) == "}" {
			st.inBlock = false
			if !st.hasFilter && isUnsafeDataSource(st.dataType) {
				w.emitDataSourceFinding(st.dataType, st.startLine)
			}
			continue
		}
		if strings.Contains(line, "filter") {
			st.hasFilter = true
		}
	}
}

func (w *fileWalker) emitDataSourceFinding(dataType string, line int) {
	w.emitFinding(analysis.Finding{
		Rule:        "terraform_conventions.data_source_safety",
		Pillar:      "terraform_conventions",
		Severity:    analysis.SeverityMajor,
		Line:        line,
		Message:     fmt.Sprintf("Data source %q lacks safety filters — may return unexpected results", dataType),
		Remediation: "Add filter block: filter { name = \"state\"; values = [\"available\"] } to constrain results",
	})
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
