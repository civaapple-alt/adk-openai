package adkopenai

import (
	"fmt"

	"github.com/openai/openai-go/v3/responses"
	"google.golang.org/genai"

	"google.golang.org/adk/v2/model"
)

func convertResponsesResult(resp *responses.Response) (*model.LLMResponse, error) {
	if resp == nil {
		return nil, fmt.Errorf("empty responses API result")
	}

	content := &genai.Content{Role: genai.RoleModel, Parts: []*genai.Part{}}
	var refusal string
	type pendingCall struct {
		id, name, args string
	}
	var calls []pendingCall

	for _, item := range resp.Output {
		switch item.Type {
		case "message":
			for _, part := range item.Content {
				switch part.Type {
				case "output_text":
					if part.Text != "" {
						content.Parts = append(content.Parts, &genai.Part{Text: part.Text})
					}
				case "refusal":
					if part.Refusal != "" {
						refusal = part.Refusal
						content.Parts = append(content.Parts, &genai.Part{Text: part.Refusal})
					}
				}
			}
		case "function_call":
			calls = append(calls, pendingCall{
				id:   item.CallID,
				name: item.Name,
				args: item.Arguments.OfString,
			})
		}
	}

	// Fallback to SDK helper when typed output parsing finds no text.
	if len(content.Parts) == 0 && len(calls) == 0 {
		if text := resp.OutputText(); text != "" {
			content.Parts = append(content.Parts, &genai.Part{Text: text})
		}
	}

	finishReason := convertResponsesFinishReason(resp, refusal != "")
	completed := append([]*genai.Part(nil), content.Parts...)
	for _, call := range calls {
		part, err := functionCallFromArgs(call.id, call.name, call.args, finishReason, completed)
		if err != nil {
			return nil, err
		}
		content.Parts = append(content.Parts, part)
		completed = append(completed, part)
	}

	llmResp := &model.LLMResponse{
		Content:       content,
		UsageMetadata: convertResponsesUsage(resp.Usage),
		FinishReason:  finishReason,
		TurnComplete:  true,
	}
	applyResponsesErrorFields(llmResp, resp)
	return llmResp, nil
}

func convertResponsesUsage(usage responses.ResponseUsage) *genai.GenerateContentResponseUsageMetadata {
	if usage.TotalTokens == 0 {
		return nil
	}
	return &genai.GenerateContentResponseUsageMetadata{
		PromptTokenCount:     int32(usage.InputTokens),
		CandidatesTokenCount: int32(usage.OutputTokens),
		TotalTokenCount:      int32(usage.TotalTokens),
	}
}

func convertResponsesFinishReason(resp *responses.Response, refused bool) genai.FinishReason {
	if refused {
		return genai.FinishReasonSafety
	}
	switch string(resp.Status) {
	case "completed":
		// Tool calls also complete with status completed.
		return genai.FinishReasonStop
	case "incomplete":
		if resp.IncompleteDetails.Reason == "max_output_tokens" {
			return genai.FinishReasonMaxTokens
		}
		if resp.IncompleteDetails.Reason == "content_filter" {
			return genai.FinishReasonSafety
		}
		return genai.FinishReasonUnspecified
	case "failed", "cancelled":
		return genai.FinishReasonUnspecified
	default:
		return convertFinishReason(string(resp.Status))
	}
}

func applyResponsesErrorFields(llmResp *model.LLMResponse, resp *responses.Response) {
	switch string(resp.Status) {
	case "failed", "cancelled":
		code := string(resp.Status)
		if resp.Error.Code != "" {
			code = string(resp.Error.Code)
		}
		llmResp.ErrorCode = code
		if resp.Error.Message != "" {
			llmResp.ErrorMessage = resp.Error.Message
		} else {
			llmResp.ErrorMessage = string(resp.Status)
		}
	}
}
