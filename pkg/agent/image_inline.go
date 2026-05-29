package agent

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"strings"

	"github.com/h2non/filetype"

	"github.com/sipeed/picoclaw/pkg/config"
)

type mediaResolveOptions struct {
	maxSourceBytes int
	imageInput     config.ResolvedImageInputConfig
}

type inlineImageCandidate struct {
	mime string
	data []byte
}

const maxInlineDecodedPixels = 40_000_000

func defaultMediaResolveOptions(maxSourceBytes int) mediaResolveOptions {
	defaults := config.AgentDefaults{
		ImageInput: config.ImageInputConfig{
			AttachUserImages: true,
			CompressionLevel: config.ImageCompressionBalanced,
		},
	}
	return newMediaResolveOptions(maxSourceBytes, defaults)
}

func newMediaResolveOptions(maxSourceBytes int, defaults config.AgentDefaults) mediaResolveOptions {
	if maxSourceBytes <= 0 {
		maxSourceBytes = config.DefaultMaxMediaSize
	}
	return mediaResolveOptions{
		maxSourceBytes: maxSourceBytes,
		imageInput:     defaults.ResolveImageInputConfig(),
	}
}

func shouldInlineImageForRole(role string, imageInput config.ResolvedImageInputConfig) bool {
	switch role {
	case "tool":
		return true
	case "user":
		return imageInput.AttachUserImages
	default:
		return false
	}
}

func encodeImageToDataURL(localPath, mime string, info os.FileInfo, opts mediaResolveOptions) (string, error) {
	candidate, err := buildInlineImageCandidate(localPath, mime, info, opts)
	if err != nil {
		return "", err
	}
	return encodeBytesToDataURL(candidate.mime, candidate.data), nil
}

func buildInlineImageCandidate(
	localPath string,
	mime string,
	info os.FileInfo,
	opts mediaResolveOptions,
) (inlineImageCandidate, error) {
	policy := opts.imageInput
	if shouldKeepOriginalInline(localPath, mime, info, policy) {
		return buildRawInlineImageCandidate(localPath, mime, info, opts)
	}

	cfg, err := decodeInlineImageConfig(localPath)
	if err != nil {
		if shouldAllowRawInlineFallback(localPath, mime) {
			return buildRawInlineImageCandidate(localPath, mime, info, opts)
		}
		return inlineImageCandidate{}, fmt.Errorf("decode inline image config: %w", err)
	}
	if err := validateInlineDecodeConfig(cfg); err != nil {
		return inlineImageCandidate{}, err
	}

	img, err := decodeInlineImage(localPath)
	if err != nil {
		if shouldAllowRawInlineFallback(localPath, mime) {
			return buildRawInlineImageCandidate(localPath, mime, info, opts)
		}
		return inlineImageCandidate{}, fmt.Errorf("decode inline image: %w", err)
	}

	return compressDecodedImage(img, mime, policy)
}

func shouldKeepOriginalInline(
	localPath string,
	mime string,
	info os.FileInfo,
	policy config.ResolvedImageInputConfig,
) bool {
	if policy.CompressionLevel != config.ImageCompressionOff || policy.TargetFormat != "auto" {
		return false
	}
	if policy.MaxInlineBytes > 0 && inlinePayloadSize(mime, int(info.Size())) > policy.MaxInlineBytes {
		return false
	}
	if policy.MaxWidth <= 0 && policy.MaxHeight <= 0 {
		return true
	}

	cfg, err := decodeInlineImageConfig(localPath)
	if err != nil {
		return false
	}
	if err := validateInlineDecodeConfig(cfg); err != nil {
		return false
	}
	width, height := fitWithin(cfg.Width, cfg.Height, policy.MaxWidth, policy.MaxHeight)
	return width == cfg.Width && height == cfg.Height
}

