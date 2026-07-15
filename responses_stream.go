package adkopenai

import (
	"context"
	"fmt"
	"iter"

	"google.golang.org/genai"

	"google.golang.org/adk/v2/model"
)

func (m *Model) generateResponsesStream(ctx context.Context, req *model.LLMRequest, modelName string) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		params, err := toResponsesRequest(req, modelName)
		if err != nil {
			yield(nil, err)
			return
		}

		stream := m.client.Responses.NewStreaming(ctx, params)
		defer stream.Close()

		aggregated := &genai.Content{Role: genai.RoleModel, Parts: []*genai.Part{}}
		var finishReason genai.FinishReason
		var usage *genai.GenerateContentResponseUsageMetadata
		toolCalls := newOrderedToolCalls()
		var refusalBuf string
		lastPartIsText := false
		var finalErrorCode, finalErrorMessage string

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
					b := toolCalls.get(id)
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
				b := toolCalls.get(event.ItemID)
				b.args += event.Delta

			case "response.function_call_arguments.done":
				lastPartIsText = false
				b := toolCalls.get(event.ItemID)
				if event.Arguments != "" {
					b.args = event.Arguments
				}
				if event.Name != "" {
					b.name = event.Name
				}

			case "response.completed", "response.incomplete", "response.failed", "response.cancelled":
				lastPartIsText = false
				if event.Response.ID != "" || event.Type == "response.failed" || event.Type == "response.cancelled" {
					finishReason = convertResponsesFinishReason(&event.Response, refusalBuf != "")
					usage = convertResponsesUsage(event.Response.Usage)
					tmp := &model.LLMResponse{}
					applyResponsesErrorFields(tmp, &event.Response)
					finalErrorCode = tmp.ErrorCode
					finalErrorMessage = tmp.ErrorMessage
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
						} else {
							yield(nil, convErr)
							return
						}
					}
					if event.Type == "response.failed" || event.Type == "response.cancelled" {
						llmResp := &model.LLMResponse{
							Content:       aggregated,
							UsageMetadata: usage,
							FinishReason:  finishReason,
							ErrorCode:     finalErrorCode,
							ErrorMessage:  finalErrorMessage,
							TurnComplete:  true,
						}
						if llmResp.ErrorCode == "" {
							llmResp.ErrorCode = string(event.Response.Status)
						}
						if llmResp.ErrorMessage == "" {
							llmResp.ErrorMessage = string(event.Response.Status)
						}
						yield(llmResp, nil)
						return
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
			finishReason = genai.FinishReasonSafety
		}

		completed := append([]*genai.Part(nil), aggregated.Parts...)
		for _, b := range toolCalls.builders() {
			if b.name == "" && b.args == "" {
				continue
			}
			part, err := functionCallFromArgs(b.id, b.name, b.args, finishReason, completed)
			if err != nil {
				yield(nil, err)
				return
			}
			aggregated.Parts = append(aggregated.Parts, part)
			completed = append(completed, part)
		}

		yield(&model.LLMResponse{
			Content:       aggregated,
			UsageMetadata: usage,
			FinishReason:  finishReason,
			ErrorCode:     finalErrorCode,
			ErrorMessage:  finalErrorMessage,
			Partial:       false,
			TurnComplete:  true,
		}, nil)
	}
}
