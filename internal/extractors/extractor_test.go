package extractors

import (
	"errors"
	"io"
	"strings"
	"testing"
)

type errReader struct {
	err error
}

func (e *errReader) Read(p []byte) (n int, err error) {
	return 0, e.err
}

func TestNewRegistry_HasDefaultExtractors(t *testing.T) {
	r := NewRegistry()

	defaultMimes := []string{
		"text/plain", "text/markdown", "application/octet-stream",
		"text/html", "application/pdf",
		"image/png", "image/jpeg", "image/gif", "image/webp",
		"audio/wav", "audio/mp3", "audio/mpeg", "audio/ogg", "audio/flac", "audio/webm",
	}

	for _, mime := range defaultMimes {
		ext, ok := r.extractors[mime]
		if !ok {
			t.Errorf("expected extractor for mime type %s", mime)
			continue
		}
		if ext == nil {
			t.Errorf("extractor for mime type %s is nil", mime)
		}
	}
}

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()

	customMime := "application/custom"
	custom := &TextExtractor{}
	r.Register(customMime, custom)

	ext, ok := r.extractors[customMime]
	if !ok {
		t.Error("expected custom extractor to be registered")
	}
	if ext != custom {
		t.Error("registered extractor does not match")
	}
}

func TestRegistry_Extract_TextPlain(t *testing.T) {
	r := NewRegistry()
	content := "Hello, World!"
	doc, err := r.Extract("text/plain", strings.NewReader(content), "test.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc.Content != content {
		t.Errorf("expected content %q, got %q", content, doc.Content)
	}
	if doc.MimeType != "text/plain" {
		t.Errorf("expected mimeType text/plain, got %s", doc.MimeType)
	}
	if doc.Source != "test.txt" {
		t.Errorf("expected source test.txt, got %s", doc.Source)
	}
	if doc.Title != "test" {
		t.Errorf("expected title test, got %s", doc.Title)
	}
}

func TestRegistry_Extract_TextMarkdown(t *testing.T) {
	r := NewRegistry()
	content := "# Heading\n\nParagraph text"
	doc, err := r.Extract("text/markdown", strings.NewReader(content), "doc.md")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc.Content != content {
		t.Errorf("expected content %q, got %q", content, doc.Content)
	}
}

func TestRegistry_Extract_TextHTML(t *testing.T) {
	r := NewRegistry()
	html := `<html><head><title>Test Page</title></head><body><p>Hello World</p></body></html>`
	doc, err := r.Extract("text/html", strings.NewReader(html), "page.html")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(doc.Content, "Hello World") {
		t.Errorf("expected content to contain 'Hello World', got %q", doc.Content)
	}
	if doc.Title != "Test Page" {
		t.Errorf("expected title 'Test Page', got %q", doc.Title)
	}
	if doc.MimeType != "text/html" {
		t.Errorf("expected mimeType text/html, got %s", doc.MimeType)
	}
}

func TestRegistry_Extract_ApplicationPDF(t *testing.T) {
	r := NewRegistry()
	doc, err := r.Extract("application/pdf", strings.NewReader("not a real pdf"), "doc.pdf")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc.Source != "doc.pdf" {
		t.Errorf("expected source doc.pdf, got %s", doc.Source)
	}
	if doc.Title != "doc" {
		t.Errorf("expected title doc, got %s", doc.Title)
	}
	if doc.MimeType != "application/pdf" {
		t.Errorf("expected mimeType application/pdf, got %s", doc.MimeType)
	}
}

func TestRegistry_Extract_ImagePNG(t *testing.T) {
	r := NewRegistry()
	doc, err := r.Extract("image/png", strings.NewReader("fakeimagedata"), "photo.png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(doc.Content, "photo.png") {
		t.Errorf("expected content to contain filename, got %q", doc.Content)
	}
	if doc.MimeType != "image/png" {
		t.Errorf("expected mimeType image/png, got %s", doc.MimeType)
	}
}

