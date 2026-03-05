// PicoClaw - Ultra-lightweight personal AI agent
// Inspired by and based on nanobot: https://github.com/HKUDS/nanobot
// License: MIT
//
// Copyright (c) 2026 PicoClaw contributors

package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ResistanceIsUseless/picoclaw/cmd/picoclaw/internal"
	"github.com/ResistanceIsUseless/picoclaw/cmd/picoclaw/internal/agent"
	"github.com/ResistanceIsUseless/picoclaw/cmd/picoclaw/internal/auth"
	"github.com/ResistanceIsUseless/picoclaw/cmd/picoclaw/internal/config"
	"github.com/ResistanceIsUseless/picoclaw/cmd/picoclaw/internal/cron"
	"github.com/ResistanceIsUseless/picoclaw/cmd/picoclaw/internal/gateway"
	"github.com/ResistanceIsUseless/picoclaw/cmd/picoclaw/internal/migrate"
	"github.com/ResistanceIsUseless/picoclaw/cmd/picoclaw/internal/onboard"
	"github.com/ResistanceIsUseless/picoclaw/cmd/picoclaw/internal/skills"
	"github.com/ResistanceIsUseless/picoclaw/cmd/picoclaw/internal/status"
	"github.com/ResistanceIsUseless/picoclaw/cmd/picoclaw/internal/version"
	pkgConfig "github.com/ResistanceIsUseless/picoclaw/pkg/config"
)

func NewPicoclawCommand() *cobra.Command {
	short := fmt.Sprintf("%s picoclaw - Personal AI Assistant v%s\n\n", internal.Logo, internal.GetVersion())

	cmd := &cobra.Command{
		Use:     "picoclaw",
		Short:   short,
		Example: "picoclaw list",
	}

	cmd.AddCommand(
		onboard.NewOnboardCommand(),
		agent.NewAgentCommand(),
		auth.NewAuthCommand(),
		config.NewConfigCommand(),
		gateway.NewGatewayCommand(),
		status.NewStatusCommand(),
		cron.NewCronCommand(),
		migrate.NewMigrateCommand(),
		skills.NewSkillsCommand(),
		version.NewVersionCommand(),
	)

	return cmd
}

func main() {
	// Check if first-run setup is needed
	if err := ensureConfigured(); err != nil {
		fmt.Fprintf(os.Stderr, "Setup failed: %v\n", err)
		os.Exit(1)
	}

	cmd := NewPicoclawCommand()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// ensureConfigured checks if config exists and is valid, runs wizard if needed
func ensureConfigured() error {
	cfgPath := internal.GetConfigPath()

	// Check if config file exists
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		// First run - trigger wizard
		return onboard.RunSetupWizard()
	}

	// Config exists - validate quality
	cfg, err := internal.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠ Config load error: %v\n", err)
		fmt.Fprintf(os.Stderr, "  Run 'picoclaw onboard' to recreate config\n\n")
		return nil // Don't block, allow other commands
	}

	// Validate and display warnings
	warnings := pkgConfig.ValidateConfigQuality(cfg)
	for _, w := range warnings {
		if w.Level == "error" {
			fmt.Fprintf(os.Stderr, "❌ %s\n", w.Message)
			fmt.Fprintf(os.Stderr, "\n")
			return onboard.RunSetupWizard()
		}
	}

	// Display non-fatal warnings
	warningText := pkgConfig.FormatWarnings(warnings)
	if warningText != "" {
		fmt.Print(warningText)
		fmt.Println()
	}

	return nil
}
