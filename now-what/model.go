package main

import (
	"github.com/charmbracelet/bubbles/textinput"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type view int

const (
	resourceEntryView = iota
	namespaceEntryView
	resourceDisplayView
)

type model struct {
	// resource you want to monitor
	resource string
	// namespace the resource lives in
	namespace string
	// current view for which to receive input
	view view
	// bubbletea components
	textInput textinput.Model
}

func initialModel() model {
	textInput := textinput.New()
	textInput.Placeholder = "nginx-dep"
	textInput.Focus()
	textInput.CharLimit = 120
	textInput.Width = 20

	return model{
		textInput: textInput,
		namespace: metav1.NamespaceDefault,
		view:      resourceEntryView,
	}
}
