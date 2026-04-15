# Tasks — Refactoring Sweep

## Phase 1: Quick Wins

- [x] T1.1: Delete empty `internal/service/message_status.go`
- [x] T1.2: Create `internal/wautil/message.go` and `internal/wautil/helpers.go` with shared functions (`ExtractMessageContent`, `ExtractMediaDownloadInfo`, `InferChatType`, `StringPtr`, `IntPtr`, `UnixTimePtr`, `FirstNonEmpty`, `Uint64ToInt64`). Update `internal/wa/events.go` and `internal/service/history.go` to import from `wautil`. Remove local copies.
- [x] T1.3: Unify `runSessionRuntime` and `runConnectedRuntime` in `internal/service/session_runtime.go` by introducing `runClientRuntime` with a `clientResolver` parameter. Simplify `runRuntimeErr` accordingly.

## Phase 2: Chatwoot Package DRY

- [x] T2.1: Add `resolveLID` method to `internal/integrations/chatwoot/jid.go`. Replace all 10+ `if strings.HasSuffix(jid, "@lid")` call sites in `inbound_events.go` and `inbound_message.go`.
- [x] T2.2: Add `cwMsgParams` struct and `newCWMsgParams` helper to `internal/integrations/chatwoot/inbound_message.go`. Replace the `messageType`/`sourceID`/`contentAttrs` boilerplate in `handleMediaMessage`, `handleStickerMessage`, `handlePollCreation`, `handleReaction`, `handleButtonResponse`, and other handler methods.
- [x] T2.3: Create `internal/imgutil/convert.go` with `ConvertWebPToPNG` and `ConvertWebPToGIF`. Move `imageToPaletted`, `buildPalette`, `rgbaKey` there. Update `inbound_message.go` to import `imgutil` and remove local image conversion code and image-related imports.
- [x] T2.4: Extract `isOutboundDuplicate` method from `HandleIncomingWebhook` in `internal/integrations/chatwoot/outbound.go`. Replace the idempotency block (sourceID check + CW msg ID cache check) with a single call.

## Phase 3: wa/events.go Decomposition

- [x] T3.1: Split `handleEvent` in `internal/wa/events.go` into `classifyEvent` (returns `classifiedEvent`), `serializeEventData`, and `dispatchEvent`. Rewrite `handleEvent` as a 3-call composition. Merge the two-pass type switch so `classifyEvent` pre-builds data for `HistorySync` and `AppState` events.

## Phase 4: Error Handling & Runtime Cleanup

- [x] T4.1: Replace critical `_ =` patterns with logged error handling in `internal/wa/connect.go` and `internal/wa/qr.go` (session repo updates). Exclude cleanup/best-effort `_ =`.
- [x] T4.2: Remove nil-receiver checks (`if r == nil { return ... }`) from internal methods on `SessionRuntime`, `MessageRuntime`, `MediaRuntime`, `StatusRuntime`, `ProfileRuntime`, and `RuntimeResolver` in `internal/service/session_runtime.go`. Keep nil checks on `Resolve()` and `resolveCapability()`.
- [x] T4.3: Simplify `Support()` methods on `MessageRuntime`, `MediaRuntime`, `StatusRuntime`, `ProfileRuntime` by removing nil-receiver guards (reducing each from 4 lines to 2 lines).
