// Package tui implements the interactive Bubble Tea dashboard: it wires per
// widget timers, async runner commands, layout, and box rendering.
package tui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/locle97/quash-board/internal/config"
	"github.com/locle97/quash-board/internal/layout"
	"github.com/locle97/quash-board/internal/runner"
	"github.com/locle97/quash-board/internal/widget"
)

// widgetState is the mutable per-widget state held by the model.
type widgetState struct {
	lastGood string
	view     widget.View
	rect     layout.Rect
}

// Model is the Bubble Tea model for normal mode.
type Model struct {
	cfg          config.Config
	states       [][]*widgetState
	termW, termH int
	started      bool
	theme        widget.Theme
}

type widgetResultMsg struct {
	row, col int
	result   runner.Result
}

type tickMsg struct {
	row, col int
}

// New builds a Model with every widget in the pending state.
func New(cfg config.Config) Model {
	states := make([][]*widgetState, len(cfg.Rows))
	for r, row := range cfg.Rows {
		states[r] = make([]*widgetState, len(row.Widgets))
		for c, w := range row.Widgets {
			states[r][c] = &widgetState{
				view: widget.View{Title: w.Title, State: widget.StatePending},
			}
		}
	}
	return Model{cfg: cfg, states: states, theme: widget.DefaultTheme()}
}

// Init does nothing until the first WindowSizeMsg, at which point widget
// rectangles are known and the initial runs are dispatched.
func (m Model) Init() tea.Cmd { return nil }

// Update advances the model in response to a message.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termW, m.termH = msg.Width, msg.Height
		m.relayout()
		if !m.started {
			m.started = true
			return m, m.initialCmds()
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
		return m, nil

	case tickMsg:
		return m, m.runWidget(msg.row, msg.col)

	case widgetResultMsg:
		st := m.states[msg.row][msg.col]
		w := m.cfg.Rows[msg.row].Widgets[msg.col]
		view, good := widget.FromResult(w.Title, st.lastGood, msg.result)
		st.view = view
		st.lastGood = good
		return m, scheduleTick(msg.row, msg.col, w.Interval)
	}
	return m, nil
}

// View renders the full frame by joining each widget box.
func (m Model) View() string {
	if !m.started {
		return ""
	}
	rowStrs := make([]string, len(m.states))
	for r := range m.states {
		cells := make([]string, len(m.states[r]))
		for c := range m.states[r] {
			st := m.states[r][c]
			cells[c] = widget.Render(st.view, st.rect, m.theme)
		}
		rowStrs[r] = lipgloss.JoinHorizontal(lipgloss.Top, cells...)
	}
	return lipgloss.JoinVertical(lipgloss.Left, rowStrs...)
}

// relayout recomputes every widget rectangle for the current terminal size.
func (m *Model) relayout() {
	rects := layout.Compute(m.termW, m.termH, m.cfg)
	for r := range m.states {
		for c := range m.states[r] {
			m.states[r][c].rect = rects[r][c]
		}
	}
}

func (m Model) initialCmds() tea.Cmd {
	var cmds []tea.Cmd
	for r := range m.cfg.Rows {
		for c := range m.cfg.Rows[r].Widgets {
			cmds = append(cmds, m.runWidget(r, c))
		}
	}
	return tea.Batch(cmds...)
}

// runWidget returns a Cmd that runs one widget's script asynchronously.
func (m Model) runWidget(r, c int) tea.Cmd {
	w := m.cfg.Rows[r].Widgets[c]
	rect := m.states[r][c].rect
	termW, termH := m.termW, m.termH
	return func() tea.Msg {
		env := runner.WidgetEnv(rect.W-2, rect.H-2, termW, termH)
		res := runner.Run(context.Background(), runner.RunSpec{
			Script:  w.Script,
			Timeout: w.Timeout,
			Env:     env,
		})
		return widgetResultMsg{row: r, col: c, result: res}
	}
}

func scheduleTick(r, c int, d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return tickMsg{row: r, col: c}
	})
}
