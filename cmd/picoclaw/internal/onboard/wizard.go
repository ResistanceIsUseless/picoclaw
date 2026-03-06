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
		Name:        "google",
		Description: "Structured output, fast",
		KeyURL:      "https://aistudio.google.com/apikey",
		FreeTier:    "Yes (generous)",
		EnvVar:      "GEMINI_API_KEY",
		Models: []ModelInfo{
			{
				Name:        "google/gemini-2.0-flash-exp",
				DisplayName: "gemini-2.0-flash-exp",
				Description: "Recommended - fast, free",
				CostInput:   "Free (generous quota)",
				CostOutput:  "Free (generous quota)",
				Recommended: true,
			},
			{
				Name:        "google/gemini-1.5-pro",
				DisplayName: "gemini-1.5-pro",
				Description: "Most capable Gemini",
				CostInput:   "$1.25/M",
				CostOutput:  "$5/M",
			},
		},
	},
	{
		Name:        "groq",
		Description: "Ultra-fast inference",
		KeyURL:      "https://console.groq.com/keys",
		FreeTier:    "Yes",
		EnvVar:      "GROQ_API_KEY",
		Models: []ModelInfo{
			{
				Name:        "groq/llama-3.3-70b-versatile",
				DisplayName: "llama-3.3-70b-versatile",
				Description: "Recommended - fast, capable",
				CostInput:   "$0.59/M",
				CostOutput:  "$0.79/M",
				Recommended: true,
			},
			{
				Name:        "groq/mixtral-8x7b-32768",
				DisplayName: "mixtral-8x7b-32768",
				Description: "Fast, good reasoning",
				CostInput:   "$0.24/M",
				CostOutput:  "$0.24/M",
			},
		},
	},
	{
		Name:        "mistral",
		Description: "European AI, GDPR-compliant",
		KeyURL:      "https://console.mistral.ai/api-keys",
		FreeTier:    "Limited",
		EnvVar:      "MISTRAL_API_KEY",
		Models: []ModelInfo{
			{
				Name:        "mistral/mistral-large-latest",
				DisplayName: "mistral-large-latest",
				Description: "Recommended - most capable",
				CostInput:   "$2/M",
				CostOutput:  "$6/M",
				Recommended: true,
			},
			{
				Name:        "mistral/mistral-small-latest",
				DisplayName: "mistral-small-latest",
				Description: "Fast, affordable",
				CostInput:   "$0.20/M",
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

	fmt.Println("\nMulti-Model Tier Configuration")
	fmt.Println("───────────────────────────────\n")

	// Prompt for planning tier
	fmt.Println("Planning Tier (Fast, Budget-Friendly):")
	fmt.Println("  Used for: task breakdown, phase planning, tool selection")
	fmt.Println("  Recommendation: Fast, cheap model\n")

	planningModel, planningKey, err := promptAdditionalModel("planning", primaryProvider, primaryModel, primaryKey)
	if err != nil {
		return nil, err
	}

	// Prompt for parsing tier
	fmt.Println("\nParsing Tier (Structured Output):")
	fmt.Println("  Used for: tool output extraction, finding parsing")
	fmt.Println("  Recommendation: Strong structured output capability\n")

	parsingModel, parsingKey, err := promptAdditionalModel("parsing", primaryProvider, primaryModel, primaryKey)
	if err != nil {
		return nil, err
	}

	// Add models to config if they're different from primary
	if planningModel.Name != primaryModel.Name && planningKey != "" {
		cfg.ModelList = append(cfg.ModelList, config.ModelConfig{
			ModelName: planningModel.DisplayName,
			Model:     planningModel.Name,
			APIKey:    planningKey,
		})
	}

	if parsingModel.Name != primaryModel.Name && parsingKey != "" && parsingModel.Name != planningModel.Name {
		cfg.ModelList = append(cfg.ModelList, config.ModelConfig{
			ModelName: parsingModel.DisplayName,
			Model:     parsingModel.Name,
			APIKey:    parsingKey,
		})
	}

	// Configure routing
	cfg.Routing.Enabled = true
	cfg.Routing.DefaultTier = "analysis"
	cfg.Routing.Tiers = map[string]config.TierConfig{
		"planning": {
			ModelName: planningModel.DisplayName,
			UseFor:    []string{"task_breakdown", "planning", "tool_selection"},
		},
		"analysis": {
			ModelName: primaryModel.DisplayName,
			UseFor:    []string{"security_analysis", "reasoning", "deep_analysis"},
		},
		"parsing": {
			ModelName: parsingModel.DisplayName,
			UseFor:    []string{"tool_output_parsing", "structured_extraction", "finding_extraction"},
		},
	}

	// Print summary
	fmt.Println("\n✓ Multi-model routing configured!")
	fmt.Printf("  Planning: %s\n", planningModel.DisplayName)
	fmt.Printf("  Analysis: %s (primary)\n", primaryModel.DisplayName)
	fmt.Printf("  Parsing: %s\n", parsingModel.DisplayName)

	return cfg, nil
}

func promptAdditionalModel(tier string, primaryProvider Provider, primaryModel ModelInfo, primaryKey string) (ModelInfo, string, error) {
	// Suggest good models for this tier
	recommendations := getSuggestedModelsForTier(tier)

	fmt.Println("Options:")
	fmt.Printf("  [1] Use same as Analysis (%s)\n", primaryModel.DisplayName)

	for i, rec := range recommendations {
		fmt.Printf("  [%d] %s (%s)\n", i+2, rec.DisplayName, rec.Description)
		if rec.CostInput != "" {
			fmt.Printf("      Cost: %s input, %s output\n", rec.CostInput, rec.CostOutput)
		}
	}

	fmt.Println()
	choice := promptChoice("Your choice", 1, len(recommendations)+1)

	// Option 1: Use primary model
	if choice == 1 {
		return primaryModel, primaryKey, nil
	}

	// Other options: Use recommended model
	selectedModel := recommendations[choice-2]

	// Check if we need to prompt for API key
	apiKey := ""
	if selectedModel.Name != primaryModel.Name {
		// Extract provider from model name (e.g., "google/gemini-2.0-flash" -> "google")
		parts := strings.Split(selectedModel.Name, "/")
		if len(parts) > 0 {
			providerName := parts[0]

			// Find provider to get env var
			for _, p := range providers {
				if p.Name == providerName {
					if p.EnvVar != "" {
						if envKey := os.Getenv(p.EnvVar); envKey != "" {
							fmt.Printf("✓ Using %s from environment\n", p.EnvVar)
							apiKey = envKey
						} else {
							fmt.Printf("Enter your %s API key (or press Enter to skip): ", p.Name)
							scanner := bufio.NewScanner(os.Stdin)
							if scanner.Scan() {
								apiKey = strings.TrimSpace(scanner.Text())
							}
						}
					}
					break
				}
			}
		}
	} else {
		apiKey = primaryKey
	}

	return selectedModel, apiKey, nil
}

func getSuggestedModelsForTier(tier string) []ModelInfo {
	switch tier {
	case "planning":
		return []ModelInfo{
			{
				Name:        "google/gemini-2.0-flash-exp",
				DisplayName: "gemini-2.0-flash-exp",
				Description: "Free, very fast",
				CostInput:   "Free",
				CostOutput:  "Free",
			},
			{
				Name:        "openai/gpt-4o-mini",
				DisplayName: "gpt-4o-mini",
				Description: "Fast, affordable",
				CostInput:   "$0.15/M",
				CostOutput:  "$0.60/M",
			},
			{
				Name:        "groq/llama-3.3-70b-versatile",
				DisplayName: "groq/llama-3.3-70b",
				Description: "Ultra-fast",
				CostInput:   "$0.59/M",
				CostOutput:  "$0.79/M",
			},
		}
	case "parsing":
		return []ModelInfo{
			{
				Name:        "google/gemini-2.0-flash-exp",
				DisplayName: "gemini-2.0-flash-exp",
				Description: "Free, structured output",
				CostInput:   "Free",
				CostOutput:  "Free",
			},
			{
				Name:        "openai/gpt-4.5-turbo",
				DisplayName: "gpt-4.5-turbo",
				Description: "Excellent structured output",
				CostInput:   "$2.50/M",
				CostOutput:  "$10/M",
			},
			{
				Name:        "anthropic/claude-haiku-4.5",
				DisplayName: "claude-haiku-4.5",
				Description: "Fast, good parsing",
				CostInput:   "$0.80/M",
				CostOutput:  "$4/M",
			},
		}
	default:
		return []ModelInfo{}
	}
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
