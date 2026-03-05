package main

import (
	"os"
	"strings"

	"github.com/ResistanceIsUseless/picoclaw/pkg/config"
)

// detectAPIKey tries to find an API key from environment variables
func detectAPIKey() string {
	// Try common environment variables in order of preference
	envVars := []string{
		"ANTHROPIC_API_KEY",
		"OPENAI_API_KEY",
		"OPENROUTER_API_KEY",
		"GEMINI_API_KEY",
		"GROQ_API_KEY",
		"DEEPSEEK_API_KEY",
	}

	for _, envVar := range envVars {
		if key := os.Getenv(envVar); key != "" {
			return key
		}
	}

	return ""
}

// detectProviderFromModel infers provider from model name
func detectProviderFromModel(model string) string {
	lowerModel := strings.ToLower(model)

	if strings.Contains(lowerModel, "claude") || strings.Contains(lowerModel, "sonnet") || strings.Contains(lowerModel, "opus") || strings.Contains(lowerModel, "haiku") {
		return "anthropic"
	}
	if strings.Contains(lowerModel, "gpt") || strings.Contains(lowerModel, "o1") {
		return "openai"
	}
	if strings.Contains(lowerModel, "gemini") {
		return "gemini"
	}
	if strings.Contains(lowerModel, "deepseek") {
		return "deepseek"
	}
	if strings.Contains(lowerModel, "llama") || strings.Contains(lowerModel, "mixtral") {
		return "groq"
	}

	return "unknown"
}

// buildProviderConfig creates a minimal config for provider factory
func buildProviderConfig(providerName, apiKey, apiBase, model string) *config.Config {
	cfg := config.DefaultConfig()

	// Auto-detect provider if not specified
	if providerName == "" {
		providerName = detectProviderFromModel(model)
	}

	// Set model
	cfg.Agents.Defaults.ModelName = model
	cfg.Agents.Defaults.Model = model
	cfg.Agents.Defaults.Provider = providerName

	// Add model to model_list (required by new provider system)
	// Format: "provider/model" (e.g., "openrouter/anthropic/claude-sonnet-4")
	fullModelName := model
	if !strings.Contains(model, "/") {
		fullModelName = providerName + "/" + model
	}

	cfg.ModelList = []config.ModelConfig{
		{
			ModelName: model,
			Model:     fullModelName,
			APIKey:    apiKey,
			APIBase:   apiBase,
		},
	}

	// Configure provider-specific settings
	switch strings.ToLower(providerName) {
	case "anthropic", "claude":
		cfg.Providers.Anthropic.APIKey = apiKey
		if apiBase != "" {
			cfg.Providers.Anthropic.APIBase = apiBase
		}

	case "openai", "gpt":
		cfg.Providers.OpenAI.APIKey = apiKey
		if apiBase != "" {
			cfg.Providers.OpenAI.APIBase = apiBase
		}

	case "openrouter":
		cfg.Providers.OpenRouter.APIKey = apiKey
		if apiBase != "" {
			cfg.Providers.OpenRouter.APIBase = apiBase
		} else {
			cfg.Providers.OpenRouter.APIBase = "https://openrouter.ai/api/v1"
		}

	case "gemini", "google":
		cfg.Providers.Gemini.APIKey = apiKey
		if apiBase != "" {
			cfg.Providers.Gemini.APIBase = apiBase
		}

	case "deepseek":
		cfg.Providers.DeepSeek.APIKey = apiKey
		if apiBase != "" {
			cfg.Providers.DeepSeek.APIBase = apiBase
		}

	case "groq":
		cfg.Providers.Groq.APIKey = apiKey
		if apiBase != "" {
			cfg.Providers.Groq.APIBase = apiBase
		}

	case "lmstudio", "ollama":
		// Local providers - no API key needed
		if apiBase != "" {
			cfg.Providers.OpenAI.APIBase = apiBase
		} else if providerName == "lmstudio" {
			cfg.Providers.OpenAI.APIBase = "http://localhost:1234/v1"
		} else {
			cfg.Providers.OpenAI.APIBase = "http://localhost:11434/v1"
		}
		cfg.Providers.OpenAI.APIKey = "local"
		cfg.Agents.Defaults.Provider = "openai" // Use OpenAI-compatible endpoint

	default:
		// Unknown provider - assume OpenAI-compatible
		cfg.Providers.OpenAI.APIKey = apiKey
		if apiBase != "" {
			cfg.Providers.OpenAI.APIBase = apiBase
		}
		cfg.Agents.Defaults.Provider = "openai"
	}

	return cfg
}
