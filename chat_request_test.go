package adkopenai

import (
	"encoding/json"
	"testing"

	"github.com/openai/openai-go/v3/packages/param"
	"google.golang.org/genai"

	"google.golang.org/adk/v2/model"
)

func TestToChatRequestToolConfig(t *testing.T) {
	base := &model.LLMRequest{
		Contents: []*genai.Content{genai.NewContentFromText("hi", genai.RoleUser)},
		Config: &genai.GenerateContentConfig{
			Tools: []*genai.Tool{{
				FunctionDeclarations: []*genai.FunctionDeclaration{{
					Name: "lookup",
					Parameters: &genai.Schema{
						Type:       genai.TypeObject,
						Properties: map[string]*genai.Schema{"q": {Type: genai.TypeString}},
					},
				}},
			}},
		},
	}

	t.Run("none", func(t *testing.T) {
		req := cloneLLMRequest(base)
		req.Config.ToolConfig = &genai.ToolConfig{
			FunctionCallingConfig: &genai.FunctionCallingConfig{Mode: genai.FunctionCallingConfigModeNone},
		}
		got, err := toChatRequest(req, "gpt-test")
		if err != nil {
			t.Fatal(err)
		}
		if param.IsOmitted(got.ToolChoice.OfAuto) || got.ToolChoice.OfAuto.Value != "none" {
			t.Fatalf("tool_choice = %#v", got.ToolChoice)
		}
	})

	t.Run("required", func(t *testing.T) {
		req := cloneLLMRequest(base)
		req.Config.ToolConfig = &genai.ToolConfig{
			FunctionCallingConfig: &genai.FunctionCallingConfig{Mode: genai.FunctionCallingConfigModeAny},
		}
		got, err := toChatRequest(req, "gpt-test")
		if err != nil {
			t.Fatal(err)
		}
		if param.IsOmitted(got.ToolChoice.OfAuto) || got.ToolChoice.OfAuto.Value != "required" {
			t.Fatalf("tool_choice = %#v", got.ToolChoice)
		}
	})

	t.Run("named", func(t *testing.T) {
		req := cloneLLMRequest(base)
		req.Config.ToolConfig = &genai.ToolConfig{
			FunctionCallingConfig: &genai.FunctionCallingConfig{
				Mode:                 genai.FunctionCallingConfigModeAny,
				AllowedFunctionNames: []string{"lookup"},
			},
		}
		got, err := toChatRequest(req, "gpt-test")
		if err != nil {
			t.Fatal(err)
		}
		if got.ToolChoice.OfFunctionToolChoice == nil || got.ToolChoice.OfFunctionToolChoice.Function.Name != "lookup" {
			t.Fatalf("tool_choice = %#v", got.ToolChoice)
		}
	})

	t.Run("multi reject", func(t *testing.T) {
		req := cloneLLMRequest(base)
		req.Config.ToolConfig = &genai.ToolConfig{
			FunctionCallingConfig: &genai.FunctionCallingConfig{
				Mode:                 genai.FunctionCallingConfigModeAny,
				AllowedFunctionNames: []string{"a", "b"},
			},
		}
		if _, err := toChatRequest(req, "gpt-test"); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestToChatMessagesMixedFunctionResponse(t *testing.T) {
	content := &genai.Content{
		Role: "user",
		Parts: []*genai.Part{
			{Text: "thanks"},
			{
				FunctionResponse: &genai.FunctionResponse{
					ID:       "call_1",
					Name:     "lookup",
					Response: map[string]any{"ok": true},
				},
			},
			{Text: "continue"},
		},
	}
	msgs, err := toChatMessages(content)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 3 {
		t.Fatalf("got %d messages: %#v", len(msgs), msgs)
	}
	if msgs[0].OfUser == nil {
		t.Fatalf("msg0 = %#v", msgs[0])
	}
	if msgs[1].OfTool == nil || msgs[1].OfTool.ToolCallID != "call_1" {
		t.Fatalf("msg1 = %#v", msgs[1])
	}
	if msgs[2].OfUser == nil {
		t.Fatalf("msg2 = %#v", msgs[2])
	}
}

func TestToChatMessagesRequiresFunctionResponseID(t *testing.T) {
	_, err := toChatMessages(&genai.Content{
		Role: "user",
		Parts: []*genai.Part{{
			FunctionResponse: &genai.FunctionResponse{Name: "lookup", Response: map[string]any{}},
		}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestToChatMessagesDeveloperRole(t *testing.T) {
	msgs, err := toChatMessages(&genai.Content{
		Role:  "developer",
		Parts: []*genai.Part{{Text: "be concise"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 || msgs[0].OfDeveloper == nil {
		t.Fatalf("got %#v", msgs)
	}
}

func TestToChatRequestRejectsUnknownRole(t *testing.T) {
	_, err := toChatRequest(&model.LLMRequest{
		Contents: []*genai.Content{{Role: "narrator", Parts: []*genai.Part{{Text: "x"}}}},
	}, "m")
	if err == nil {
		t.Fatal("expected error")
	}
}

func cloneLLMRequest(in *model.LLMRequest) *model.LLMRequest {
	raw, err := json.Marshal(in)
	if err != nil {
		panic(err)
	}
	var out model.LLMRequest
	if err := json.Unmarshal(raw, &out); err != nil {
		panic(err)
	}
	// Tools and ToolConfig are not always round-tripped via JSON the way we need;
	// copy Config pointers manually for tests that mutate ToolConfig.
	if in.Config != nil {
		cfg := *in.Config
		if in.Config.ToolConfig != nil {
			tc := *in.Config.ToolConfig
			if in.Config.ToolConfig.FunctionCallingConfig != nil {
				fcc := *in.Config.ToolConfig.FunctionCallingConfig
				tc.FunctionCallingConfig = &fcc
			}
			cfg.ToolConfig = &tc
		}
		out.Config = &cfg
	}
	return &out
}
