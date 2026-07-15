package adkopenai

import (
	"errors"

	"github.com/openai/openai-go/v3"
	"google.golang.org/genai"

	"google.golang.org/adk/v2/model"
)

var errNoChoices = errors.New("no choices in OpenAI response")

func convertChatResponse(resp *openai.ChatCompletion) (*model.LLMResponse, error) {
	if len(resp.Choices) == 0 {
		return nil, errNoChoices
	}
	choice := resp.Choices[0]
	content := &genai.Content{Role: genai.RoleModel, Parts: []*genai.Part{}}

	if choice.Message.Content != "" {
		content.Parts = append(content.Parts, &genai.Part{Text: choice.Message.Content})
	}
	if choice.Message.Refusal != "" {
		content.Parts = append(content.Parts, &genai.Part{Text: choice.Message.Refusal})
	}

	finishReason := convertFinishReason(choice.FinishReason)
	if choice.Message.Refusal != "" {
		finishReason = genai.FinishReasonSafety
	}

	completed := append([]*genai.Part(nil), content.Parts...)
	for _, tc := range choice.Message.ToolCalls {
		if tc.Type != "" && tc.Type != "function" {
			continue
		}
		part, err := functionCallFromArgs(tc.ID, tc.Function.Name, tc.Function.Arguments, finishReason, completed)
		if err != nil {
			return nil, err
		}
		content.Parts = append(content.Parts, part)
		completed = append(completed, part)
	}

	return &model.LLMResponse{
		Content:       content,
		UsageMetadata: convertChatUsage(resp.Usage),
		FinishReason:  finishReason,
		TurnComplete:  true,
	}, nil
}

func convertChatUsage(usage openai.CompletionUsage) *genai.GenerateContentResponseUsageMetadata {
	if usage.TotalTokens == 0 {
		return nil
	}
	return &genai.GenerateContentResponseUsageMetadata{
		PromptTokenCount:     int32(usage.PromptTokens),
		CandidatesTokenCount: int32(usage.CompletionTokens),
		TotalTokenCount:      int32(usage.TotalTokens),
	}
}
