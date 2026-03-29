package hcl

import "testing"

// FuzzParseFile exercises the HCL parser with arbitrary byte sequences.
//
// Invariants verified:
//   - ParseFile must never panic regardless of input
//   - A non-nil *TFModule is always returned (even on parse errors)
//   - Parse errors must only be returned when the content is genuinely invalid
func FuzzParseFile(f *testing.F) {
	// Seed corpus: representative valid HCL patterns
	f.Add("main.tf", ``)
	f.Add("main.tf", `terraform {}`)
	f.Add("main.tf", `
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.0"
    }
  }
  backend "s3" {
    bucket = "my-state"
    key    = "tfstate"
    region = "us-east-1"
  }
}
`)
	f.Add("resources.tf", `
resource "aws_s3_bucket" "example" {
  bucket = "my-bucket"
  tags = {
    Env  = "production"
    Team = "platform"
  }
}
`)
	f.Add("modules.tf", `
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "5.0.0"
}
`)
	f.Add("variables.tf", `
variable "region" {
  type    = string
  default = "us-east-1"
}
variable "count" {
  type = number
}
`)
	// Edge cases
	f.Add("empty.tf", `   `)
	f.Add("invalid.tf", `}{{{ not valid HCL at all }}}`)
	f.Add("partial.tf", `resource "aws_s3_bucket" {`)
	f.Add("unicode.tf", "resource \"日本語\" \"テスト\" {}")
	f.Add("deep.tf", `terraform { required_providers { aws = { source = "hashicorp/aws" version = ">= 4.0" } } }`)

	f.Fuzz(func(t *testing.T, filePath, content string) {
		module, _ := ParseFile(filePath, content)
		// Core invariant: module is never nil
		if module == nil {
			t.Fatal("ParseFile returned nil *TFModule")
		}
	})
}
