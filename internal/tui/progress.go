package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	ghclient "github.com/gvns/gh-repo-defaults/internal/github"
)

type allDoneMsg struct {
	url string
	err error
}

type ProgressModel struct {
	styles *Styles
	create CreateModel
	done   bool
	url    string
	err    error
}

func NewProgressModel(styles *Styles, create CreateModel) ProgressModel {
	return ProgressModel{
		styles: styles,
		create: create,
	}
}

func (m ProgressModel) Init() tea.Cmd {
	return m.startCreation()
}

func (m ProgressModel) startCreation() tea.Cmd {
	return func() tea.Msg {
		client := ghclient.NewClient()

		opts := ghclient.CreateOpts{
			Name:        m.create.Name,
			Description: m.create.Description,
			Public:      m.create.IsPublic(),
			Profile:     m.create.Profile(),
			OnProgress:  func(s ghclient.StepStatus) {},
		}

		url, err := client.CreateWithDefaults(opts)
		return allDoneMsg{url: url, err: err}
	}
}

func (m ProgressModel) Update(msg tea.Msg) (ProgressModel, tea.Cmd) {
	switch msg := msg.(type) {
	case allDoneMsg:
		m.done = true
		m.url = msg.url
		m.err = msg.err
		return m, nil
	}
	return m, nil
}

func (m ProgressModel) View() string {
	var b strings.Builder

	b.WriteString(m.styles.Header.Render("Creating repository..."))
	b.WriteString("\n\n")

	if !m.done {
		b.WriteString("  Working...\n")
	} else {
		if m.err != nil {
			b.WriteString(m.styles.Error.Render(fmt.Sprintf("  Error: %s\n", m.err)))
		}
		if m.url != "" {
			b.WriteString(m.styles.Success.Render(fmt.Sprintf("\n  Done! %s\n", m.url)))
		}
		b.WriteString("\n  Press q to exit.\n")
	}

	return b.String()
}

func (m ProgressModel) Done() bool {
	return m.done
}
