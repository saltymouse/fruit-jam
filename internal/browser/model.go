package browser

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"fruit-jam/internal/render"
)

type mode int

const (
	modeView   mode = iota // scrolling through content
	modeURL                // editing the URL bar
	modeLink               // typing a link number to follow
	modeForm               // filling out a form
	modeSearch             // typing a search query
)

type model struct {
	mode        mode
	url         string
	urlInput    textinput.Model
	viewport    viewport.Model
	spinner     spinner.Model
	status      string
	links       []render.Link
	forms       []render.Form
	formInputs  []textinput.Model // one per visible field in the active form
	formIdx     int               // which form is active
	formFocus   int               // which field within the form has focus
	history     []string
	width       int
	height      int
	rawMarkdown string // stored for re-render on resize
	rendered    string // ANSI output set as viewport content
	loading     bool
	ready       bool // true after first tea.WindowSizeMsg
	linkInput   string
	showHelp    bool
	err         error
	searchInput textinput.Model
	searchQuery string
	searchHits  []int // 0-indexed line numbers in ANSI-stripped rendered content
	searchIdx   int   // current position within searchHits
}

const welcomeMarkdown = `# fruit-jam

A terminal browser that renders pages as Markdown.

## Keybindings

| Key | Action |
|-----|--------|
| **g** / **ctrl+l** | Open URL bar (new URL) |
| **G** | Edit current URL |
| **f** | Follow a numbered link |
| **/** | Search this page |
| **n** / **N** | Next / previous match |
| **i** | Fill a form |
| **r** | Reload |
| **h** / **backspace** | Go back |
| **j** / **k** | Scroll |
| **?** | Toggle this help |
| **q** | Quit |
`

// New creates the initial browser model, optionally loading startURL.
func New(startURL string) model {
	if startURL != "" {
		startURL = resolveInput(startURL)
	}

	ti := textinput.New()
	ti.Placeholder = "https://"
	ti.CharLimit = 512
	if startURL != "" {
		ti.SetValue(startURL)
	}

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	si := textinput.New()
	si.Placeholder = "/search…"
	si.CharLimit = 256

	m := model{
		url:         startURL,
		urlInput:    ti,
		spinner:     sp,
		searchInput: si,
		loading:     startURL != "",
		status:      "Press g to open a URL  ·  ? for help",
	}
	if startURL == "" {
		m.rawMarkdown = welcomeMarkdown
	}
	return m
}
