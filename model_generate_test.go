package adkopenai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"google.golang.org/genai"

	"google.golang.org/adk/v2/model"
)

func TestGenerateContentNilRequest(t *testing.T) {
	llm := New(openai.Client{}, "m")
	for _, err := range llm.GenerateContent(context.Background(), nil, false) {
		if err == nil {
			t.Fatal("expected error")
		}
		return
	}
	t.Fatal("expected yield")
}

func TestGenerateContentInvalidAPIMode(t *testing.T) {
	llm := New(openai.Client{}, "m", WithAPIMode(APIMode("nope")))
	for _, err := range llm.GenerateContent(context.Background(), &model.LLMRequest{}, false) {
		if err == nil {
			t.Fatal("expected error")
		}
		return
	}
	t.Fatal("expected yield")
}

func TestGenerateChatNonStreamAndStream(t *testing.T) {
	var sawStream bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			http.NotFound(w, r)
			return
		}
		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		_ = json.Unmarshal(body, &payload)
		if stream, _ := payload["stream"].(bool); stream {
			sawStream = true
			w.Header().Set("Content-Type", "text/event-stream")
			flusher, _ := w.(http.Flusher)
			writeSSE(w, flusher, `{"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"Hel"},"finish_reason":null}]}`)
			writeSSE(w, flusher, `{"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"lo"},"finish_reason":null}]}`)
			writeSSE(w, flusher, `{"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`)
			writeSSE(w, flusher, `[DONE]`)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"1","object":"chat.completion","model":"gpt-test",
			"choices":[{"index":0,"message":{"role":"assistant","content":"Hello"},"finish_reason":"stop"}],
			"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}
		}`))
	}))
	defer srv.Close()

	client := openai.NewClient(option.WithBaseURL(srv.URL), option.WithAPIKey("test"))
	llm := New(client, "gpt-test")
	req := &model.LLMRequest{
		Contents: []*genai.Content{genai.NewContentFromText("hi", genai.RoleUser)},
	}

	var nonStream string
	for resp, err := range llm.GenerateContent(context.Background(), req, false) {
		if err != nil {
			t.Fatal(err)
		}
		if resp != nil && resp.Content != nil {
			for _, p := range resp.Content.Parts {
				nonStream += p.Text
			}
		}
	}
	if nonStream != "Hello" {
		t.Fatalf("non-stream = %q", nonStream)
	}

	var partials []string
	var final string
	for resp, err := range llm.GenerateContent(context.Background(), req, true) {
		if err != nil {
			t.Fatal(err)
		}
		if resp == nil || resp.Content == nil {
			continue
		}
		text := ""
		for _, p := range resp.Content.Parts {
			text += p.Text
		}
		if resp.Partial {
			partials = append(partials, text)
		} else {
			final = text
		}
	}
	if !sawStream {
		t.Fatal("expected stream request")
	}
	if len(partials) < 2 || final != "Hello" {
		t.Fatalf("partials=%v final=%q", partials, final)
	}
}

func TestGenerateChatStreamEarlyExit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		writeSSE(w, flusher, `{"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"a"},"finish_reason":null}]}`)
		writeSSE(w, flusher, `{"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"b"},"finish_reason":null}]}`)
		writeSSE(w, flusher, `{"id":"1","object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`)
		writeSSE(w, flusher, `[DONE]`)
	}))
	defer srv.Close()

	client := openai.NewClient(option.WithBaseURL(srv.URL), option.WithAPIKey("test"))
	llm := New(client, "gpt-test")
	req := &model.LLMRequest{
		Contents: []*genai.Content{genai.NewContentFromText("hi", genai.RoleUser)},
	}
	n := 0
	for range llm.GenerateContent(context.Background(), req, true) {
		n++
		break
	}
	if n != 1 {
		t.Fatalf("n=%d", n)
	}
}

func TestGenerateChatParallelToolCallsOrder(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"1","object":"chat.completion","model":"gpt-test",
			"choices":[{
				"index":0,
				"message":{
					"role":"assistant",
					"content":"",
					"tool_calls":[
						{"id":"c1","type":"function","function":{"name":"a","arguments":"{}"}},
						{"id":"c2","type":"function","function":{"name":"b","arguments":"{}"}}
					]
				},
				"finish_reason":"tool_calls"
			}],
			"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}
		}`))
	}))
	defer srv.Close()

	client := openai.NewClient(option.WithBaseURL(srv.URL), option.WithAPIKey("test"))
	llm := New(client, "gpt-test")
	for resp, err := range llm.GenerateContent(context.Background(), &model.LLMRequest{
		Contents: []*genai.Content{genai.NewContentFromText("hi", genai.RoleUser)},
	}, false) {
		if err != nil {
			t.Fatal(err)
		}
		if len(resp.Content.Parts) != 2 {
			t.Fatalf("%#v", resp.Content)
		}
		if resp.Content.Parts[0].FunctionCall.Name != "a" || resp.Content.Parts[1].FunctionCall.Name != "b" {
			t.Fatalf("%#v", resp.Content.Parts)
		}
	}
}

func TestGenerateResponsesFailed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/responses") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id":"resp_1","object":"response","status":"failed",
			"error":{"code":"server_error","message":"boom"},
			"output":[],
			"usage":{"input_tokens":1,"output_tokens":0,"total_tokens":1}
		}`))
	}))
	defer srv.Close()

	client := openai.NewClient(option.WithBaseURL(srv.URL), option.WithAPIKey("test"))
	llm := New(client, "gpt-test", WithAPIMode(APIModeResponses))
	for resp, err := range llm.GenerateContent(context.Background(), &model.LLMRequest{
		Contents: []*genai.Content{genai.NewContentFromText("hi", genai.RoleUser)},
	}, false) {
		if err != nil {
			t.Fatal(err)
		}
		if resp.ErrorCode != "server_error" || resp.ErrorMessage != "boom" {
			t.Fatalf("%#v", resp)
		}
	}
}

func TestGenerateChatCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be reached with cancelled context")
	}))
	defer srv.Close()

	client := openai.NewClient(option.WithBaseURL(srv.URL), option.WithAPIKey("test"))
	llm := New(client, "gpt-test")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, err := range llm.GenerateContent(ctx, &model.LLMRequest{
		Contents: []*genai.Content{genai.NewContentFromText("hi", genai.RoleUser)},
	}, false) {
		if err == nil {
			t.Fatal("expected cancel error")
		}
		return
	}
	t.Fatal("expected yield")
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, data string) {
	_, _ = io.WriteString(w, "data: "+data+"\n\n")
	if flusher != nil {
		flusher.Flush()
	}
}
