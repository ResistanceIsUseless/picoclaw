package config

import (
	"fmt"
	"strings"
)

// Warning represents a configuration warning
type Warning struct {
	Level   string // "error", "warning", "info"
	Message string
}

// ValidateConfigQuality checks if the config is suitable for security assessment
func ValidateConfigQuality(cfg *Config) []Warning {
	warnings := []Warning{}

	// Check 1: No models configured
	if len(cfg.ModelList) == 0 {
		warnings = append(warnings, Warning{
			Level:   "error",
			Message: "No models configured. Run: picoclaw config setup",
		})
		return warnings // Fatal error, no point checking further
	}

	// Check 2: No API keys configured
	hasValidKey := false
	for _, model := range cfg.ModelList {
		if model.APIKey != "" {
			hasValidKey = true
			break
		}
	}

	if !hasValidKey {
		warnings = append(warnings, Warning{
			Level:   "error",
			Message: "No API keys configured. Run: picoclaw config setup",
		})
		return warnings
	}

	// Check 3: Using weak models for security assessment
	for _, model := range cfg.ModelList {
		if isWeakForSecurity(model.Model) {
			warnings = append(warnings, Warning{
				Level: "warning",
				Message: fmt.Sprintf(
					"Model '%s' may struggle with complex security analysis.\n  "+
						"Consider: claude-sonnet-4.6, gpt-4.5-turbo, or deepseek-chat",
					model.ModelName,
				),
			})
		}
	}

	// Check 4: Single-model setup with complex tasks
	if len(cfg.ModelList) == 1 && !cfg.Routing.Enabled {
		warnings = append(warnings, Warning{
			Level: "info",
			Message: "Tip: Enable multi-model routing for better performance.\n  " +
				"Run: picoclaw config setup",
		})
	}

	// Check 5: Routing enabled but not properly configured
	if cfg.Routing.Enabled && len(cfg.Routing.Tiers) == 0 {
		warnings = append(warnings, Warning{
			Level: "warning",
			Message: "Multi-model routing is enabled but no tiers are configured.\n  " +
				"Run: picoclaw config setup",
		})
	}

	return warnings
}

// isWeakForSecurity checks if a model is known to be weak for security tasks
func isWeakForSecurity(modelName string) bool {
	lowerModel := strings.ToLower(modelName)

	weakPatterns := []string{
		"gpt-3.5",
		"gpt-4-turbo-preview", // Old preview model
		"llama-3.1-8b",
		"llama-2",
		"mistral-7b",
		"mixtral-8x7b", // Weak compared to newer models
		"gemma-7b",
		"phi-3",
	}

	for _, pattern := range weakPatterns {
		if strings.Contains(lowerModel, pattern) {
			return true
		}
	}

	return false
}

// ModelRecommendation represents a recommended model configuration
type ModelRecommendation struct {
	Name        string
	Provider    string
	Strengths   []string
	Cost        string // "Low", "Medium", "High"
	Description string
}

// RecommendedModels returns models recommended for security assessment
func RecommendedModels() []ModelRecommendation {
	return []ModelRecommendation{
		{
			Name:        "claude-sonnet-4.6",
			Provider:    "anthropic",
			Strengths:   []string{"reasoning", "security analysis", "tool use"},
			Cost:        "Medium",
			Description: "Best overall for security assessment. Excellent reasoning and tool use.",
		},
		{
			Name:        "claude-opus-4.6",
			Provider:    "anthropic",
			Strengths:   []string{"deep analysis", "complex reasoning"},
			Cost:        "High",
			Description: "Most capable model. Use for complex analysis that requires maximum intelligence.",
		},
		{
			Name:        "deepseek-chat",
			Provider:    "deepseek",
			Strengths:   []string{"cost-effective", "reasoning", "coding"},
			Cost:        "Low",
			Description: "Excellent value. Strong reasoning at very low cost. Great for budget-conscious users.",
		},
		{
			Name:        "gpt-4.5-turbo",
			Provider:    "openai",
			Strengths:   []string{"structured output", "reliable", "fast"},
			Cost:        "Medium",
			Description: "Latest GPT-4. Good all-around performance with fast response times.",
		},
		{
			Name:        "claude-haiku-4.5",
			Provider:    "anthropic",
			Strengths:   []string{"speed", "cost-effective", "parsing"},
			Cost:        "Low",
			Description: "Fastest Claude model. Ideal for planning tier and parsing tool outputs.",
		},
		{
			Name:        "gpt-4o-mini",
			Provider:    "openai",
			Strengths:   []string{"speed", "cost-effective"},
			Cost:        "Low",
			Description: "Fast and cheap. Good for planning and simple tasks.",
		},
		{
			Name:        "gemini-2.0-flash-exp",
			Provider:    "google",
			Strengths:   []string{"structured output", "cost-effective", "fast"},
			Cost:        "Low",
			Description: "Excellent for parsing tool outputs. Strong structured output capabilities.",
		},
	}
}

// SuggestModelsForTier suggests models suitable for a specific routing tier
func SuggestModelsForTier(tier string) []string {
	switch strings.ToLower(tier) {
	case "planning":
		return []string{
			"claude-haiku-4.5",
			"gpt-4o-mini",
			"groq/llama-3.3-70b",
		}
	case "analysis":
		return []string{
			"claude-sonnet-4.6",
			"claude-opus-4.6",
			"deepseek-chat",
			"gpt-4.5-turbo",
		}
	case "parsing":
		return []string{
			"gpt-4.5-turbo",
			"gemini-2.0-flash-exp",
			"claude-haiku-4.5",
		}
	default:
		return []string{"claude-sonnet-4.6", "gpt-4.5-turbo"}
	}
}

// FormatWarnings formats warnings for display
func FormatWarnings(warnings []Warning) string {
	if len(warnings) == 0 {
		return ""
	}

	var sb strings.Builder

	for _, w := range warnings {
		var prefix string
		switch w.Level {
		case "error":
			prefix = "❌ Error:"
		case "warning":
			prefix = "⚠  Warning:"
		case "info":
			prefix = "ℹ  Info:"
		}

		sb.WriteString(fmt.Sprintf("%s %s\n", prefix, w.Message))
	}

	return sb.String()
}
