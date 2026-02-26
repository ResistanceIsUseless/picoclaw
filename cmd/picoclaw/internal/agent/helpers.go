package agent

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/chzyer/readline"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	"github.com/sipeed/picoclaw/pkg/agent"
	"github.com/sipeed/picoclaw/pkg/bus"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/providers"
	"github.com/sipeed/picoclaw/pkg/tui"
)

func agentCmd(message, sessionKey, model string, debug, useTUI bool, workflowName, target string) error {
	if sessionKey == "" {
		sessionKey = "cli:default"
	}

	if debug {
		logger.SetLevel(logger.DEBUG)
		fmt.Println("🔍 Debug mode enabled")
	}

	// Validate workflow flags
	if workflowName != "" && target == "" {
		return fmt.Errorf("--target is required when using --workflow")
	}

	cfg, err := internal.LoadConfig()
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	if model != "" {
		cfg.Agents.Defaults.ModelName = model
	}

	provider, modelID, err := providers.CreateProvider(cfg)
	if err != nil {
		return fmt.Errorf("error creating provider: %w", err)
	}

	// Use the resolved model ID from provider creation
	if modelID != "" {
		cfg.Agents.Defaults.ModelName = modelID
	}

	msgBus := bus.NewMessageBus()
	agentLoop := agent.NewAgentLoop(cfg, msgBus, provider)

	// Load workflow if specified
	if workflowName != "" {
		defaultAgent := agentLoop.GetRegistry().GetDefaultAgent()
		if defaultAgent == nil {
			return fmt.Errorf("failed to get default agent for workflow loading")
		}

		err := defaultAgent.LoadWorkflow(workflowName, target)
		if err != nil {
			return fmt.Errorf("failed to load workflow '%s': %w", workflowName, err)
		}

		logger.InfoCF("agent", "Workflow loaded", map[string]any{
			"workflow": workflowName,
			"target":   target,
		})
		fmt.Printf("📋 Loaded workflow: %s (target: %s)\n", workflowName, target)
	}

	// Print agent startup info (only for interactive mode)
	startupInfo := agentLoop.GetStartupInfo()
	logger.InfoCF("agent", "Agent initialized",
		map[string]any{
			"tools_count":      startupInfo["tools"].(map[string]any)["count"],
			"skills_total":     startupInfo["skills"].(map[string]any)["total"],
			"skills_available": startupInfo["skills"].(map[string]any)["available"],
		})

	if message != "" {
		// Single message mode (non-interactive)
		ctx := context.Background()
		response, err := agentLoop.ProcessDirect(ctx, message, sessionKey)
		if err != nil {
			return fmt.Errorf("error processing message: %w", err)
		}
		fmt.Printf("\n%s %s\n", internal.Logo, response)
		return nil
	}

	// Interactive mode
	if useTUI {
		// TUI mode
		return tuiMode(agentLoop, sessionKey)
	}

	// Traditional readline mode
	fmt.Printf("%s Interactive mode (Ctrl+C to exit)\n\n", internal.Logo)
	interactiveMode(agentLoop, sessionKey)

	return nil
}

func interactiveMode(agentLoop *agent.AgentLoop, sessionKey string) {
	prompt := fmt.Sprintf("%s You: ", internal.Logo)

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          prompt,
		HistoryFile:     filepath.Join(os.TempDir(), ".picoclaw_history"),
		HistoryLimit:    100,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		fmt.Printf("Error initializing readline: %v\n", err)
		fmt.Println("Falling back to simple input mode...")
		simpleInteractiveMode(agentLoop, sessionKey)
		return
	}
	defer rl.Close()

	for {
		line, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt || err == io.EOF {
				fmt.Println("\nGoodbye!")
				return
			}
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}

		if input == "exit" || input == "quit" {
			fmt.Println("Goodbye!")
			return
		}

		ctx := context.Background()
		response, err := agentLoop.ProcessDirect(ctx, input, sessionKey)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Printf("\n%s %s\n\n", internal.Logo, response)
	}
}

func simpleInteractiveMode(agentLoop *agent.AgentLoop, sessionKey string) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print(fmt.Sprintf("%s You: ", internal.Logo))
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				fmt.Println("\nGoodbye!")
				return
			}
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}

		input := strings.TrimSpace(line)
		if input == "" {
			continue
		}

		if input == "exit" || input == "quit" {
			fmt.Println("Goodbye!")
			return
		}

		ctx := context.Background()
		response, err := agentLoop.ProcessDirect(ctx, input, sessionKey)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Printf("\n%s %s\n\n", internal.Logo, response)
	}
}

func tuiMode(agentLoop *agent.AgentLoop, sessionKey string) error {
	// Create TUI program
	program := tui.NewProgram()

	// Set up input handler with closure
	var programRef *tui.Program = program
	handler := func(input string) {
		// Send user message to chat
		programRef.Send(tui.SendChatMessage("user", input, ""))

		// Process with agent
		ctx := context.Background()
		response, err := agentLoop.ProcessDirect(ctx, input, sessionKey)
		if err != nil {
			programRef.Send(tui.SendChatMessage("system", fmt.Sprintf("Error: %v", err), ""))
			return
		}

		// Send assistant response
		programRef.Send(tui.SendChatMessage("assistant", response, ""))
	}

	// Set the handler
	program.SetInputHandler(handler)

	// Set up workflow engine if loaded
	defaultAgent := agentLoop.GetRegistry().GetDefaultAgent()
	if defaultAgent != nil && defaultAgent.WorkflowEngine != nil {
		program.SetWorkflowEngine(defaultAgent.WorkflowEngine)
	}

	// Set up tier router if enabled
	if tierRouter := agentLoop.GetTierRouter(); tierRouter != nil {
		program.SetTierRouter(tierRouter)
	}

	// Run TUI
	return program.Run()
}
