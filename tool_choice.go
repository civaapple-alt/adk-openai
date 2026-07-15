package adkopenai

import (
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/responses"
	"google.golang.org/genai"
)

type toolChoiceKind int

const (
	toolChoiceUnset toolChoiceKind = iota
	toolChoiceNone
	toolChoiceAuto
	toolChoiceRequired
	toolChoiceNamed
)

type resolvedToolChoice struct {
	kind toolChoiceKind
	name string
}

func resolveToolChoice(config *genai.ToolConfig) (resolvedToolChoice, error) {
	if config == nil || config.FunctionCallingConfig == nil {
		return resolvedToolChoice{kind: toolChoiceUnset}, nil
	}

	fcc := config.FunctionCallingConfig
	if len(fcc.AllowedFunctionNames) > 1 {
		return resolvedToolChoice{}, fmt.Errorf(
			"OpenAI-compatible backends do not support multiple AllowedFunctionNames (got %d); use a single function name or remove the restriction",
			len(fcc.AllowedFunctionNames),
		)
	}

	switch fcc.Mode {
	case genai.FunctionCallingConfigModeNone:
		return resolvedToolChoice{kind: toolChoiceNone}, nil
	case genai.FunctionCallingConfigModeAuto:
		return resolvedToolChoice{kind: toolChoiceAuto}, nil
	case genai.FunctionCallingConfigModeAny:
		if len(fcc.AllowedFunctionNames) == 1 {
			return resolvedToolChoice{kind: toolChoiceNamed, name: fcc.AllowedFunctionNames[0]}, nil
		}
		return resolvedToolChoice{kind: toolChoiceRequired}, nil
	case "", genai.FunctionCallingConfigModeUnspecified:
		return resolvedToolChoice{kind: toolChoiceUnset}, nil
	default:
		return resolvedToolChoice{}, fmt.Errorf(
			"unsupported FunctionCallingConfig mode %q; supported modes are: ModeNone, ModeAuto, ModeAny",
			fcc.Mode,
		)
	}
}

func chatToolChoice(config *genai.ToolConfig) (openai.ChatCompletionToolChoiceOptionUnionParam, bool, error) {
	resolved, err := resolveToolChoice(config)
	if err != nil {
		return openai.ChatCompletionToolChoiceOptionUnionParam{}, false, err
	}
	switch resolved.kind {
	case toolChoiceUnset:
		return openai.ChatCompletionToolChoiceOptionUnionParam{}, false, nil
	case toolChoiceNone:
		return openai.ChatCompletionToolChoiceOptionUnionParam{
			OfAuto: openai.String(string(openai.ChatCompletionToolChoiceOptionAutoNone)),
		}, true, nil
	case toolChoiceAuto:
		return openai.ChatCompletionToolChoiceOptionUnionParam{
			OfAuto: openai.String(string(openai.ChatCompletionToolChoiceOptionAutoAuto)),
		}, true, nil
	case toolChoiceRequired:
		return openai.ChatCompletionToolChoiceOptionUnionParam{
			OfAuto: openai.String(string(openai.ChatCompletionToolChoiceOptionAutoRequired)),
		}, true, nil
	case toolChoiceNamed:
		return openai.ToolChoiceOptionFunctionToolChoice(openai.ChatCompletionNamedToolChoiceFunctionParam{
			Name: resolved.name,
		}), true, nil
	default:
		return openai.ChatCompletionToolChoiceOptionUnionParam{}, false, fmt.Errorf("unexpected tool choice kind: %d", resolved.kind)
	}
}

func responsesToolChoice(config *genai.ToolConfig) (responses.ResponseNewParamsToolChoiceUnion, bool, error) {
	resolved, err := resolveToolChoice(config)
	if err != nil {
		return responses.ResponseNewParamsToolChoiceUnion{}, false, err
	}
	switch resolved.kind {
	case toolChoiceUnset:
		return responses.ResponseNewParamsToolChoiceUnion{}, false, nil
	case toolChoiceNone:
		return responses.ResponseNewParamsToolChoiceUnion{
			OfToolChoiceMode: openai.Opt(responses.ToolChoiceOptionsNone),
		}, true, nil
	case toolChoiceAuto:
		return responses.ResponseNewParamsToolChoiceUnion{
			OfToolChoiceMode: openai.Opt(responses.ToolChoiceOptionsAuto),
		}, true, nil
	case toolChoiceRequired:
		return responses.ResponseNewParamsToolChoiceUnion{
			OfToolChoiceMode: openai.Opt(responses.ToolChoiceOptionsRequired),
		}, true, nil
	case toolChoiceNamed:
		return responses.ResponseNewParamsToolChoiceUnion{
			OfFunctionTool: &responses.ToolChoiceFunctionParam{Name: resolved.name},
		}, true, nil
	default:
		return responses.ResponseNewParamsToolChoiceUnion{}, false, fmt.Errorf("unexpected tool choice kind: %d", resolved.kind)
	}
}
