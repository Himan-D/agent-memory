package connectors

import (
	"bufio"
	"context"
	"fmt"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

type WebCrawler struct {
	httpClient *http.Client
	userAgent string
	timeout   time.Duration
}

type CrawledPage struct {
	URL       string
	Title     string
	Content   string
	Links     []string
	Images    []string
	Metadata map[string]string
	Status   int
	Error    error
}

type CrawlConfig struct {
	MaxDepth    int
	MaxPages    int
	Concurrent int
	Timeout    time.Duration
	FollowExternal bool
}

type CrawlResult struct {
	Pages   []CrawledPage
	Errors  []string
	Summary struct {
		TotalPages int `json:"totalPages"`
		Failed     int `json:"failed"`
		Duration   int `json:"durationMs"`
	} `json:"summary"`
}

func NewWebCrawler() *WebCrawler {
	return &WebCrawler{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) > 10 {
					return fmt.Errorf("too many Redirects")
				}
				return nil
			},
		},
		userAgent: "Hystersis/1.0 (Web Crawler; +https://github.com/Himan-D/agent-memory)",
		timeout:   30 * time.Second,
	}
}

func (w *WebCrawler) CrawlURL(ctx context.Context, url string) (*CrawledPage, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", w.userAgent)

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	page := &CrawledPage{
		URL:     url,
		Status:  resp.StatusCode,
	}

	reader := bufio.NewReader(resp.Body)
	tokenizer := html.NewTokenizer(reader)

	var title string
	var content strings.Builder
	var inBody bool
	var inTitle bool

	for {
		tt := tokenizer.Next()
		switch tt {
		case html.ErrorToken:
			err := tokenizer.Err()
			if err == io.EOF {
				break
			}
			log.Printf("HTML parse error: %v", err)
			break
		case html.StartTagToken:
			token := tokenizer.Token()
			switch token.DataAtom {
			case atom.Body:
				inBody = true
			case atom.Title:
				inTitle = true
			case atom.A:
				for _, attr := range token.Attr {
					if attr.Key == "href" {
						page.Links = append(page.Links, attr.Val)
					}
				}
			case atom.Img:
				for _, attr := range token.Attr {
					if attr.Key == "src" {
						page.Images = append(page.Images, attr.Val)
					}
				}
			case atom.Meta:
				var name, content string
				for _, attr := range token.Attr {
					if attr.Key == "name" {
						name = attr.Val
					}
					if attr.Key == "content" {
						content = attr.Val
					}
					if attr.Key == "property" && attr.Val == "og:title" {
						name = "og:title"
					}
				}
				if name != "" && content != "" {
					if page.Metadata == nil {
						page.Metadata = make(map[string]string)
					}
					page.Metadata[name] = content
				}
			}
		case html.EndTagToken:
			token := tokenizer.Token()
			switch token.DataAtom {
			case atom.Body:
				inBody = false
			case atom.Title:
				inTitle = false
				title = content.String()
				content.Reset()
			}
		case html.TextToken:
			if inBody || inTitle {
				text := tokenizer.Token().Data
				text = strings.TrimSpace(text)
				if text != "" {
					if inTitle {
						content.WriteString(text)
					} else {
						content.WriteString(text)
						content.WriteString(" ")
					}
				}
			}
		}
	}

	page.Title = title
	if ogTitle, ok := page.Metadata["og:title"]; ok {
		page.Title = ogTitle
	}

	page.Content = normalizeText(content.String())

	return page, nil
}