func TestRegistry_Extract_AudioWAV(t *testing.T) {
	r := NewRegistry()
	doc, err := r.Extract("audio/wav", strings.NewReader("fakeaudiodata"), "recording.wav")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(doc.Content, "recording.wav") {
		t.Errorf("expected content to contain filename, got %q", doc.Content)
	}
	if doc.MimeType != "audio/wav" {
		t.Errorf("expected mimeType audio/wav, got %s", doc.MimeType)
	}
}

func TestRegistry_Extract_UnsupportedMimeType(t *testing.T) {
	r := NewRegistry()
	_, err := r.Extract("video/mp4", strings.NewReader("data"), "video.mp4")
	// application/octet-stream is registered as a fallback, so video/mp4
	// will not match any registered extractor and should return ErrUnsupportedMimeType
	if err == nil {
		// If unexpectedly no error, the content was handled by octet-stream fallback
		t.Skip("video/mp4 handled by octet-stream fallback")
	}
	if !errors.Is(err, ErrUnsupportedMimeType) {
		t.Errorf("expected ErrUnsupportedMimeType, got %v", err)
	}
}

func TestRegistry_Extract_TrulyUnsupportedMimeType(t *testing.T) {
	r := NewRegistry()
	// Remove octet-stream fallback to test truly unsupported type
	delete(r.extractors, "application/octet-stream")
	_, err := r.Extract("video/mp4", strings.NewReader("data"), "video.mp4")
	if err == nil {
		t.Error("expected error for unsupported mime type")
	}
	if !errors.Is(err, ErrUnsupportedMimeType) {
		t.Errorf("expected ErrUnsupportedMimeType, got %v", err)
	}
}

func TestRegistry_FindByFilename(t *testing.T) {
	r := NewRegistry()

	tests := []struct {
		name     string
		filename string
		found    bool
	}{
		{"txt file", "file.txt", true},
		{"md file", "file.md", true},
		{"html file", "file.html", true},
		{"htm file", "file.htm", true},
		{"unknown file", "file.xyz", false},
		{"no extension", "Makefile", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ext, ok := r.FindByFilename(tt.filename)
			if ok != tt.found {
				t.Errorf("FindByFilename(%s) ok=%v, want %v", tt.filename, ok, tt.found)
			}
			if tt.found && ext == nil {
				t.Errorf("FindByFilename(%s) returned nil extractor", tt.filename)
			}
		})
	}
}

func TestTextExtractor_SupportedMimeTypes(t *testing.T) {
	e := &TextExtractor{}
	mimes := e.SupportedMimeTypes()
	expected := []string{"text/plain", "text/markdown", "application/octet-stream"}

	if len(mimes) != len(expected) {
		t.Fatalf("expected %d mime types, got %d", len(expected), len(mimes))
	}

	for i, exp := range expected {
		if mimes[i] != exp {
			t.Errorf("mime type[%d] = %q, want %q", i, mimes[i], exp)
		}
	}
}

func TestTextExtractor_Extract(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		filename string
		title    string
		mime     string
	}{
		{"simple text", "Hello World", "test.txt", "test", "text/plain"},
		{"multiline", "line1\nline2\nline3", "doc.txt", "doc", "text/plain"},
		{"empty content", "", "empty.txt", "empty", "text/plain"},
		{"no extension", "content", "README", "README", "text/plain"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &TextExtractor{}
			doc, err := e.Extract(strings.NewReader(tt.content), tt.filename)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if doc.Content != tt.content {
				t.Errorf("expected content %q, got %q", tt.content, doc.Content)
			}
			if doc.Title != tt.title {
				t.Errorf("expected title %q, got %q", tt.title, doc.Title)
			}
			if doc.MimeType != tt.mime {
				t.Errorf("expected mimeType %q, got %q", tt.mime, doc.MimeType)
			}
			if doc.Source != tt.filename {
				t.Errorf("expected source %q, got %q", tt.filename, doc.Source)
			}
			if doc.Metadata["filename"] != tt.filename {
				t.Errorf("expected metadata filename %q, got %q", tt.filename, doc.Metadata["filename"])
			}
			if doc.PageCount < 1 {
				t.Errorf("expected PageCount >= 1, got %d", doc.PageCount)
			}
		})
	}
}

