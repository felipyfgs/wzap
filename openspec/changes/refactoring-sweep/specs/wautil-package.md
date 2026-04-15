# Spec: wautil Package

## Capability: Shared WhatsApp Message Utilities

### Purpose

Consolidate WhatsApp message parsing functions that are currently duplicated across `internal/wa/` and `internal/service/` into a single shared package.

### Interface

```go
package wautil

// ExtractMessageContent extracts the type, body text, and media MIME type
// from a WhatsApp E2E protocol message.
func ExtractMessageContent(msg *waE2E.Message) (msgType, body, mediaType string)

// ExtractMediaDownloadInfo extracts download parameters from a media-bearing message.
// Returns (directPath, encFileHash, fileHash, mediaKey, fileLength, hasMedia).
func ExtractMediaDownloadInfo(msg *waE2E.Message) (string, []byte, []byte, []byte, int, bool)

// InferChatType returns the chat type string based on JID suffix conventions.
func InferChatType(chatJID string) string

// Helper functions for pointer conversions.
func StringPtr(value string) *string
func IntPtr(value int) *int
func UnixTimePtr(timestamp int64) *time.Time
func FirstNonEmpty(values ...string) string
func Uint64ToInt64(value uint64) int64
```

### Dependencies

- `go.mau.fi/whatsmeow/proto/waE2E` — for `*waE2E.Message` parameter type

### Consumers

- `internal/wa/events.go` — event classification
- `internal/service/history.go` — history sync processing

### Migration

1. Create `internal/wautil/message.go` and `internal/wautil/helpers.go`
2. Copy functions with exported names (uppercase first letter)
3. Update `wa/events.go` to import `wautil` and call exported functions
4. Update `service/history.go` to import `wautil` and call exported functions
5. Remove local copies from both files
6. Verify `go build ./...` and `go test ./...`
