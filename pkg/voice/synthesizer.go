package voice

import "context"

// Synthesizer converts text to audio and returns the file path of the resulting audio file.
// The caller is responsible for cleaning up the returned temp file.
type Synthesizer interface {
	Synthesize(ctx context.Context, text string) (filePath string, err error)
	IsAvailable() bool
}
