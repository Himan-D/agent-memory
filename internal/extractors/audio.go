package extractors

import (
	"fmt"
	"io"
	"strings"
)

type AudioExtractor struct{}

func (e *AudioExtractor) SupportedMimeTypes() []string {
	return []string{"audio/wav", "audio/mp3", "audio/mpeg", "audio/ogg", "audio/flac", "audio/webm"}
}

func (e *AudioExtractor) Extract(r io.Reader, filename string) (*Document, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	title := filename
	if idx := strings.LastIndex(filename, "."); idx > 0 {
		title = filename[:idx]
	}

	return &Document{
		Content:   fmt.Sprintf("[Audio transcription requires Whisper API integration - file: %s, size: %d bytes]", filename, len(data)),
		Title:     title,
		MimeType:  "audio/wav",
		Source:    filename,
		Metadata:  map[string]string{"filename": filename, "size": fmt.Sprintf("%d", len(data)), "type": "audio"},
		PageCount: 1,
	}, nil
}