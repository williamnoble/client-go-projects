package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"io"
	"strings"
)

var docStyle = lipgloss.NewStyle().Margin(1, 2)

const listHeight = 14

var (
	titleStyle            = lipgloss.NewStyle().MarginLeft(2)
	itemStyle             = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle     = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	defaultNamespaceStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("#FAAE72"))
	favouriteItemStyle    = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color(lipgloss.Color("#ff593f")))
	paginationStyle       = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle             = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
	quitTextStyle         = lipgloss.NewStyle().Margin(1, 0, 2, 4)
)

type itemDelegate struct{}

func (i item) FilterValue() string { return string(i) }

func (i item) String() string { return string(i) }

func (d itemDelegate) Height() int                               { return 1 }
func (d itemDelegate) Spacing() int                              { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	//str := fmt.Sprintf("%d  %s", index+1, i)
	str := fmt.Sprintf("  %s", i)

	fn := itemStyle.Render

	if i == "default" {
		fn = func(s ...string) string {
			return defaultNamespaceStyle.Render("  " + strings.Join(s, " "))
		}
	}

	if isFavourite(i.String()) {
		fn = func(s ...string) string {
			return defaultNamespaceStyle.Render(" ★" + strings.Join(s, " "))
		}
	}
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	if index == m.Index() && isFavourite(i.String()) {
		fn = func(s ...string) string {
			return selectedItemStyle.Render(">★" + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

type item string

type Model struct {
	list           list.Model
	showFavourites bool
	msg            string
	quitting       bool
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.showFavourites = !m.showFavourites
			if m.showFavourites {
				m.list.Title = "[FAVOURITE] Namespaces"
				m.list.SetItems(getFiltered())
				return m, nil

			} else {
				m.list.Title = "[ALL] Namespaces"
				m.list.SetItems(getNamespaces())
				return m, nil
			}
		case "f":
			item, ok := m.list.SelectedItem().(item)
			if ok {
				setFavourites(item.String())
			}

			if m.list.Title == "[ALL] Namespaces" {
				m.list.SetItems(getNamespaces())
			} else {
				m.list.SetItems(getFiltered())
			}

			return m, nil
		case "enter":
			item, ok := m.list.SelectedItem().(item)
			if ok {
				switchContext(item.String())
				return m, tea.Quit
			}
		}

	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	return docStyle.Render(m.list.View())

}

func getFiltered() []list.Item {
	unfilteredNamespaces := getNamespaces()
	favouriteNamespaces := getFavourites()
	var filteredItems []list.Item
	for _, ns := range unfilteredNamespaces {
		for _, f := range favouriteNamespaces {
			if f == ns.(item).String() {
				filteredItems = append(filteredItems, ns)
			}
		}
	}
	return filteredItems
}
