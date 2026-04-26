package browser

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"fruit-jam/internal/fetch"
	"fruit-jam/internal/render"
)

// resolveInput turns user input into a navigable URL.
// If the input looks like a search query (has spaces, or has no dot and
// isn't localhost/an IP), it routes to DuckDuckGo Lite.
func resolveInput(input string) string {
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		return input
	}
	isQuery := strings.ContainsAny(input, " \t") ||
		(!strings.Contains(input, ".") && input != "localhost")
	if isQuery {
		return "https://lite.duckduckgo.com/lite/?q=" + url.QueryEscape(input)
	}
	return "https://" + input
}

// pageLoadedMsg is sent when an async fetch+convert completes.
type pageLoadedMsg struct {
	url         string
	page        render.Page
	rendered    string
	err         error
}

// renderedMsg is sent when Glamour re-renders existing Markdown (on resize).
type renderedMsg struct {
	rendered string
}

func postPage(rawURL string, values url.Values, width int) tea.Cmd {
	return func() tea.Msg {
		result, err := fetch.Post(rawURL, values)
		if err != nil {
			return pageLoadedMsg{url: rawURL, err: err}
		}
		page, err := render.HTMLToPage(result.Body, result.URL)
		if err != nil {
			return pageLoadedMsg{url: rawURL, err: err}
		}
		rendered, err := glamourRender(page.Markdown, width)
		if err != nil {
			return pageLoadedMsg{url: rawURL, err: err}
		}
		return pageLoadedMsg{url: result.URL, page: page, rendered: rendered}
	}
}

func loadPage(rawURL string, width int) tea.Cmd {
	return func() tea.Msg {
		result, err := fetch.Get(rawURL)
		if err != nil {
			return pageLoadedMsg{url: rawURL, err: err}
		}

		page, err := render.HTMLToPage(result.Body, result.URL)
		if err != nil {
			return pageLoadedMsg{url: rawURL, err: err}
		}

		rendered, err := glamourRender(page.Markdown, width)
		if err != nil {
			return pageLoadedMsg{url: rawURL, err: err}
		}

		return pageLoadedMsg{url: result.URL, page: page, rendered: rendered}
	}
}

func rerenderCmd(md string, width int) tea.Cmd {
	return func() tea.Msg {
		rendered, err := glamourRender(md, width)
		if err != nil {
			return nil
		}
		return renderedMsg{rendered: rendered}
	}
}

func glamourRender(md string, width int) (string, error) {
	if width <= 0 {
		width = 80
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width-2),
	)
	if err != nil {
		return "", err
	}
	return r.Render(md)
}

func (m model) Init() tea.Cmd {
	cmds := []tea.Cmd{textinput.Blink, m.spinner.Tick}
	if m.url != "" {
		cmds = append(cmds, loadPage(m.url, 80))
	}
	return tea.Batch(cmds...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		headerHeight := 1 // URL bar
		footerHeight := 1 // status / link prompt
		vpHeight := m.height - headerHeight - footerHeight

		if !m.ready {
			m.viewport = viewport.New(m.width, vpHeight)
			m.viewport.SetContent(m.rendered)
			m.ready = true
		} else {
			m.viewport.Width = m.width
			m.viewport.Height = vpHeight
		}

		if m.rawMarkdown != "" && !m.loading {
			return m, rerenderCmd(m.rawMarkdown, m.width)
		}
		return m, nil

	case renderedMsg:
		m.rendered = msg.rendered
		m.viewport.SetContent(m.rendered)
		return m, nil

	case pageLoadedMsg:
		m.loading = false
		m.searchHits = nil
		m.searchIdx = 0
		m.searchQuery = ""
		if msg.err != nil {
			m.err = msg.err
			m.status = fmt.Sprintf("Error: %v", msg.err)
			return m, nil
		}
		m.err = nil
		m.url = msg.url
		m.urlInput.SetValue(msg.url)
		m.links = msg.page.Links
		m.forms = msg.page.Forms
		m.rawMarkdown = msg.page.Markdown
		m.rendered = msg.rendered
		if msg.page.Title != "" {
			m.status = msg.page.Title
		} else {
			m.status = msg.url
		}
		m.viewport.SetContent(m.rendered)
		m.viewport.GotoTop()
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Delegate remaining messages (mouse scroll etc.) to the viewport.
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case modeURL:
		return m.handleURLKey(msg)
	case modeLink:
		return m.handleLinkKey(msg)
	case modeForm:
		return m.handleFormKey(msg)
	case modeSearch:
		return m.handleSearchKey(msg)
	default:
		return m.handleViewKey(msg)
	}
}

