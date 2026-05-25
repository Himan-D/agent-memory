package extractors

import (
	"fmt"
	"io"
	"strings"
)

type TextExtractor struct{}

func (e *TextExtractor) SupportedMimeTypes() []string {
	return []string{"text/plain", "text/markdown", "application/octet-stream"}
}

func (e *TextExtractor) Extract(r io.Reader, filename string) (*Document, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	content := string(data)
	title := filename
	if idx := strings.LastIndex(filename, "."); idx > 0 {
		title = filename[:idx]
	}

	return &Document{
		Content:   content,
		Title:     title,
		MimeType:  "text/plain",
		Source:    filename,
		Metadata:  map[string]string{"filename": filename, "lines": fmt.Sprintf("%d", strings.Count(content, "\n")+1)},
		PageCount: max(1, strings.Count(content, "\n\n")+1),
	}, nil
}