func decodeInlineImage(localPath string) (image.Image, error) {
	f, err := os.Open(localPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	return img, err
}

func decodeInlineImageConfig(localPath string) (image.Config, error) {
	f, err := os.Open(localPath)
	if err != nil {
		return image.Config{}, err
	}
	defer f.Close()

	cfg, _, err := image.DecodeConfig(f)
	return cfg, err
}

func shouldAllowRawInlineFallback(localPath, mime string) bool {
	// Raw passthrough after decode failure is intentionally narrow so
	// mislabeled or truncated images do not reach providers as malformed
	// data URLs. WebP is allowed because providers accept it and the Go
	// stdlib decoder used here does not register WebP support by default.
	switch strings.ToLower(strings.TrimSpace(mime)) {
	case "image/webp":
		return detectRawImageMIME(localPath) == "image/webp"
	default:
		return false
	}
}

func detectRawImageMIME(localPath string) string {
	kind, err := filetype.MatchFile(localPath)
	if err != nil || kind == filetype.Unknown {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(kind.MIME.Value))
}

func buildRawInlineImageCandidate(
	localPath string,
	mime string,
	info os.FileInfo,
	opts mediaResolveOptions,
) (inlineImageCandidate, error) {
	// max_media_size remains the guardrail for paths that need to read the
	// original file bytes directly into memory. Decode+resize can handle
	// larger source files safely via decoded-pixel validation below.
	if info.Size() > int64(opts.maxSourceBytes) {
		return inlineImageCandidate{}, fmt.Errorf(
			"raw inline source exceeds max_media_size (%d > %d bytes)",
			info.Size(),
			opts.maxSourceBytes,
		)
	}
	return readRawInlineImage(localPath, mime, opts.imageInput.MaxInlineBytes)
}

func readRawInlineImage(
	localPath string,
	mime string,
	maxInlineBytes int,
) (inlineImageCandidate, error) {
	data, err := os.ReadFile(localPath)
	if err != nil {
		return inlineImageCandidate{}, err
	}
	if maxInlineBytes > 0 && inlinePayloadSize(mime, len(data)) > maxInlineBytes {
		return inlineImageCandidate{}, fmt.Errorf(
			"inline payload exceeds configured image_input.max_inline_bytes (%d > %d bytes)",
			inlinePayloadSize(mime, len(data)),
			maxInlineBytes,
		)
	}
	return inlineImageCandidate{mime: mime, data: data}, nil
}

func validateInlineDecodeConfig(cfg image.Config) error {
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return fmt.Errorf("decoded image has invalid dimensions (%dx%d)", cfg.Width, cfg.Height)
	}
	pixels := int64(cfg.Width) * int64(cfg.Height)
	if pixels > maxInlineDecodedPixels {
		return fmt.Errorf(
			"decoded image exceeds safety pixel limit (%d > %d pixels)",
			pixels,
			maxInlineDecodedPixels,
		)
	}
	return nil
}

