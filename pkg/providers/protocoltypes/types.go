package protocoltypes

type ToolCall struct {
	ID               string         `json:"id"`
	Type             string         `json:"type,omitempty"`
	Function         *FunctionCall  `json:"function,omitempty"`
	Name             string         `json:"-"`
	Arguments        map[string]any `json:"-"`
	ThoughtSignature string         `json:"-"` // Internal use only
	ExtraContent     *ExtraContent  `json:"extra_content,omitempty"`
}

type ExtraContent struct {
	Google *GoogleExtra `json:"google,omitempty"`
}

type GoogleExtra struct {
	ThoughtSignature string `json:"thought_signature,omitempty"`
}

type FunctionCall struct {
	Name             string `json:"name"`
	Arguments        string `json:"arguments"`
	ThoughtSignature string `json:"thought_signature,omitempty"`
}

type LLMResponse struct {
	Content          string     `json:"content"`
	ReasoningContent string     `json:"reasoning_content,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
	FinishReason     string     `json:"finish_reason"`
	Usage            *UsageInfo `json:"usage,omitempty"`
}

type UsageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// CacheControl marks a content block for LLM-side prefix caching.
// Currently only "ephemeral" is supported (used by Anthropic).
type CacheControl struct {
	Type string `json:"type"` // "ephemeral"
}

// ContentBlock represents a structured segment of a system message.
// Adapters that understand SystemParts can use these blocks to set
// per-block cache control (e.g. Anthropic's cache_control: ephemeral).
type ContentBlock struct {
	Type         string        `json:"type"` // "text"
	Text         string        `json:"text"`
	CacheControl *CacheControl `json:"cache_control,omitempty"`
}

type Message struct {
	Role             string         `json:"role"`
	Content          string         `json:"content"`
	ReasoningContent string         `json:"reasoning_content,omitempty"`
	SystemParts      []ContentBlock `json:"system_parts,omitempty"` // structured system blocks for cache-aware adapters
	ToolCalls        []ToolCall     `json:"tool_calls,omitempty"`
	ToolCallID       string         `json:"tool_call_id,omitempty"`
}

// DeepCopy returns a fully independent copy of the Message, including all
// nested slices, maps, and pointer fields. This is used by the session
// manager to isolate stored history from caller mutations.
func (m Message) DeepCopy() Message {
	cp := m // copies all value fields (Role, Content, etc.)

	if len(m.ToolCalls) > 0 {
		cp.ToolCalls = make([]ToolCall, len(m.ToolCalls))
		for i, tc := range m.ToolCalls {
			cp.ToolCalls[i] = tc
			if tc.Function != nil {
				fn := *tc.Function
				cp.ToolCalls[i].Function = &fn
			}
			if tc.Arguments != nil {
				args := make(map[string]any, len(tc.Arguments))
				for k, v := range tc.Arguments {
					args[k] = v
				}
				cp.ToolCalls[i].Arguments = args
			}
			if tc.ExtraContent != nil {
				ec := *tc.ExtraContent
				if tc.ExtraContent.Google != nil {
					g := *tc.ExtraContent.Google
					ec.Google = &g
				}
				cp.ToolCalls[i].ExtraContent = &ec
			}
		}
	}

	if len(m.SystemParts) > 0 {
		cp.SystemParts = make([]ContentBlock, len(m.SystemParts))
		for i, sp := range m.SystemParts {
			cp.SystemParts[i] = sp
			if sp.CacheControl != nil {
				cc := *sp.CacheControl
				cp.SystemParts[i].CacheControl = &cc
			}
		}
	}

	return cp
}

type ToolDefinition struct {
	Type     string                 `json:"type"`
	Function ToolFunctionDefinition `json:"function"`
}

type ToolFunctionDefinition struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}
