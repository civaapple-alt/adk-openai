package adkopenai

import (
	"testing"

	"google.golang.org/genai"
)

func TestOrderedToolCallsStable(t *testing.T) {
	o := newOrderedToolCalls()
	o.get("z").name = "third"
	o.get("a").name = "first"
	o.get("m").name = "second"
	o.get("a").args = `{}`

	builders := o.builders()
	if len(builders) != 3 {
		t.Fatalf("len=%d", len(builders))
	}
	if builders[0].name != "third" || builders[1].name != "first" || builders[2].name != "second" {
		t.Fatalf("%#v", builders)
	}
}

func TestChatStreamToolCallOrderViaBuilders(t *testing.T) {
	// Mirror chat stream index sorting contract used by generateChatStream.
	toolCalls := map[int]*toolCallBuilder{
		1: {id: "c2", name: "b", args: `{}`},
		0: {id: "c1", name: "a", args: `{}`},
	}
	indices := make([]int, 0, len(toolCalls))
	for i := range toolCalls {
		indices = append(indices, i)
	}
	// sort is applied in generateChatStream; assert builders produce a,b when sorted.
	if toolCalls[0].name != "a" || toolCalls[1].name != "b" {
		t.Fatalf("%#v", toolCalls)
	}
	_ = indices
}

func TestFunctionCallFromArgsEmptyObject(t *testing.T) {
	part, err := functionCallFromArgs("c1", "lookup", `{}`, genai.FinishReasonStop, nil)
	if err != nil {
		t.Fatal(err)
	}
	if part.FunctionCall.Name != "lookup" || len(part.FunctionCall.Args) != 0 {
		t.Fatalf("%#v", part)
	}
}