func compressDecodedImage(
	img image.Image,
	sourceMIME string,
	policy config.ResolvedImageInputConfig,
) (inlineImageCandidate, error) {
	origBounds := img.Bounds()
	width, height := fitWithin(origBounds.Dx(), origBounds.Dy(), policy.MaxWidth, policy.MaxHeight)
	if width <= 0 || height <= 0 {
		width = maxInt(origBounds.Dx(), 1)
		height = maxInt(origBounds.Dy(), 1)
	}

	hasAlpha := imageHasTransparency(img)
	formatOrder := preferredInlineFormats(policy.TargetFormat, hasAlpha)
	var best inlineImageCandidate
	bestSize := math.MaxInt

	for attempt := 0; attempt < 5; attempt++ {
		working := img
		if width != origBounds.Dx() || height != origBounds.Dy() {
			working = resizeImageBilinear(img, width, height)
		}

		for _, format := range formatOrder {
			if format == "jpeg" {
				for quality := policy.JPEGQuality; quality >= 40; quality -= 10 {
					candidate, err := encodeDecodedImage(working, format, quality)
					if err != nil {
						return inlineImageCandidate{}, err
					}
					size := inlinePayloadSize(candidate.mime, len(candidate.data))
					if size < bestSize {
						best = candidate
						bestSize = size
					}
					if policy.MaxInlineBytes <= 0 || size <= policy.MaxInlineBytes {
						return candidate, nil
					}
				}
				continue
			}

			candidate, err := encodeDecodedImage(working, format, policy.JPEGQuality)
			if err != nil {
				return inlineImageCandidate{}, err
			}
			size := inlinePayloadSize(candidate.mime, len(candidate.data))
			if size < bestSize {
				best = candidate
				bestSize = size
			}
			if policy.MaxInlineBytes <= 0 || size <= policy.MaxInlineBytes {
				return candidate, nil
			}
		}

		if width <= 320 && height <= 320 {
			break
		}

		nextWidth := maxInt(int(math.Round(float64(width)*0.85)), 1)
		nextHeight := maxInt(int(math.Round(float64(height)*0.85)), 1)
		if nextWidth == width && nextHeight == height {
			break
		}
		width = nextWidth
		height = nextHeight
	}

	if best.data != nil {
		return inlineImageCandidate{}, fmt.Errorf(
			"best-effort inline payload still too large (%d > %d bytes)",
			bestSize,
			policy.MaxInlineBytes,
		)
	}

	return inlineImageCandidate{}, fmt.Errorf("failed to compress inline image")
}

func preferredInlineFormats(targetFormat string, hasAlpha bool) []string {
	switch strings.ToLower(strings.TrimSpace(targetFormat)) {
	case "jpeg":
		return []string{"jpeg"}
	case "png":
		return []string{"png"}
	default:
		if hasAlpha {
			return []string{"png", "jpeg"}
		}
		return []string{"jpeg", "png"}
	}
}

func encodeDecodedImage(img image.Image, format string, quality int) (inlineImageCandidate, error) {
	var buf bytes.Buffer

	switch format {
	case "jpeg":
		if err := jpeg.Encode(&buf, flattenImageForJPEG(img), &jpeg.Options{Quality: quality}); err != nil {
			return inlineImageCandidate{}, err
		}
		return inlineImageCandidate{mime: "image/jpeg", data: buf.Bytes()}, nil
	case "png":
		encoder := png.Encoder{CompressionLevel: png.DefaultCompression}
		if err := encoder.Encode(&buf, img); err != nil {
			return inlineImageCandidate{}, err
		}
		return inlineImageCandidate{mime: "image/png", data: buf.Bytes()}, nil
	default:
		return inlineImageCandidate{}, fmt.Errorf("unsupported target image format %q", format)
	}
}

func flattenImageForJPEG(img image.Image) image.Image {
	bounds := img.Bounds()
	dst := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			src := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			a := float64(src.A) / 255.0
			r := uint8(math.Round(float64(src.R)*a + 255*(1-a)))
			g := uint8(math.Round(float64(src.G)*a + 255*(1-a)))
			b := uint8(math.Round(float64(src.B)*a + 255*(1-a)))
			dst.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
	return dst
}

