package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ResistanceIsUseless/picoclaw/pkg/artifacts"
	"github.com/ResistanceIsUseless/picoclaw/pkg/blackboard"
	"github.com/ResistanceIsUseless/picoclaw/pkg/logger"
	"github.com/ResistanceIsUseless/picoclaw/pkg/orchestrator"
	anthropicprovider "github.com/ResistanceIsUseless/picoclaw/pkg/providers/anthropic"
	"github.com/ResistanceIsUseless/picoclaw/pkg/registry"
)

func main() {
	// Parse command line flags
	target := flag.String("target", "", "Target domain (e.g., careers.draftkings.com)")
	pipelineName := flag.String("pipeline", "web_quick", "Pipeline to use (web_quick or web_full)")
	apiKey := flag.String("api-key", os.Getenv("ANTHROPIC_API_KEY"), "Anthropic API key")
	persistDir := flag.String("persist-dir", filepath.Join(os.Getenv("HOME"), ".picoclaw-test", "blackboard"), "Blackboard persistence directory")
	dryRun := flag.Bool("dry-run", false, "Dry run - show what would be executed without calling LLM")
	flag.Parse()

	if *target == "" {
		fmt.Println("Usage: test-claw -target <domain>")
		fmt.Println("\nExample:")
		fmt.Println("  test-claw -target careers.draftkings.com")
		fmt.Println("  test-claw -target careers.draftkings.com -pipeline web_full")
		fmt.Println("  test-claw -target example.com -dry-run")
		fmt.Println("\nEnvironment variables:")
		fmt.Println("  ANTHROPIC_API_KEY - Required for LLM calls (unless -dry-run)")
		fmt.Println("  PATH - Must include ~/go/bin for security tools")
		os.Exit(1)
	}

	// Configure logger
	logger.SetLevel(logger.INFO)

	fmt.Printf("╔═══════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║           CLAW Real-World Integration Test               ║\n")
	fmt.Printf("╚═══════════════════════════════════════════════════════════╝\n")
	fmt.Printf("\n")
	fmt.Printf("Target:   %s\n", *target)
	fmt.Printf("Pipeline: %s\n", *pipelineName)
	fmt.Printf("Mode:     %s\n", map[bool]string{true: "DRY RUN (no LLM)", false: "LIVE (with LLM)"}[*dryRun])
	fmt.Printf("\n")

	// Verify tools are available
	fmt.Println("═══ Pre-flight Checks ═══")
	requiredTools := []string{"subfinder", "amass", "nmap", "httpx", "nuclei"}
	for _, tool := range requiredTools {
		if _, err := os.Stat(filepath.Join(os.Getenv("HOME"), "go", "bin", tool)); err == nil {
			fmt.Printf("✓ %s found\n", tool)
		} else if _, err := os.Stat(filepath.Join("/opt/homebrew/bin", tool)); err == nil {
			fmt.Printf("✓ %s found\n", tool)
		} else if _, err := os.Stat(filepath.Join("/usr/local/bin", tool)); err == nil {
			fmt.Printf("✓ %s found\n", tool)
		} else {
			fmt.Printf("⚠ %s not found (may cause issues)\n", tool)
		}
	}
	fmt.Println()

	// Setup CLAW components
	ctx := context.Background()

	// Create blackboard
	fmt.Println("═══ Initializing CLAW Components ═══")
	os.MkdirAll(*persistDir, 0755)
	persister, err := blackboard.NewFilePersister(*persistDir)
	if err != nil {
		fmt.Printf("✗ Failed to create persister: %v\n", err)
		os.Exit(1)
	}
	bb := blackboard.New(persister)
	fmt.Printf("✓ Blackboard initialized: %s\n", *persistDir)

	// Create tool registry and register security tools
	toolRegistry := registry.NewToolRegistry()
	if err := registry.RegisterSecurityTools(toolRegistry); err != nil {
		fmt.Printf("✗ Failed to register security tools: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Security tools registered: 5 tools\n")

	// Load pipeline
	pipeline, err := orchestrator.GetPredefinedPipeline(*pipelineName)
	if err != nil {
		fmt.Printf("✗ Failed to load pipeline %q: %v\n", *pipelineName, err)
		os.Exit(1)
	}
	fmt.Printf("✓ Pipeline loaded: %s (%d phases)\n", pipeline.Name, len(pipeline.Phases))

	// Create orchestrator
	orch := orchestrator.NewOrchestrator(pipeline, bb, toolRegistry)
	fmt.Printf("✓ Orchestrator created\n")

	// Setup provider (if not dry run)
	if !*dryRun {
		if *apiKey == "" {
			fmt.Println("\n✗ ANTHROPIC_API_KEY required for live mode")
			fmt.Println("  Set with: export ANTHROPIC_API_KEY=your-key")
			fmt.Println("  Or use: -dry-run flag")
			os.Exit(1)
		}
		provider := anthropicprovider.NewProvider(*apiKey)
		orch.SetProvider(provider)
		fmt.Printf("✓ LLM provider configured: Anthropic (Claude)\n")
	} else {
		fmt.Printf("⚠ Dry run mode - no LLM provider\n")
	}
	fmt.Println()

	// Publish initial target
	fmt.Println("═══ Starting CLAW Pipeline ═══")
	fmt.Printf("Publishing OperatorTarget artifact...\n")
	targetArtifact := artifacts.NewOperatorTarget(*target, "web", "recon")
	if err := bb.Publish(ctx, targetArtifact); err != nil {
		fmt.Printf("✗ Failed to publish target: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Target published to blackboard\n\n")

	if *dryRun {
		fmt.Println("═══ CLAW Methodology Overview ═══")
		fmt.Println()
		fmt.Println("Key Concepts:")
		fmt.Println("  • Phase Isolation: Each phase has isolated context (no prompt pollution)")
		fmt.Println("  • Contract-Driven: Explicit success criteria per phase")
		fmt.Println("  • Tool Security: 5-tier model (Tier 0 invisible, Tier 1 auto-approve)")
		fmt.Println("  • Knowledge Graph: Persistent discovery state across phases")
		fmt.Println("  • Frontier-Based: Intelligent property exploration")
		fmt.Println()

		fmt.Println("═══ Pipeline Execution Plan ═══")
		fmt.Printf("Pipeline: %s (%s domain)\n", pipeline.Name, pipeline.Domain)
		fmt.Printf("Total Phases: %d\n\n", len(pipeline.Phases))

		for i, phaseDef := range pipeline.Phases {
			fmt.Printf("┌─ Phase %d: %s\n", i+1, phaseDef.Name)
			fmt.Printf("│\n")
			fmt.Printf("│  Objective:\n")
			fmt.Printf("│    %s\n", phaseDef.Objective)
			fmt.Printf("│\n")

			// Show tool tier information
			fmt.Printf("│  Tools Available:\n")
			for _, toolName := range phaseDef.Tools {
				toolDef, err := toolRegistry.Get(toolName)
				if err != nil {
					fmt.Printf("│    - %s (unknown)\n", toolName)
					continue
				}
				tierName := map[registry.ToolTier]string{
					registry.TierOrchestrator: "Orchestrator",
					registry.TierHardwired:    "Hardwired (invisible to LLM)",
					registry.TierAutoApprove:  "Auto-approve",
					registry.TierHuman:        "Human approval",
					registry.TierBanned:       "Banned",
				}[toolDef.Tier]
				fmt.Printf("│    - %s [Tier %d: %s]\n", toolName, toolDef.Tier, tierName)
				fmt.Printf("│      → %s\n", toolDef.Description)
			}
			fmt.Printf("│\n")

			// Contract requirements
			fmt.Printf("│  Contract Requirements:\n")
			fmt.Printf("│    Required Tools: %v\n", phaseDef.RequiredTools)
			fmt.Printf("│    Required Artifacts: %v\n", phaseDef.RequiredArtifacts)
			fmt.Printf("│    Iteration Bounds: %d-%d\n", phaseDef.MinIterations, phaseDef.MaxIterations)
			fmt.Printf("│    Token Budget: %d tokens\n", phaseDef.TokenBudget)
			fmt.Printf("│\n")

			// Dependencies
			if len(phaseDef.DependsOn) > 0 {
				fmt.Printf("│  Dependencies:\n")
				fmt.Printf("│    Requires completion of: %v\n", phaseDef.DependsOn)
				fmt.Printf("│    (Receives input artifacts from previous phases)\n")
				fmt.Printf("│\n")
			}

			// Context building
			fmt.Printf("│  Context Building (per iteration):\n")
			fmt.Printf("│    ✓ System prompt + phase objective\n")
			fmt.Printf("│    ✓ Input artifacts from previous phases\n")
			fmt.Printf("│    ✓ Current knowledge graph state\n")
			fmt.Printf("│    ✓ Frontier (unknown properties to discover)\n")
			fmt.Printf("│    ✓ DAGState (tool execution tracking)\n")
			fmt.Printf("│    ✓ Contract status (progress toward completion)\n")
			fmt.Printf("│\n")

			// Execution flow
			fmt.Printf("│  Execution Flow:\n")
			fmt.Printf("│    1. Build context from graph/artifacts/frontier\n")
			fmt.Printf("│    2. Call LLM with tool definitions\n")
			fmt.Printf("│    3. Execute requested tools\n")
			fmt.Printf("│       → Parse output to typed artifacts\n")
			fmt.Printf("│       → Publish artifacts to blackboard\n")
			fmt.Printf("│       → Extract graph mutations (nodes/edges)\n")
			fmt.Printf("│       → Update knowledge graph\n")
			fmt.Printf("│    4. Update DAGState with tool status\n")
			fmt.Printf("│    5. Check contract satisfaction\n")
			fmt.Printf("│    6. Repeat until contract satisfied or max iterations\n")
			fmt.Printf("│\n")

			// What LLM sees
			if i == 0 {
				fmt.Printf("│  What LLM Sees (Phase 1):\n")
				fmt.Printf("│    • Target: %s (from OperatorTarget artifact)\n", *target)
				fmt.Printf("│    • Available tools: ")
				visibleTools := []string{}
				for _, toolName := range phaseDef.Tools {
					toolDef, _ := toolRegistry.Get(toolName)
					if toolDef != nil && toolDef.Tier != registry.TierHardwired {
						visibleTools = append(visibleTools, toolName)
					}
				}
				if len(visibleTools) > 0 {
					fmt.Printf("%v\n", visibleTools)
				} else {
					fmt.Printf("(none - Tier 0 tools are invisible)\n")
				}
				fmt.Printf("│    • Empty knowledge graph (nothing discovered yet)\n")
			} else {
				fmt.Printf("│  What LLM Sees (Phase %d):\n", i+1)
				fmt.Printf("│    • All artifacts from previous phases\n")
				fmt.Printf("│    • Updated knowledge graph with discovered entities\n")
				fmt.Printf("│    • Frontier of unknown properties to explore\n")
				fmt.Printf("│    • Available tools: ")
				visibleTools := []string{}
				for _, toolName := range phaseDef.Tools {
					toolDef, _ := toolRegistry.Get(toolName)
					if toolDef != nil && toolDef.Tier != registry.TierHardwired {
						visibleTools = append(visibleTools, toolName)
					}
				}
				fmt.Printf("%v\n", visibleTools)
			}

			if i < len(pipeline.Phases)-1 {
				fmt.Printf("└─ ▼\n\n")
			} else {
				fmt.Printf("└─ [END]\n\n")
			}
		}

		fmt.Println("═══ Expected Outputs ═══")
		fmt.Println()
		fmt.Printf("Artifacts: ~/.picoclaw-test/blackboard/\n")
		fmt.Printf("  • OperatorTarget (initial target spec)\n")
		fmt.Printf("  • SubdomainList (from recon phase)\n")
		fmt.Printf("  • ServiceFingerprint (from scan phase)\n")
		fmt.Printf("  • VulnerabilityList (if vulnerabilities found)\n")
		fmt.Println()
		fmt.Printf("Knowledge Graph:\n")
		fmt.Printf("  Nodes: domain, subdomain, IP, service, endpoint\n")
		fmt.Printf("  Edges: subdomain_of, resolves_to, hosts_service, etc.\n")
		fmt.Printf("  Properties: known (discovered) vs unknown (frontier)\n")
		fmt.Println()

		fmt.Println("═══ Key Features Demonstrated ═══")
		fmt.Println()
		fmt.Println("✓ Phase Isolation")
		fmt.Println("  Each phase has clean context - no prompt pollution from previous phases")
		fmt.Println("  Phase 1 sees only target, Phase 2 sees Phase 1 artifacts + graph")
		fmt.Println()
		fmt.Println("✓ Contract Validation")
		fmt.Println("  Phase cannot complete without required tools executing")
		fmt.Println("  Must produce required artifact types")
		fmt.Println("  Success criteria explicitly validated")
		fmt.Println()
		fmt.Println("✓ Tool Security Tiers")
		fmt.Println("  Tier 0 (Hardwired): Invisible to LLM, always executed")
		fmt.Println("  Tier 1 (AutoApprove): LLM can see and call without operator approval")
		fmt.Println("  Tier 2+ require escalating levels of human confirmation")
		fmt.Println()
		fmt.Println("✓ Knowledge Graph")
		fmt.Println("  Persistent state of discovered entities and relationships")
		fmt.Println("  Enables frontier-based exploration (what's unknown?)")
		fmt.Println("  Shared across all phases in pipeline")
		fmt.Println()
		fmt.Println("✓ Structured Artifacts")
		fmt.Println("  All tool outputs parsed into typed artifacts")
		fmt.Println("  Queryable by phase, type, metadata")
		fmt.Println("  Enables reproducibility and auditing")
		fmt.Println()

		fmt.Println("✓ Dry run complete - use without -dry-run to execute with Claude")
		fmt.Println()
		fmt.Printf("To run live: export ANTHROPIC_API_KEY=sk-... && ./build/test-claw -target %s\n", *target)
		os.Exit(0)
	}

	// Execute pipeline
	startTime := time.Now()
	fmt.Printf("Executing %s pipeline...\n", pipeline.Name)
	fmt.Println("(This may take several minutes depending on tools and target)")
	fmt.Println()

	if err := orch.Execute(ctx); err != nil {
		fmt.Printf("\n✗ Pipeline execution failed: %v\n", err)
		os.Exit(1)
	}

	duration := time.Since(startTime)
	fmt.Println()
	fmt.Println("═══ Pipeline Execution Complete ═══")
	fmt.Printf("Duration: %s\n", duration.Round(time.Second))
	fmt.Println()

	// Display results
	fmt.Println("═══ Results Summary ═══")

	// Completed phases
	completedPhases := orch.GetCompletedPhases()
	fmt.Printf("Phases completed: %d/%d\n", len(completedPhases), len(pipeline.Phases))
	for i, phase := range completedPhases {
		fmt.Printf("  %d. %s ✓\n", i+1, phase)
	}
	fmt.Println()

	// Artifacts by phase
	fmt.Println("Artifacts produced:")
	for _, phaseName := range completedPhases {
		artifacts, err := bb.GetByPhase(phaseName)
		if err != nil {
			continue
		}
		if len(artifacts) > 0 {
			fmt.Printf("  %s: %d artifacts\n", phaseName, len(artifacts))
			typeCount := make(map[string]int)
			for _, artifact := range artifacts {
				typeCount[artifact.Metadata.Type]++
			}
			for artifactType, count := range typeCount {
				fmt.Printf("    - %s: %d\n", artifactType, count)
			}
		}
	}
	fmt.Println()

	// Knowledge graph stats
	kg := orch.GetGraph()
	if kg != nil {
		fmt.Println("Knowledge Graph:")
		fmt.Printf("  Nodes: %d\n", kg.NodeCount())
		fmt.Printf("  Edges: %d\n", kg.EdgeCount())

		// Show some discovered entities
		if kg.NodeCount() > 0 {
			fmt.Println("  Sample entities discovered:")
			// Show a few subdomain nodes as examples
			shown := 0
			for _, phaseName := range completedPhases {
				artifacts, _ := bb.GetByPhase(phaseName)
				for _, artifact := range artifacts {
					if artifact.Metadata.Type == "SubdomainList" && shown < 3 {
						fmt.Printf("    - Subdomains found via %s phase\n", phaseName)
						shown++
					}
				}
			}
		}
	}
	fmt.Println()

	// Output location
	fmt.Println("═══ Data Location ═══")
	fmt.Printf("Blackboard artifacts: %s\n", *persistDir)
	fmt.Println()

	fmt.Println("✓ CLAW integration test complete!")
}
