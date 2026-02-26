package config

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	pkgconfig "github.com/sipeed/picoclaw/pkg/config"
)

func newDiscoverCommand() *cobra.Command {
	var (
		provider     string
		interactive  bool
		outputConfig string
	)

	cmd := &cobra.Command{
		Use:   "discover",
		Short: "Discover available models from providers",
		Long: `Query AI providers to discover available models and optionally add them to your configuration.

Supported providers:
  - lmstudio: Local LM Studio instance
  - openrouter: OpenRouter API
  - anthropic: Anthropic API

Examples:
  picoclaw config discover --provider lmstudio    # List LM Studio models
  picoclaw config discover --provider openrouter  # List OpenRouter models
  picoclaw config discover --provider anthropic   # List Anthropic models
  picoclaw config discover --interactive          # Discover all and select interactively`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return discoverCmd(provider, interactive, outputConfig)
		},
	}

	cmd.Flags().StringVarP(&provider, "provider", "p", "", "Provider to query (lmstudio, openrouter, anthropic)")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Interactive mode to select models")
	cmd.Flags().StringVarP(&outputConfig, "output", "o", "", "Output updated config to file (default: update config.json)")

	return cmd
}

type ProviderModels struct {
	Provider string
	Models   []DiscoveredModel
	Error    error
}

type DiscoveredModel struct {
	ID          string
	Name        string
	Description string
	Context     int
	Pricing     *ModelPricing
}

type ModelPricing struct {
	Prompt     float64
	Completion float64
}

func discoverCmd(provider string, interactive bool, outputConfig string) error {
	cfg, err := internal.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Println("🔍 Discovering Available Models\n")

	var results []ProviderModels

	// Discover from specified provider or all
	if provider != "" {
		models, err := discoverProvider(cfg, provider)
		results = append(results, ProviderModels{
			Provider: provider,
			Models:   models,
			Error:    err,
		})
	} else {
		// Discover from all available providers
		for _, providerName := range []string{"lmstudio", "openrouter", "anthropic"} {
			models, err := discoverProvider(cfg, providerName)
			results = append(results, ProviderModels{
				Provider: providerName,
				Models:   models,
				Error:    err,
			})
		}
	}

	// Display results
	if !interactive {
		for _, result := range results {
			displayProviderModels(result)
		}
		return nil
	}

	// Interactive mode: let user select models
	return interactiveSelection(cfg, results, outputConfig)
}

