package openai_compat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/ResistanceIsUseless/picoclaw/pkg/providers/protocoltypes"
)

type (
	ToolCall               = protocoltypes.ToolCall
	FunctionCall           = protocoltypes.FunctionCall
	LLMResponse            = protocoltypes.LLMResponse
	UsageInfo              = protocoltypes.UsageInfo
	Message                = protocoltypes.Message
	ToolDefinition         = protocoltypes.ToolDefinition
	ToolFunctionDefinition = protocoltypes.ToolFunctionDefinition
	ExtraContent           = protocoltypes.ExtraContent
	GoogleExtra            = protocoltypes.GoogleExtra
)

type Provider struct {
	apiKey         string
	apiBase        string
	maxTokensField string // Field name for max tokens (e.g., "max_completion_tokens" for o1/glm models)
	httpClient     *http.Client
}

func NewProvider(apiKey, apiBase, proxy string) *Provider {
	return NewProviderWithMaxTokensField(apiKey, apiBase, proxy, "")
}

func NewProviderWithMaxTokensField(apiKey, apiBase, proxy, maxTokensField string) *Provider {
	client := &http.Client{
		Timeout: 120 * time.Second,
	}

	if proxy != "" {
		parsed, err := url.Parse(proxy)
		if err == nil {
			client.Transport = &http.Transport{
				Proxy: http.ProxyURL(parsed),
			}
		} else {
			log.Printf("openai_compat: invalid proxy URL %q: %v", proxy, err)
		}
	}

	return &Provider{
		apiKey:         apiKey,
		apiBase:        strings.TrimRight(apiBase, "/"),
		maxTokensField: maxTokensField,
		httpClient:     client,
	}
}

func (p *Provider) Chat(
	ctx context.Context,
	messages []Message,
	tools []ToolDefinition,
	model string,
	options map[string]any,
) (*LLMResponse, error) {
	if p.apiBase == "" {
		return nil, fmt.Errorf("API base not configured")
	}

	model = normalizeModel(model, p.apiBase)

	requestBody := map[string]any{
		"model":    model,
		"messages": stripSystemParts(messages),
	}

	if len(tools) > 0 {
		requestBody["tools"] = tools
		requestBody["tool_choice"] = "auto"
	}

	if maxTokens, ok := asInt(options["max_tokens"]); ok {
		// Use configured maxTokensField if specified, otherwise fallback to model-based detection
		fieldName := p.maxTokensField
		if fieldName == "" {
			// Fallback: detect from model name for backward compatibility
			lowerModel := strings.ToLower(model)
			if strings.Contains(lowerModel, "glm") || strings.Contains(lowerModel, "o1") ||
				strings.Contains(lowerModel, "gpt-5") {
				fieldName = "max_completion_tokens"
			} else {
				fieldName = "max_tokens"
			}
		}
		requestBody[fieldName] = maxTokens
	}

	if temperature, ok := asFloat(options["temperature"]); ok {
		lowerModel := strings.ToLower(model)
		// Kimi k2 models only support temperature=1.
		if strings.Contains(lowerModel, "kimi") && strings.Contains(lowerModel, "k2") {
			requestBody["temperature"] = 1.0
		} else {
			requestBody["temperature"] = temperature
		}
	}

	// Prompt caching: pass a stable cache key so OpenAI can bucket requests
	// with the same key and reuse prefix KV cache across calls.
	// The key is typically the agent ID — stable per agent, shared across requests.
	// See: https://platform.openai.com/docs/guides/prompt-caching
	if cacheKey, ok := options["prompt_cache_key"].(string); ok && cacheKey != "" {
		requestBody["prompt_cache_key"] = cacheKey
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.apiBase+"/chat/completions", bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed:\n  Status: %d\n  Body:   %s", resp.StatusCode, string(body))
	}

	return parseResponse(body)
}

func parseResponse(body []byte) (*LLMResponse, error) {
	var apiResponse struct {
		Choices []struct {
			Message struct {
				Content          string `json:"content"`
				ReasoningContent string `json:"reasoning_content"`
				ToolCalls        []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function *struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
					ExtraContent *struct {
						Google *struct {
							ThoughtSignature string `json:"thought_signature"`
						} `json:"google"`
					} `json:"extra_content"`
				} `json:"tool_calls"`
			} `json:"message"`
			FinishReason string `json:"finish_reason"`
		} `json:"choices"`
		Usage *UsageInfo `json:"usage"`
	}

	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(apiResponse.Choices) == 0 {
		return &LLMResponse{
			Content:      "",
			FinishReason: "stop",
		}, nil
	}

	choice := apiResponse.Choices[0]
	toolCalls := make([]ToolCall, 0, len(choice.Message.ToolCalls))
	for _, tc := range choice.Message.ToolCalls {
		arguments := make(map[string]any)
		name := ""

		// Extract thought_signature from Gemini/Google-specific extra content
		thoughtSignature := ""
		if tc.ExtraContent != nil && tc.ExtraContent.Google != nil {
			thoughtSignature = tc.ExtraContent.Google.ThoughtSignature
		}

		if tc.Function != nil {
			name = tc.Function.Name
			if tc.Function.Arguments != "" {
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &arguments); err != nil {
					log.Printf("openai_compat: failed to decode tool call arguments for %q: %v", name, err)
					arguments["raw"] = tc.Function.Arguments
				}
			}
		}

		// Build ToolCall with ExtraContent for Gemini 3 thought_signature persistence
		toolCall := ToolCall{
			ID:               tc.ID,
			Name:             name,
			Arguments:        arguments,
			ThoughtSignature: thoughtSignature,
		}

		if thoughtSignature != "" {
			toolCall.ExtraContent = &ExtraContent{
				Google: &GoogleExtra{
					ThoughtSignature: thoughtSignature,
				},
			}
		}

		toolCalls = append(toolCalls, toolCall)
	}

	// Fallback: if no structured tool calls were returned but the content
	// contains text-formatted tool calls (common with local models like
	// codestral, qwen, etc.), parse them from the text.
	if len(toolCalls) == 0 && choice.Message.Content != "" {
		if extracted := extractToolCallsFromText(choice.Message.Content); len(extracted) > 0 {
			log.Printf("openai_compat: extracted %d tool call(s) from text output (model did not use structured tool calling)", len(extracted))
			toolCalls = extracted
			// Clear the content since it was a tool call, not a real response
			choice.Message.Content = ""
			choice.FinishReason = "tool_calls"
		}
	}

	return &LLMResponse{
		Content:          choice.Message.Content,
		ReasoningContent: choice.Message.ReasoningContent,
		ToolCalls:        toolCalls,
		FinishReason:     choice.FinishReason,
		Usage:            apiResponse.Usage,
	}, nil
}

