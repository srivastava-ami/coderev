package treesitter

import (
	"testing"

	"github.com/srivastava-ami/coderev/internal/analysis"
)

// TestTFHardcodedValuesRegion detects hardcoded AWS regions.
func TestTFHardcodedValuesRegion(t *testing.T) {
	src := `provider "aws" {
  region = "us-east-1"
}`
	findings := findingsForSrc(t, src, analysis.LangTerraform)
	if !hasRule(findings, "terraform_conventions.hardcoded_values") {
		t.Error("must flag hardcoded AWS region us-east-1")
	}
}

// TestTFHardcodedValuesVPC detects hardcoded VPC IDs.
func TestTFHardcodedValuesVPC(t *testing.T) {
	src := `resource "aws_instance" "web" {
  vpc_id = "vpc-0123456789abcdef0"
}`
	findings := findingsForSrc(t, src, analysis.LangTerraform)
	if !hasRule(findings, "terraform_conventions.hardcoded_values") {
		t.Error("must flag hardcoded VPC ID")
	}
}

// TestTFProviderVersionPinning detects unpinned provider versions.
func TestTFProviderVersionPinning(t *testing.T) {
	src := `provider "aws" {
  region = var.aws_region
}`
	findings := findingsForSrc(t, src, analysis.LangTerraform)
	if !hasRule(findings, "terraform_conventions.provider_version_pinning") {
		t.Error("must flag provider without required_version")
	}
}

// TestTFProviderVersionPinnedNoFP allows pinned provider versions.
func TestTFProviderVersionPinnedNoFP(t *testing.T) {
	src := `provider "aws" {
  required_version = "~> 5.0"
  region = var.aws_region
}`
	findings := findingsForSrc(t, src, analysis.LangTerraform)
	if hasRule(findings, "terraform_conventions.provider_version_pinning") {
		t.Error("must NOT flag provider with required_version")
	}
}

// TestTFVariableDefaultsSensitivePassword detects password defaults.
func TestTFVariableDefaultsSensitivePassword(t *testing.T) {
	src := `variable "db_password" {
  type = string
  default = "mypassword123"
}`
	findings := findingsForSrc(t, src, analysis.LangTerraform)
	if !hasRule(findings, "terraform_conventions.variable_defaults_sensitive") {
		t.Error("must flag password default value")
	}
}

// TestTFVariableDefaultsSensitiveAPIKey detects API key defaults.
func TestTFVariableDefaultsSensitiveAPIKey(t *testing.T) {
	src := `variable "api_key" {
  type = string
  default = "sk-1234567890abcdef"
}`
	findings := findingsForSrc(t, src, analysis.LangTerraform)
	if !hasRule(findings, "terraform_conventions.variable_defaults_sensitive") {
		t.Error("must flag API key default value")
	}
}

// TestTFStateFileExposure detects state file exposure patterns.
func TestTFStateFileExposure(t *testing.T) {
	src := `terraform {
  backend "local" {
    path = "terraform.tfstate"
  }
}`
	findings := findingsForSrc(t, src, analysis.LangTerraform)
	if !hasRule(findings, "terraform_conventions.state_file_exposure") {
		t.Error("must flag tfstate file path reference")
	}
}

// TestTFResourceNamingCamelCase detects non-snake_case resource names.
func TestTFResourceNamingCamelCase(t *testing.T) {
	src := `resource "aws_instance" "webServer" {
  instance_type = "t2.micro"
}`
	findings := findingsForSrc(t, src, analysis.LangTerraform)
	if !hasRule(findings, "terraform_conventions.resource_naming") {
		t.Error("must flag camelCase resource name webServer")
	}
}

// TestTFResourceNamingSnakeCaseOK allows snake_case names.
func TestTFResourceNamingSnakeCaseOK(t *testing.T) {
	src := `resource "aws_instance" "web_server" {
  instance_type = "t2.micro"
}`
	findings := findingsForSrc(t, src, analysis.LangTerraform)
	if hasRule(findings, "terraform_conventions.resource_naming") {
		t.Error("must NOT flag snake_case resource name web_server")
	}
}

// TestTFCountVsForEach detects count.index usage.
func TestTFCountVsForEach(t *testing.T) {
	src := `resource "aws_instance" "servers" {
  count = length(var.servers)
  instance_type = var.instance_types[count.index]
}`
	findings := findingsForSrc(t, src, analysis.LangTerraform)
	if !hasRule(findings, "terraform_conventions.count_vs_for_each") {
		t.Error("must flag count.index usage")
	}
}

// TestTFModuleCoupling detects relative module paths.
func TestTFModuleCoupling(t *testing.T) {
	src := `module "network" {
  source = "../network"
  vpc_cidr = var.vpc_cidr
}`
	findings := findingsForSrc(t, src, analysis.LangTerraform)
	if !hasRule(findings, "terraform_conventions.module_coupling") {
		t.Error("must flag relative module path ../network")
	}
}

// TestTFDataSourceSafetyAMI detects unsafe AMI data source.
func TestTFDataSourceSafetyAMI(t *testing.T) {
	src := `data "aws_ami" "ubuntu" {
  owners = ["099720109477"]
}`
	findings := findingsForSrc(t, src, analysis.LangTerraform)
	if !hasRule(findings, "terraform_conventions.data_source_safety") {
		t.Error("must flag aws_ami data source without safety filters")
	}
}

