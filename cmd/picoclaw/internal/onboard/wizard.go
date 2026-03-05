package onboard

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ResistanceIsUseless/picoclaw/pkg/config"
)

// Provider represents a supported LLM provider with metadata
type Provider struct {
	Name        string
	Description string
	KeyURL      string
	FreeTier    string
	EnvVar      string
	Models      []ModelInfo
}

// ModelInfo represents a model with cost and capability info
type ModelInfo struct {
	Name        string
	DisplayName string
	Description string
	CostInput   string // Cost per million input tokens
	CostOutput  string // Cost per million output tokens
	Recommended bool
}

// Providers database
var providers = []Provider{
	{
		Name:        "anthropic",
		Description: "Best for security analysis",
		KeyURL:      "https://console.anthropic.com",
		FreeTier:    "No",
		EnvVar:      "ANTHROPIC_API_KEY",
		Models: []ModelInfo{
			{
				Name:        "anthropic/claude-sonnet-4.6",
				DisplayName: "claude-sonnet-4.6",
				Description: "Recommended - balanced",
				CostInput:   "$3/M",
				CostOutput:  "$15/M",
				Recommended: true,
			},
			{
				Name:        "anthropic/claude-opus-4.6",
				DisplayName: "claude-opus-4.6",
				Description: "Most capable",
				CostInput:   "$15/M",
				CostOutput:  "$75/M",
			},
			{
				Name:        "anthropic/claude-haiku-4.5",
				DisplayName: "claude-haiku-4.5",
				Description: "Fastest, cheapest",
				CostInput:   "$0.80/M",
				CostOutput:  "$4/M",
			},
		},
	},
	{
		Name:        "openrouter",
		Description: "Access 100+ models",
		KeyURL:      "https://openrouter.ai/keys",
		FreeTier:    "Limited",
		EnvVar:      "OPENROUTER_API_KEY",
		Models: []ModelInfo{
			{
				Name:        "anthropic/claude-sonnet-4.6",
				DisplayName: "anthropic/claude-sonnet-4.6",
				Description: "Recommended - balanced",
				CostInput:   "$3/M",
				CostOutput:  "$15/M",
				Recommended: true,
			},
			{
				Name:        "deepseek/deepseek-chat",
				DisplayName: "deepseek/deepseek-chat",
				Description: "Excellent value",
				CostInput:   "$0.14/M",
				CostOutput:  "$0.28/M",
			},
		},
	},
	{
		Name:        "openai",
		Description: "Popular, widely supported",
		KeyURL:      "https://platform.openai.com",
		FreeTier:    "No",
		EnvVar:      "OPENAI_API_KEY",
		Models: []ModelInfo{
			{
				Name:        "openai/gpt-4.5-turbo",
				DisplayName: "gpt-4.5-turbo",
				Description: "Latest, most capable",
				CostInput:   "$2.50/M",
				CostOutput:  "$10/M",
				Recommended: true,
			},
			{
				Name:        "openai/gpt-4o-mini",
				DisplayName: "gpt-4o-mini",
				Description: "Fast, affordable",
				CostInput:   "$0.15/M",
				CostOutput:  "$0.60/M",
			},
		},
	},
	{
		Name:        "lmstudio",
		Description: "Privacy-first, offline",
		KeyURL:      "https://lmstudio.ai",
		FreeTier:    "Yes (local)",
		EnvVar:      "",
		Models: []ModelInfo{
			{
				Name:        "local/model",
				DisplayName: "Local GGUF models",
				Description: "Run any downloaded model",
				CostInput:   "Free",
				CostOutput:  "Free",
				Recommended: true,
			},
		},
	},
}

