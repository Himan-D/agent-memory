package extractors

import (
	"io"
	"strings"
)

type Document struct {
	Content    string
	Title      string
	MimeType  string
	Source     string
	Metadata   map[string]string
	PageCount  int
}

type Extractor interface {
	Extract(r io.Reader, filename string) (*Document, error)
	SupportedMimeTypes() []string
}

type Registry struct {
	extractors map[string]Extractor
}

func NewRegistry() *Registry {
	r := &Registry{
		extractors: make(map[string]Extractor),
	}
	r.registerDefaults()
	return r
}

func (r *Registry) registerDefaults() {
	text := &TextExtractor{}
	for _, mt := range text.SupportedMimeTypes() {
		r.extractors[mt] = text
	}

	html := &HTMLExtractor{}
	for _, mt := range html.SupportedMimeTypes() {
		r.extractors[mt] = html
	}

	pdf := &PDFExtractor{}
	for _, mt := range pdf.SupportedMimeTypes() {
		r.extractors[mt] = pdf
	}

	img := &ImageExtractor{}
	for _, mt := range img.SupportedMimeTypes() {
		r.extractors[mt] = img
	}

	audio := &AudioExtractor{}
	for _, mt := range audio.SupportedMimeTypes() {
		r.extractors[mt] = audio
	}
}

func (r *Registry) Register(mimeType string, ext Extractor) {
	r.extractors[mimeType] = ext
}

func (r *Registry) Extract(mimeType string, reader io.Reader, filename string) (*Document, error) {
	ext, ok := r.extractors[mimeType]
	if !ok {
		ext, ok = r.extractors["application/octet-stream"]
		if !ok {
			return nil, ErrUnsupportedMimeType
		}
	}
	return ext.Extract(reader, filename)
}

func (r *Registry) FindByFilename(filename string) (Extractor, bool) {
	if strings.HasSuffix(filename, ".txt") {
		return r.extractors["text/plain"], true
	}
	if strings.HasSuffix(filename, ".md") {
		return r.extractors["text/markdown"], true
	}
	if strings.HasSuffix(filename, ".html") || strings.HasSuffix(filename, ".htm") {
		return r.extractors["text/html"], true
	}
	return nil, false
}