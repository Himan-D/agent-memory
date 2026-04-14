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
	"time"

	"agent-memory/internal/memory/datapoint"
)

type Loader interface {
	Load(ctx context.Context, source interface{}) ([]*datapoint.DataPoint, error)
	Name() string
	Supports(sourceType string) bool
}

type LoaderRegistry struct {
	loaders map[string]Loader
}

func NewLoaderRegistry() *LoaderRegistry {
	return &LoaderRegistry{
		loaders: make(map[string]Loader),
	}
}

func (r *LoaderRegistry) Register(loader Loader) {
	r.loaders[loader.Name()] = loader
}

func (r *LoaderRegistry) Get(name string) (Loader, bool) {
	loader, ok := r.loaders[name]
	return loader, ok
}

func (r *LoaderRegistry) Load(ctx context.Context, source string) ([]*datapoint.DataPoint, error) {
	sourceType := detectSourceType(source)
	return r.LoadByType(ctx, source, sourceType)
}

func (r *LoaderRegistry) LoadByType(ctx context.Context, source string, sourceType string) ([]*datapoint.DataPoint, error) {
	for _, loader := range r.loaders {
		if loader.Supports(sourceType) {
			return loader.Load(ctx, source)
		}
	}
	return nil, fmt.Errorf("no loader found for source type: %s", sourceType)
}

func detectSourceType(source string) string {
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		return "url"
	}

	if strings.HasPrefix(source, "{") || strings.HasPrefix(source, "[") {
		return "json"
	}

	ext := filepath.Ext(source)
	switch strings.ToLower(ext) {
	case ".txt", ".md":
		return "text"
	case ".pdf":
		return "pdf"
	case ".doc", ".docx":
		return "docx"
	case ".csv":
		return "csv"
	case ".json":
		return "json"
	case ".xml":
		return "xml"
	case ".html", ".htm":
		return "html"
	case ".png", ".jpg", ".jpeg", ".gif", ".bmp":
		return "image"
	case ".mp3", ".wav", ".m4a", ".flac":
		return "audio"
	default:
		return "text"
	}
}

type TextLoader struct{}

func NewTextLoader() *TextLoader {
	return &TextLoader{}
}

func (l *TextLoader) Name() string {
	return "text"
}

func (l *TextLoader) Supports(sourceType string) bool {
	return sourceType == "text"
}

func (l *TextLoader) Load(ctx context.Context, source interface{}) ([]*datapoint.DataPoint, error) {
	var content string

	switch s := source.(type) {
	case string:
		if _, err := os.Stat(s); err == nil {
			data, err := os.ReadFile(s)
			if err != nil {
				return nil, fmt.Errorf("failed to read file: %w", err)
			}
			content = string(data)
		} else {
			content = s
		}
	case []byte:
		content = string(s)
	default:
		return nil, fmt.Errorf("unsupported source type: %T", source)
	}

	lines := strings.Split(content, "\n")
	points := make([]*datapoint.DataPoint, 0, len(lines))

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		dp := datapoint.New(line, datapoint.DataPointTypeChunk)
		dp.SetSource("text", "unknown", "")
		dp.Metadata["line_number"] = i + 1
		dp.Metadata["source_type"] = "text"
		points = append(points, dp)
	}

	return points, nil
}

type JSONLoader struct{}

func NewJSONLoader() *JSONLoader {
	return &JSONLoader{}
}

func (l *JSONLoader) Name() string {
	return "json"
}

func (l *JSONLoader) Supports(sourceType string) bool {
	return sourceType == "json"
}

func (l *JSONLoader) Load(ctx context.Context, source interface{}) ([]*datapoint.DataPoint, error) {
	var data []byte

	switch s := source.(type) {
	case string:
		if _, err := os.Stat(s); err == nil {
			var err error
			data, err = os.ReadFile(s)
			if err != nil {
				return nil, fmt.Errorf("failed to read file: %w", err)
			}
		} else {
			data = []byte(s)
		}
	case []byte:
		data = s
	case io.Reader:
		data, _ = io.ReadAll(s)
	default:
		return nil, fmt.Errorf("unsupported source type: %T", source)
	}

	var jsonData interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	return l.parseJSONValue(jsonData, "root")
}

