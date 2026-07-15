package adkopenai

import (
	"testing"

	"github.com/openai/openai-go/v3"
	"google.golang.org/genai"
)

func TestConvertChatResponseHandlesRefusal(t *testing.T) {
	resp := &openai.ChatCompletion{
		Choices: []openai.ChatCompletionChoice{{
			Message: openai.ChatCompletionMessage{
				Refusal: "I cannot help with that.",
			},
			FinishReason: "stop",
		}},
		Usage: openai.CompletionUsage{PromptTokens: 1, CompletionTokens: 2, TotalTokens: 3},
	}
	got, err := convertChatResponse(resp)
	if err != nil {
		t.Fatalf("convertChatResponse() error = %v", err)
	}
	if got.Content == nil || len(got.Content.Parts) == 0 || got.Content.Parts[0].Text != "I cannot help with that." {
		t.Fatalf("content = %#v", got.Content)
	}
	if got.UsageMetadata == nil || got.UsageMetadata.TotalTokenCount != 3 {
		t.Fatalf("usage = %#v", got.UsageMetadata)
	}
	if got.FinishReason != genai.FinishReasonSafety {
		t.Fatalf("finish = %v, want Safety", got.FinishReason)
	}
}

func TestConvertChatResponseInvalidToolArgs(t *testing.T) {
	resp := &openai.ChatCompletion{
		Choices: []openai.ChatCompletionChoice{{
			Message: openai.ChatCompletionMessage{
				ToolCalls: []openai.ChatCompletionMessageToolCallUnion{{
					ID:   "c1",
					Type: "function",
					Function: openai.ChatCompletionMessageFunctionToolCallFunction{
						Name:      "lookup",
						Arguments: `{`,
					},
				}},
			},
			FinishReason: "stop",
		}},
	}
	_, err := convertChatResponse(resp)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestConvertFinishReason(t *testing.T) {
	if got := convertFinishReason("content_filter"); got != genai.FinishReasonSafety {
		t.Errorf("got %v", got)
	}
	if got := convertFinishReason("length"); got != genai.FinishReasonMaxTokens {
		t.Errorf("got %v", got)
	}
}
