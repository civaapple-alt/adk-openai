package adkopenai

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
	"github.com/openai/openai-go/v3/shared"
	"google.golang.org/genai"

	"google.golang.org/adk/v2/model"
)

func toResponsesRequest(req *model.LLMRequest, modelName string) (responses.ResponseNewParams, error) {
	if req == nil {
		return responses.ResponseNewParams{}, fmt.Errorf("LLMRequest is nil")
	}
	resolved, err := resolveModelName(req, modelName)
	if err != nil {
		return responses.ResponseNewParams{}, err
	}

	input, err := toResponsesInput(req)
	if err != nil {
		return responses.ResponseNewParams{}, err
	}

	out := responses.ResponseNewParams{
		Model: resolved,
		Input: responses.ResponseNewParamsInputUnion{
			OfInputItemList: input,
		},
	}

	cfg := req.Config
	if cfg == nil {
		return out, nil
	}

	if cfg.SystemInstruction != nil {
		if text := extractText(cfg.SystemInstruction); text != "" {
			out.Instructions = openai.String(text)
		}
	}
	if cfg.Temperature != nil {
		out.Temperature = openai.Float(float64(*cfg.Temperature))
	}
	if cfg.MaxOutputTokens > 0 {
		out.MaxOutputTokens = openai.Int(int64(cfg.MaxOutputTokens))
	}
	if cfg.TopP != nil {
		out.TopP = openai.Float(float64(*cfg.TopP))
	}

	if cfg.ThinkingConfig != nil {
		effort := shared.ReasoningEffortMedium
		switch cfg.ThinkingConfig.ThinkingLevel {
		case genai.ThinkingLevelLow:
			effort = shared.ReasoningEffortLow
		case genai.ThinkingLevelHigh:
			effort = shared.ReasoningEffortHigh
		}
		out.Reasoning = shared.ReasoningParam{Effort: effort}
	}

	if cfg.ResponseMIMEType == "application/json" {
		out.Text = responses.ResponseTextConfigParam{
			Format: responses.ResponseFormatTextConfigUnionParam{
				OfJSONObject: &shared.ResponseFormatJSONObjectParam{},
			},
		}
	}
	if cfg.ResponseSchema != nil {
		schema, err := convertSchema(cfg.ResponseSchema)
		if err != nil {
			return responses.ResponseNewParams{}, err
		}
		out.Text = responses.ResponseTextConfigParam{
			Format: responses.ResponseFormatTextConfigUnionParam{
				OfJSONSchema: &responses.ResponseFormatTextJSONSchemaConfigParam{
					Name:        "response",
					Description: openai.String(cfg.ResponseSchema.Description),
					Strict:      openai.Bool(false),
					Schema:      schema,
				},
			},
		}
	}

	if len(cfg.Tools) > 0 {
		tools, err := convertResponsesTools(cfg.Tools)
		if err != nil {
			return responses.ResponseNewParams{}, err
		}
		out.Tools = tools
	}

	if choice, ok, err := responsesToolChoice(cfg.ToolConfig); err != nil {
		return responses.ResponseNewParams{}, err
	} else if ok {
		out.ToolChoice = choice
	}

	return out, nil
}

func toResponsesInput(req *model.LLMRequest) (responses.ResponseInputParam, error) {
	var out responses.ResponseInputParam
	for _, content := range req.Contents {
		items, err := toResponsesInputItems(content)
		if err != nil {
			return nil, err
		}
		out = append(out, items...)
	}
	return out, nil
}

func toResponsesInputItems(content *genai.Content) ([]responses.ResponseInputItemUnionParam, error) {
	if content == nil {
		return nil, nil
	}

	role, err := roleToOpenAI(content.Role)
	if err != nil {
		return nil, err
	}

	var items []responses.ResponseInputItemUnionParam
	var pending []*genai.Part

	flushPending := func() error {
		if len(pending) == 0 {
			return nil
		}
		built, err := buildResponsesItemsFromParts(role, pending)
		if err != nil {
			return err
		}
		items = append(items, built...)
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
			items = append(items, responses.ResponseInputItemParamOfFunctionCallOutput(part.FunctionResponse.ID, string(raw)))
			continue
		}
		pending = append(pending, part)
	}
	if err := flushPending(); err != nil {
		return nil, err
	}
	return items, nil
}

func buildResponsesItemsFromParts(role string, parts []*genai.Part) ([]responses.ResponseInputItemUnionParam, error) {
	var items []responses.ResponseInputItemUnionParam
	var textBuf string
	var contentParts responses.ResponseInputMessageContentListParam
	var functionCalls []responses.ResponseInputItemUnionParam
	hasImage := false

	for _, part := range parts {
		if part == nil {
			continue
		}
		if part.Text != "" {
			textBuf += part.Text
			contentParts = append(contentParts, responses.ResponseInputContentParamOfInputText(part.Text))
		}
		if part.FunctionCall != nil {
			args, err := json.Marshal(part.FunctionCall.Args)
			if err != nil {
				return nil, fmt.Errorf("marshal function args: %w", err)
			}
			functionCalls = append(functionCalls, responses.ResponseInputItemParamOfFunctionCall(
				string(args),
				part.FunctionCall.ID,
				part.FunctionCall.Name,
			))
		}
		if part.InlineData != nil {
			mt := part.InlineData.MIMEType
			switch mt {
			case "image/jpg", "image/jpeg", "image/png", "image/gif", "image/webp":
				b64 := base64.StdEncoding.EncodeToString(part.InlineData.Data)
				img := responses.ResponseInputContentParamOfInputImage(responses.ResponseInputImageDetailAuto)
				img.OfInputImage.ImageURL = openai.String(fmt.Sprintf("data:%s;base64,%s", mt, b64))
				contentParts = append(contentParts, img)
				hasImage = true
			default:
				return nil, fmt.Errorf("unsupported inline MIME type: %s", mt)
			}
		}
	}

	// Assistant prior function calls are sent as standalone input items.
	if role == "assistant" && len(functionCalls) > 0 {
		if textBuf != "" {
			items = append(items, responses.ResponseInputItemParamOfMessage(textBuf, responses.EasyInputMessageRoleAssistant))
		}
		items = append(items, functionCalls...)
		return items, nil
	}

	if hasImage || len(contentParts) > 1 {
		items = append(items, responses.ResponseInputItemParamOfMessage(contentParts, easyRole(role)))
	} else if textBuf != "" || role == "user" || role == "system" || role == "assistant" || role == "developer" {
		items = append(items, responses.ResponseInputItemParamOfMessage(textBuf, easyRole(role)))
	}

	return items, nil
}

func easyRole(role string) responses.EasyInputMessageRole {
	switch role {
	case "system":
		return responses.EasyInputMessageRoleSystem
	case "developer":
		return responses.EasyInputMessageRoleDeveloper
	case "assistant":
		return responses.EasyInputMessageRoleAssistant
	default:
		return responses.EasyInputMessageRoleUser
	}
}

func convertResponsesTools(genaiTools []*genai.Tool) ([]responses.ToolUnionParam, error) {
	var out []responses.ToolUnionParam
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
			tool := responses.ToolParamOfFunction(decl.Name, params, false)
			if decl.Description != "" {
				tool.OfFunction.Description = openai.String(decl.Description)
			}
			out = append(out, tool)
		}
	}
	return out, nil
}
