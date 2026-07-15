package adkopenai

import (
	"testing"

	"github.com/openai/openai-go/v3/packages/param"
	"google.golang.org/genai"

	"google.golang.org/adk/v2/model"
)

func TestToResponsesRequestToolConfig(t *testing.T) {
	req := &model.LLMRequest{
		Contents: []*genai.Content{genai.NewContentFromText("hi", genai.RoleUser)},
		Config: &genai.GenerateContentConfig{
			Tools: []*genai.Tool{{
				FunctionDeclarations: []*genai.FunctionDeclaration{{Name: "lookup"}},
			}},
			ToolConfig: &genai.ToolConfig{
				FunctionCallingConfig: &genai.FunctionCallingConfig{Mode: genai.FunctionCallingConfigModeNone},
			},
		},
	}
	got, err := toResponsesRequest(req, "gpt-test")
	if err != nil {
		t.Fatal(err)
	}
	if param.IsOmitted(got.ToolChoice.OfToolChoiceMode) || got.ToolChoice.OfToolChoiceMode.Value != "none" {
		t.Fatalf("tool_choice = %#v", got.ToolChoice)
	}

	req.Config.ToolConfig.FunctionCallingConfig = &genai.FunctionCallingConfig{
		Mode:                 genai.FunctionCallingConfigModeAny,
		AllowedFunctionNames: []string{"lookup"},
	}
	got, err = toResponsesRequest(req, "gpt-test")
	if err != nil {
		t.Fatal(err)
	}
	if got.ToolChoice.OfFunctionTool == nil || got.ToolChoice.OfFunctionTool.Name != "lookup" {
		t.Fatalf("tool_choice = %#v", got.ToolChoice)
	}
}

func TestToResponsesInputItemsMixedFunctionResponse(t *testing.T) {
	items, err := toResponsesInputItems(&genai.Content{
		Role: "user",
		Parts: []*genai.Part{
			{Text: "before"},
			{
				FunctionResponse: &genai.FunctionResponse{
					ID:       "call_1",
					Name:     "lookup",
					Response: map[string]any{"ok": true},
				},
			},
			{Text: "after"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("got %d items", len(items))
	}
	if items[1].OfFunctionCallOutput == nil || items[1].OfFunctionCallOutput.CallID != "call_1" {
		t.Fatalf("item1 = %#v", items[1])
	}
}

func TestToResponsesInputItemsDeveloperRole(t *testing.T) {
	items, err := toResponsesInputItems(&genai.Content{
		Role:  "developer",
		Parts: []*genai.Part{{Text: "rules"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].OfMessage == nil {
		t.Fatalf("%#v", items)
	}
}
