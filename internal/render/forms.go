package render

import (
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// FormField is a single input within an HTML form.
type FormField struct {
	Name        string
	Type        string // "text", "search", "email", "password", "hidden", "textarea"
	Value       string
	Placeholder string
}

// Form represents a parsed HTML <form>.
type Form struct {
	Action string
	Method string // "get" or "post"
	Fields []FormField
}

// VisibleFields returns only the fields a user should interact with.
func (f Form) VisibleFields() []FormField {
	var out []FormField
	skip := map[string]bool{
		"hidden": true, "submit": true, "button": true,
		"reset": true, "image": true, "checkbox": true, "radio": true, "file": true,
	}
	for _, ff := range f.Fields {
		if !skip[ff.Type] {
			out = append(out, ff)
		}
	}
	return out
}

func parseForms(htmlStr, baseURL string) []Form {
	base, _ := url.Parse(baseURL)

	doc, err := html.Parse(strings.NewReader(htmlStr))
	if err != nil {
		return nil
	}

	var forms []Form
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "form" {
			f := extractForm(n, base)
			if len(f.VisibleFields()) > 0 {
				forms = append(forms, f)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return forms
}

func extractForm(n *html.Node, base *url.URL) Form {
	f := Form{Method: "get"}

	for _, a := range n.Attr {
		switch strings.ToLower(a.Key) {
		case "action":
			if base != nil {
				if ref, err := url.Parse(a.Val); err == nil {
					f.Action = base.ResolveReference(ref).String()
				}
			} else {
				f.Action = a.Val
			}
		case "method":
			f.Method = strings.ToLower(a.Val)
		}
	}

	var walkInputs func(*html.Node)
	walkInputs = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "input":
				ff := FormField{Type: "text"}
				for _, a := range n.Attr {
					switch strings.ToLower(a.Key) {
					case "name":
						ff.Name = a.Val
					case "type":
						ff.Type = strings.ToLower(a.Val)
					case "value":
						ff.Value = a.Val
					case "placeholder":
						ff.Placeholder = a.Val
					}
				}
				if ff.Name != "" {
					f.Fields = append(f.Fields, ff)
				}
			case "textarea":
				ff := FormField{Type: "textarea"}
				for _, a := range n.Attr {
					switch strings.ToLower(a.Key) {
					case "name":
						ff.Name = a.Val
					case "placeholder":
						ff.Placeholder = a.Val
					}
				}
				if ff.Name != "" {
					f.Fields = append(f.Fields, ff)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walkInputs(c)
		}
	}
	walkInputs(n)
	return f
}