func resizeImageBilinear(src image.Image, width, height int) *image.NRGBA {
	srcBounds := src.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()
	if srcW == width && srcH == height {
		return toNRGBA(src)
	}

	dst := image.NewNRGBA(image.Rect(0, 0, width, height))
	scaleX := float64(srcW) / float64(width)
	scaleY := float64(srcH) / float64(height)

	for y := 0; y < height; y++ {
		srcY := (float64(y)+0.5)*scaleY - 0.5
		y0 := clampCoord(int(math.Floor(srcY)), srcH)
		y1 := clampCoord(y0+1, srcH)
		wy := srcY - math.Floor(srcY)
		for x := 0; x < width; x++ {
			srcX := (float64(x)+0.5)*scaleX - 0.5
			x0 := clampCoord(int(math.Floor(srcX)), srcW)
			x1 := clampCoord(x0+1, srcW)
			wx := srcX - math.Floor(srcX)

			c00 := color.NRGBAModel.Convert(src.At(srcBounds.Min.X+x0, srcBounds.Min.Y+y0)).(color.NRGBA)
			c10 := color.NRGBAModel.Convert(src.At(srcBounds.Min.X+x1, srcBounds.Min.Y+y0)).(color.NRGBA)
			c01 := color.NRGBAModel.Convert(src.At(srcBounds.Min.X+x0, srcBounds.Min.Y+y1)).(color.NRGBA)
			c11 := color.NRGBAModel.Convert(src.At(srcBounds.Min.X+x1, srcBounds.Min.Y+y1)).(color.NRGBA)

			dst.SetNRGBA(x, y, bilinearInterpolate(c00, c10, c01, c11, wx, wy))
		}
	}

	return dst
}

func bilinearInterpolate(c00, c10, c01, c11 color.NRGBA, wx, wy float64) color.NRGBA {
	return color.NRGBA{
		R: bilinearChannel(c00.R, c10.R, c01.R, c11.R, wx, wy),
		G: bilinearChannel(c00.G, c10.G, c01.G, c11.G, wx, wy),
		B: bilinearChannel(c00.B, c10.B, c01.B, c11.B, wx, wy),
		A: bilinearChannel(c00.A, c10.A, c01.A, c11.A, wx, wy),
	}
}

func bilinearChannel(c00, c10, c01, c11 uint8, wx, wy float64) uint8 {
	top := float64(c00)*(1-wx) + float64(c10)*wx
	bottom := float64(c01)*(1-wx) + float64(c11)*wx
	return uint8(math.Round(top*(1-wy) + bottom*wy))
}

func toNRGBA(src image.Image) *image.NRGBA {
	bounds := src.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			dst.SetNRGBA(x-bounds.Min.X, y-bounds.Min.Y, color.NRGBAModel.Convert(src.At(x, y)).(color.NRGBA))
		}
	}
	return dst
}

func imageHasTransparency(img image.Image) bool {
	if opaque, ok := img.(interface{ Opaque() bool }); ok && opaque.Opaque() {
		return false
	}

	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a != 0xffff {
				return true
			}
		}
	}
	return false
}

func fitWithin(width, height, maxWidth, maxHeight int) (int, int) {
	if width <= 0 || height <= 0 {
		return width, height
	}

	scale := 1.0
	if maxWidth > 0 && width > maxWidth {
		scale = math.Min(scale, float64(maxWidth)/float64(width))
	}
	if maxHeight > 0 && height > maxHeight {
		scale = math.Min(scale, float64(maxHeight)/float64(height))
	}
	if scale >= 1 {
		return width, height
	}

	return maxInt(int(math.Round(float64(width)*scale)), 1), maxInt(int(math.Round(float64(height)*scale)), 1)
}

func inlinePayloadSize(mime string, dataLen int) int {
	return len("data:"+mime+";base64,") + base64.StdEncoding.EncodedLen(dataLen)
}

func encodeBytesToDataURL(mime string, data []byte) string {
	if len(data) == 0 {
		return ""
	}
	var buf bytes.Buffer
	buf.Grow(inlinePayloadSize(mime, len(data)))
	buf.WriteString("data:")
	buf.WriteString(mime)
	buf.WriteString(";base64,")

	encoder := base64.NewEncoder(base64.StdEncoding, &buf)
	_, _ = encoder.Write(data)
	_ = encoder.Close()
	return buf.String()
}

func clampCoord(v, limit int) int {
	if limit <= 0 {
		return 0
	}
	if v < 0 {
		return 0
	}
	if v >= limit {
		return limit - 1
	}
	return v
}

func maxInt(v, fallback int) int {
	if v > 0 {
		return v
	}
	return fallback
}
