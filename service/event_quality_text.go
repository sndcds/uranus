package service

import (
	"bytes"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/sndcds/uranus/model"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	rendererhtml "github.com/yuin/goldmark/renderer/html"
	"golang.org/x/net/html"
)

// Raw HTML is retained only for analysis; rendered markup is never returned or executed.
var eventQualityMarkdown = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithRendererOptions(rendererhtml.WithUnsafe()),
)

// analyzeEventText supports the dashboard's Markdown, HTML and plain text.
// Paragraphs count non-empty blocks separated by HTML blocks/br or blank lines.
func analyzeEventText(input string) model.EventTextMetrics {
	metrics := model.EventTextMetrics{}
	var rendered bytes.Buffer
	if err := eventQualityMarkdown.Convert([]byte(input), &rendered); err != nil {
		return metrics
	}
	root, err := html.Parse(&rendered)
	if err != nil {
		return metrics
	} // A bytes.Buffer cannot return an I/O error.
	var text strings.Builder
	emphasized, visible := 0, 0
	var walk func(*html.Node, bool)
	walk = func(n *html.Node, emphasis bool) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "script", "style", "template", "head":
				return
			}
			block := false
			switch n.Data {
			case "p", "div", "section", "article", "blockquote", "li", "ul", "ol", "pre", "table", "tr", "h1", "h2", "h3", "h4", "h5", "h6", "br", "hr":
				block = true
			}
			if block {
				text.WriteString("\n\n")
			}
			if len(n.Data) == 2 && n.Data[0] == 'h' && n.Data[1] >= '1' && n.Data[1] <= '6' {
				metrics.Headings++
			}
			if n.Data == "b" || n.Data == "strong" || n.Data == "i" || n.Data == "em" {
				metrics.EmphasisCount++
				emphasis = true
			}
			switch n.Data {
			case "html", "body", "p", "div", "span", "br", "b", "strong", "i", "em":
			default:
				metrics.OtherFormatting++
			}
			for _, attr := range n.Attr {
				if attr.Key == "style" && strings.TrimSpace(attr.Val) != "" {
					metrics.OtherFormatting++
					break
				}
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				walk(c, emphasis)
			}
			if block {
				text.WriteString("\n\n")
			}
			return
		}
		if n.Type == html.TextNode {
			text.WriteString(n.Data)
			for _, r := range n.Data {
				if !unicode.IsSpace(r) {
					visible++
					if emphasis {
						emphasized++
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, emphasis)
		}
	}
	walk(root, false)
	raw := strings.ReplaceAll(text.String(), "\r\n", "\n")
	// Treat whitespace-only lines as paragraph separators as well.
	var block strings.Builder
	flush := func() {
		if strings.TrimSpace(block.String()) != "" {
			metrics.Paragraphs++
		}
		block.Reset()
	}
	for _, line := range strings.Split(raw, "\n") {
		if strings.TrimSpace(line) == "" {
			flush()
		} else {
			block.WriteString(line)
			block.WriteByte(' ')
		}
	}
	flush()
	plain := strings.Join(strings.Fields(raw), " ")
	metrics.Characters = utf8.RuneCountInString(plain)
	metrics.Words = len(strings.Fields(plain))
	inSentence := false
	for _, r := range plain {
		switch r {
		case '.', '!', '?', '。', '！', '？':
			if inSentence {
				metrics.Sentences++
				inSentence = false
			}
		default:
			if unicode.IsLetter(r) || unicode.IsNumber(r) {
				inSentence = true
			}
		}
	}
	if inSentence {
		metrics.Sentences++
	}
	if visible > 0 {
		metrics.EmphasisRatio = math.Round(float64(emphasized)/float64(visible)*10000) / 10000
	}
	return metrics
}
