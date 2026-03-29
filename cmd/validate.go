package cmd

import (
	"fmt"
	"os"

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
	registry, loadErr := rule.Load(validateRules)

	hasErrors := false

	if loadErr != nil {
		fmt.Fprintf(os.Stderr, "load error: %v\n", loadErr)
		hasErrors = true
	}

	if errs := rule.Validate(registry); len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "validation error: %v\n", e)
		}
		hasErrors = true
	}

	if hasErrors {
		return fmt.Errorf("validation failed")
	}

	fmt.Println("All rules are valid.")
	return nil
}