func discoverProvider(cfg *pkgconfig.Config, provider string) ([]DiscoveredModel, error) {
	switch strings.ToLower(provider) {
	case "lmstudio":
		return discoverLMStudio(cfg)
	case "openrouter":
		return discoverOpenRouter(cfg)
	case "anthropic":
		return discoverAnthropic(cfg)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

func discoverLMStudio(cfg *pkgconfig.Config) ([]DiscoveredModel, error) {
	// Try to find LM Studio API base from config or environment
	apiBase := os.Getenv("LM_STUDIO_BASE_URL")
	if apiBase == "" {
		// Check if any model has localhost API base
		for _, m := range cfg.ModelList {
			if strings.Contains(m.APIBase, "localhost") || strings.Contains(m.APIBase, "127.0.0.1") {
				apiBase = m.APIBase
				break
			}
		}
	}
	if apiBase == "" {
		apiBase = "http://localhost:1234/v1"
	}

	// Query LM Studio models endpoint
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", apiBase+"/models", nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to LM Studio at %s: %w", apiBase, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("LM Studio returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	models := make([]DiscoveredModel, 0, len(result.Data))
	for _, m := range result.Data {
		models = append(models, DiscoveredModel{
			ID:   m.ID,
			Name: m.ID,
		})
	}

	return models, nil
}

func discoverOpenRouter(cfg *pkgconfig.Config) ([]DiscoveredModel, error) {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		// Try to find in config
		for _, m := range cfg.ModelList {
			if strings.Contains(strings.ToLower(m.APIBase), "openrouter") && m.APIKey != "" {
				apiKey = m.APIKey
				break
			}
		}
	}

	if apiKey == "" {
		return nil, fmt.Errorf("OPENROUTER_API_KEY not found in environment or config")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://openrouter.ai/api/v1/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("OpenRouter returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			Context     int    `json:"context_length"`
			Pricing     struct {
				Prompt     string `json:"prompt"`
				Completion string `json:"completion"`
			} `json:"pricing"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	models := make([]DiscoveredModel, 0, len(result.Data))
	for _, m := range result.Data {
		models = append(models, DiscoveredModel{
			ID:          m.ID,
			Name:        m.Name,
			Description: m.Description,
			Context:     m.Context,
		})
	}

	return models, nil
}

func discoverAnthropic(cfg *pkgconfig.Config) ([]DiscoveredModel, error) {
	// Anthropic doesn't have a models list API, so return known models
	return []DiscoveredModel{
		{
			ID:          "claude-opus-4-6",
			Name:        "Claude Opus 4.6",
			Description: "Most capable model for complex tasks",
			Context:     200000,
			Pricing: &ModelPricing{
				Prompt:     15.0,
				Completion: 75.0,
			},
		},
		{
			ID:          "claude-sonnet-4-6",
			Name:        "Claude Sonnet 4.6",
			Description: "Balanced performance and speed",
			Context:     200000,
			Pricing: &ModelPricing{
				Prompt:     3.0,
				Completion: 15.0,
			},
		},
		{
			ID:          "claude-haiku-4-5-20251001",
			Name:        "Claude Haiku 4.5",
			Description: "Fast and cost-effective",
			Context:     200000,
			Pricing: &ModelPricing{
				Prompt:     0.8,
				Completion: 4.0,
			},
		},
		{
			ID:          "claude-sonnet-3-5-20241022",
			Name:        "Claude Sonnet 3.5",
			Description: "Previous generation Sonnet",
			Context:     200000,
			Pricing: &ModelPricing{
				Prompt:     3.0,
				Completion: 15.0,
			},
		},
	}, nil
}

func displayProviderModels(result ProviderModels) {
	providerName := strings.ToUpper(result.Provider[:1]) + result.Provider[1:]
	fmt.Printf("🔌 %s\n", providerName)

	if result.Error != nil {
		fmt.Printf("  ❌ Error: %v\n\n", result.Error)
		return
	}

	if len(result.Models) == 0 {
		fmt.Printf("  ℹ️  No models found\n\n")
		return
	}

	fmt.Printf("  Found %d model%s:\n\n", len(result.Models), plural(len(result.Models)))

	for i, model := range result.Models {
		if i >= 20 && len(result.Models) > 20 {
			fmt.Printf("  ... and %d more models\n\n", len(result.Models)-20)
			break
		}

		fmt.Printf("  • %s", model.ID)
		if model.Name != "" && model.Name != model.ID {
			fmt.Printf(" (%s)", model.Name)
		}
		fmt.Println()

		if model.Description != "" {
			desc := model.Description
			if len(desc) > 80 {
				desc = desc[:77] + "..."
			}
			fmt.Printf("    %s\n", desc)
		}

		if model.Context > 0 {
			fmt.Printf("    Context: %d tokens", model.Context)
		}

		if model.Pricing != nil {
			if model.Context > 0 {
				fmt.Print(" | ")
			} else {
				fmt.Print("    ")
			}
			fmt.Printf("$%.2f/$%.2f per M tokens\n", model.Pricing.Prompt, model.Pricing.Completion)
		} else if model.Context > 0 {
			fmt.Println()
		}

		fmt.Println()
	}
}

func interactiveSelection(cfg *pkgconfig.Config, results []ProviderModels, outputPath string) error {
	fmt.Println("\n📝 Interactive Model Selection\n")

	// Collect all available models
	var allModels []struct {
		Provider string
		Model    DiscoveredModel
	}

	for _, result := range results {
		if result.Error != nil {
			fmt.Printf("⚠️  Skipping %s due to error: %v\n", result.Provider, result.Error)
			continue
		}
		for _, model := range result.Models {
			allModels = append(allModels, struct {
				Provider string
				Model    DiscoveredModel
			}{
				Provider: result.Provider,
				Model:    model,
			})
		}
	}

	if len(allModels) == 0 {
		return fmt.Errorf("no models available for selection")
	}

	fmt.Printf("Found %d total models across all providers\n\n", len(allModels))

	// Display models with numbers
	for i, item := range allModels {
		fmt.Printf("[%d] %s - %s", i+1, item.Provider, item.Model.ID)
		if item.Model.Name != "" && item.Model.Name != item.Model.ID {
			fmt.Printf(" (%s)", item.Model.Name)
		}
		fmt.Println()
	}

	fmt.Println("\nEnter model numbers to add (comma-separated, e.g., 1,3,5), 'all', or 'done' to finish:")
	fmt.Print("> ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %w", err)
	}

	input = strings.TrimSpace(input)
	if input == "done" || input == "" {
		fmt.Println("No models selected")
		return nil
	}

	// Parse selections
	var selectedModels []struct {
		Provider string
		Model    DiscoveredModel
	}

	if input == "all" {
		selectedModels = allModels
	} else {
		// Parse comma-separated numbers
		parts := strings.Split(input, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			num, err := strconv.Atoi(part)
			if err != nil || num < 1 || num > len(allModels) {
				fmt.Printf("⚠️  Invalid selection: %s\n", part)
				continue
			}
			selectedModels = append(selectedModels, allModels[num-1])
		}
	}

	if len(selectedModels) == 0 {
		fmt.Println("No valid models selected")
		return nil
	}

	fmt.Printf("\n✅ Selected %d model%s\n\n", len(selectedModels), plural(len(selectedModels)))

	// Add models to config
	for _, item := range selectedModels {
		modelConfig := createModelConfig(item.Provider, item.Model)

		// Check if model already exists
		exists := false
		for _, existing := range cfg.ModelList {
			if existing.ModelName == modelConfig.ModelName {
				exists = true
				fmt.Printf("⚠️  Model %s already exists in config, skipping\n", modelConfig.ModelName)
				break
			}
		}

		if !exists {
			cfg.ModelList = append(cfg.ModelList, modelConfig)
			fmt.Printf("➕ Added: %s\n", modelConfig.ModelName)
		}
	}

	// Save config
	configPath := outputPath
	if configPath == "" {
		configPath = internal.GetConfigPath()
	}

	fmt.Printf("\n💾 Saving configuration to: %s\n", configPath)

	// Marshal config to JSON with indentation
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Println("✅ Configuration saved successfully!")
	fmt.Println("\nRun 'picoclaw config models' to view your updated configuration")

	return nil
}

func createModelConfig(provider string, model DiscoveredModel) pkgconfig.ModelConfig {
	config := pkgconfig.ModelConfig{
		ModelName: sanitizeModelName(model.ID),
		Model:     model.ID,
	}

	switch strings.ToLower(provider) {
	case "lmstudio":
		// Use LM Studio base URL
		apiBase := os.Getenv("LM_STUDIO_BASE_URL")
		if apiBase == "" {
			apiBase = "http://localhost:1234/v1"
		}
		config.APIBase = apiBase

	case "openrouter":
		config.APIBase = "https://openrouter.ai/api/v1"
		config.APIKey = os.Getenv("OPENROUTER_API_KEY")

	case "anthropic":
		// Anthropic uses default settings, just need API key
		config.APIKey = os.Getenv("ANTHROPIC_API_KEY")
	}

	return config
}

func sanitizeModelName(id string) string {
	// Convert model ID to a friendly name
	name := id

	// Remove provider prefixes
	name = strings.TrimPrefix(name, "openai/")
	name = strings.TrimPrefix(name, "anthropic/")
	name = strings.TrimPrefix(name, "google/")

	// Replace slashes with dashes
	name = strings.ReplaceAll(name, "/", "-")

	return name
}
