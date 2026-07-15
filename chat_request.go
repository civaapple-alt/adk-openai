package adkopenai

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/shared"
	"google.golang.org/genai"

	"google.golang.org/adk/v2/model"
)

func toChatRequest(req *model.LLMRequest, modelName string) (openai.ChatCompletionNewParams, error) {
	if req == nil {
		return openai.ChatCompletionNewParams{}, fmt.Errorf("LLMRequest is nil")
	}
	resolved, err := resolveModelName(req, modelName)
	if err != nil {
		return openai.ChatCompletionNewParams{}, err
	}

	messages := make([]openai.ChatCompletionMessageParamUnion, 0, len(req.Contents)+1)
	for _, content := range req.Contents {
		msgs, err := toChatMessages(content)
		if err != nil {
			return openai.ChatCompletionNewParams{}, err
		}
		messages = append(messages, msgs...)
	}

	out := openai.ChatCompletionNewParams{
		Model:    resolved,
		Messages: messages,
	}

	cfg := req.Config
	if cfg == nil {
		return out, nil
	}

	if cfg.SystemInstruction != nil {
		sys := openai.SystemMessage(extractText(cfg.SystemInstruction))
		out.Messages = append([]openai.ChatCompletionMessageParamUnion{sys}, out.Messages...)
	}

	if cfg.Temperature != nil {
		out.Temperature = openai.Float(float64(*cfg.Temperature))
	}
	if cfg.MaxOutputTokens > 0 {
		out.MaxCompletionTokens = openai.Int(int64(cfg.MaxOutputTokens))
	}
	if cfg.TopP != nil {
		out.TopP = openai.Float(float64(*cfg.TopP))
	}
	if len(cfg.StopSequences) > 0 {
		out.Stop = openai.ChatCompletionNewParamsStopUnion{OfStringArray: cfg.StopSequences}
	}

	if cfg.ThinkingConfig != nil {
		switch cfg.ThinkingConfig.ThinkingLevel {
		case genai.ThinkingLevelLow:
			out.ReasoningEffort = shared.ReasoningEffortLow
		case genai.ThinkingLevelHigh:
			out.ReasoningEffort = shared.ReasoningEffortHigh
		default:
			out.ReasoningEffort = shared.ReasoningEffortMedium
		}
	}

	if cfg.ResponseMIMEType == "application/json" {
		out.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: &shared.ResponseFormatJSONObjectParam{},
		}
	}
	if cfg.ResponseSchema != nil {
		schema, err := toChatJSONSchema(cfg.ResponseSchema)
		if err != nil {
			return openai.ChatCompletionNewParams{}, err
		}
		out.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: schema,
		}
	}

	if len(cfg.Tools) > 0 {
		tools, err := convertChatTools(cfg.Tools)
		if err != nil {
			return openai.ChatCompletionNewParams{}, err
		}
		out.Tools = tools
	}

	if choice, ok, err := chatToolChoice(cfg.ToolConfig); err != nil {
		return openai.ChatCompletionNewParams{}, err
	} else if ok {
		out.ToolChoice = choice
	}

	return out, nil
}

func toChatMessages(content *genai.Content) ([]openai.ChatCompletionMessageParamUnion, error) {
	if content == nil {
		return nil, nil
	}

	role, err := roleToOpenAI(content.Role)
	if err != nil {
		return nil, err
	}

	var out []openai.ChatCompletionMessageParamUnion
	var pending []*genai.Part

	flushPending := func() error {
		if len(pending) == 0 {
			return nil
		}
		msg, err := buildChatMessageFromParts(role, pending)
		if err != nil {
			return err
		}
		out = append(out, msg)
		pending = nil
		return nil
	}

	for _, part := range content.Parts {
		if part == nil {
			continue
		}
		if part.FunctionResponse != nil {
			if err := flushPending(); err != nil {
				return nil, err
			}
			if part.FunctionResponse.ID == "" {
				return nil, fmt.Errorf("FunctionResponse.ID is required for tool call correlation (function: %s)", part.FunctionResponse.Name)
			}
			raw, err := json.Marshal(part.FunctionResponse.Response)
			if err != nil {
				return nil, fmt.Errorf("marshal function response: %w", err)
			}
			out = append(out, openai.ToolMessage(string(raw), part.FunctionResponse.ID))
			continue
		}
		pending = append(pending, part)
	}
	if err := flushPending(); err != nil {
		return nil, err
	}
	return out, nil
}

