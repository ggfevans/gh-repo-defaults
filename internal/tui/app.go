package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gvns/gh-repo-defaults/internal/config"
)

type screen int

const (
	screenMode screen = iota
	screenCreate
	screenProgress
)

type App struct {
	screen   screen
	styles   *Styles
	lg       *lipgloss.Renderer
	config   *config.Config
	width    int
	mode     ModeModel
	create   CreateModel
	progress ProgressModel
}

func NewApp(cfg *config.Config) App {
	lg := lipgloss.DefaultRenderer()
	styles := NewStyles(lg)
	a := App{
		screen: screenMode,
		styles: styles,
		lg:     lg,
		config: cfg,
		width:  80,
	}
	a.mode = NewModeModel(styles)
	return a
}

func (a App) Init() tea.Cmd {
	return a.mode.Init()
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return a, tea.Interrupt
		}
	}

	switch a.screen {
	case screenMode:
		return a.updateMode(msg)
	case screenCreate:
		return a.updateCreate(msg)
	case screenProgress:
		return a.updateProgress(msg)
	}
	return a, nil
}

func (a App) View() string {
	switch a.screen {
	case screenMode:
		return a.mode.View()
	case screenCreate:
		return a.create.View()
	case screenProgress:
		return a.progress.View()
	}
	return ""
}

func (a App) updateMode(msg tea.Msg) (tea.Model, tea.Cmd) {
	mode, cmd := a.mode.Update(msg)
	a.mode = mode

	if a.mode.Done() {
		choice := a.mode.Choice()
		switch choice {
		case modeCreate:
			a.screen = screenCreate
			a.create = NewCreateModel(a.styles, a.config)
			return a, a.create.Init()
		case modeProfiles:
			return a, tea.Quit
		}
	}
	return a, cmd
}

func (a App) updateCreate(msg tea.Msg) (tea.Model, tea.Cmd) {
	create, cmd := a.create.Update(msg)
	a.create = create

	if a.create.Done() {
		if !a.create.Confirmed() {
			a.screen = screenMode
			a.mode = NewModeModel(a.styles)
			return a, a.mode.Init()
		}
		a.screen = screenProgress
		a.progress = NewProgressModel(a.styles, a.create)
		return a, a.progress.Init()
	}
	return a, cmd
}

func (a App) updateProgress(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if a.progress.Done() && (msg.String() == "q" || msg.String() == "enter") {
			return a, tea.Quit
		}
	}
	progress, cmd := a.progress.Update(msg)
	a.progress = progress
	return a, cmd
}
