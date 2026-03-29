package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type contextKey string

const loggerKey contextKey = "logger"

var logLevel string

var rootCmd = &cobra.Command{
	Use:   "platform-guardian",
	Short: "Platform Guardian — enforce repository standards across your org",
	Long: `platform-guardian checks repositories against configurable rules,
enforcing structure, content, Terraform, and policy standards.`,
	SilenceUsage: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		logger, err := buildLogger(logLevel)
		if err != nil {
			return fmt.Errorf("building logger: %w", err)
		}
		ctx := context.WithValue(cmd.Context(), loggerKey, logger)
		cmd.SetContext(ctx)
		return nil
	}

	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(scanOrgCmd)
	rootCmd.AddCommand(validateCmd)
}

func buildLogger(level string) (*zap.Logger, error) {
	var lvl zapcore.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		return nil, fmt.Errorf("invalid log level %q: %w", level, err)
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = zap.NewAtomicLevelAt(lvl)
	return cfg.Build()
}

func getLogger(cmd *cobra.Command) *zap.Logger {
	if cmd.Context() != nil {
		if logger, ok := cmd.Context().Value(loggerKey).(*zap.Logger); ok {
			return logger
		}
	}
	logger, _ := zap.NewProduction()
	return logger
}
