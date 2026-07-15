package adkopenai

import (
	"context"
	"iter"
	"sort"

	"github.com/openai/openai-go/v3"
	"google.golang.org/genai"

	"google.golang.org/adk/v2/model"
)

func (m *Model) generateChatStream(ctx context.Context, req *model.LLMRequest, modelName string) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		openaiReq, err := toChatRequest(req, modelName)
		if err != nil {
			yield(nil, err)
			return
		}
		openaiReq.StreamOptions = openai.ChatCompletionStreamOptionsParam{
			IncludeUsage: openai.Bool(true),
		}

		stream := m.client.Chat.Completions.NewStreaming(ctx, openaiReq)
		defer stream.Close()

		aggregated := &genai.Content{Role: genai.RoleModel, Parts: []*genai.Part{}}
		var finishReason genai.FinishReason
		var usage *genai.GenerateContentResponseUsageMetadata
		toolCalls := map[int]*toolCallBuilder{}
		lastPartIsText := false
		var refusalBuf string

		for stream.Next() {
			chunk := stream.Current()
			if len(chunk.Choices) == 0 {
				if chunk.Usage.TotalTokens > 0 {
					usage = convertChatUsage(chunk.Usage)
				}
				continue
			}
			choice := chunk.Choices[0]
			delta := choice.Delta

			if delta.Content != "" {
				part := &genai.Part{Text: delta.Content}
				if lastPartIsText {
					aggregated.Parts[len(aggregated.Parts)-1].Text += part.Text
				} else {
					aggregated.Parts = append(aggregated.Parts, part)
				}
				lastPartIsText = true
				if !yield(&model.LLMResponse{
					Content:      newContent(part.Text),
					Partial:      true,
					TurnComplete: false,
				}, nil) {
					return
				}
			} else {
				lastPartIsText = false
			}

			if delta.Refusal != "" {
				refusalBuf += delta.Refusal
			}

			for _, tc := range delta.ToolCalls {
				idx := int(tc.Index)
				if idx < 0 {
					idx = 0
				}
				b, ok := toolCalls[idx]
				if !ok {
					b = &toolCallBuilder{}
					toolCalls[idx] = b
				}
				if tc.ID != "" {
					b.id = tc.ID
				}
				if tc.Function.Name != "" {
					b.name = tc.Function.Name
				}
				b.args += tc.Function.Arguments
			}

			if choice.FinishReason != "" {
				finishReason = convertFinishReason(choice.FinishReason)
			}
			if chunk.Usage.TotalTokens > 0 {
				usage = convertChatUsage(chunk.Usage)
			}
		}

		if err := stream.Err(); err != nil {
			yield(nil, err)
			return
		}

		if refusalBuf != "" {
			aggregated.Parts = append(aggregated.Parts, &genai.Part{Text: refusalBuf})
			finishReason = genai.FinishReasonSafety
		}

		if len(toolCalls) > 0 {
			indices := make([]int, 0, len(toolCalls))
			for i := range toolCalls {
				indices = append(indices, i)
			}
			sort.Ints(indices)
			completed := append([]*genai.Part(nil), aggregated.Parts...)
			for _, i := range indices {
				b := toolCalls[i]
				part, err := functionCallFromArgs(b.id, b.name, b.args, finishReason, completed)
				if err != nil {
					yield(nil, err)
					return
				}
				aggregated.Parts = append(aggregated.Parts, part)
				completed = append(completed, part)
			}
		}

		yield(&model.LLMResponse{
			Content:       aggregated,
			UsageMetadata: usage,
			FinishReason:  finishReason,
			Partial:       false,
			TurnComplete:  true,
		}, nil)
	}
}
