# adk-openai

[![Go Reference](https://pkg.go.dev/badge/github.com/civaapple-alt/adk-openai.svg)](https://pkg.go.dev/github.com/civaapple-alt/adk-openai)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

[English](README.md) | [简体中文](README.zh_CN.md)

面向 [ADK Go](https://pkg.go.dev/google.golang.org/adk/v2)（`google.golang.org/adk/v2`）的 OpenAI 兼容 [`model.LLM`](https://pkg.go.dev/google.golang.org/adk/v2/model) 适配器。

基于官方 [`openai-go`](https://github.com/openai/openai-go) 客户端，可对接 OpenAI 以及任意 OpenAI 兼容端点（如 xAI、Ollama、自建网关等）。

## 功能

- **Chat Completions**（`POST /v1/chat/completions`）— 默认
- **Responses API**（`POST /v1/responses`）
- 文本生成（流式 / 非流式）
- Function tools（支持 `*jsonschema.Schema` 参数）
- 图片输入（inline data URL）
- JSON / JSON Schema 结构化输出
- Refusal 映射
- Chat Completions 流式 `include_usage`

## 环境要求

- Go 1.25+
- OpenAI 兼容的 API Key 与 Base URL

## 安装

```bash
go get github.com/civaapple-alt/adk-openai
```

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

	// 默认：Chat Completions
	llm := adkopenai.New(client, "grok-4.5")

	// 或使用 Responses API
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

调用方负责配置 `openai.Client`（API Key、Base URL、HTTP 选项等）。本包仅将其适配为 ADK 的 `model.LLM` 接口。

## API 模式

| 模式 | 常量 | 端点 |
|---|---|---|
| `chat_completions`（默认） | `APIModeChatCompletions` | `/v1/chat/completions` |
| `responses` | `APIModeResponses` | `/v1/responses` |

```go
mode, err := adkopenai.ParseAPIMode("responses")
if err != nil {
	panic(err)
}
llm := adkopenai.New(client, "grok-4.5", adkopenai.WithAPIMode(mode))
```

`ParseAPIMode` 支持常见别名：`chat_completions`、`chat`、`completions`、`responses`、`response`。

## 配置

推荐环境变量（与 [`examples/hello`](examples/hello) 一致）：

```bash
export OPENAI_BASE_URL=https://api.x.ai/v1
export OPENAI_API_KEY=...
export OPENAI_MODEL=grok-4.5
export OPENAI_API_MODE=chat_completions   # 或 responses
```

运行示例：

```bash
go run ./examples/hello
```

## 本地开发

若使用本地 ADK Go 源码，可在 `go.mod` 中添加 `replace`：

```go
replace google.golang.org/adk/v2 => /path/to/adk-go
```

然后运行测试：

```bash
go test ./...
```

## 许可证

Apache License 2.0，详见 [LICENSE](LICENSE)。