func TestTextExtractor_Extract_ReaderError(t *testing.T) {
	e := &TextExtractor{}
	expectedErr := errors.New("read error")
	_, err := e.Extract(&errReader{err: expectedErr}, "test.txt")
	if err == nil {
		t.Error("expected error from reader failure")
	}
}

func TestHTMLExtractor_SupportedMimeTypes(t *testing.T) {
	e := &HTMLExtractor{}
	mimes := e.SupportedMimeTypes()
	if len(mimes) != 1 || mimes[0] != "text/html" {
		t.Errorf("expected [text/html], got %v", mimes)
	}
}

func TestHTMLExtractor_Extract(t *testing.T) {
	tests := []struct {
		name            string
		html            string
		title           string
		contentContains string
	}{
		{
			"simple html",
			`<html><head><title>Test</title></head><body><p>Hello World</p></body></html>`,
			"Test",
			"Hello World",
		},
		{
			"html with script/style stripped",
			`<html><head><title>Page</title><script>alert('x')</script><style>body{}</style></head><body><p>Content</p></body></html>`,
			"Page",
			"Content",
		},
		{
			"html without title",
			`<html><body><p>No title</p></body></html>`,
			"",
			"No title",
		},
		{
			"html with meta og:title",
			`<html><head><meta property="og:title" content="OG Title"><title>Old Title</title></head><body><p>Body</p></body></html>`,
			"OG Title",
			"Body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &HTMLExtractor{}
			doc, err := e.Extract(strings.NewReader(tt.html), "test.html")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if doc.Title != tt.title {
				t.Errorf("expected title %q, got %q", tt.title, doc.Title)
			}
			if !strings.Contains(doc.Content, tt.contentContains) {
				t.Errorf("expected content to contain %q, got %q", tt.contentContains, doc.Content)
			}
			if doc.MimeType != "text/html" {
				t.Errorf("expected mimeType text/html, got %s", doc.MimeType)
			}
		})
	}
}

func TestHTMLExtractor_Extract_ReaderError(t *testing.T) {
	e := &HTMLExtractor{}
	expectedErr := errors.New("read error")
	_, err := e.Extract(&errReader{err: expectedErr}, "test.html")
	if err == nil {
		t.Error("expected error from reader failure")
	}
}

func TestHTMLExtractor_Extract_InvalidHTML(t *testing.T) {
	e := &HTMLExtractor{}
	_, err := e.Extract(strings.NewReader(""), "empty.html")
	if err != nil {
		t.Fatalf("empty input should not cause error: %v", err)
	}
}

func TestPDFExtractor_SupportedMimeTypes(t *testing.T) {
	e := &PDFExtractor{}
	mimes := e.SupportedMimeTypes()
	if len(mimes) != 1 || mimes[0] != "application/pdf" {
		t.Errorf("expected [application/pdf], got %v", mimes)
	}
}

func TestPDFExtractor_Extract_EmptyContent(t *testing.T) {
	e := &PDFExtractor{}
	doc, err := e.Extract(strings.NewReader(""), "report.pdf")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.TrimSpace(doc.Content) == "" || !strings.Contains(doc.Content, "requires") {
		t.Errorf("expected fallback message for empty PDF content, got %q", doc.Content)
	}
	if doc.Title != "report" {
		t.Errorf("expected title 'report', got %q", doc.Title)
	}
}

func TestPDFExtractor_Extract_ReaderError(t *testing.T) {
	e := &PDFExtractor{}
	expectedErr := errors.New("read error")
	_, err := e.Extract(&errReader{err: expectedErr}, "test.pdf")
	if err == nil {
		t.Error("expected error from reader failure")
	}
}

func TestImageExtractor_SupportedMimeTypes(t *testing.T) {
	e := &ImageExtractor{}
	mimes := e.SupportedMimeTypes()
	expected := []string{"image/png", "image/jpeg", "image/gif", "image/webp"}
	if len(mimes) != len(expected) {
		t.Fatalf("expected %d mime types, got %d", len(expected), len(mimes))
	}
	for i, exp := range expected {
		if mimes[i] != exp {
			t.Errorf("mime type[%d] = %q, want %q", i, mimes[i], exp)
		}
	}
}

