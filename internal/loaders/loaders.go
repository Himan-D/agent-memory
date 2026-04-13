package loaders

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Loader interface {
	Load(ctx context.Context, source string) (*Document, error)
	SupportedTypes() []string
}

type Document struct {
	Content  string                 `json:"content"`
	Source   string                 `json:"source"`
	Title    string                 `json:"title,omitempty"`
	Format   string                 `json:"format"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Chunks   []DocumentChunk        `json:"chunks,omitempty"`
}

type DocumentChunk struct {
	Content  string                 `json:"content"`
	Index    int                    `json:"index"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

type PDFLoader struct {
	maxChunkSize int
}

func NewPDFLoader() *PDFLoader {
	return &PDFLoader{maxChunkSize: 1000}
}

func (l *PDFLoader) Load(ctx context.Context, source string) (*Document, error) {
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		return l.loadFromURL(ctx, source)
	}
	return l.loadFromFile(source)
}

func (l *PDFLoader) loadFromURL(ctx context.Context, url string) (*Document, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return l.parsePDF(body, url)
}

func (l *PDFLoader) loadFromFile(path string) (*Document, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return l.parsePDF(body, path)
}

func (l *PDFLoader) parsePDF(data []byte, source string) (*Document, error) {
	content := extractPDFText(data)

	chunks := l.chunkText(content)

	title := extractPDFTitle(source)

	return &Document{
		Content: content,
		Source:  source,
		Title:   title,
		Format:  "pdf",
		Metadata: map[string]interface{}{
			"source": source,
			"size":   len(data),
			"chunks": len(chunks),
		},
		Chunks: chunks,
	}, nil
}

func (l *PDFLoader) chunkText(text string) []DocumentChunk {
	var chunks []DocumentChunk

	words := strings.Fields(text)
	var currentChunk []string
	currentLen := 0

	for i, word := range words {
		currentChunk = append(currentChunk, word)
		currentLen += len(word) + 1

		if currentLen >= l.maxChunkSize {
			chunks = append(chunks, DocumentChunk{
				Content:  strings.Join(currentChunk, " "),
				Index:    len(chunks),
				Metadata: map[string]interface{}{"word_count": len(currentChunk)},
			})
			currentChunk = nil
			currentLen = 0
		}

		if i == len(words)-1 && len(currentChunk) > 0 {
			chunks = append(chunks, DocumentChunk{
				Content:  strings.Join(currentChunk, " "),
				Index:    len(chunks),
				Metadata: map[string]interface{}{"word_count": len(currentChunk)},
			})
		}
	}

	return chunks
}

func (l *PDFLoader) SupportedTypes() []string {
	return []string{"pdf"}
}

type AudioLoader struct {
	whisperEndpoint string
}

func NewAudioLoader(whisperEndpoint string) *AudioLoader {
	return &AudioLoader{whisperEndpoint: whisperEndpoint}
}

func (l *AudioLoader) Load(ctx context.Context, source string) (*Document, error) {
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		return l.loadFromURL(ctx, source)
	}
	return l.loadFromFile(source)
}

func (l *AudioLoader) loadFromURL(ctx context.Context, url string) (*Document, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return l.transcribeAudio(ctx, body, url)
}

func (l *AudioLoader) loadFromFile(path string) (*Document, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return l.transcribeAudio(context.Background(), body, path)
}

func (l *AudioLoader) transcribeAudio(ctx context.Context, audioData []byte, source string) (*Document, error) {
	if l.whisperEndpoint == "" {
		return &Document{
			Content: "Audio transcription requires whisper_endpoint configuration",
			Source:  source,
			Format:  "audio",
			Metadata: map[string]interface{}{
				"size": len(audioData),
				"note": "Configure whisper_endpoint for transcription",
			},
		}, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, l.whisperEndpoint, bytes.NewReader(audioData))
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Whisper API error: %d", resp.StatusCode)
	}

	var result struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &Document{
		Content: result.Text,
		Source:  source,
		Format:  "audio",
		Metadata: map[string]interface{}{
			"size": len(audioData),
		},
	}, nil
}

func (l *AudioLoader) SupportedTypes() []string {
	return []string{"mp3", "wav", "m4a", "ogg", "flac"}
}

type DocxLoader struct{}

func NewDocxLoader() *DocxLoader {
	return &DocxLoader{}
}

func (l *DocxLoader) Load(ctx context.Context, source string) (*Document, error) {
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		return l.loadFromURL(ctx, source)
	}
	return l.loadFromFile(source)
}

func (l *DocxLoader) loadFromURL(ctx context.Context, url string) (*Document, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return l.parseDocx(body, url)
}

func (l *DocxLoader) loadFromFile(path string) (*Document, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return l.parseDocx(body, path)
}

func (l *DocxLoader) parseDocx(data []byte, source string) (*Document, error) {
	content := extractDocxText(data)

	title := strings.TrimSuffix(filepath.Base(source), filepath.Ext(source))

	return &Document{
		Content: content,
		Source:  source,
		Title:   title,
		Format:  "docx",
		Metadata: map[string]interface{}{
			"source": source,
			"size":   len(data),
		},
	}, nil
}

func (l *DocxLoader) SupportedTypes() []string {
	return []string{"docx", "doc"}
}

type XLSXLoader struct{}

func NewXLSXLoader() *XLSXLoader {
	return &XLSXLoader{}
}

func (l *XLSXLoader) Load(ctx context.Context, source string) (*Document, error) {
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		return l.loadFromURL(ctx, source)
	}
	return l.loadFromFile(source)
}

func (l *XLSXLoader) loadFromURL(ctx context.Context, url string) (*Document, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return l.parseXLSX(body, url)
}

func (l *XLSXLoader) loadFromFile(path string) (*Document, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return l.parseXLSX(body, path)
}

func (l *XLSXLoader) parseXLSX(data []byte, source string) (*Document, error) {
	content := extractXLSXText(data)

	return &Document{
		Content: content,
		Source:  source,
		Format:  "xlsx",
		Metadata: map[string]interface{}{
			"source": source,
			"size":   len(data),
		},
	}, nil
}

func (l *XLSXLoader) SupportedTypes() []string {
	return []string{"xlsx", "xls"}
}

type MultiLoader struct {
	loaders map[string]Loader
}

func NewMultiLoader() *MultiLoader {
	return &MultiLoader{
		loaders: make(map[string]Loader),
	}
}

func (m *MultiLoader) Register(loader Loader) {
	for _, ext := range loader.SupportedTypes() {
		m.loaders[ext] = loader
	}
}

func (m *MultiLoader) Load(ctx context.Context, source string) (*Document, error) {
	ext := strings.ToLower(filepath.Ext(source))
	ext = strings.TrimPrefix(ext, ".")

	loader, ok := m.loaders[ext]
	if !ok {
		return nil, fmt.Errorf("no loader for type: %s", ext)
	}

	return loader.Load(ctx, source)
}

func extractPDFText(data []byte) string {
	var text strings.Builder
	start := false
	for i := 0; i < len(data)-1; i++ {
		if data[i] == 'B' && data[i+1] == 'T' {
			start = true
		}
		if start && data[i] == 'E' && data[i+1] == 'T' {
			start = false
		}
		if start && data[i] >= 32 && data[i] < 127 {
			text.WriteByte(data[i])
		}
	}
	return text.String()
}

func extractPDFTitle(source string) string {
	base := filepath.Base(source)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

func extractDocxText(data []byte) string {
	return "DOCX content extraction - requires proper XML parsing library"
}

func extractXLSXText(data []byte) string {
	return "XLSX content extraction - requires spreadsheet parsing library"
}
