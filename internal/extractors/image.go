package extractors

import (
	"fmt"
	"io"
	"strings"
)

type ImageExtractor struct{}

func (e *ImageExtractor) SupportedMimeTypes() []string {
	return []string{"image/png", "image/jpeg", "image/gif", "image/webp"}
}

func (e *ImageExtractor) Extract(r io.Reader, filename string) (*Document, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	title := filename
	if idx := strings.LastIndex(filename, "."); idx > 0 {
		title = filename[:idx]
	}

	return &Document{
		Content:   fmt.Sprintf("[Image: %s, size: %d bytes]", filename, len(data)),
		Title:     title,
		MimeType:  "image/png",
		Source:    filename,
		Metadata:  map[string]string{"filename": filename, "size": fmt.Sprintf("%d", len(data)), "type": "image"},
		PageCount: 1,
	}, nil
}