// openaiMessage is the wire-format message for OpenAI-compatible APIs.
// It mirrors protocoltypes.Message but omits SystemParts, which is an
// internal field that would be unknown to third-party endpoints.
type openaiMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// stripSystemParts converts []Message to []openaiMessage, dropping the
// SystemParts field so it doesn't leak into the JSON payload sent to
// OpenAI-compatible APIs (some strict endpoints reject unknown fields).
func stripSystemParts(messages []Message) []openaiMessage {
	out := make([]openaiMessage, len(messages))
	for i, m := range messages {
		out[i] = openaiMessage{
			Role:       m.Role,
			Content:    m.Content,
			ToolCalls:  m.ToolCalls,
			ToolCallID: m.ToolCallID,
		}
	}
	return out
}

func normalizeModel(model, apiBase string) string {
	idx := strings.Index(model, "/")
	if idx == -1 {
		return model
	}

	if strings.Contains(strings.ToLower(apiBase), "openrouter.ai") {
		return model
	}

	prefix := strings.ToLower(model[:idx])
	switch prefix {
	case "moonshot", "nvidia", "groq", "ollama", "deepseek", "google", "openrouter", "zhipu", "mistral":
		return model[idx+1:]
	default:
		return model
	}
}

func asInt(v any) (int, bool) {
	switch val := v.(type) {
	case int:
		return val, true
	case int64:
		return int(val), true
	case float64:
		return int(val), true
	case float32:
		return int(val), true
	default:
		return 0, false
	}
}

func asFloat(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	default:
		return 0, false
	}
}

// textToolCallTagPattern matches the opening tags of text-formatted tool calls.
var textToolCallTagPattern = regexp.MustCompile(`<(?:functioncall|tool_call)>\s*|` +
	`\[TOOL_CALL\]\s*`)

// extractToolCallsFromText parses tool calls embedded in the response text.
// Many local models (codestral, qwen, mistral, etc.) emit tool calls as
// text like <functioncall>{"name":"exec","arguments":{"command":"ls"}}
// rather than using the API's structured tool_calls field.
//
// This uses brace-counting to correctly extract nested JSON objects
// (e.g., {"name":"exec","arguments":{"command":"ls /tmp"}}).
func extractToolCallsFromText(content string) []ToolCall {
	var toolCalls []ToolCall

	// Find all opening tags and extract JSON after each one
	tagLocs := textToolCallTagPattern.FindAllStringIndex(content, -1)
	for _, loc := range tagLocs {
		remaining := content[loc[1]:]

		// Extract balanced JSON object using brace counting
		jsonStr := extractBalancedJSON(remaining)
		if jsonStr == "" {
			continue
		}

		// Try to parse as {"name":"...","arguments":{...}} or
		// {"name":"...","arguments":"..."} (stringified JSON)
		var call struct {
			Name      string `json:"name"`
			Arguments any    `json:"arguments"`
		}
		if err := json.Unmarshal([]byte(jsonStr), &call); err != nil {
			log.Printf("openai_compat: failed to parse text tool call: %v", err)
			continue
		}
		if call.Name == "" {
			continue
		}

		arguments := make(map[string]any)
		switch args := call.Arguments.(type) {
		case map[string]any:
			arguments = args
		case string:
			// Some models stringify the arguments JSON
			if err := json.Unmarshal([]byte(args), &arguments); err != nil {
				arguments["raw"] = args
			}
		}

		toolCalls = append(toolCalls, ToolCall{
			ID:        fmt.Sprintf("textcall_%d", len(toolCalls)),
			Name:      call.Name,
			Arguments: arguments,
		})
	}

	return toolCalls
}

// extractBalancedJSON finds the first balanced JSON object in s.
// Returns the complete JSON string including outer braces, or "" if not found.
func extractBalancedJSON(s string) string {
	start := strings.IndexByte(s, '{')
	if start == -1 {
		return ""
	}

	depth := 0
	inString := false
	escaped := false

	for i := start; i < len(s); i++ {
		ch := s[i]

		if escaped {
			escaped = false
			continue
		}

		if ch == '\\' && inString {
			escaped = true
			continue
		}

		if ch == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		switch ch {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}

	return "" // unbalanced braces
}
