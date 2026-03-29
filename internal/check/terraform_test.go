package check

import (
	"testing"

	"github.com/ffreis/platform-guardian/internal/hcl"
	"github.com/ffreis/platform-guardian/internal/scanner"
)

const (
	testRepo = "test/repo"

	providerAWS    = "hashicorp/aws"
	providerGoogle = "hashicorp/google"

	backendS3Type    = "s3"
	backendBucketKey = "bucket"
	backendBucket    = "my-state-bucket"

	passResultFmt = "expected Pass, got %s: %s"
)

func tfMockSnapshot() *scanner.RepoSnapshot {
	snap := scanner.NewSnapshot(testRepo)
	snap.TFModules = []hcl.TFModule{
		{
			Path: "main.tf",
			Variables: []hcl.TFVariable{
				{Name: "environment", Type: "string"},
				{Name: "region", Type: "string"},
			},
			Providers: []hcl.TFProvider{
				{Source: providerAWS, Version: ">= 4.0"},
			},
			Backend: &hcl.TFBackend{
				Type: backendS3Type,
				Config: map[string]string{
					backendBucketKey: backendBucket,
					"key":            "terraform.tfstate",
					"region":         "us-east-1",
				},
			},
			Resources: []hcl.TFResource{
				{
					Type: "aws_s3_bucket",
					Name: "my_bucket",
					Labels: map[string]string{
						"Environment": "production",
						"Team":        "platform",
					},
				},
				{
					Type: "aws_iam_role",
					Name: "my_role",
					Labels: map[string]string{
						"Environment": "production",
						"Team":        "platform",
					},
				},
			},
			Modules: []hcl.TFModuleCall{
				{Name: "vpc", Source: "terraform-aws-modules/vpc/aws"},
			},
		},
	}
	return snap
}

func TestTFProviderRequiredPass(t *testing.T) {
	snap := tfMockSnapshot()
	checker := &TFProviderReqChecker{Source: providerAWS}
	result := checker.Evaluate(snap)
	if result.Status != Pass {
		t.Errorf(passResultFmt, result.Status, result.Message)
	}
}

func TestTFProviderRequiredFail(t *testing.T) {
	snap := tfMockSnapshot()
	checker := &TFProviderReqChecker{Source: providerGoogle}
	result := checker.Evaluate(snap)
	if result.Status != Fail {
		t.Errorf("expected Fail, got %s: %s", result.Status, result.Message)
	}
}

func TestTFBackendConfigPass(t *testing.T) {
	snap := tfMockSnapshot()
	checker := &TFBackendConfigChecker{
		Type:   backendS3Type,
		Fields: map[string]string{backendBucketKey: backendBucket},
	}
	result := checker.Evaluate(snap)
	if result.Status != Pass {
		t.Errorf(passResultFmt, result.Status, result.Message)
	}
}

func TestTFResourceForbidPass(t *testing.T) {
	snap := tfMockSnapshot()
	checker := &TFResourceForbidChecker{Type: "aws_lambda_function"}
	result := checker.Evaluate(snap)
	if result.Status != Pass {
		t.Errorf("expected Pass (no forbidden resource), got %s: %s", result.Status, result.Message)
	}
}

func TestTFResourceForbidFail(t *testing.T) {
	snap := tfMockSnapshot()
	checker := &TFResourceForbidChecker{Type: "aws_s3_bucket"}
	result := checker.Evaluate(snap)
	if result.Status != Fail {
		t.Errorf("expected Fail (forbidden resource found), got %s: %s", result.Status, result.Message)
	}
}

// TestTFProviderVersionConstraintWrongVersionFails ensures that a required
// version that does not appear in the provider's declared version string causes
// a failure. This is a regression test for the bug where || p.Version != ""
// made the check always pass for any provider with any version.
func TestTFProviderVersionConstraintWrongVersionFails(t *testing.T) {
	snap := tfMockSnapshot() // provider hashicorp/aws version ">= 4.0"
	checker := &TFProviderReqChecker{Source: providerAWS, Version: ">= 5.0"}
	result := checker.Evaluate(snap)
	if result.Status != Fail {
		t.Errorf("expected Fail when version constraint does not match, got %s: %s", result.Status, result.Message)
	}
}

func TestTFProviderVersionConstraintMatchingVersionPasses(t *testing.T) {
	snap := tfMockSnapshot() // provider hashicorp/aws version ">= 4.0"
	checker := &TFProviderReqChecker{Source: providerAWS, Version: ">= 4.0"}
	result := checker.Evaluate(snap)
	if result.Status != Pass {
		t.Errorf("expected Pass when version constraint matches, got %s: %s", result.Status, result.Message)
	}
}

func TestTFRequiredTagsFail(t *testing.T) {
	snap := tfMockSnapshot()
	// Add a resource missing the required tag
	snap.TFModules[0].Resources = append(snap.TFModules[0].Resources, hcl.TFResource{
		Type:   "aws_ec2_instance",
		Name:   "bad_instance",
		Labels: map[string]string{}, // missing all tags
	})

	checker := &TFRequiredTagsChecker{Tags: []string{"Environment", "Team"}}
	result := checker.Evaluate(snap)
	if result.Status != Fail {
		t.Errorf("expected Fail (missing tags), got %s: %s", result.Status, result.Message)
	}
}

func TestTFVariableRequiredPass(t *testing.T) {
	snap := tfMockSnapshot() // has variable "environment"
	checker := &TFVariableReqChecker{Name: "environment"}
	result := checker.Evaluate(snap)
	if result.Status != Pass {
		t.Errorf(passResultFmt, result.Status, result.Message)
	}
}

func TestTFVariableRequiredFail(t *testing.T) {
	snap := tfMockSnapshot()
	checker := &TFVariableReqChecker{Name: "missing_var"}
	result := checker.Evaluate(snap)
	if result.Status != Fail {
		t.Errorf("expected Fail (variable absent), got %s: %s", result.Status, result.Message)
	}
}

func TestTFVariableRequiredTypeMismatch(t *testing.T) {
	snap := tfMockSnapshot() // environment has type "string"
	checker := &TFVariableReqChecker{Name: "environment", Type: "number"}
	result := checker.Evaluate(snap)
	if result.Status != Fail {
		t.Errorf("expected Fail (type mismatch), got %s: %s", result.Status, result.Message)
	}
}

func TestTFVariableRequiredTypeMatch(t *testing.T) {
	snap := tfMockSnapshot() // environment has type "string"
	checker := &TFVariableReqChecker{Name: "environment", Type: "string"}
	result := checker.Evaluate(snap)
	if result.Status != Pass {
		t.Errorf("expected Pass (type matches), got %s: %s", result.Status, result.Message)
	}
}