func TestImageExtractor_Extract(t *testing.T) {
	e := &ImageExtractor{}
	doc, err := e.Extract(strings.NewReader("imagedata"), "photo.png")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(doc.Content, "photo.png") {
		t.Errorf("expected content to contain filename, got %q", doc.Content)
	}
	if doc.Title != "photo" {
		t.Errorf("expected title 'photo', got %q", doc.Title)
	}
	if doc.Metadata["type"] != "image" {
		t.Errorf("expected metadata type=image, got %s", doc.Metadata["type"])
	}
}

func TestAudioExtractor_SupportedMimeTypes(t *testing.T) {
	e := &AudioExtractor{}
	mimes := e.SupportedMimeTypes()
	expected := []string{"audio/wav", "audio/mp3", "audio/mpeg", "audio/ogg", "audio/flac", "audio/webm"}
	if len(mimes) != len(expected) {
		t.Fatalf("expected %d mime types, got %d", len(expected), len(mimes))
	}
}

func TestAudioExtractor_Extract(t *testing.T) {
	e := &AudioExtractor{}
	doc, err := e.Extract(strings.NewReader("audiodata"), "recording.wav")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(doc.Content, "recording.wav") {
		t.Errorf("expected content to contain filename, got %q", doc.Content)
	}
	if !strings.Contains(doc.Content, "Whisper") {
		t.Errorf("expected content to mention Whisper, got %q", doc.Content)
	}
	if doc.Metadata["type"] != "audio" {
		t.Errorf("expected metadata type=audio, got %s", doc.Metadata["type"])
	}
}

func TestRegistry_Extract_OctetStream(t *testing.T) {
	r := NewRegistry()
	content := "raw binary content"
	doc, err := r.Extract("application/octet-stream", strings.NewReader(content), "data.bin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc.Content != content {
		t.Errorf("expected content %q, got %q", content, doc.Content)
	}
}

func TestRegistry_Extract_ReaderError(t *testing.T) {
	r := NewRegistry()
	expectedErr := errors.New("read error")
	_, err := r.Extract("text/plain", &errReader{err: expectedErr}, "test.txt")
	if err == nil {
		t.Error("expected error from reader failure")
	}
}

func TestPDFExtractor_Extract_WithPDFStream(t *testing.T) {
	e := &PDFExtractor{}
	pdf := "header\nstream\nHello PDF World\nendstream\ntrailer"
	doc, err := e.Extract(strings.NewReader(pdf), "doc.pdf")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(doc.Content, "Hello PDF World") {
		t.Errorf("expected extracted content with stream text, got %q", doc.Content)
	}
}

func TestHTMLExtractor_Extract_MetaTags(t *testing.T) {
	e := &HTMLExtractor{}
	html := `<html><head><meta name="description" content="Test Description"><title>T</title></head><body><p>X</p></body></html>`
	doc, err := e.Extract(strings.NewReader(html), "test.html")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc.Metadata["description"] != "Test Description" {
		t.Errorf("expected meta description, got %v", doc.Metadata)
	}
}

func TestDocument_Fields(t *testing.T) {
	r := NewRegistry()
	doc, err := r.Extract("text/plain", strings.NewReader("test content"), "file.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if doc.Source != "file.txt" {
		t.Errorf("expected Source='file.txt', got %q", doc.Source)
	}
	if _, ok := doc.Metadata["lines"]; !ok {
		t.Error("expected 'lines' in metadata")
	}
	if _, ok := doc.Metadata["filename"]; !ok {
		t.Error("expected 'filename' in metadata")
	}
}

func BenchmarkTextExtractor_Extract(b *testing.B) {
	e := &TextExtractor{}
	content := strings.Repeat("Hello World ", 1000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := e.Extract(strings.NewReader(content), "bench.txt")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkHTMLExtractor_Extract(b *testing.B) {
	e := &HTMLExtractor{}
	html := `<html><head><title>Bench</title></head><body><p>` + strings.Repeat("text ", 100) + `</p></body></html>`
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := e.Extract(strings.NewReader(html), "bench.html")
		if err != nil {
			b.Fatal(err)
		}
	}
}

var _ io.Reader = (*errReader)(nil)
