# adk-openai

[![Go Reference](https://pkg.go.dev/badge/github.com/civaapple-alt/adk-openai.svg)](https://pkg.go.dev/github.com/civaapple-alt/adk-openai)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

OpenAI-compatible [`model.LLM`](https://pkg.go.dev/google.golang.org/adk/v2/model) adapter for [ADK Go](https://pkg.go.dev/google.golang.org/adk/v2) (`google.golang.org/adk/v2`).

Built on the official [`openai-go`](https://github.com/openai/openai-go) client, this package lets ADK agents talk to OpenAI and any OpenAI-compatible endpoint (for example xAI, Ollama, or self-hosted gateways).

## Related projects

For Anthropic Claude, see [adk-anthropic-go](https://github.com/Alcova-AI/adk-anthropic-go). It implements the same ADK Go `model.LLM` adapter pattern for Anthropic (direct API and Vertex AI), with a design approach closely aligned with this package. Provider-specific API behavior remains implemented independently.

See the [development roadmap](ROADMAP.md) for planned compatibility, correctness, and feature work.

## Features

- **Chat Completions** (`POST /v1/chat/completions`) — default
- **Responses API** (`POST /v1/responses`)
- Text generation (streaming and non-streaming)
- Function tools with `*jsonschema.Schema` parameters
- ADK `ToolConfig` mapping (`none` / `auto` / `required` / named function)
- Mixed `FunctionResponse` history with required call IDs
- Diagnostic errors for malformed or truncated tool arguments
- Image input via inline data URLs
- JSON / JSON Schema structured output
- Refusal, failed, cancelled, and incomplete response mapping
- Chat Completions streaming with `include_usage`

## Requirements

- Go 1.25+
- An OpenAI-compatible API key and base URL

## Installation

```bash
go get github.com/civaapple-alt/adk-openai
```

## Quick start

```go
package main

import (
	"context"

	adkopenai "github.com/civaapple-alt/adk-openai"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"

	"google.golang.org/adk/v2/agent/llmagent"
)

func main() {
	client := openai.NewClient(
		option.WithAPIKey("your-api-key"),
		option.WithBaseURL("https://api.x.ai/v1"),
	)

	// Default: Chat Completions
	llm := adkopenai.New(client, "grok-4.5")

	// Or use the Responses API
	// llm := adkopenai.New(client, "grok-4.5", adkopenai.WithAPIMode(adkopenai.APIModeResponses))

	agent, err := llmagent.New(llmagent.Config{
		Name:        "assistant",
		Model:       llm,
		Instruction: "You are a helpful assistant.",
	})
	if err != nil {
		panic(err)
	}
	_ = agent
	_ = context.Background()
}
```

You own the `openai.Client` configuration (API key, base URL, HTTP options). This package only adapts that client to ADK's `model.LLM` interface.

## API modes

| Mode | Constant | Endpoint |
|---|---|---|
| `chat_completions` (default) | `APIModeChatCompletions` | `/v1/chat/completions` |
| `responses` | `APIModeResponses` | `/v1/responses` |

```go
mode, err := adkopenai.ParseAPIMode("responses")
if err != nil {
	panic(err)
}
llm := adkopenai.New(client, "grok-4.5", adkopenai.WithAPIMode(mode))
```

`ParseAPIMode` accepts common aliases: `chat_completions`, `chat`, `completions`, `responses`, and `response`.

## Configuration

Suggested environment variables (as used by [`examples/hello`](examples/hello)):

```bash
export OPENAI_BASE_URL=https://api.x.ai/v1
export OPENAI_API_KEY=...
export OPENAI_MODEL=grok-4.5
export OPENAI_API_MODE=chat_completions   # or responses
```

Run the example:

```bash
go run ./examples/hello
```

## Local development

If you develop against a local checkout of ADK Go, add a `replace` directive in `go.mod`:

```go
replace google.golang.org/adk/v2 => /path/to/adk-go
```

Then run tests:

```bash
go test ./...
```

## License

Apache License 2.0. See [LICENSE](LICENSE).
