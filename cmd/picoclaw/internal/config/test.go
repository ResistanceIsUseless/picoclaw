package config

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/sipeed/picoclaw/cmd/picoclaw/internal"
	pkgconfig "github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/providers"
)

func newTestCommand() *cobra.Command {
	var testAll bool

	cmd := &cobra.Command{
		Use:   "test [model-name]",
		Short: "Test API connections and model availability",
		Long: `Test connections to configured AI providers and models.

Examples:
  picoclaw config test                    # Test default model
  picoclaw config test --all              # Test all configured models
  picoclaw config test claude-sonnet-4    # Test specific model`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return testCmd(args, testAll)
		},
	}

	cmd.Flags().BoolVar(&testAll, "all", false, "Test all configured models")

	return cmd
}

func testCmd(args []string, testAll bool) error {
	cfg, err := internal.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if len(cfg.ModelList) == 0 {
		return fmt.Errorf("no models configured in config.json")
	}

	fmt.Println("🔍 Testing API Connections\n")

	// Test specific model if provided
	if len(args) > 0 {
		modelName := args[0]
		return testModel(cfg, modelName)
	}

	// Test all models if --all flag
	if testAll {
		var successCount, failCount int
		for _, modelCfg := range cfg.ModelList {
			if err := testModel(cfg, modelCfg.ModelName); err != nil {
				failCount++
			} else {
				successCount++
			}
			fmt.Println()
		}
		fmt.Printf("Summary: %d/%d models tested successfully\n", successCount, successCount+failCount)
		return nil
	}

	// Test default model
	defaultModel := cfg.Agents.Defaults.GetModelName()
	fmt.Printf("Testing default model: %s\n\n", defaultModel)
	return testModel(cfg, defaultModel)
}

func testModel(cfg *pkgconfig.Config, modelName string) error {
	// Find model in config
	var modelCfg *pkgconfig.ModelConfig
	for _, m := range cfg.ModelList {
		if m.ModelName == modelName {
			modelCfg = &m
			break
		}
	}

	if modelCfg == nil {
		return fmt.Errorf("❌ Model '%s' not found in config", modelName)
	}

	fmt.Printf("Testing: %s\n", modelName)
	fmt.Printf("  Model ID: %s\n", modelCfg.Model)
	if modelCfg.APIBase != "" {
		fmt.Printf("  API Base: %s\n", modelCfg.APIBase)
	}

	// Create provider for this model
	provider, resolvedModel, err := providers.CreateProvider(cfg)
	if err != nil {
		fmt.Printf("  ❌ Failed to create provider: %v\n", err)
		return err
	}

	// Test with simple prompt
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testMessage := providers.Message{
		Role:    "user",
		Content: "Respond with exactly: 'Connection successful'",
	}

	fmt.Printf("  🔄 Sending test request...\n")
	start := time.Now()

	response, err := provider.Chat(ctx, []providers.Message{testMessage}, nil, resolvedModel, nil)
	elapsed := time.Since(start)

	if err != nil {
		fmt.Printf("  ❌ Request failed: %v\n", err)
		return err
	}

	fmt.Printf("  ✅ Connection successful!\n")
	fmt.Printf("  Response time: %v\n", elapsed.Round(time.Millisecond))
	if response.Usage.PromptTokens > 0 {
		fmt.Printf("  Tokens: %d prompt + %d completion = %d total\n",
			response.Usage.PromptTokens,
			response.Usage.CompletionTokens,
			response.Usage.PromptTokens+response.Usage.CompletionTokens)
	}
	fmt.Printf("  Response: %s\n", truncate(response.Content, 100))

	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