func (l *JSONLoader) parseJSONValue(value interface{}, path string) ([]*datapoint.DataPoint, error) {
	points := make([]*datapoint.DataPoint, 0)

	switch v := value.(type) {
	case string:
		dp := datapoint.New(v, datapoint.DataPointTypeChunk)
		dp.SetSource("json", path, "")
		dp.Metadata["json_path"] = path
		points = append(points, dp)

	case map[string]interface{}:
		for key, val := range v {
			childPath := fmt.Sprintf("%s.%s", path, key)
			childPoints, err := l.parseJSONValue(val, childPath)
			if err != nil {
				continue
			}
			points = append(points, childPoints...)
		}

	case []interface{}:
		for i, item := range v {
			itemPath := fmt.Sprintf("%s[%d]", path, i)
			itemPoints, err := l.parseJSONValue(item, itemPath)
			if err != nil {
				continue
			}
			points = append(points, itemPoints...)
		}

	case float64, int, bool:
		dp := datapoint.New(fmt.Sprintf("%v", v), datapoint.DataPointTypeChunk)
		dp.SetSource("json", path, "")
		dp.Metadata["json_path"] = path
		points = append(points, dp)
	}

	return points, nil
}

type CSVLoader struct {
	delimiter string
}

func NewCSVLoader(delimiter string) *CSVLoader {
	if delimiter == "" {
		delimiter = ","
	}
	return &CSVLoader{delimiter: delimiter}
}

func (l *CSVLoader) Name() string {
	return "csv"
}

func (l *CSVLoader) Supports(sourceType string) bool {
	return sourceType == "csv"
}

func (l *CSVLoader) Load(ctx context.Context, source interface{}) ([]*datapoint.DataPoint, error) {
	var reader io.Reader

	switch s := source.(type) {
	case string:
		if _, err := os.Stat(s); err == nil {
			file, err := os.Open(s)
			if err != nil {
				return nil, fmt.Errorf("failed to open file: %w", err)
			}
			defer file.Close()
			reader = file
		} else {
			reader = bytes.NewReader([]byte(s))
		}
	case []byte:
		reader = bytes.NewReader(s)
	default:
		return nil, fmt.Errorf("unsupported source type: %T", source)
	}

	var lines []string
	buf := make([]byte, 1024)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			lines = append(lines, strings.Split(string(buf[:n]), "\n")...)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}

	points := make([]*datapoint.DataPoint, 0)
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		dp := datapoint.New(line, datapoint.DataPointTypeChunk)
		dp.SetSource("csv", "unknown", "")
		dp.Metadata["row_number"] = i + 1
		dp.Metadata["source_type"] = "csv"
		points = append(points, dp)
	}

	return points, nil
}

type URLLoader struct {
	userAgent string
}

func NewURLLoader() *URLLoader {
	return &URLLoader{
		userAgent: "Mozilla/5.0 (compatible; AgentMemory/1.0)",
	}
}

func (l *URLLoader) Name() string {
	return "url"
}

func (l *URLLoader) Supports(sourceType string) bool {
	return sourceType == "url"
}

func (l *URLLoader) Load(ctx context.Context, source interface{}) ([]*datapoint.DataPoint, error) {
	url, ok := source.(string)
	if !ok {
		return nil, fmt.Errorf("URL loader requires string source")
	}

	resp, err := fetchURL(ctx, url, l.userAgent)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch URL: %w", err)
	}

	dp := datapoint.New(resp, datapoint.DataPointTypeDocument)
	dp.SetSource("url", url, "")
	dp.Metadata["source_type"] = "url"

	return []*datapoint.DataPoint{dp}, nil
}

func fetchURL(ctx context.Context, url, userAgent string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}

type MultiLoader struct {
	registry *LoaderRegistry
}

func NewMultiLoader(registry *LoaderRegistry) *MultiLoader {
	return &MultiLoader{registry: registry}
}

func (l *MultiLoader) Load(ctx context.Context, source interface{}) ([]*datapoint.DataPoint, error) {
	var sources []string

	switch s := source.(type) {
	case string:
		if strings.HasSuffix(s, "/*") || strings.HasSuffix(s, "/**") {
			dir := strings.TrimSuffix(s, "/*")
			if dir == "" {
				dir = "."
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				return nil, fmt.Errorf("failed to read directory: %w", err)
			}
			for _, entry := range entries {
				if !entry.IsDir() {
					sources = append(sources, filepath.Join(dir, entry.Name()))
				}
			}
		} else if _, err := os.Stat(s); err == nil {
			sources = []string{s}
		} else {
			sources = []string{s}
		}
	case []string:
		sources = s
	default:
		return nil, fmt.Errorf("unsupported source type: %T", source)
	}

	allPoints := make([]*datapoint.DataPoint, 0)

	for _, src := range sources {
		points, err := l.registry.Load(ctx, src)
		if err != nil {
			continue
		}
		allPoints = append(allPoints, points...)
	}

	return allPoints, nil
}

func NewDefaultRegistry() *LoaderRegistry {
	registry := NewLoaderRegistry()
	registry.Register(NewTextLoader())
	registry.Register(NewJSONLoader())
	registry.Register(NewCSVLoader(","))
	registry.Register(NewURLLoader())
	return registry
}
