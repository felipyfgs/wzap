# Refactoring Sweep

## Problem

The wzap Go backend has accumulated technical debt across several dimensions:

1. **Code duplication**: `extractMessageContent` is copy-pasted verbatim across `internal/wa/events.go` and `internal/service/history.go`. The `@lid` JID resolution pattern is repeated 10+ times in the Chatwoot package. `runSessionRuntime` and `runConnectedRuntime` differ by one line.

2. **Monolithic functions**: `Manager.handleEvent` in `wa/events.go` is a single ~530-line method with two sequential type switches. `Service.handleMessage` in Chatwoot is 256 lines mixing LID resolution, idempotency, conversation management, and message type dispatch.

3. **Dead code**: `internal/service/message_status.go` contains only `package service` — no types, functions, or variables.

4. **Silently discarded errors**: 127 instances of `_ =` across 28 non-test files. Critical ones include `_ = s.msgRepo.UpdateChatwootRef(...)` (8 occurrences) and `_ = msg.Nak()` (16 occurrences), which hide DB/NATS failures.

5. **Misplaced concerns**: Image conversion logic (WebP→PNG/GIF with palette building, ~60 lines) lives inside the Chatwoot message handler rather than a utility package.

6. **Structural inefficiency**: `handleEvent` does two full type switches on every event — first to classify, then to serialize — matching each event type twice.

## Proposed Solution

A phased refactoring across 4 phases, each independently shippable:

- **Phase 1 (Quick Wins)**: Remove dead file, extract shared `extractMessageContent` to `internal/wautil`, unify runtime dispatch functions.
- **Phase 2 (Chatwoot DRY)**: Extract `@lid` resolution helper, message param builders, image conversion to utility package, idempotency check.
- **Phase 3 (wa/events.go Decomposition)**: Split `handleEvent` into classify + serialize + dispatch; merge two-pass type switch into single pass.
- **Phase 4 (Error Handling & Cleanup)**: Log silently discarded errors on critical paths, remove excessive nil-receiver checks, deduplicate `Support()` methods.

## Scope

- **In scope**: `internal/wa/`, `internal/service/`, `internal/integrations/chatwoot/`, new utility packages (`wautil`, `imgutil`)
- **Out of scope**: Frontend (`web/`), handler layer, repo layer, provider layer, no new features

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Moving shared functions breaks callers | Low | Medium | Single PR updates all call sites; `go build` catches compile errors |
| Chatwoot refactoring changes message processing behavior | Low | High | Each extraction preserves exact logic; existing tests pass |
| `handleEvent` split changes event dispatch order | Low | Medium | Dispatch is sequential; split preserves order |
| Removing nil-receiver checks causes panics | Medium | Low | Audit all call sites; add test coverage |

## Success Criteria

- All existing tests pass without modification
- No function exceeds 150 lines
- Zero duplicate `extractMessageContent` implementations
- Critical `_ =` patterns replaced with logged error handling
- `message_status.go` removed
