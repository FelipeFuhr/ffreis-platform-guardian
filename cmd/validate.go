package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ffreis/platform-guardian/internal/rule"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate rule YAML files",
	RunE:  runValidate,
}

var validateRules []string

func init() {
	validateCmd.Flags().StringSliceVar(&validateRules, "rules", nil, "Rule directories or files (required)")
	_ = validateCmd.MarkFlagRequired("rules")
}

func runValidate(cmd *cobra.Command, args []string) error {
	out := newCommandOutput(cmd, getPresenter(cmd))
	registry, loadErr := rule.Load(validateRules)

	hasErrors := false

	if loadErr != nil {
		out.ErrStatus("error", "fail", "load error: "+loadErr.Error())
		hasErrors = true
	}

	if errs := rule.Validate(registry); len(errs) > 0 {
		for _, e := range errs {
			out.ErrStatus("error", "fail", "validation error: "+e.Error())
		}
		hasErrors = true
	}

	if hasErrors {
		return fmt.Errorf("validation failed")
	}

	out.Status("ok", "ok", "all rules are valid")
	return nil
}
