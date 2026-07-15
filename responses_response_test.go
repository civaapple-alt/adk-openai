package adkopenai

import (
	"errors"
	"testing"

	"github.com/openai/openai-go/v3/responses"
	"google.golang.org/genai"
)

func TestConvertResponsesResultRefusal(t *testing.T) {
	resp := &responses.Response{
		Status: "completed",
		Output: []responses.ResponseOutputItemUnion{{
			Type: "message",
			Content: []responses.ResponseOutputMessageContentUnion{{
				Type:    "refusal",
				Refusal: "nope",
			}},
		}},
	}
	got, err := convertResponsesResult(resp)
	if err != nil {
		t.Fatal(err)
	}
	if got.FinishReason != genai.FinishReasonSafety {
		t.Fatalf("finish = %v", got.FinishReason)
	}
	if got.Content.Parts[0].Text != "nope" {
		t.Fatalf("content = %#v", got.Content)
	}
}

func TestConvertResponsesResultFailed(t *testing.T) {
	resp := &responses.Response{
		Status: "failed",
		Error:  responses.ResponseError{Code: "server_error", Message: "boom"},
	}
	got, err := convertResponsesResult(resp)
	if err != nil {
		t.Fatal(err)
	}
	if got.ErrorCode != "server_error" || got.ErrorMessage != "boom" {
		t.Fatalf("%#v", got)
	}
	if got.FinishReason != genai.FinishReasonUnspecified {
		t.Fatalf("finish = %v", got.FinishReason)
	}
}

func TestConvertResponsesResultIncompleteMaxTokens(t *testing.T) {
	resp := &responses.Response{
		Status:            "incomplete",
		IncompleteDetails: responses.ResponseIncompleteDetails{Reason: "max_output_tokens"},
		Output: []responses.ResponseOutputItemUnion{{
			Type:      "function_call",
			CallID:    "c1",
			Name:      "lookup",
			Arguments: responses.ResponseOutputItemUnionArguments{OfString: `{"q":`},
		}},
	}
	_, err := convertResponsesResult(resp)
	var interrupted *OutputInterruptedError
	if !errors.As(err, &interrupted) {
		t.Fatalf("got %T %v", err, err)
	}
}

func TestConvertResponsesResultInvalidArgs(t *testing.T) {
	resp := &responses.Response{
		Status: "completed",
		Output: []responses.ResponseOutputItemUnion{{
			Type:      "function_call",
			CallID:    "c1",
			Name:      "lookup",
			Arguments: responses.ResponseOutputItemUnionArguments{OfString: `{`},
		}},
	}
	_, err := convertResponsesResult(resp)
	var invalid *InvalidToolArgumentsError
	if !errors.As(err, &invalid) {
		t.Fatalf("got %T %v", err, err)
	}
}

func TestConvertResponsesResultToolOrder(t *testing.T) {
	resp := &responses.Response{
		Status: "completed",
		Output: []responses.ResponseOutputItemUnion{
			{Type: "function_call", CallID: "c1", Name: "a", Arguments: responses.ResponseOutputItemUnionArguments{OfString: `{}`}},
			{Type: "function_call", CallID: "c2", Name: "b", Arguments: responses.ResponseOutputItemUnionArguments{OfString: `{}`}},
		},
	}
	got, err := convertResponsesResult(resp)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Content.Parts) != 2 {
		t.Fatalf("%#v", got.Content)
	}
	if got.Content.Parts[0].FunctionCall.Name != "a" || got.Content.Parts[1].FunctionCall.Name != "b" {
		t.Fatalf("%#v", got.Content.Parts)
	}
}
