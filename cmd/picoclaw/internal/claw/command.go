package claw

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/ResistanceIsUseless/picoclaw/cmd/picoclaw/internal"
	"github.com/ResistanceIsUseless/picoclaw/pkg/integration"
	"github.com/ResistanceIsUseless/picoclaw/pkg/providers"
)

func NewClawCommand() *cobra.Command {
	var pipelineFlag string
	var webUIFlag string

	cmd := &cobra.Command{
		Use:   "claw [pipeline] [target]",
		Short: "Run structured security assessment with CLAW orchestrator",
		Long: `Execute a structured security assessment pipeline using the CLAW orchestrator.

CLAW (Context-as-Artifacts, LLM-Advised Workflow) provides predefined assessment
pipelines with phase contracts, tool execution, and artifact management.

This is different from the default agent mode which provides flexible, LLM-driven
tool selection. Use CLAW when you want repeatable, structured assessments.`,
		Example: `  # Quick web reconnaissance
  picoclaw claw web_quick example.com

  # Full web security assessment
  picoclaw claw web_full example.com

  # Specify pipeline explicitly
  picoclaw claw --pipeline web_quick example.com

  # With Web UI
  picoclaw claw web_quick example.com --webui :8080`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runClaw(cmd.Context(), pipelineFlag, webUIFlag, args)
		},
	}

	cmd.Flags().StringVar(&pipelineFlag, "pipeline", "web_quick", "Pipeline to execute (web_quick, web_full)")
	cmd.Flags().StringVar(&webUIFlag, "webui", "", "Start Web UI on specified address (e.g., :8080)")

	return cmd
}

func runClaw(ctx context.Context, pipelineFlag string, webUIFlag string, args []string) error {
	// Parse arguments
	var pipeline, target string

	if len(args) == 1 {
		// Single arg: use pipeline flag
		pipeline = pipelineFlag
		target = args[0]
	} else {
		// Two+ args: first is pipeline, rest is target
		pipeline = args[0]
		target = args[1]
		if len(args) > 2 {
			// Join remaining args as target
			for i := 2; i < len(args); i++ {
				target += " " + args[i]
			}
		}
	}

	fmt.Printf("🦞 CLAW Mode: %s pipeline\n", pipeline)
	fmt.Printf("📍 Target: %s\n\n", target)

	// Load config
	cfg, err := internal.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get provider
	provider, _, err := providers.CreateProvider(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize provider: %w", err)
	}

	// Setup CLAW adapter
	persistenceDir := filepath.Join(os.ExpandEnv("$HOME"), ".picoclaw", "blackboard")
	if cfg.Agents.Defaults.CLAWMode != nil && cfg.Agents.Defaults.CLAWMode.PersistenceDir != "" {
		persistenceDir = os.ExpandEnv(cfg.Agents.Defaults.CLAWMode.PersistenceDir)
	}

	adapterCfg := &integration.CLAWConfig{
		Enabled:        true,
		Pipeline:       pipeline,
		PersistenceDir: persistenceDir,
	}

	clawAdapter, err := integration.NewCLAWAdapter(adapterCfg, provider)
	if err != nil {
		return fmt.Errorf("failed to create CLAW adapter: %w", err)
	}

	// Start Web UI if requested
	if webUIFlag != "" {
		fmt.Printf("🌐 Web UI: http://localhost%s\n\n", webUIFlag)
		// TODO: Integrate webui startup here
		// For now, just note it's not implemented
		fmt.Println("⚠️  Web UI integration not yet implemented in claw command")
	}

	// Execute assessment
	fmt.Println("🚀 Starting assessment...\n")
	response, err := clawAdapter.ProcessMessage(ctx, target)
	if err != nil {
		return fmt.Errorf("CLAW execution failed: %w", err)
	}

	// Print results
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Println("📊 Assessment Results")
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Println(response)
	fmt.Println()

	return nil
}
