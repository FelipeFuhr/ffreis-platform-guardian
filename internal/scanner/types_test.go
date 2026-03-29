package scanner

import "testing"

func TestScanners_Type(t *testing.T) {
	t.Parallel()

	snap := NewSnapshot("org/repo")

	if NewStructureScanner(snap).Type() != ScannerTypeStructure {
		t.Fatalf("unexpected type for structure scanner")
	}
	if NewContentScanner(snap).Type() != ScannerTypeContent {
		t.Fatalf("unexpected type for content scanner")
	}
	if NewTerraformScanner(snap).Type() != ScannerTypeTerraform {
		t.Fatalf("unexpected type for terraform scanner")
	}
	if NewPolicyScanner(snap).Type() != ScannerTypePolicy {
		t.Fatalf("unexpected type for policy scanner")
	}
}
