package browser

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	urlBarStyle      = lipgloss.NewStyle().Background(lipgloss.Color("62")).Foreground(lipgloss.Color("255")).Padding(0, 1)
	urlBarEditStyle  = lipgloss.NewStyle().Background(lipgloss.Color("57")).Foreground(lipgloss.Color("255")).Padding(0, 1)
	statusStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	errStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	linkStyle        = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3"))
	helpStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	formLabelStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Width(12)
	formFocusedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Bold(true).Width(12)
	formBorderStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("62"))
)

const helpText = "  g url  G edit url  f follow  / search  n/N next/prev  i form  h/bksp back  j/k scroll  ? help  q quit"


func (m model) View() string {
	if !m.ready {
		return "\n  Initializing…"
	}

	// URL bar
	var topBar string
	if m.mode == modeURL {
		topBar = urlBarEditStyle.Width(m.width).Render("▸ " + m.urlInput.View())
	} else {
		label := m.url
		if m.loading {
			label = m.spinner.View() + " " + m.url
		}
		if label == "" {
			label = "fruit-jam  —  press g to open a URL"
		}
		topBar = urlBarStyle.Width(m.width).Render(label)
	}

	// Viewport
	content := m.viewport.View()

	// Bottom bar
	var bottom string
	switch {
	case m.mode == modeSearch:
		bottom = linkStyle.Render("/ " + m.searchInput.View() + "  esc to cancel")
	case m.showHelp:
		bottom = helpStyle.Render(helpText)
	case m.mode == modeLink:
		prompt := fmt.Sprintf("Follow link [1-%d]: %s_  esc to cancel", len(m.links), m.linkInput)
		bottom = linkStyle.Render(prompt)
	case m.err != nil:
		bottom = errStyle.Render(truncate(fmt.Sprintf("Error: %v", m.err), m.width))
	default:
		pct := fmt.Sprintf("%.0f%%", m.viewport.ScrollPercent()*100)
		meta := pct
		if len(m.links) > 0 || len(m.forms) > 0 {
			meta = fmt.Sprintf("%d links", len(m.links))
			if len(m.forms) > 0 {
				meta += fmt.Sprintf("  %d form", len(m.forms))
				if len(m.forms) > 1 {
					meta += "s"
				}
			}
			meta += "  " + pct
		}
		if len(m.searchHits) > 0 {
			meta = fmt.Sprintf("%d/%d  ", m.searchIdx+1, len(m.searchHits)) + meta
		}
		bottom = statusStyle.Render(truncate(m.status, m.width-len(meta)-2) + "  " + meta)
	}

	parts := []string{topBar, content}
	if m.mode == modeForm {
		parts = append(parts, m.formPanel())
	}
	parts = append(parts, bottom)
	return strings.Join(parts, "\n")
}

func (m model) formPanel() string {
	if m.formIdx >= len(m.forms) || len(m.formInputs) == 0 {
		return ""
	}
	form := m.forms[m.formIdx]
	visible := form.VisibleFields()

	var sb strings.Builder
	sb.WriteString(formBorderStyle.Render("  ─ Form ") + formBorderStyle.Render(strings.Repeat("─", max(0, m.width-10))))
	for i, input := range m.formInputs {
		name := ""
		if i < len(visible) {
			name = visible[i].Name
		}
		var label string
		if i == m.formFocus {
			label = formFocusedStyle.Render("▶ " + name)
		} else {
			label = formLabelStyle.Render("  " + name)
		}
		sb.WriteString("\n" + label + "  " + input.View())
	}
	sb.WriteString("\n" + helpStyle.Render("  Tab next · Enter submit · Esc cancel"))
	return sb.String()
}

func truncate(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
