// Package adkopenai 提供 ADK Go 的 OpenAI 兼容 model.LLM 实现。
//
// 支持两种 API 模式：
//   - chat_completions：POST /v1/chat/completions（默认）
//   - responses：POST /v1/responses
//
// 调用方负责配置 openai.Client（API Key、Base URL 等）：
//
//	client := openai.NewClient(
//	    option.WithAPIKey(apiKey),
//	    option.WithBaseURL(baseURL),
//	)
//	llm := adkopenai.New(client, "grok-4.5", adkopenai.WithAPIMode(adkopenai.APIModeResponses))
package adkopenai

import (
	"context"
	"fmt"
	"iter"
	"strings"

	"github.com/openai/openai-go/v3"

	"google.golang.org/adk/v2/model"
)

var _ model.LLM = (*Model)(nil)

// APIMode 选择底层 OpenAI 兼容 API。
type APIMode string

const (
	// APIModeChatCompletions 使用 /v1/chat/completions（默认）。
	APIModeChatCompletions APIMode = "chat_completions"
	// APIModeResponses 使用 /v1/responses。
	APIModeResponses APIMode = "responses"
)

// Option 配置 Model。
type Option func(*Model)

// WithAPIMode 设置 API 模式。
func WithAPIMode(mode APIMode) Option {
	return func(m *Model) {
		m.apiMode = mode
	}
}

// ParseAPIMode 解析环境变量风格的模式字符串。
// 支持：chat_completions / chat / completions / responses / response。
func ParseAPIMode(s string) (APIMode, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "chat_completions", "chat", "completions", "chat.completions":
		return APIModeChatCompletions, nil
	case "responses", "response":
		return APIModeResponses, nil
	default:
		return "", fmt.Errorf("unsupported OPENAI_API_MODE %q (want chat_completions or responses)", s)
	}
}

// Model 实现 model.LLM。
type Model struct {
	client    openai.Client
	modelName string
	apiMode   APIMode
}

// New 使用已配置好的 openai.Client 创建模型。
func New(client openai.Client, modelName string, opts ...Option) *Model {
	m := &Model{
		client:    client,
		modelName: modelName,
		apiMode:   APIModeChatCompletions,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(m)
		}
	}
	if m.apiMode == "" {
		m.apiMode = APIModeChatCompletions
	}
	return m
}

// Name implements model.LLM.
func (m *Model) Name() string { return m.modelName }

// APIMode 返回当前 API 模式。
func (m *Model) APIMode() APIMode { return m.apiMode }

// GenerateContent implements model.LLM.
func (m *Model) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		modelName, err := validateGenerateRequest(m.apiMode, req, m.modelName)
		if err != nil {
			yield(nil, err)
			return
		}

		switch m.apiMode {
		case APIModeResponses:
			if stream {
				m.generateResponsesStream(ctx, req, modelName)(yield)
				return
			}
			m.generateResponses(ctx, req, modelName)(yield)
		case APIModeChatCompletions:
			if stream {
				m.generateChatStream(ctx, req, modelName)(yield)
				return
			}
			m.generateChat(ctx, req, modelName)(yield)
		default:
			yield(nil, fmt.Errorf("unsupported API mode %q", m.apiMode))
		}
	}
}