// TestTFDataSourceSafetyWithFiltersNoFP allows safe data sources.
func TestTFDataSourceSafetyWithFiltersNoFP(t *testing.T) {
	src := `data "aws_ami" "ubuntu" {
  owners = ["099720109477"]
  filter {
    name   = "state"
    values = ["available"]
  }
}`
	findings := findingsForSrc(t, src, analysis.LangTerraform)
	if hasRule(findings, "terraform_conventions.data_source_safety") {
		t.Error("must NOT flag aws_ami with safety filters")
	}
}

// TestTFPublicResourceExposurePubliclyAccessible detects publicly_accessible true.
func TestTFPublicResourceExposurePubliclyAccessible(t *testing.T) {
	src := `resource "aws_rds_cluster" "db" {
  publicly_accessible = true
}`
	findings := findingsForSrc(t, src, analysis.LangTerraform)
	if !hasRule(findings, "terraform_conventions.public_resource_exposure") {
		t.Error("must flag publicly_accessible = true")
	}
}

// TestTFPublicResourceExposureCIDR detects 0.0.0.0/0 in security group.
func TestTFPublicResourceExposureCIDR(t *testing.T) {
	src := `resource "aws_security_group_rule" "allow_ssh" {
  cidr_blocks = ["0.0.0.0/0"]
  from_port   = 22
  to_port     = 22
}`
	findings := findingsForSrc(t, src, analysis.LangTerraform)
	if !hasRule(findings, "terraform_conventions.public_resource_exposure") {
		t.Error("must flag CIDR 0.0.0.0/0")
	}
}

// TestTFEncryptionDisabledFalse detects encrypted = false.
func TestTFEncryptionDisabledFalse(t *testing.T) {
	src := `resource "aws_s3_bucket" "logs" {
  encrypted = false
}`
	findings := findingsForSrc(t, src, analysis.LangTerraform)
	if !hasRule(findings, "terraform_conventions.encryption_disabled") {
		t.Error("must flag encrypted = false")
	}
}

// TestTFEncryptionDisabledKMSEmpty detects empty KMS key.
func TestTFEncryptionDisabledKMSEmpty(t *testing.T) {
	src := `resource "aws_ebs_volume" "data" {
  kms_key_id = ""
}`
	findings := findingsForSrc(t, src, analysis.LangTerraform)
	if !hasRule(findings, "terraform_conventions.encryption_disabled") {
		t.Error("must flag empty kms_key_id")
	}
}

// TestTFLoggingDisabledFalse detects enable_logging = false.
func TestTFLoggingDisabledFalse(t *testing.T) {
	src := `resource "aws_s3_bucket" "app" {
  enable_logging = false
}`
	findings := findingsForSrc(t, src, analysis.LangTerraform)
	if !hasRule(findings, "terraform_conventions.logging_disabled") {
		t.Error("must flag enable_logging = false")
	}
}

// TestTFBackupMissingSkipFinalSnapshot detects skip_final_snapshot = true.
func TestTFBackupMissingSkipFinalSnapshot(t *testing.T) {
	src := `resource "aws_db_instance" "db" {
  skip_final_snapshot = true
}`
	findings := findingsForSrc(t, src, analysis.LangTerraform)
	if !hasRule(findings, "terraform_conventions.backup_missing") {
		t.Error("must flag skip_final_snapshot = true")
	}
}

// TestTFBackupMissingRetentionZero detects backup_retention_days = 0.
func TestTFBackupMissingRetentionZero(t *testing.T) {
	src := `resource "aws_rds_cluster" "db" {
  backup_retention_days = 0
}`
	findings := findingsForSrc(t, src, analysis.LangTerraform)
	if !hasRule(findings, "terraform_conventions.backup_missing") {
		t.Error("must flag backup_retention_days = 0")
	}
}

// TestTFCommentedLineIgnored skips commented lines.
func TestTFCommentedLineIgnored(t *testing.T) {
	src := `# provider "aws" {
#   region = "us-east-1"
# }`
	findings := findingsForSrc(t, src, analysis.LangTerraform)
	if hasRule(findings, "terraform_conventions.hardcoded_values") {
		t.Error("must NOT flag commented-out code")
	}
}

// TestTFMultipleViolations detects multiple violations in one file.
func TestTFMultipleViolations(t *testing.T) {
	src := `provider "aws" {
  region = "us-east-1"
}

resource "aws_instance" "webServer" {
  vpc_id = "vpc-0123456789abcdef0"
  publicly_accessible = true
}`
	findings := findingsForSrc(t, src, analysis.LangTerraform)
	if !hasRule(findings, "terraform_conventions.hardcoded_values") {
		t.Error("must flag hardcoded values")
	}
	if !hasRule(findings, "terraform_conventions.resource_naming") {
		t.Error("must flag resource naming")
	}
	if !hasRule(findings, "terraform_conventions.public_resource_exposure") {
		t.Error("must flag public resource exposure")
	}
}