func (m model) handleViewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", keyQuit:
		return m, tea.Quit
	case keyURL, "ctrl+l":
		m.mode = modeURL
		m.urlInput.SetValue("")
		m.urlInput.Focus()
		return m, nil
	case keyURLEdit:
		m.mode = modeURL
		m.urlInput.SetValue(m.url)
		m.urlInput.CursorEnd()
		m.urlInput.Focus()
		return m, nil
	case "r":
		if m.url != "" && !m.loading {
			m.loading = true
			m.status = "Loading…"
			return m, loadPage(m.url, m.width)
		}
		return m, nil
	case keyFollow:
		if len(m.links) > 0 {
			m.mode = modeLink
			m.linkInput = ""
		}
		return m, nil
	case keyBack, "backspace":
		return m.goBack()
	case keyHelp:
		m.showHelp = !m.showHelp
		return m, nil
	case keyForm:
		return m.enterForm()
	case keySearch:
		m.mode = modeSearch
		m.searchInput.SetValue("")
		m.searchHits = nil
		m.searchIdx = 0
		return m, m.searchInput.Focus()
	case keyNextHit:
		return m.nextHit(+1)
	case keyPrevHit:
		return m.nextHit(-1)
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m model) handleURLKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		rawURL := strings.TrimSpace(m.urlInput.Value())
		if rawURL == "" {
			m.mode = modeView
			m.urlInput.Blur()
			return m, nil
		}
		rawURL = resolveInput(rawURL)
		if m.url != "" && m.url != rawURL {
			m.history = append(m.history, m.url)
		}
		m.url = rawURL
		m.loading = true
		m.status = "Loading…"
		m.mode = modeView
		m.urlInput.Blur()
		return m, loadPage(rawURL, m.width)

	case "esc":
		m.mode = modeView
		m.urlInput.SetValue(m.url)
		m.urlInput.Blur()
		return m, nil
	}

	var cmd tea.Cmd
	m.urlInput, cmd = m.urlInput.Update(msg)
	return m, cmd
}

func (m model) handleLinkKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.mode = modeView
		m.linkInput = ""
	case tea.KeyEnter:
		return m.followLinkInput()
	case tea.KeyBackspace:
		if len(m.linkInput) > 0 {
			m.linkInput = m.linkInput[:len(m.linkInput)-1]
		}
	case tea.KeyRunes:
		r := msg.Runes[0]
		if r >= '0' && r <= '9' {
			m.linkInput += string(r)
		}
	}
	return m, nil
}

func (m model) followLinkInput() (tea.Model, tea.Cmd) {
	m.mode = modeView
	n, err := strconv.Atoi(m.linkInput)
	m.linkInput = ""
	if err != nil || n < 1 || n > len(m.links) {
		return m, nil
	}
	href := m.links[n-1].URL
	if m.url != "" {
		m.history = append(m.history, m.url)
	}
	m.url = href
	m.urlInput.SetValue(href)
	m.loading = true
	m.status = "Loading…"
	return m, loadPage(href, m.width)
}

func (m model) enterForm() (tea.Model, tea.Cmd) {
	if len(m.forms) == 0 {
		m.status = "No forms on this page"
		return m, nil
	}
	m.mode = modeForm
	m.formIdx = 0
	m.formFocus = 0
	m.formInputs = buildFormInputs(m.forms[0])
	m.viewport.Height = m.contentHeight()
	if len(m.formInputs) == 0 {
		m.mode = modeView
		m.status = "Form has no fillable fields"
		return m, nil
	}
	return m, m.formInputs[0].Focus()
}

func buildFormInputs(f render.Form) []textinput.Model {
	var inputs []textinput.Model
	for _, field := range f.VisibleFields() {
		ti := textinput.New()
		ti.SetValue(field.Value)
		ph := field.Placeholder
		if ph == "" {
			ph = field.Name
		}
		ti.Placeholder = ph
		inputs = append(inputs, ti)
	}
	return inputs
}

