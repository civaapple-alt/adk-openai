package adkopenai

import (
	"errors"
	"testing"

	"google.golang.org/genai"

	"google.golang.org/adk/v2/model"
)

func TestRoleToOpenAI(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"", "user", false},
		{"user", "user", false},
		{"model", "assistant", false},
		{"assistant", "assistant", false},
		{"system", "system", false},
		{"developer", "developer", false},
		{"tool", "", true},
	}
	for _, tt := range tests {
		got, err := roleToOpenAI(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("roleToOpenAI(%q) expected error", tt.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("roleToOpenAI(%q) error = %v", tt.in, err)
		}
		if got != tt.want {
			t.Errorf("roleToOpenAI(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestParseJSONArgs(t *testing.T) {
	got, err := parseJSONArgs(`{"a":1}`)
	if err != nil {
		t.Fatal(err)
	}
	if got["a"] != float64(1) {
		t.Fatalf("got %#v", got)
	}
	if _, err := parseJSONArgs(`{`); err == nil {
		t.Fatal("expected error for malformed JSON")
	}
	empty, err := parseJSONArgs("")
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty = %#v err=%v", empty, err)
	}
}

func TestFunctionCallFromArgsInterrupted(t *testing.T) {
	_, err := functionCallFromArgs("c1", "lookup", `{"q":`, genai.FinishReasonMaxTokens, nil)
	var interrupted *OutputInterruptedError
	if !errors.As(err, &interrupted) {
		t.Fatalf("got %T %v", err, err)
	}
	if interrupted.ToolName != "lookup" || interrupted.ToolID != "c1" {
		t.Fatalf("%#v", interrupted)
	}

	_, err = functionCallFromArgs("c1", "lookup", `{"q":`, genai.FinishReasonStop, nil)
	var invalid *InvalidToolArgumentsError
	if !errors.As(err, &invalid) {
		t.Fatalf("got %T %v", err, err)
	}
}

func TestValidateGenerateRequest(t *testing.T) {
	if _, err := validateGenerateRequest(APIModeChatCompletions, nil, "m"); err == nil {
		t.Fatal("expected nil request error")
	}
	if _, err := validateGenerateRequest(APIModeChatCompletions, &model.LLMRequest{}, ""); err == nil {
		t.Fatal("expected empty model error")
	}
	if _, err := validateGenerateRequest(APIMode("bogus"), &model.LLMRequest{}, "m"); err == nil {
		t.Fatal("expected bad api mode error")
	}
	got, err := validateGenerateRequest(APIModeResponses, &model.LLMRequest{Model: "override"}, "default")
	if err != nil || got != "override" {
		t.Fatalf("got %q err=%v", got, err)
	}
}

func TestResolveToolChoice(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *genai.ToolConfig
		want    toolChoiceKind
		wantErr bool
	}{
		{"nil", nil, toolChoiceUnset, false},
		{"none", &genai.ToolConfig{FunctionCallingConfig: &genai.FunctionCallingConfig{Mode: genai.FunctionCallingConfigModeNone}}, toolChoiceNone, false},
		{"auto", &genai.ToolConfig{FunctionCallingConfig: &genai.FunctionCallingConfig{Mode: genai.FunctionCallingConfigModeAuto}}, toolChoiceAuto, false},
		{"any", &genai.ToolConfig{FunctionCallingConfig: &genai.FunctionCallingConfig{Mode: genai.FunctionCallingConfigModeAny}}, toolChoiceRequired, false},
		{"named", &genai.ToolConfig{FunctionCallingConfig: &genai.FunctionCallingConfig{Mode: genai.FunctionCallingConfigModeAny, AllowedFunctionNames: []string{"f"}}}, toolChoiceNamed, false},
		{"multi", &genai.ToolConfig{FunctionCallingConfig: &genai.FunctionCallingConfig{Mode: genai.FunctionCallingConfigModeAny, AllowedFunctionNames: []string{"a", "b"}}}, toolChoiceUnset, true},
		{"validated", &genai.ToolConfig{FunctionCallingConfig: &genai.FunctionCallingConfig{Mode: genai.FunctionCallingConfigModeValidated}}, toolChoiceUnset, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveToolChoice(tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.kind != tt.want {
				t.Fatalf("kind = %d, want %d", got.kind, tt.want)
			}
		})
	}
}
