package imgutil

import (
	"bytes"
	"image"
	"image/color"
	"image/gif"
	"image/png"

	_ "golang.org/x/image/webp"
)

// ConvertWebPToPNG converts WebP image data to PNG.
func ConvertWebPToPNG(data []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ConvertWebPToGIF converts WebP image data to GIF.
func ConvertWebPToGIF(data []byte) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	g := &gif.GIF{
		Image: []*image.Paletted{imageToPaletted(img)},
		Delay: []int{10},
	}
	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, g); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func imageToPaletted(img image.Image) *image.Paletted {
	bounds := img.Bounds()
	pm := image.NewPaletted(bounds, buildPalette(img, bounds))
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			pm.Set(x, y, img.At(x, y))
		}
	}
	return pm
}

type rgbaKey struct{ R, G, B, A uint8 }

func buildPalette(img image.Image, bounds image.Rectangle) color.Palette {
	seen := make(map[rgbaKey]struct{}, 256)
	palette := make(color.Palette, 0, 256)
	for y := bounds.Min.Y; y < bounds.Max.Y && len(palette) < 256; y++ {
		for x := bounds.Min.X; x < bounds.Max.X && len(palette) < 256; x++ {
			r32, g32, b32, a32 := img.At(x, y).RGBA()
			k := rgbaKey{uint8((r32 >> 8) & 0xFF), uint8((g32 >> 8) & 0xFF), uint8((b32 >> 8) & 0xFF), uint8((a32 >> 8) & 0xFF)}
			if _, ok := seen[k]; !ok {
				seen[k] = struct{}{}
				palette = append(palette, color.RGBA{R: k.R, G: k.G, B: k.B, A: k.A})
			}
		}
	}
	for len(palette) < 256 {
		palette = append(palette, color.RGBA{})
	}
	return palette
}
