package main

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
)

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.QuitMsg:
		fmt.Println("Existing")
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			switch m.view {
			case resourceEntryView:
				m.resource = m.textInput.Value()
				m.textInput.Reset()
				m.textInput.Placeholder = metav1.NamespaceDefault
				m.view = namespaceEntryView
			case namespaceEntryView:
				if m.textInput.Value() == "" {
					m.namespace = metav1.NamespaceDefault
				} else {
					m.namespace = m.textInput.Value()
				}
				m.view = resourceDisplayView
			case resourceDisplayView:
				return m, m.printOut()
			}

		}

	}
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m model) printOut() tea.Cmd {
	return tea.Sequence(
		tea.ExitAltScreen,
		tea.Printf("Resource: %s, NS: %s", m.resource, m.namespace),
		tea.Quit,
	)
}

func (m model) View() string {
	switch m.view {
	case resourceEntryView:
		return fmt.Sprintf("Your resource name? this could be a deployment or statefulset\n%s", m.textInput.View())
	case namespaceEntryView:
		return fmt.Sprintf("The given namespace?\n%s", m.textInput.View())
	case resourceDisplayView:
		return fmt.Sprintf("Resource: %s, Namespace: %s", m.resource, m.namespace)
	default:
		return fmt.Sprintf("Thinking...\n")
	}

}

func main() {
	if _, err := tea.NewProgram(initialModel(), tea.WithAltScreen()).Run(); err != nil {
		fmt.Errorf("encountered an error when attempting to run now-what %v", err)
		os.Exit(1)
	}
}
