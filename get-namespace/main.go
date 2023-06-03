package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"os"
)

func main() {
	l := list.New(getNamespaces(), itemDelegate{}, 0, 0)
	l.Title = "[ALL] Namespaces"
	m := Model{
		list:           l,
		showFavourites: false,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)

		os.Exit(1)
	}
}

func contains(s []string, key string) bool {
	for _, v := range s {
		if v == key {
			return true
		}
	}
	return false
}