func (w *WebCrawler) CrawlWithConfig(ctx context.Context, url string, config *CrawlConfig) (*CrawlResult, error) {
	if config == nil {
		config = &CrawlConfig{
			MaxDepth:  2,
			MaxPages: 10,
			Timeout:  30 * time.Second,
		}
	}

	start := time.Now()
	result := &CrawlResult{}

	visited := make(map[string]bool)
	var crawlQueue []string
	crawlQueue = append(crawlQueue, url)

	for len(crawlQueue) > 0 && len(result.Pages) < config.MaxPages {
		currentURL := crawlQueue[0]
		crawlQueue = crawlQueue[1:]

		if visited[currentURL] {
			continue
		}
		visited[currentURL] = true

		page, err := w.CrawlURL(ctx, currentURL)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", currentURL, err))
			continue
		}

		if page.Status != http.StatusOK {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: status %d", currentURL, page.Status))
			continue
		}

		result.Pages = append(result.Pages, *page)

		for _, link := range page.Links {
			if config.MaxDepth > 1 && !visited[link] && isInternal(link, url) {
				crawlQueue = append(crawlQueue, link)
			}
		}
	}

	result.Summary.TotalPages = len(result.Pages)
	result.Summary.Failed = len(result.Errors)
	result.Summary.Duration = int(time.Since(start).Milliseconds())

	return result, nil
}

func isInternal(link, baseURL string) bool {
	if !strings.HasPrefix(link, "http") {
		return false
	}
	
	baseDomain := extractDomain(baseURL)
	linkDomain := extractDomain(link)
	
	return baseDomain == linkDomain
}

func extractDomain(url string) string {
	parts := strings.SplitN(strings.TrimPrefix(url, "https://"), "/", 2)
	if len(parts) > 0 {
		return strings.Split(parts[0], ":")[0]
	}
	return ""
}

func normalizeText(text string) string {
	text = strings.Join(strings.Fields(text), " ")
	text = strings.ReplaceAll(text, "  ", " ")

	if !utf8.ValidString(text) {
		b := make([]byte, 0, len(text))
		for i := 0; i < len(text); {
			r, n := utf8.DecodeRuneInString(text[i:])
			if r != utf8.RuneError {
				b = append(b, text[i:i+n]...)
			}
			i += n
		}
		text = string(b)
	}

	return text
}

func (w *WebCrawler) ConvertToMemory(page *CrawledPage) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("URL: %s\n", page.URL))
	b.WriteString(fmt.Sprintf("Title: %s\n", page.Title))

	if page.Content != "" {
		b.WriteString("\nContent:\n")
		b.WriteString(truncate(page.Content, 5000))
	}

	return b.String()
}

type WebCrawlerConfig struct {
	StartURL      string `json:"startUrl"`
	MaxDepth     int    `json:"maxDepth,omitempty"`
	MaxPages     int    `json:"maxPages,omitempty"`
	Concurrent   int    `json:"concurrent,omitempty"`
	ExcludePaths []string `json:"excludePaths,omitempty"`
}

func (c *WebCrawlerConfig) Validate() error {
	if c.StartURL == "" {
		return fmt.Errorf("startUrl required")
	}
	if !strings.HasPrefix(c.StartURL, "http") {
		return fmt.Errorf("startUrl must be a valid URL")
	}
	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	return s[:maxLen] + "... [truncated]"
}

type SitemapEntry struct {
	Loc        string     `xml:"loc"`
	LastMod    string     `xml:"lastmod"`
	ChangeFreq string    `xml:"changefreq"`
	Priority  float64    `xml:"priority"`
}

func (w *WebCrawler) ParseSitemap(sitemapURL string) ([]string, error) {
	req, _ := http.NewRequest("GET", sitemapURL, nil)
	req.Header.Set("User-Agent", w.userAgent)

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sitemap fetch failed: %d", resp.StatusCode)
	}

	var urls []string
	tokenizer := html.NewTokenizer(resp.Body)

	for {
		tt := tokenizer.Next()
		if tt == html.ErrorToken {
			break
		}
		if tt == html.StartTagToken {
			tok := tokenizer.Token()
			if tok.DataAtom == atom.Link {
				for _, attr := range tok.Attr {
					if attr.Key == "" {
						urls = append(urls, attr.Val)
						break
					}
				}
			}
		}
	}

	return urls, nil
}