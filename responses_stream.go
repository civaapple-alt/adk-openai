package adkopenai

import (
	"context"
	"fmt"
	"iter"

	"google.golang.org/genai"

	"google.golang.org/adk/v2/model"
)

func (m *Model) generateResponsesStream(ctx context.Context, req *model.LLMRequest) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		params, err := toResponsesRequest(req, m.modelName)
		if err != nil {
			yield(nil, err)
			return
		}

		stream := m.client.Responses.NewStreaming(ctx, params)
		defer stream.Close()

		aggregated := &genai.Content{Role: genai.RoleModel, Parts: []*genai.Part{}}
		var finishReason genai.FinishReason
		var usage *genai.GenerateContentResponseUsageMetadata
		toolCalls := map[string]*toolCallBuilder{} // keyed by item_id
		var refusalBuf string
		lastPartIsText := false

		for stream.Next() {
			event := stream.Current()
			switch event.Type {
			case "response.output_text.delta":
				if event.Delta == "" {
					continue
				}
				part := &genai.Part{Text: event.Delta}
				if lastPartIsText && len(aggregated.Parts) > 0 {
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

			case "response.refusal.delta":
				lastPartIsText = false
				refusalBuf += event.Delta

			case "response.refusal.done":
				lastPartIsText = false
				if event.Refusal != "" {
					refusalBuf = event.Refusal
				}

			case "response.output_item.added", "response.output_item.done":
				lastPartIsText = false
				if event.Item.Type == "function_call" {
					id := event.Item.ID
					if id == "" {
						id = event.Item.CallID
					}
					b, ok := toolCalls[id]
					if !ok {
						b = &toolCallBuilder{}
						toolCalls[id] = b
					}
					if event.Item.CallID != "" {
						b.id = event.Item.CallID
					}
					if event.Item.Name != "" {
						b.name = event.Item.Name
					}
					if event.Item.Arguments.OfString != "" {
						b.args = event.Item.Arguments.OfString
					}
				}

			case "response.function_call_arguments.delta":
				lastPartIsText = false
				id := event.ItemID
				b, ok := toolCalls[id]
				if !ok {
					b = &toolCallBuilder{}
					toolCalls[id] = b
				}
				b.args += event.Delta

			case "response.function_call_arguments.done":
				lastPartIsText = false
				id := event.ItemID
				b, ok := toolCalls[id]
				if !ok {
					b = &toolCallBuilder{}
					toolCalls[id] = b
				}
				if event.Arguments != "" {
					b.args = event.Arguments
				}
				if event.Name != "" {
					b.name = event.Name
				}

			case "response.completed", "response.incomplete", "response.failed":
				lastPartIsText = false
				if event.Response.ID != "" {
					finishReason = convertResponsesFinishReason(&event.Response, refusalBuf != "")
					usage = convertResponsesUsage(event.Response.Usage)
					// Prefer final response conversion when available.
					if event.Type == "response.completed" || event.Type == "response.incomplete" {
						if llmResp, convErr := convertResponsesResult(&event.Response); convErr == nil {
							if refusalBuf != "" && llmResp.Content != nil {
								hasRefusal := false
								for _, p := range llmResp.Content.Parts {
									if p != nil && p.Text == refusalBuf {
										hasRefusal = true
										break
									}
								}
								if !hasRefusal {
									llmResp.Content.Parts = append(llmResp.Content.Parts, &genai.Part{Text: refusalBuf})
								}
							}
							yield(llmResp, nil)
							return
						}
					}
				}

			case "error":
				yield(nil, fmt.Errorf("responses stream error: %s (%s)", event.Message, event.Code))
				return
			}
		}

		if err := stream.Err(); err != nil {
			yield(nil, err)
			return
		}

		if refusalBuf != "" {
			aggregated.Parts = append(aggregated.Parts, &genai.Part{Text: refusalBuf})
			if finishReason == "" || finishReason == genai.FinishReasonUnspecified {
				finishReason = genai.FinishReasonSafety
			}
		}

		for _, b := range toolCalls {
			if b.name == "" && b.args == "" {
				continue
			}
			aggregated.Parts = append(aggregated.Parts, &genai.Part{
				FunctionCall: &genai.FunctionCall{
					ID:   b.id,
					Name: b.name,
					Args: parseJSONArgs(b.args),
				},
			})
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
