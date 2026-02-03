package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

type modeChoice string

const (
	modeCreate   modeChoice = "create"
	modeApply    modeChoice = "apply"
	modeProfiles modeChoice = "profiles"
)

type ModeModel struct {
	form   *huh.Form
	choice modeChoice
	styles *Styles
}

func NewModeModel(styles *Styles) ModeModel {
	m := ModeModel{styles: styles}
	f := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[modeChoice]().
				Key("mode").
				Title("gh repo-defaults").
				Description("What would you like to do?").
				Options(
					huh.NewOption("Create new repo", modeCreate),
					huh.NewOption("Apply to existing repo", modeApply),
					huh.NewOption("Manage profiles", modeProfiles),
				).
				Value(&m.choice),
		),
	).WithShowHelp(false)
	m.form = f
	return m
}

func (m ModeModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m ModeModel) Update(msg tea.Msg) (ModeModel, tea.Cmd) {
	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}
	return m, cmd
}

func (m ModeModel) View() string {
	return m.form.View()
}

func (m ModeModel) Done() bool {
	return m.form.State == huh.StateCompleted
}

func (m ModeModel) Choice() modeChoice {
	return m.choice
}
