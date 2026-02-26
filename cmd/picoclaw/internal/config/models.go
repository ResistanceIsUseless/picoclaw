package config

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	pkgconfig "github.com/sipeed/picoclaw/pkg/config"
)

func newModelsCommand() *cobra.Command {
	var showRouting bool

	cmd := &cobra.Command{
		Use:   "models",
		Short: "List configured models and their routing assignments",
		Long: `List all configured models showing their provider, routing tier, and enabled status.

Examples:
  picoclaw config models              # List all models
  picoclaw config models --routing    # Show routing tier assignments`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return modelsCmd(showRouting)
		},
	}

	cmd.Flags().BoolVar(&showRouting, "routing", false, "Show tier routing assignments")

	return cmd
}

func modelsCmd(showRouting bool) error {
	cfg, err := internal.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if len(cfg.ModelList) == 0 {
		return fmt.Errorf("no models configured in config.json")
	}

	fmt.Println("📋 Configured Models\n")

	// Group models by provider
	providerModels := make(map[string][]pkgconfig.ModelConfig)
	for _, m := range cfg.ModelList {
		provider := detectProvider(m)
		providerModels[provider] = append(providerModels[provider], m)
	}

	// Sort provider names
	providers := make([]string, 0, len(providerModels))
	for provider := range providerModels {
		providers = append(providers, provider)
	}
	sort.Strings(providers)

	// Display by provider
	for _, provider := range providers {
		models := providerModels[provider]
		fmt.Printf("🔌 %s (%d model%s)\n", provider, len(models), plural(len(models)))

		for _, m := range models {
			// Check if this model is used in routing
			tier := ""
			if showRouting && cfg.Routing.Enabled {
				tier = getModelTier(cfg, m.ModelName)
			}

			// Format output
			status := "✓"
			modelLine := fmt.Sprintf("  %s %s", status, m.ModelName)

			if m.APIBase != "" {
				modelLine += fmt.Sprintf(" → %s", m.APIBase)
			} else {
				modelLine += fmt.Sprintf(" (%s)", m.Model)
			}

			if tier != "" {
				modelLine += fmt.Sprintf(" [%s tier]", tier)
			}

			fmt.Println(modelLine)
		}
		fmt.Println()
	}

	// Show routing summary if requested
	if showRouting && cfg.Routing.Enabled {
		fmt.Println("🎯 Routing Configuration\n")
		fmt.Printf("  Default Tier: %s\n", cfg.Routing.DefaultTier)
		fmt.Println()

		// Show tier assignments
		tierNames := []string{"heavy", "medium", "light"}
		for _, tierName := range tierNames {
			tier, ok := getTierConfig(cfg, tierName)
			if ok {
				fmt.Printf("  %s tier:\n", strings.ToUpper(tierName))
				fmt.Printf("    Model: %s\n", tier.ModelName)
				if len(tier.UseFor) > 0 {
					fmt.Printf("    Use for: %s\n", strings.Join(tier.UseFor, ", "))
				}
				fmt.Printf("    Cost: $%.2f/M input, $%.2f/M output\n",
					tier.CostPerM.Input, tier.CostPerM.Output)
				fmt.Println()
			}
		}
	}

	// Show default model
	defaultModel := cfg.Agents.Defaults.GetModelName()
	fmt.Printf("⭐ Default Model: %s\n", defaultModel)

	return nil
}

func detectProvider(m pkgconfig.ModelConfig) string {
	// Detect provider from API base or model ID
	if m.APIBase != "" {
		apiBase := strings.ToLower(m.APIBase)
		if strings.Contains(apiBase, "localhost") || strings.Contains(apiBase, "127.0.0.1") {
			return "LM Studio (Local)"
		}
		if strings.Contains(apiBase, "openrouter") {
			return "OpenRouter"
		}
		if strings.Contains(apiBase, "anthropic") {
			return "Anthropic"
		}
		if strings.Contains(apiBase, "openai") {
			return "OpenAI"
		}
		return "Custom API"
	}

	// Detect from model ID
	modelID := strings.ToLower(m.Model)
	if strings.Contains(modelID, "claude") || strings.Contains(modelID, "anthropic") {
		return "Anthropic"
	}
	if strings.Contains(modelID, "gpt") || strings.Contains(modelID, "openai") {
		return "OpenAI"
	}
	if strings.Contains(modelID, "openrouter") {
		return "OpenRouter"
	}
	if strings.Contains(modelID, "gemini") || strings.Contains(modelID, "google") {
		return "Google"
	}

	return "Unknown"
}

func getModelTier(cfg *pkgconfig.Config, modelName string) string {
	if !cfg.Routing.Enabled {
		return ""
	}

	// Check each tier
	if tier, ok := cfg.Routing.Tiers["heavy"]; ok && tier.ModelName == modelName {
		return "heavy"
	}
	if tier, ok := cfg.Routing.Tiers["medium"]; ok && tier.ModelName == modelName {
		return "medium"
	}
	if tier, ok := cfg.Routing.Tiers["light"]; ok && tier.ModelName == modelName {
		return "light"
	}

	return ""
}

func getTierConfig(cfg *pkgconfig.Config, tierName string) (pkgconfig.TierConfig, bool) {
	if cfg.Routing.Tiers == nil {
		return pkgconfig.TierConfig{}, false
	}

	tier, ok := cfg.Routing.Tiers[tierName]
	return tier, ok
}

func plural(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
