package hcl

import (
	"strings"
	"testing"
)

const (
	tfFileName    = "test.tf"
	invalidTFFile = "invalid.tf"
)

func parseModule(t *testing.T, content string) *TFModule {
	t.Helper()
	module, err := ParseFile(tfFileName, content)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	return module
}

func TestParseProviders(t *testing.T) {
	content := `
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 4.0"
    }
  }
}
`
	module := parseModule(t, content)

	if len(module.Providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(module.Providers))
	}

	p := module.Providers[0]
	if p.Source != "hashicorp/aws" {
		t.Errorf("expected source 'hashicorp/aws', got %q", p.Source)
	}
	if p.Version != ">= 4.0" {
		t.Errorf("expected version '>= 4.0', got %q", p.Version)
	}
}

func TestParseBackend(t *testing.T) {
	content := `
terraform {
  backend "s3" {
    bucket = "my-terraform-state"
    key    = "terraform.tfstate"
    region = "us-east-1"
  }
}
`
	module := parseModule(t, content)

	if module.Backend == nil {
		t.Fatal("expected backend, got nil")
	}
	if module.Backend.Type != "s3" {
		t.Errorf("expected backend type 's3', got %q", module.Backend.Type)
	}
	if module.Backend.Config["bucket"] != "my-terraform-state" {
		t.Errorf("expected bucket 'my-terraform-state', got %q", module.Backend.Config["bucket"])
	}
}

func TestParseResources(t *testing.T) {
	content := `
resource "aws_s3_bucket" "my_bucket" {
  bucket = "my-bucket"

  tags = {
    Environment = "production"
    Team        = "platform"
  }
}
`
	module := parseModule(t, content)

	if len(module.Resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(module.Resources))
	}

	res := module.Resources[0]
	if res.Type != "aws_s3_bucket" {
		t.Errorf("expected type 'aws_s3_bucket', got %q", res.Type)
	}
	if res.Name != "my_bucket" {
		t.Errorf("expected name 'my_bucket', got %q", res.Name)
	}
	if res.Labels["Environment"] != "production" {
		t.Errorf("expected Environment=production, got %q", res.Labels["Environment"])
	}
	if res.Labels["Team"] != "platform" {
		t.Errorf("expected Team=platform, got %q", res.Labels["Team"])
	}
}

func TestParseModuleCall(t *testing.T) {
	content := `
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "3.0.0"
}
`
	module := parseModule(t, content)

	if len(module.Modules) != 1 {
		t.Fatalf("expected 1 module call, got %d", len(module.Modules))
	}

	mc := module.Modules[0]
	if mc.Name != "vpc" {
		t.Errorf("expected name 'vpc', got %q", mc.Name)
	}
	if mc.Source != "terraform-aws-modules/vpc/aws" {
		t.Errorf("expected source 'terraform-aws-modules/vpc/aws', got %q", mc.Source)
	}
}

func TestParseInvalidHCL(t *testing.T) {
	content := `
this is not valid HCL }{{{ garbage
`
	module, err := ParseFile(invalidTFFile, content)
	if err == nil {
		t.Fatal("expected parse error for invalid HCL")
	}
	if module == nil {
		t.Fatal("expected non-nil module even on parse error")
	}
	if !strings.HasSuffix(module.Path, ":parse-error") {
		t.Errorf("expected path to end with ':parse-error', got %q", module.Path)
	}
}