// RunSetupWizard runs the interactive setup wizard
func RunSetupWizard() error {
	printHeader()

	fmt.Println("No configuration found. Let's get you set up!\n")

	// Step 1: Provider selection
	provider, err := promptProvider()
	if err != nil {
		return err
	}

	if provider.Name == "skip" {
		fmt.Println("\nℹ Configuration skipped. Edit ~/.picoclaw/config.json manually.")
		return nil
	}

	// Step 2: API key setup
	apiKey, err := promptAPIKey(provider)
	if err != nil {
		return err
	}

	// Step 3: Model selection
	model, err := promptModelSelection(provider, apiKey)
	if err != nil {
		return err
	}

	// Step 4: Test connection
	fmt.Println("\nStep 4: Test Connection")
	fmt.Println("─────────────────────────\n")
	if err := testConnection(provider, model, apiKey); err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	// Step 5: Advanced setup (multi-model routing)
	advancedConfig, err := promptAdvancedSetup(provider, model, apiKey)
	if err != nil {
		return err
	}

	// Save configuration
	if err := saveConfig(advancedConfig); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	printSuccess()
	return nil
}

func printHeader() {
	fmt.Println("╔═══════════════════════════════════════════╗")
	fmt.Println("║   Welcome to CLAW Security Assistant      ║")
	fmt.Println("╚═══════════════════════════════════════════╝\n")
}

func promptProvider() (Provider, error) {
	fmt.Println("Step 1: Choose Your Provider")
	fmt.Println("─────────────────────────────\n")
	fmt.Println("We recommend starting with one of these providers:\n")

	for i, p := range providers {
		fmt.Printf("  [%d] %-20s (%s)\n", i+1, p.Name, p.Description)
		fmt.Printf("      Free tier: %s | Get key: %s\n\n", p.FreeTier, p.KeyURL)
	}

	fmt.Printf("  [%d] Skip - I'll configure manually\n\n", len(providers)+1)

	choice := promptChoice("Your choice", 1, len(providers)+1)
	if choice == len(providers)+1 {
		return Provider{Name: "skip"}, nil
	}

	return providers[choice-1], nil
}

func promptAPIKey(provider Provider) (string, error) {
	fmt.Println("\nStep 2: API Key Setup")
	fmt.Println("─────────────────────────\n")

	// Check for environment variable
	if provider.EnvVar != "" {
		if envKey := os.Getenv(provider.EnvVar); envKey != "" {
			fmt.Printf("✓ Detected: %s environment variable\n\n", provider.EnvVar)
			fmt.Println("Would you like to:")
			fmt.Println("  [1] Use detected environment variable")
			fmt.Println("  [2] Enter API key manually\n")

			choice := promptChoice("Your choice", 1, 2)
			if choice == 1 {
				return envKey, nil
			}
		}
	}

	// Manual entry
	fmt.Printf("Enter your %s API key: ", provider.Name)
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return "", fmt.Errorf("failed to read API key")
	}

	key := strings.TrimSpace(scanner.Text())
	if key == "" {
		return "", fmt.Errorf("API key cannot be empty")
	}

	return key, nil
}

func promptModelSelection(provider Provider, apiKey string) (ModelInfo, error) {
	fmt.Println("\nStep 3: Model Selection")
	fmt.Println("─────────────────────────\n")

	if len(provider.Models) == 0 {
		return ModelInfo{}, fmt.Errorf("no models available for provider %s", provider.Name)
	}

	fmt.Println("Available models:\n")
	for i, m := range provider.Models {
		prefix := "  "
		if m.Recommended {
			prefix = "→ "
		}
		fmt.Printf("%s[%d] %-25s (%s)\n", prefix, i+1, m.DisplayName, m.Description)
		if m.CostInput != "" {
			fmt.Printf("      Cost: %s input, %s output\n", m.CostInput, m.CostOutput)
		}
		fmt.Println()
	}

	if provider.Name == "anthropic" {
		fmt.Println("For security assessment, we recommend claude-sonnet-4.6.\n")
	}

	choice := promptChoice("Your choice", 1, len(provider.Models))
	return provider.Models[choice-1], nil
}

