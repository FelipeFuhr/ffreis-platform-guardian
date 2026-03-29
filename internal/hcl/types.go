package hcl

type TFModule struct {
	Path      string // file path relative to repo root
	Providers []TFProvider
	Resources []TFResource
	Modules   []TFModuleCall
	Backend   *TFBackend
	Variables []TFVariable
}

type TFProvider struct {
	Source  string // e.g. "hashicorp/aws"
	Version string // e.g. ">= 4.0"
}

type TFResource struct {
	Type   string // e.g. "aws_s3_bucket"
	Name   string
	Labels map[string]string // parsed from `tags` or `labels` attribute
}

type TFModuleCall struct {
	Name   string
	Source string
}

type TFBackend struct {
	Type   string
	Config map[string]string
}

type TFVariable struct {
	Name string
	Type string
}
