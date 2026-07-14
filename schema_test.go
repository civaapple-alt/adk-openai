package adkopenai

import (
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"google.golang.org/genai"
)

func TestConvertChatToolsPreservesTypedJSONSchema(t *testing.T) {
	schema := &jsonschema.Schema{
		Type:     "object",
		Required: []string{"course_id"},
		Properties: map[string]*jsonschema.Schema{
			"course_id": {Type: "string"},
		},
	}
	genaiTools := []*genai.Tool{{
		FunctionDeclarations: []*genai.FunctionDeclaration{{
			Name:                 "get_course_price",
			ParametersJsonSchema: schema,
		}},
	}}

	tools, err := convertChatTools(genaiTools)
	if err != nil {
		t.Fatalf("convertChatTools() error = %v", err)
	}
	if len(tools) != 1 {
		t.Fatalf("convertChatTools() returned %d tools, want 1", len(tools))
	}
	function := tools[0].GetFunction()
	if function == nil {
		t.Fatal("converted tool is not a function tool")
	}
	required, ok := function.Parameters["required"].([]any)
	if !ok || len(required) != 1 || required[0] != "course_id" {
		t.Errorf("parameters required = %#v, want [course_id]", function.Parameters["required"])
	}
}

func TestConvertResponsesToolsPreservesTypedJSONSchema(t *testing.T) {
	schema := &jsonschema.Schema{
		Type:     "object",
		Required: []string{"course_id"},
		Properties: map[string]*jsonschema.Schema{
			"course_id": {Type: "string"},
		},
	}
	genaiTools := []*genai.Tool{{
		FunctionDeclarations: []*genai.FunctionDeclaration{{
			Name:                 "get_course_price",
			Description:          "lookup price",
			ParametersJsonSchema: schema,
		}},
	}}

	tools, err := convertResponsesTools(genaiTools)
	if err != nil {
		t.Fatalf("convertResponsesTools() error = %v", err)
	}
	if len(tools) != 1 || tools[0].OfFunction == nil {
		t.Fatalf("convertResponsesTools() returned %#v", tools)
	}
	required, ok := tools[0].OfFunction.Parameters["required"].([]any)
	if !ok || len(required) != 1 || required[0] != "course_id" {
		t.Errorf("parameters required = %#v, want [course_id]", tools[0].OfFunction.Parameters["required"])
	}
}

func TestConvertSchemaKeepsConstraints(t *testing.T) {
	minLen := int64(1)
	format := "email"
	nullable := true
	schema := &genai.Schema{
		Type:        genai.TypeObject,
		Description: "person",
		Required:    []string{"email"},
		Properties: map[string]*genai.Schema{
			"email": {
				Type:        genai.TypeString,
				Format:      format,
				MinLength:   &minLen,
				Nullable:    &nullable,
				Description: "email address",
			},
		},
		AnyOf: []*genai.Schema{
			{Type: genai.TypeObject},
		},
	}
	got, err := convertSchema(schema)
	if err != nil {
		t.Fatalf("convertSchema() error = %v", err)
	}
	props := got["properties"].(map[string]any)
	email := props["email"].(map[string]any)
	if email["format"] != "email" {
		t.Errorf("format = %v, want email", email["format"])
	}
	if email["minLength"] != int64(1) && email["minLength"] != float64(1) {
		t.Errorf("minLength = %v, want 1", email["minLength"])
	}
	if email["nullable"] != true {
		t.Errorf("nullable = %v, want true", email["nullable"])
	}
	if _, ok := got["anyOf"]; !ok {
		t.Error("missing anyOf")
	}
}

func TestParseAPIMode(t *testing.T) {
	tests := []struct {
		in   string
		want APIMode
	}{
		{"", APIModeChatCompletions},
		{"chat", APIModeChatCompletions},
		{"chat_completions", APIModeChatCompletions},
		{"responses", APIModeResponses},
		{"response", APIModeResponses},
	}
	for _, tt := range tests {
		got, err := ParseAPIMode(tt.in)
		if err != nil {
			t.Fatalf("ParseAPIMode(%q) error = %v", tt.in, err)
		}
		if got != tt.want {
			t.Errorf("ParseAPIMode(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
	if _, err := ParseAPIMode("bogus"); err == nil {
		t.Fatal("ParseAPIMode(bogus) expected error")
	}
}
