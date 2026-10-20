Bug classes seen in LLM gateways (tingly-box's tracker and similar tools:
litellm, claude-code-router, new-api, cc-switch). State each as a general
property of the function at hand; do not encode trigger values: go2lean's
search finds the inputs.

Request conversion (Anthropic, OpenAI Chat, OpenAI Responses, any direction)
- Conservation: every message, every content block the target format can
  carry (text, image, tool_use, tool_result, thinking), every tool, and every
  sampling or control parameter the target has (temperature, top_p, stop
  sequences, max tokens, tool_choice, parallel tool calls) reaches the
  converted request.
- Meaning: tool_choice modes keep their meaning across formats (must call a
  tool / may / must not / this tool); reasoning budgets map monotonically to
  efforts.
- Pairing: tool call ids and tool result ids still pair up after conversion,
  within the target's id limits on both sides.

Response conversion, non-streaming
- Conservation of text, refusals, reasoning/thinking text (as text, not as
  JSON), tool calls (id, name, arguments) and usage counts.
- Stop / finish reasons map totally and consistently in both directions.

Response conversion, streaming
- Agreement: for any legal event sequence of a response (values split into
  deltas or sent whole, in any legal order), what a client assembles from
  the converted stream equals the non-streaming conversion of the same
  response. Derive the sequence's shape from a generated field.
- Protocol shape: the converted events obey the target protocol: one
  message start and stop, one content block open at a time, deltas only to
  the open block and of its type, indexes contiguous, tool calls with
  distinct indexes.

Usage
- Normalised counts follow the documented basis (input excludes cache
  reads, cache writes are a subset of input, reasoning a subset of output);
  converting back reproduces the wire counts; streaming and non-streaming
  paths record the same numbers for the same upstream usage.

Keep the spec inside go2lean's modelled subset (see the system prompt).
