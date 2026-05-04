package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	platformui "github.com/ffreis/platform-guardian/internal/ui"
)

type contextKey string

const loggerKey contextKey = "logger"
const presenterKey contextKey = "presenter"

var logLevel string
var uiMode string

var (
	version   string
	commit    string
	buildTime string
)

const (
	exitOK    = 0
	exitError = 1
)

type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e == nil || e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e *ExitError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

var rootCmd = &cobra.Command{
	Use:   "platform-guardian",
	Short: "Platform Guardian — enforce repository standards across your org",
	Long: `platform-guardian checks repositories against configurable rules,
enforcing structure, content, Terraform, and policy standards.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() int {
	return executeCommand(rootCmd, os.Stderr)
}

func executeCommand(cmd *cobra.Command, stderr io.Writer) int {
	if err := cmd.Execute(); err != nil {
		if message := err.Error(); message != "" {
			_, _ = io.WriteString(stderr, "error: "+message+"\n")
		}
		return exitCodeForError(err)
	}
	return exitOK
}

func exitCodeForError(err error) int {
	var exitErr *ExitError
	if errors.As(err, &exitErr) && exitErr != nil && exitErr.Code != 0 {
		return exitErr.Code
	}
	return exitError
}

func init() {
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().StringVar(&uiMode, "ui", "auto", "UI mode: auto, plain, rich")
	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		presenter, err := platformui.New(uiMode)
		if err != nil {
			return fmt.Errorf("building ui: %w", err)
		}
		logger, err := buildLogger(logLevel)
		if err != nil {
			return fmt.Errorf("building logger: %w", err)
		}
		ctx := context.WithValue(cmd.Context(), loggerKey, logger)
		ctx = context.WithValue(ctx, presenterKey, presenter)
		cmd.SetContext(ctx)
		return nil
	}

	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(scanOrgCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(versionCmd)
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

func getPresenter(cmd *cobra.Command) *platformui.Presenter {
	if cmd.Context() != nil {
		if presenter, ok := cmd.Context().Value(presenterKey).(*platformui.Presenter); ok {
			return presenter
		}
	}
	presenter, _ := platformui.New(platformui.ModePlain)
	return presenter
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print build information",
	RunE: func(cmd *cobra.Command, _ []string) error {
		v := strings.TrimSpace(version)
		if v == "" {
			v = "dev"
		}
		c := strings.TrimSpace(commit)
		if c == "" {
			c = "unknown"
		}
		t := strings.TrimSpace(buildTime)
		if t == "" {
			t = "unknown"
		}

		newCommandOutput(cmd, getPresenter(cmd)).Line(v + " (commit=" + c + " built=" + t + ")")
		return nil
	},
}