func buildChatMessageFromParts(role string, parts []*genai.Part) (openai.ChatCompletionMessageParamUnion, error) {
	var textBuf string
	var userParts []openai.ChatCompletionContentPartUnionParam
	var assistantParts []openai.ChatCompletionAssistantMessageParamContentArrayOfContentPartUnion
	var toolCalls []openai.ChatCompletionMessageToolCallUnionParam

	for _, part := range parts {
		if part == nil {
			continue
		}
		if part.Text != "" {
			textBuf += part.Text
			userParts = append(userParts, openai.TextContentPart(part.Text))
			assistantParts = append(assistantParts, openai.ChatCompletionAssistantMessageParamContentArrayOfContentPartUnion{
				OfText: &openai.ChatCompletionContentPartTextParam{Text: part.Text},
			})
		}
		if part.FunctionCall != nil {
			args, err := json.Marshal(part.FunctionCall.Args)
			if err != nil {
				return openai.ChatCompletionMessageParamUnion{}, fmt.Errorf("marshal function args: %w", err)
			}
			toolCalls = append(toolCalls, openai.ChatCompletionMessageToolCallUnionParam{
				OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
					ID: part.FunctionCall.ID,
					Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
						Name:      part.FunctionCall.Name,
						Arguments: string(args),
					},
				},
			})
		}
		if part.InlineData != nil {
			mt := part.InlineData.MIMEType
			switch mt {
			case "image/jpg", "image/jpeg", "image/png", "image/gif", "image/webp":
				b64 := base64.StdEncoding.EncodeToString(part.InlineData.Data)
				userParts = append(userParts, openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{
					URL:    fmt.Sprintf("data:%s;base64,%s", mt, b64),
					Detail: "auto",
				}))
			default:
				return openai.ChatCompletionMessageParamUnion{}, fmt.Errorf("unsupported inline MIME type: %s", mt)
			}
		}
	}

	return buildChatMessageForRole(role, textBuf, userParts, assistantParts, toolCalls), nil
}

func buildChatMessageForRole(
	role, text string,
	userParts []openai.ChatCompletionContentPartUnionParam,
	assistantParts []openai.ChatCompletionAssistantMessageParamContentArrayOfContentPartUnion,
	toolCalls []openai.ChatCompletionMessageToolCallUnionParam,
) openai.ChatCompletionMessageParamUnion {
	switch role {
	case "system":
		return openai.SystemMessage(text)
	case "developer":
		return openai.DeveloperMessage(text)
	case "assistant":
		var assistant openai.ChatCompletionMessageParamUnion
		if len(assistantParts) > 0 {
			assistant = openai.AssistantMessage(assistantParts)
		} else {
			assistant = openai.AssistantMessage("")
		}
		assistant.OfAssistant.ToolCalls = toolCalls
		return assistant
	default:
		if len(userParts) > 0 {
			return openai.UserMessage(userParts)
		}
		return openai.UserMessage(text)
	}
}

func convertChatTools(genaiTools []*genai.Tool) ([]openai.ChatCompletionToolUnionParam, error) {
	var out []openai.ChatCompletionToolUnionParam
	for _, t := range genaiTools {
		if t == nil {
			continue
		}
		if t.GoogleSearch != nil || t.CodeExecution != nil || t.FileSearch != nil ||
			t.Retrieval != nil || t.ComputerUse != nil {
			return nil, fmt.Errorf("OpenAI-compatible backend does not support Gemini-native tools (GoogleSearch 等)")
		}
		for _, decl := range t.FunctionDeclarations {
			if decl == nil {
				continue
			}
			params, err := functionParametersFromDecl(decl)
			if err != nil {
				return nil, fmt.Errorf("convert parameters schema for tool %q: %w", decl.Name, err)
			}
			out = append(out, openai.ChatCompletionFunctionTool(shared.FunctionDefinitionParam{
				Name:        decl.Name,
				Description: openai.String(decl.Description),
				Parameters:  shared.FunctionParameters(params),
			}))
		}
	}
	return out, nil
}

func toChatJSONSchema(schema *genai.Schema) (*shared.ResponseFormatJSONSchemaParam, error) {
	converted, err := convertSchema(schema)
	if err != nil {
		return nil, err
	}
	// Prefer non-strict mode to avoid OpenAI strict schema constraints
	// (additionalProperties:false etc.) that break genai.Schema conversion.
	return &shared.ResponseFormatJSONSchemaParam{
		JSONSchema: shared.ResponseFormatJSONSchemaJSONSchemaParam{
			Name:        "response",
			Description: openai.String(schema.Description),
			Strict:      openai.Bool(false),
			Schema:      converted,
		},
	}, nil
}
