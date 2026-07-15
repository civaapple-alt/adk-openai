package adkopenai

import (
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/genai"

	"google.golang.org/adk/v2/model"
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

func parseJSONArgs(s string) (map[string]any, error) {
	if s == "" {
		return map[string]any{}, nil
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(s), &args); err != nil {
		return nil, err
	}
	if args == nil {
		return map[string]any{}, nil
	}
	return args, nil
}

func functionCallFromArgs(id, name, rawArgs string, finishReason genai.FinishReason, completedParts []*genai.Part) (*genai.Part, error) {
	args, err := parseJSONArgs(rawArgs)
	if err != nil {
		if finishReason == genai.FinishReasonMaxTokens {
			return nil, &OutputInterruptedError{
				ToolName:     name,
				ToolID:       id,
				PartialInput: rawArgs,
				FinishReason: finishReason,
				Parts:        completedParts,
				Cause:        err,
			}
		}
		return nil, &InvalidToolArgumentsError{
			ToolName:     name,
			ToolID:       id,
			RawArguments: rawArgs,
			Cause:        err,
		}
	}
	return &genai.Part{
		FunctionCall: &genai.FunctionCall{
			ID:   id,
			Name: name,
			Args: args,
		},
	}, nil
}

func newContent(text string) *genai.Content {
	return &genai.Content{
		Role:  genai.RoleModel,
		Parts: []*genai.Part{{Text: text}},
	}
}

func roleToOpenAI(role string) (string, error) {
	switch role {
	case "", "user":
		return "user", nil
	case "model", "assistant":
		return "assistant", nil
	case "system":
		return "system", nil
	case "developer":
		return "developer", nil
	default:
		return "", fmt.Errorf("unsupported role: %s", role)
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

func resolveModelName(req *model.LLMRequest, defaultName string) (string, error) {
	name := strings.TrimSpace(defaultName)
	if req != nil && strings.TrimSpace(req.Model) != "" {
		name = strings.TrimSpace(req.Model)
	}
	if name == "" {
		return "", fmt.Errorf("model name is required")
	}
	return name, nil
}

func validateGenerateRequest(apiMode APIMode, req *model.LLMRequest, defaultModel string) (string, error) {
	if req == nil {
		return "", fmt.Errorf("LLMRequest is nil")
	}
	switch apiMode {
	case APIModeChatCompletions, APIModeResponses:
	default:
		return "", fmt.Errorf("unsupported API mode %q (want %q or %q)", apiMode, APIModeChatCompletions, APIModeResponses)
	}
	return resolveModelName(req, defaultModel)
}

type toolCallBuilder struct {
	id, name, args string
}

type orderedToolCalls struct {
	order []string
	byID  map[string]*toolCallBuilder
}

func newOrderedToolCalls() *orderedToolCalls {
	return &orderedToolCalls{byID: make(map[string]*toolCallBuilder)}
}

func (o *orderedToolCalls) get(id string) *toolCallBuilder {
	if id == "" {
		id = fmt.Sprintf("_anon_%d", len(o.order))
	}
	if b, ok := o.byID[id]; ok {
		return b
	}
	b := &toolCallBuilder{}
	o.byID[id] = b
	o.order = append(o.order, id)
	return b
}

func (o *orderedToolCalls) builders() []*toolCallBuilder {
	out := make([]*toolCallBuilder, 0, len(o.order))
	for _, id := range o.order {
		out = append(out, o.byID[id])
	}
	return out
}