func (m model) handleFormKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeView
		m.viewport.Height = m.contentHeight()
		return m, nil
	case "enter":
		return m.submitForm()
	case "tab", "down":
		m.formInputs[m.formFocus].Blur()
		m.formFocus = (m.formFocus + 1) % len(m.formInputs)
		return m, m.formInputs[m.formFocus].Focus()
	case "shift+tab", "up":
		m.formInputs[m.formFocus].Blur()
		m.formFocus = (m.formFocus - 1 + len(m.formInputs)) % len(m.formInputs)
		return m, m.formInputs[m.formFocus].Focus()
	}

	// Delegate keystrokes to the focused input.
	var cmd tea.Cmd
	m.formInputs[m.formFocus], cmd = m.formInputs[m.formFocus].Update(msg)
	return m, cmd
}

func (m model) submitForm() (tea.Model, tea.Cmd) {
	m.mode = modeView
	m.viewport.Height = m.contentHeight()

	form := m.forms[m.formIdx]
	values := url.Values{}

	skip := map[string]bool{
		"submit": true, "button": true, "reset": true,
		"image": true, "checkbox": true, "radio": true, "file": true,
	}
	visibleIdx := 0
	for _, field := range form.Fields {
		if field.Type == "hidden" {
			values.Set(field.Name, field.Value)
		} else if !skip[field.Type] {
			if visibleIdx < len(m.formInputs) {
				values.Set(field.Name, m.formInputs[visibleIdx].Value())
			}
			visibleIdx++
		}
	}

	target := form.Action
	if target == "" {
		target = m.url
	}
	if m.url != "" {
		m.history = append(m.history, m.url)
	}
	m.loading = true
	m.status = "Loading…"

	if form.Method == "post" {
		return m, postPage(target, values, m.width)
	}

	// GET: append query string to action URL.
	u, err := url.Parse(target)
	if err != nil {
		m.status = "Form error: " + err.Error()
		return m, nil
	}
	u.RawQuery = values.Encode()
	navURL := u.String()
	m.url = navURL
	m.urlInput.SetValue(navURL)
	return m, loadPage(navURL, m.width)
}

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[mGKHF]`)

func runSearch(rendered, query string) []int {
	if query == "" {
		return nil
	}
	lower := strings.ToLower(query)
	var hits []int
	for i, line := range strings.Split(rendered, "\n") {
		stripped := ansiRe.ReplaceAllString(line, "")
		if strings.Contains(strings.ToLower(stripped), lower) {
			hits = append(hits, i)
		}
	}
	return hits
}

func (m model) nextHit(dir int) (model, tea.Cmd) {
	if len(m.searchHits) == 0 {
		return m, nil
	}
	m.searchIdx = (m.searchIdx + dir + len(m.searchHits)) % len(m.searchHits)
	line := m.searchHits[m.searchIdx]
	offset := line - 2
	if offset < 0 {
		offset = 0
	}
	m.viewport.SetYOffset(offset)
	return m, nil
}

func (m model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeView
		m.searchInput.Blur()
		m.searchHits = nil
		m.searchIdx = 0
		m.searchQuery = ""
		return m, nil
	case "enter":
		m.mode = modeView
		m.searchInput.Blur()
		m.searchQuery = m.searchInput.Value()
		return m, nil
	}
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	q := m.searchInput.Value()
	m.searchQuery = q
	m.searchHits = runSearch(m.rendered, q)
	m.searchIdx = 0
	if len(m.searchHits) > 0 {
		line := m.searchHits[0]
		offset := line - 2
		if offset < 0 {
			offset = 0
		}
		m.viewport.SetYOffset(offset)
	}
	return m, cmd
}

// contentHeight returns the viewport height accounting for form panel.
func (m model) contentHeight() int {
	h := m.height - 2 // URL bar + status bar
	if m.mode == modeForm && len(m.formInputs) > 0 {
		h -= len(m.formInputs) + 1 // one line per field + help line
	}
	if h < 1 {
		h = 1
	}
	return h
}

func (m model) goBack() (tea.Model, tea.Cmd) {
	if len(m.history) == 0 {
		return m, nil
	}
	prev := m.history[len(m.history)-1]
	m.history = m.history[:len(m.history)-1]
	m.url = prev
	m.urlInput.SetValue(prev)
	m.loading = true
	m.status = "Loading…"
	return m, loadPage(prev, m.width)
}
