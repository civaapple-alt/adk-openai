package adkopenai

import (
	"encoding/json"
	"strings"

	"google.golang.org/genai"
)

func extractText(content *genai.Content) string {
	if content == nil {
		return ""
	}
	var parts []string
	for _, p := range content.Parts {
		if p != nil && p.Text != "" {
			parts = append(parts, p.Text)
		}
	}
	return strings.Join(parts, "\n")
}

func parseJSONArgs(s string) map[string]any {
	if s == "" {
		return map[string]any{}
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(s), &args); err != nil {
		return map[string]any{}
	}
	return args
}

func newContent(text string) *genai.Content {
	return &genai.Content{
		Role:  genai.RoleModel,
		Parts: []*genai.Part{{Text: text}},
	}
}

func roleToOpenAI(role string) string {
	switch role {
	case "user":
		return "user"
	case "model", "assistant":
		return "assistant"
	case "system":
		return "system"
	default:
		return "user"
	}
}

func convertFinishReason(reason string) genai.FinishReason {
	switch reason {
	case "stop", "tool_calls", "function_call", "completed":
		return genai.FinishReasonStop
	case "length", "max_output_tokens":
		return genai.FinishReasonMaxTokens
	case "content_filter":
		return genai.FinishReasonSafety
	default:
		return genai.FinishReasonUnspecified
	}
}

type toolCallBuilder struct {
	id, name, args string
}
