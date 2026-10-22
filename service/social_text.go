package service

import (
	"bytes"
	"golang.org/x/net/html"
	"strings"
)

// Reuse Uranus's Markdown/GFM parser. Unlike text metrics, social output keeps
// paragraph breaks and link destinations so conversion does not lose URLs.
func socialPlainText(input string) string {
	var rendered bytes.Buffer
	if err := eventQualityMarkdown.Convert([]byte(input), &rendered); err != nil {
		return ""
	}
	root, err := html.Parse(&rendered)
	if err != nil {
		return ""
	}
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "script", "style", "template", "head":
				return
			}
		}
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
		if n.Type == html.ElementNode {
			switch n.Data {
			case "a":
				for _, attr := range n.Attr {
					if attr.Key == "href" {
						if _, err := socialURL(attr.Val); err == nil && !strings.HasSuffix(b.String(), attr.Val) {
							b.WriteString(" (" + attr.Val + ")")
						}
					}
				}
			case "p", "div", "section", "article", "blockquote", "li", "ul", "ol", "pre", "table", "tr", "h1", "h2", "h3", "h4", "h5", "h6", "br", "hr":
				b.WriteByte('\n')
			}
		}
	}
	walk(root)
	lines := strings.Split(strings.TrimSpace(b.String()), "\n")
	for i := range lines {
		lines[i] = strings.Join(strings.Fields(lines[i]), " ")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
