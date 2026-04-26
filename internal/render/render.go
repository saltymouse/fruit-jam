package render

import (
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
)

// Link is a numbered hyperlink extracted from the page.
type Link struct {
	N    int
	URL  string
	Text string
}

// Page holds the converted Markdown, extracted links, and forms.
type Page struct {
	Markdown string
	Links    []Link
	Forms    []Form
	Title    string
}

// linkRe matches both image links (![ ]) and regular links ([ ]).
var linkRe = regexp.MustCompile(`!?\[([^\]]*)\]\(([^)]+)\)`)

// newConverter builds the html-to-markdown converter with layout-table awareness.
// Table rows (<tr>) are registered as block elements so layout-table pages
// (e.g. DDG Lite search results) produce one paragraph per row instead of
// collapsing everything into a single line of text.
func newConverter() *converter.Converter {
	conv := converter.NewConverter(
		converter.WithPlugins(
			base.NewBasePlugin(),
			commonmark.NewCommonmarkPlugin(),
		),
	)
	conv.Register.TagType("tr", converter.TagTypeBlock, 0)
	return conv
}

// HTMLToPage converts raw HTML into a Page, absolutising and numbering links.
func HTMLToPage(htmlStr, baseURL string) (Page, error) {
	base, _ := url.Parse(baseURL)

	md, err := newConverter().ConvertString(htmlStr)
	if err != nil {
		return Page{}, fmt.Errorf("html→markdown: %w", err)
	}

	title := extractTitle(htmlStr, baseURL)

	var links []Link
	n := 0

	numbered := linkRe.ReplaceAllStringFunc(md, func(match string) string {
		// Leave image links untouched.
		if strings.HasPrefix(match, "!") {
			return match
		}

		subs := linkRe.FindStringSubmatch(match)
		if len(subs) < 3 {
			return match
		}
		text, rawHref := subs[1], subs[2]

		if strings.HasPrefix(rawHref, "#") ||
			strings.HasPrefix(rawHref, "mailto:") ||
			strings.HasPrefix(rawHref, "javascript:") {
			return match
		}

		href, err := resolveURL(base, rawHref)
		if err != nil || href == "" {
			return match
		}

		n++
		links = append(links, Link{N: n, URL: href, Text: text})
		return fmt.Sprintf("[[%d] %s](%s)", n, text, href)
	})

	forms := parseForms(htmlStr, baseURL)
	return Page{Markdown: numbered, Links: links, Forms: forms, Title: title}, nil
}

func resolveURL(base *url.URL, rawHref string) (string, error) {
	ref, err := url.Parse(rawHref)
	if err != nil {
		return "", err
	}
	if base == nil {
		return ref.String(), nil
	}
	return base.ResolveReference(ref).String(), nil
}

func extractTitle(htmlStr, fallback string) string {
	lower := strings.ToLower(htmlStr)
	start := strings.Index(lower, "<title>")
	if start == -1 {
		return fallback
	}
	start += len("<title>")
	end := strings.Index(lower[start:], "</title>")
	if end == -1 {
		return fallback
	}
	return html.UnescapeString(strings.TrimSpace(htmlStr[start : start+end]))
}
