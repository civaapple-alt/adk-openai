# Changelog

## [v0.2.0] - P0 correctness and regression coverage

- Map ADK `ToolConfig` to OpenAI `tool_choice` for Chat Completions and
  Responses (`none`, `auto`, `required`, and a single named function). Reject
  multiple `AllowedFunctionNames` and unsupported modes instead of relaxing them.
- Process `FunctionResponse` parts throughout mixed content, require call IDs,
  and preserve Chat / Responses message ordering.
- Replace silent empty-map fallback for malformed tool arguments with
  `InvalidToolArgumentsError` and `OutputInterruptedError` (max-token truncation).
- Preserve Responses API tool-call order deterministically in the streaming
  fallback path.
- Support the `developer` role; reject unknown roles, nil requests, empty model
  names, and invalid API modes instead of silently coercing them.
- Map refusal to `FinishReasonSafety`; populate `ErrorCode` / `ErrorMessage` for
  failed and cancelled Responses API results; keep incomplete max-token mapping.
- Expand conversion and httptest/SSE regression tests for both API modes.

## [v0.1.0] - Initial release

- OpenAI-compatible ADK `model.LLM` adapter with Chat Completions and Responses
  API modes, streaming, function tools, inline images, structured output, usage,
  and refusal mapping.
