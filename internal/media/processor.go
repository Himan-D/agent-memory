package media

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"time"

	"agent-memory/internal/llm"
)

type Processor struct {
	httpClient *http.Client
	llmClient llm.Provider
	config    *Config
}

type Config struct {
	VisionProvider string `env:"VISION_PROVIDER" envDefault:"openai"`
	VisionModel   string `env:"VISION_MODEL" envDefault:"gpt-4o"`
	AudioModel   string `env:"AUDIO_MODEL" envDefault:"whisper-1"`
	MaxFileSize  int64  `env:"MEDIA_MAX_FILE_SIZE" envDefault:"52428800"`
}

type ExtractionResult struct {
	Text     string                  `json:"text"`
	Type     string                  `json:"type"`
	Metadata map[string]interface{} `json:"metadata"`
	Errors   []string               `json:"errors,omitempty"`
}

type MediaType string

const (
	MediaTypePDF   MediaType = "pdf"
	MediaTypeImage MediaType = "image"
	MediaTypeAudio MediaType = "audio"
	MediaTypeVideo MediaType = "video"
	MediaTypeDoc  MediaType = "document"
)

func NewProcessor(cfg *Config, llmClient llm.Provider) *Processor {
	return &Processor{
		httpClient: &http.Client{Timeout: 120 * time.Second},
		llmClient: llmClient,
		config:    cfg,
	}
}

func (p *Processor) Process(ctx context.Context, filePath string) (*ExtractionResult, error) {
	mediaType, err := p.detectMediaType(filePath)
	if err != nil {
		return nil, err
	}

	switch mediaType {
	case MediaTypePDF:
		return p.ExtractPDF(ctx, filePath)
	case MediaTypeImage:
		return p.ExtractImage(ctx, filePath)
	case MediaTypeAudio:
		return p.ExtractAudio(ctx, filePath)
	case MediaTypeVideo:
		return p.ExtractVideo(ctx, filePath)
	default:
		return nil, fmt.Errorf("unsupported media type: %s", mediaType)
	}
}

func (p *Processor) detectMediaType(filePath string) (MediaType, error) {
	ext := getExt(filePath)

	switch ext {
	case ".pdf":
		return MediaTypePDF, nil
	case ".png", ".jpg", ".jpeg", ".gif", ".bmp", ".webp", ".tiff":
		return MediaTypeImage, nil
	case ".mp3", ".wav", ".m4a", ".flac", ".ogg":
		return MediaTypeAudio, nil
	case ".mp4", ".avi", ".mov", ".mkv", ".webm":
		return MediaTypeVideo, nil
	case ".doc", ".docx", ".txt", ".rtf":
		return MediaTypeDoc, nil
	default:
		return "", fmt.Errorf("unknown file extension: %s", ext)
	}
}

func getExt(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '.' {
			return path[i:]
		}
		if path[i] == '/' || path[i] == '\\' {
			break
		}
	}
	return ""
}

func (p *Processor) ExtractPDF(ctx context.Context, filePath string) (*ExtractionResult, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	maxSize := p.config.MaxFileSize
	if int64(len(data)) > maxSize {
		return &ExtractionResult{
			Text:     "",
			Type:     string(MediaTypePDF),
			Metadata: map[string]interface{}{"size": len(data), "truncated": true},
			Errors:   []string{"file size exceeds max"},
		}, nil
	}

	text := string(data)
	if len(text) > 10000 {
		text = text[:10000] + "\n...[truncated]"
	}

	return &ExtractionResult{
		Text:     text,
		Type:     string(MediaTypePDF),
		Metadata: map[string]interface{}{"size": len(data)},
	}, nil
}

func (p *Processor) ExtractImage(ctx context.Context, filePath string) (*ExtractionResult, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	if p.llmClient == nil {
		return &ExtractionResult{
			Text:     "",
			Type:     string(MediaTypeImage),
			Metadata: map[string]interface{}{"size": len(data), "skipped": true},
			Errors:   []string{"no LLM client configured"},
		}, nil
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	prompt := "Describe this image in detail. Extract all text, labels, signs, objects, or any readable content."

	resp, err := p.llmClient.Complete(ctx, &llm.CompletionRequest{
		Model: p.config.VisionModel,
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		Temperature: 0.3,
		MaxTokens:   2000,
	})
	if err != nil {
		return &ExtractionResult{
			Text:     "",
			Type:     string(MediaTypeImage),
			Metadata: map[string]interface{}{"size": len(data)},
			Errors:   []string{err.Error()},
		}, nil
	}

	_ = encoded
	return &ExtractionResult{
		Text:     resp.Content,
		Type:     string(MediaTypeImage),
		Metadata: map[string]interface{}{"size": len(data), "model": p.config.VisionModel},
	}, nil
}

func (p *Processor) ExtractAudio(ctx context.Context, filePath string) (*ExtractionResult, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	if p.llmClient == nil {
		return &ExtractionResult{
			Text:     "",
			Type:     string(MediaTypeAudio),
			Metadata: map[string]interface{}{"size": len(data), "skipped": true},
			Errors:   []string{"no LLM client configured"},
		}, nil
	}

	return &ExtractionResult{
		Text:     "[Audio transcription requires OpenAI API endpoint]",
		Type:     string(MediaTypeAudio),
		Metadata: map[string]interface{}{"size": len(data), "model": p.config.AudioModel},
	}, nil
}

func (p *Processor) ExtractVideo(ctx context.Context, filePath string) (*ExtractionResult, error) {
	return &ExtractionResult{
		Text:     "[Video extraction requires ffmpeg + frame processing]",
		Type:     string(MediaTypeVideo),
		Metadata: map[string]interface{}{"file": filePath},
	}, nil
}

func init() {
	_ = http.DefaultTransport
}