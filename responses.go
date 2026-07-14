package adkopenai

import (
	"context"
	"iter"

	"google.golang.org/adk/v2/model"
)

func (m *Model) generateResponses(ctx context.Context, req *model.LLMRequest) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		params, err := toResponsesRequest(req, m.modelName)
		if err != nil {
			yield(nil, err)
			return
		}

		resp, err := m.client.Responses.New(ctx, params)
		if err != nil {
			yield(nil, err)
			return
		}

		llmResp, err := convertResponsesResult(resp)
		if err != nil {
			yield(nil, err)
			return
		}
		yield(llmResp, nil)
	}
}
