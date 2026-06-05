package extractors

import (
	"fmt"
	"io"
	"strings"
)

type PDFExtractor struct{}

func (e *PDFExtractor) SupportedMimeTypes() []string {
	return []string{"application/pdf"}
}

func (e *PDFExtractor) Extract(r io.Reader, filename string) (*Document, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	content := extractPDFText(data)

	return &Document{
		Content:   content,
		Title:     strings.TrimSuffix(filename, ".pdf"),
		MimeType:  "application/pdf",
		Source:    filename,
		Metadata:  map[string]string{"filename": filename, "size": fmt.Sprintf("%d", len(data))},
		PageCount: max(1, strings.Count(content, "\n\n")+1),
	}, nil
}

func extractPDFText(data []byte) string {
	startMarker := []byte("stream\n")
	endMarker := []byte("\nendstream")

	var content strings.Builder
	pos := 0
	for pos < len(data) {
		startIdx := indexOf(data, startMarker, pos)
		if startIdx == -1 {
			break
		}
		startIdx += len(startMarker)

		endIdx := indexOf(data, endMarker, startIdx)
		if endIdx == -1 {
			break
		}

		stream := string(data[startIdx:endIdx])

		for _, line := range strings.Split(stream, "\n") {
			line = strings.TrimSpace(line)
			if len(line) > 1 && !strings.HasPrefix(line, "/") && !strings.HasPrefix(line, "%") {
				if isPrintableText(line) {
					content.WriteString(line)
					content.WriteString(" ")
				}
			}
		}

		pos = endIdx + len(endMarker)
	}

	result := content.String()
	if strings.TrimSpace(result) == "" {
		result = "[PDF content extraction requires a dedicated PDF parser - install pdftotext or similar tool]"
	}

	return result
}

func indexOf(data []byte, marker []byte, start int) int {
	for i := start; i <= len(data)-len(marker); i++ {
		match := true
		for j := 0; j < len(marker); j++ {
			if data[i+j] != marker[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

func isPrintableText(s string) bool {
	printable := 0
	for _, r := range s {
		if r >= 32 && r < 127 {
			printable++
		}
	}
	if len(s) == 0 {
		return false
	}
	return float64(printable)/float64(len(s)) > 0.7
}
