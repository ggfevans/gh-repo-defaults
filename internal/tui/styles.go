package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	indigo = lipgloss.AdaptiveColor{Light: "#5A56E0", Dark: "#7571F9"}
	green  = lipgloss.AdaptiveColor{Light: "#02BA84", Dark: "#02BF87"}
	red    = lipgloss.AdaptiveColor{Light: "#FE5F86", Dark: "#FE5F86"}
)

type Styles struct {
	Base,
	Header,
	Success,
	Error,
	Help,
	Highlight lipgloss.Style
}

func NewStyles(lg *lipgloss.Renderer) *Styles {
	s := Styles{}
	s.Base = lg.NewStyle().Padding(1, 2)
	s.Header = lg.NewStyle().Foreground(indigo).Bold(true).Padding(0, 1)
	s.Success = lg.NewStyle().Foreground(green)
	s.Error = lg.NewStyle().Foreground(red)
	s.Help = lg.NewStyle().Foreground(lipgloss.Color("240"))
	s.Highlight = lg.NewStyle().Foreground(lipgloss.Color("212"))
	return &s
}
