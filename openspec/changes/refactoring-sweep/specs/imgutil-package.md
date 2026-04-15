# Spec: imgutil Package

## Capability: Image Format Conversion Utilities

### Purpose

Extract WebP→PNG/GIF image conversion logic from the Chatwoot inbound message handler into a reusable utility package.

### Interface

```go
package imgutil

// ConvertWebPToPNG converts WebP image data to PNG format.
func ConvertWebPToPNG(data []byte) ([]byte, error)

// ConvertWebPToGIF converts WebP image data to animated GIF format.
func ConvertWebPToGIF(data []byte) ([]byte, error)
```

### Dependencies

- `golang.org/x/image/webp` — for WebP decoding (blank import)
- `image`, `image/color`, `image/gif`, `image/png` — stdlib

### Consumers

- `internal/integrations/chatwoot/inbound_message.go` — sticker message handler

### Migration

1. Create `internal/imgutil/convert.go` with exported functions and unexported helpers (`imageToPaletted`, `buildPalette`, `rgbaKey`)
2. Update `inbound_message.go` to import `imgutil` and call `imgutil.ConvertWebPToPNG(data)` / `imgutil.ConvertWebPToGIF(data)`
3. Remove local `convertWebPToPNG`, `convertWebPToGIF`, `imageToPaletted`, `buildPalette`, `rgbaKey` from `inbound_message.go`
4. Remove image-related imports (`image`, `image/color`, `image/gif`, `image/png`, `golang.org/x/image/webp`) from `inbound_message.go`
5. Verify `go build ./...` and `go test ./...`
