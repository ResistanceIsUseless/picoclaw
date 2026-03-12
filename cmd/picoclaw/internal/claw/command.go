package claw

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ResistanceIsUseless/picoclaw/cmd/picoclaw/internal"
	"github.com/ResistanceIsUseless/picoclaw/pkg/tools"
)

func NewClawCommand() *cobra.Command {
	var pipelineFlag string
	var webUIFlag string
	var autoOpenWebUI bool

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
  picoclaw claw web_quick example.com --webui :8080 --open-webui`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runClaw(cmd.Context(), pipelineFlag, webUIFlag, autoOpenWebUI, args)
		},
	}

	cmd.Flags().StringVar(&pipelineFlag, "pipeline", "web_quick", "Pipeline to execute (web_quick, web_full)")
	cmd.Flags().StringVar(&webUIFlag, "webui", "", "Start Web UI on specified address (e.g., :8080)")
	cmd.Flags().BoolVar(&autoOpenWebUI, "open-webui", false, "Open the embedded web UI in your browser after startup")

	return cmd
}

func runClaw(ctx context.Context, pipelineFlag string, webUIFlag string, autoOpenWebUI bool, args []string) error {
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

	runtime, err := internal.BootstrapAgentRuntime("")
	if err != nil {
		return err
	}

	preflight, err := internal.PreflightCLAWPipeline(pipeline)
	if err != nil {
		return fmt.Errorf("failed to validate pipeline: %w", err)
	}
	if preflight.HasBlockingIssues() {
		summary := internal.BuildPreflightSummary("claw", extractMissingProfiles(preflight.MissingRequired), runtime.ProfileReadiness)
		return fmt.Errorf("%s\n%s", preflight.BlockingMessage(pipeline), summary.Guidance)
	}

	var sharedExecRegistry *tools.ToolRegistry
	defaultAgent := runtime.AgentLoop.GetRegistry().GetDefaultAgent()
	if defaultAgent != nil {
		sharedExecRegistry = defaultAgent.Tools
	}
	clawAdapter, err := internal.BuildCLAWAdapter(runtime.Config, runtime.Provider, sharedExecRegistry, pipeline)
	if err != nil {
		return fmt.Errorf("failed to create CLAW adapter: %w", err)
	}

	// Start Web UI if requested
	if webUIFlag != "" {
		url, _, err := internal.StartEmbeddedCLAWWebUI(webUIFlag, clawAdapter)
		if err != nil {
			return fmt.Errorf("failed to start web UI: %w", err)
		}
		fmt.Printf("🌐 Web UI: %s\n\n", url)
		if autoOpenWebUI {
			if err := internal.OpenBrowser(url); err != nil {
				fmt.Printf("⚠ Failed to open browser automatically: %v\n", err)
			}
		}
	}

	// Execute assessment
	fmt.Print("🚀 Starting assessment...\n\n")
	if len(preflight.MissingOptional) > 0 {
		fmt.Printf("⚠ Optional tools unavailable: %s\n\n", strings.Join(preflight.MissingOptional, ", "))
	}
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

func extractMissingProfiles(items []string) []string {
	missingProfiles := make([]string, 0)
	for _, item := range items {
		if strings.HasSuffix(item, " profile") {
			missingProfiles = append(missingProfiles, strings.TrimSuffix(item, " profile"))
		}
	}
	return missingProfiles
}
