package adkopenai

import (
	"context"
	"iter"

	"google.golang.org/adk/v2/model"
)

func (m *Model) generateChat(ctx context.Context, req *model.LLMRequest) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		openaiReq, err := toChatRequest(req, m.modelName)
		if err != nil {
			yield(nil, err)
			return
		}

		resp, err := m.client.Chat.Completions.New(ctx, openaiReq)
		if err != nil {
			yield(nil, err)
			return
		}

		llmResp, err := convertChatResponse(resp)
		if err != nil {
			yield(nil, err)
			return
		}
		yield(llmResp, nil)
	}
}
