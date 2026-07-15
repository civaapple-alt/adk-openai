package adkopenai

import (
	"fmt"

	"google.golang.org/genai"
)

// InvalidToolArgumentsError reports that a tool call's arguments were not valid JSON.
type InvalidToolArgumentsError struct {
	ToolName     string
	ToolID       string
	RawArguments string
	Cause        error
}

func (e *InvalidToolArgumentsError) Error() string {
	name := e.ToolName
	if name == "" {
		name = "<unknown>"
	}
	return fmt.Sprintf("invalid tool arguments for %q (id=%s): %v", name, e.ToolID, e.Cause)
}

func (e *InvalidToolArgumentsError) Unwrap() error { return e.Cause }

// OutputInterruptedError reports that generation was cut off (typically at a
// max-token limit) before tool-call arguments could complete.
type OutputInterruptedError struct {
	ToolName     string
	ToolID       string
	PartialInput string
	FinishReason genai.FinishReason
	Parts        []*genai.Part
	Cause        error
}

func (e *OutputInterruptedError) Error() string {
	if e.ToolName != "" {
		return fmt.Sprintf("model output interrupted (finish_reason=%s): tool call %q truncated after %d bytes of input",
			e.FinishReason, e.ToolName, len(e.PartialInput))
	}
	return fmt.Sprintf("model output interrupted (finish_reason=%s)", e.FinishReason)
}

func (e *OutputInterruptedError) Unwrap() error { return e.Cause }
