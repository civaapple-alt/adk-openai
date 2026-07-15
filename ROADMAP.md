# Development Roadmap

This roadmap is ordered by priority, not by release date. It draws on lessons
from [adk-anthropic-go](https://github.com/Alcova-AI/adk-anthropic-go) while
keeping OpenAI and OpenAI-compatible API semantics independent.

## Current baseline

- Go 1.25
- ADK Go v2.0.0
- `openai-go/v3`
- Chat Completions and Responses API modes
- Streaming and non-streaming text
- Function tools, inline images, structured output, usage, and refusal mapping

The adapter already implements the current ADK `model.LLM` interface. Tracking
newer ADK revisions is mainly a behavioral compatibility and testing concern,
not an interface migration blocker.

## Guiding principles

- Preserve caller-owned `openai.Client` configuration and custom base URLs.
- Keep Chat Completions and Responses protocol handling separate where their
  capabilities differ.
- Never silently weaken requested ADK behavior.
- Gate optional features by provider capability so official OpenAI, xAI,
  Ollama, and self-hosted gateways can behave predictably.
- Add regression tests before refactoring conversion code.

## P0: Correctness and regression coverage

### Work

- Add table-driven request and response conversion tests for both API modes.
- Add fake HTTP/SSE tests for streaming, cancellation, early iterator exit,
  refusals, failures, and multiple parallel tool calls.
- Map ADK `ToolConfig` explicitly:
  - `ModeNone` to `none`
  - `ModeAuto` to `auto`
  - `ModeAny` to `required`
  - one allowed function to a provider-specific named function choice
  - reject unsupported multi-function restrictions instead of relaxing them
- Process `FunctionResponse` parts throughout mixed content, validate call IDs,
  and preserve protocol-required message ordering.
- Replace silent malformed tool argument fallback with diagnostic errors.
  Distinguish invalid provider output from output truncated at a token limit.
- Preserve Responses API tool-call order deterministically.
- Correct `developer` role conversion and validate unknown roles, nil requests,
  empty model names, and invalid API modes.
- Map refusal, failed, cancelled, and incomplete responses to consistent ADK
  finish and error fields.

### Primary files

- `model.go`
- `chat_request.go`, `chat_response.go`, `chat_stream.go`
- `responses_request.go`, `responses_response.go`, `responses_stream.go`
- `util.go`
- New request, response, streaming, and integration test files

### Exit criteria

- Chat and Responses modes pass the same core behavioral test matrix.
- No malformed tool call can be executed as an accidental empty argument map.
- Parallel tool-call order is stable across repeated test runs.
- `go test ./...`, `go test -race ./...`, and `go vet ./...` pass.

## P1: ADK response semantics and endpoint capabilities

### Work

- Populate `LLMResponse.ModelVersion` from the resolved response model.
- Map cached input tokens to `CachedContentTokenCount`.
- Map Responses API URL annotations and citations to `CitationMetadata`.
- Make `ThinkingConfig` handling model- and API-aware:
  - map supported minimal, low, medium, and high effort levels faithfully
  - define explicit behavior for unsupported `ThinkingBudget`
  - only expose reasoning summaries as `genai.Part{Thought: true}` when the
    provider returns content intended for callers
- Add an endpoint capability profile for features that are not uniformly
  supported by OpenAI-compatible services.
- Add opt-in strict structured output. Normalize object schemas recursively for
  OpenAI strict mode while retaining a compatible non-strict mode.

### Primary files

- `model.go`
- `schema.go`
- Chat and Responses request/response converters
- New capability and metadata tests

### Exit criteria

- Unsupported requested behavior produces a clear error or documented,
  configurable downgrade.
- Structured output tests cover nested objects, arrays, unions, required
  fields, and compatibility mode.
- Response metadata is consistent in streaming and non-streaming paths.

## P2: Internal conversion architecture

### Work

- Move shared role, schema, tool, validation, and error logic into private
  `internal/converters` packages.
- Retain dedicated Chat Completions and Responses conversion layers.
- Introduce provider-neutral typed errors for invalid tool calls and interrupted
  output, preserving tool name, call ID, raw partial arguments, and completed
  content where possible.
- Keep conversion APIs private until their stability requirements are clear.

### Exit criteria

- Existing P0 and P1 behavior remains unchanged through the refactor.
- Shared ADK rules have one implementation without hiding protocol differences.

## P3: Multimodal and Responses API expansion

### Work

- Support `FileData` image URLs where the selected API mode allows them.
- Evaluate Responses API input files, PDFs, file IDs, and supported audio input.
- Add capability-gated mappings for relevant Responses built-in tools such as
  web search, file search, and computer use.
- Cover additional Responses item and stream event types without silently
  dropping meaningful output.

### Exit criteria

- Every advertised content or tool type has request, response, and unsupported
  endpoint tests.
- Chat-only and Responses-only features are documented separately.

## P4: Maintenance and releases

### Work

- Add package-level Go documentation and a changelog.
- Add CI for formatting, tests, race tests, and vet.
- Test ADK compatibility on three tracks:
  - the minimum supported ADK v2 version
  - the latest ADK v2 release
  - scheduled, non-blocking checks against ADK `main`
- Document explicit use of the newer ADK model registry. Do not register a
  factory automatically from `init()`.
- Use release automation that requires an intentional semantic version decision
  instead of tagging every merged change as a patch.
- Check README feature claims against tests during releases.

### Versioning note

This module started on ADK v2 and is currently in its own v0 release line. An
ADK major version does not by itself require this module to adopt the same major
version. A `/v2` module path should only be introduced for this package's own
v2 release.

## Provider-specific non-goals

The following Anthropic features should not be copied unless OpenAI exposes an
equivalent with matching semantics:

- Vertex AI backend variants
- Explicit Anthropic prompt-cache breakpoints
- Anthropic thinking signatures and redacted-thinking blocks
- Anthropic message alternation workarounds

Only the portable ADK behavior, validation strategy, test discipline, and
release practices should be shared.
