package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"fruit-jam/internal/browser"
)

func main() {
	var startURL string
	if len(os.Args) > 1 {
		startURL = os.Args[1]
	}

	p := tea.NewProgram(
		browser.New(startURL),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
