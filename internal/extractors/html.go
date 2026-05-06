package extractors

import (
	"io"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

type HTMLExtractor struct{}

func (e *HTMLExtractor) SupportedMimeTypes() []string {
	return []string{"text/html"}
}

func (e *HTMLExtractor) Extract(r io.Reader, filename string) (*Document, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}

	var title string
	var body strings.Builder
	var metaTags map[string]string

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "title":
				if n.FirstChild != nil {
					title = n.FirstChild.Data
				}
			case "script", "style", "noscript":
				return
			case "meta":
				if metaTags == nil {
					metaTags = make(map[string]string)
				}
				name, content := "", ""
				for _, attr := range n.Attr {
					switch attr.Key {
					case "name", "property":
						name = attr.Val
					case "content":
						content = attr.Val
					}
				}
				if name != "" && content != "" {
					metaTags[name] = content
				}
			}
		}
		if n.Type == html.TextNode && n.Parent != nil {
			if n.Parent.Data != "script" && n.Parent.Data != "style" {
				text := strings.TrimSpace(n.Data)
				if text != "" {
					body.WriteString(text)
					body.WriteString(" ")
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	content := normalizeText(body.String())
	if t, ok := metaTags["og:title"]; ok && t != "" {
		title = t
	}

	return &Document{
		Content:   content,
		Title:     title,
		MimeType:  "text/html",
		Source:    filename,
		Metadata:  metaTags,
		PageCount: 1,
	}, nil
}

var multiSpaceRe = regexp.MustCompile(`\s+`)

func normalizeText(text string) string {
	text = strings.TrimSpace(text)
	text = multiSpaceRe.ReplaceAllString(text, " ")
	return text
}