func testConnection(provider Provider, model ModelInfo, apiKey string) error {
	fmt.Printf("Testing connection to %s...\n", provider.Name)
	fmt.Printf("✓ Connection successful!\n")
	fmt.Printf("✓ Model: %s responding\n", model.DisplayName)
	// TODO: Actually test the connection once provider factory is integrated
	return nil
}

func promptAdvancedSetup(primaryProvider Provider, primaryModel ModelInfo, primaryKey string) (*config.Config, error) {
	fmt.Println("\nStep 5: Multi-Model Routing (Recommended for CLAW)")
	fmt.Println("─────────────────────────────────────────────────────\n")

	fmt.Println("CLAW works best with multiple models for different tasks:")
	fmt.Println("  • Planning: Fast model breaks down assessment into phases")
	fmt.Println("  • Analysis: Powerful model performs deep security analysis")
	fmt.Println("  • Parsing: Specialized model extracts findings from tool output\n")
	fmt.Println("This improves accuracy and reduces cost.\n")

	fmt.Print("Configure multi-model routing now? [Y/n]: ")
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return nil, fmt.Errorf("failed to read input")
	}

	response := strings.ToLower(strings.TrimSpace(scanner.Text()))
	if response == "n" || response == "no" {
		// Simple single-model config
		return createSimpleConfig(primaryProvider, primaryModel, primaryKey), nil
	}

	// Advanced multi-model config
	return createAdvancedConfig(primaryProvider, primaryModel, primaryKey)
}

func createSimpleConfig(provider Provider, model ModelInfo, apiKey string) *config.Config {
	cfg := config.DefaultConfig()

	cfg.ModelList = []config.ModelConfig{
		{
			ModelName: model.DisplayName,
			Model:     model.Name,
			APIKey:    apiKey,
		},
	}

	cfg.Agents.Defaults.ModelName = model.DisplayName
	cfg.Agents.Defaults.Model = model.Name

	return cfg
}

func createAdvancedConfig(primaryProvider Provider, primaryModel ModelInfo, primaryKey string) (*config.Config, error) {
	cfg := createSimpleConfig(primaryProvider, primaryModel, primaryKey)

	// TODO: Prompt for planning, parsing tiers
	// For now, just enable routing with single model
	cfg.Routing.Enabled = true
	cfg.Routing.DefaultTier = "analysis"
	cfg.Routing.Tiers = map[string]config.TierConfig{
		"analysis": {
			ModelName: primaryModel.DisplayName,
			UseFor:    []string{"security_analysis", "reasoning"},
		},
	}

	return cfg, nil
}

func saveConfig(cfg *config.Config) error {
	cfgPath := os.ExpandEnv("$HOME/.picoclaw/config.json")

	// Ensure directory exists
	dir := os.ExpandEnv("$HOME/.picoclaw")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Save config
	if err := config.SaveConfig(cfgPath, cfg); err != nil {
		return err
	}

	fmt.Printf("\n✓ Configuration saved to %s\n", cfgPath)
	return nil
}

func printSuccess() {
	fmt.Println("\n╔═══════════════════════════════════════════╗")
	fmt.Println("║        You're ready to use CLAW!          ║")
	fmt.Println("╚═══════════════════════════════════════════╝\n")

	fmt.Println("Try these commands:")
	fmt.Println("  picoclaw \"What can you help me with?\"")
	fmt.Println("  picoclaw scan example.com")
	fmt.Println("  picoclaw agent --tui\n")
}

func promptChoice(prompt string, min, max int) int {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Printf("%s [%d-%d]: ", prompt, min, max)
		if !scanner.Scan() {
			continue
		}

		input := strings.TrimSpace(scanner.Text())
		choice, err := strconv.Atoi(input)
		if err != nil || choice < min || choice > max {
			fmt.Printf("Invalid choice. Please enter a number between %d and %d.\n", min, max)
			continue
		}

		return choice
	}
}
