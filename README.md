# adk-openai

[![Go Reference](https://pkg.go.dev/badge/github.com/civaapple-alt/adk-openai.svg)](https://pkg.go.dev/github.com/civaapple-alt/adk-openai)

ADK Go (`google.golang.org/adk/v2`) 的 OpenAI 兼容 `model.LLM` 实现。

基于官方 [`github.com/openai/openai-go/v3`](https://github.com/openai/openai-go)，支持：

- **Chat Completions**：`POST /v1/chat/completions`（默认）
- **Responses API**：`POST /v1/responses`

可用于 OpenAI、xAI、Ollama 等 OpenAI 兼容端点。

## 安装

```bash
go get github.com/civaapple-alt/adk-openai
```

需要 Go 1.25+。

## 快速开始

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

	// 默认 Chat Completions
	llm := adkopenai.New(client, "grok-4.5")

	// 或使用 Responses API
	// llm := adkopenai.New(client, "grok-4.5", adkopenai.WithAPIMode(adkopenai.APIModeResponses))

	agent, err := llmagent.New(llmagent.Config{
		Name:        "assistant",
		Model:       llm,
		Instruction: "You are a helpful assistant.",
	})
	_ = agent
	_ = err
	_ = context.Background()
}
```

## API 模式

| 值 | 实际请求 |
|---|---|
| `chat_completions`（默认） | `/v1/chat/completions` |
| `responses` | `/v1/responses` |

```go
mode, err := adkopenai.ParseAPIMode("responses")
llm := adkopenai.New(client, "grok-4.5", adkopenai.WithAPIMode(mode))
```

环境变量建议：

```bash
OPENAI_BASE_URL=https://api.x.ai/v1
OPENAI_API_KEY=...
OPENAI_MODEL=grok-4.5
OPENAI_API_MODE=chat_completions   # 或 responses
```

## 功能

- 文本生成（非流式 / 流式）
- Function tools（含 `*jsonschema.Schema` 参数）
- 图片输入（inline data URL）
- JSON / JSON Schema 结构化输出
- Refusal 映射
- Chat Completions 流式 `include_usage`

## 本地开发

若使用本地 ADK：

```go
// go.mod
replace google.golang.org/adk/v2 => D:/gh-ws/adk-go
```

```bash
go test ./...
```

## License

Apache-2.0
