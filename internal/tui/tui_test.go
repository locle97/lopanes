package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/locle97/quash-board/internal/config"
	"github.com/locle97/quash-board/internal/runner"
	"github.com/locle97/quash-board/internal/widget"
)

func twoWidgetCfg() config.Config {
	return config.Config{
		DefaultInterval: time.Second,
		DefaultTimeout:  time.Second,
		Rows: []config.Row{
			{HeightWeight: 1, Widgets: []config.Widget{
				{Name: "a", Title: "a", Script: "echo a", Interval: time.Second, Timeout: time.Second, WidthWeight: 1},
				{Name: "b", Title: "b", Script: "echo b", Interval: time.Second, Timeout: time.Second, WidthWeight: 1},
			}},
		},
	}
}

func TestInitReturnsNilUntilSized(t *testing.T) {
	m := New(twoWidgetCfg())
	if cmd := m.Init(); cmd != nil {
		t.Fatal("Init should defer initial runs until first WindowSizeMsg")
	}
	if m.View() != "" {
		t.Fatal("View before sizing should be empty")
	}
}

func TestWindowSizeStartsAndLaysOut(t *testing.T) {
	m := New(twoWidgetCfg())
	updated, cmd := m.Update(tea.WindowSizeMsg{Width: 40, Height: 10})
	m = updated.(Model)
	if cmd == nil {
		t.Fatal("first WindowSizeMsg should dispatch initial run commands")
	}
	if !m.started {
		t.Fatal("model should be marked started")
	}
	// Two widgets share width 40 -> 20 each.
	if m.states[0][0].rect.W != 20 || m.states[0][1].rect.W != 20 {
		t.Fatalf("layout not applied: %+v", m.states[0])
	}
	if got := m.View(); got == "" || !strings.Contains(got, "…") {
		t.Fatalf("view should show pending placeholders:\n%s", got)
	}
}

func TestWidgetResultUpdatesStateAndSchedulesTick(t *testing.T) {
	m := New(twoWidgetCfg())
	u, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 10})
	m = u.(Model)

	u, cmd := m.Update(widgetResultMsg{row: 0, col: 0, result: runner.Result{Stdout: "hello", ExitCode: 0}})
	m = u.(Model)
	if m.states[0][0].view.State != widget.StateOK {
		t.Fatalf("state not OK: %+v", m.states[0][0].view)
	}
	if m.states[0][0].lastGood != "hello" {
		t.Fatalf("lastGood = %q", m.states[0][0].lastGood)
	}
	if cmd == nil {
		t.Fatal("a successful result should schedule the next tick")
	}
}

func TestErrorResultKeepsLastGood(t *testing.T) {
	m := New(twoWidgetCfg())
	u, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 10})
	m = u.(Model)
	u, _ = m.Update(widgetResultMsg{row: 0, col: 0, result: runner.Result{Stdout: "good", ExitCode: 0}})
	m = u.(Model)
	u, _ = m.Update(widgetResultMsg{row: 0, col: 0, result: runner.Result{ExitCode: 1, Stderr: "boom"}})
	m = u.(Model)
	st := m.states[0][0]
	if st.view.State != widget.StateError || st.view.Body != "good" || st.lastGood != "good" {
		t.Fatalf("error should retain last good output: %+v", st)
	}
}

func TestQuitKeys(t *testing.T) {
	m := New(twoWidgetCfg())
	for _, key := range []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune{'q'}},
		{Type: tea.KeyCtrlC},
	} {
		_, cmd := m.Update(key)
		if cmd == nil {
			t.Fatalf("key %v should produce a quit command", key)
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Fatalf("key %v cmd is not tea.Quit", key)
		}
	}
}

func TestResizeRecomputesLayoutWithoutRestart(t *testing.T) {
	m := New(twoWidgetCfg())
	u, _ := m.Update(tea.WindowSizeMsg{Width: 40, Height: 10})
	m = u.(Model)
	u, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	m = u.(Model)
	if cmd != nil {
		t.Fatal("resize after start should not re-dispatch initial runs")
	}
	if m.states[0][0].rect.W != 40 {
		t.Fatalf("resize not applied: %+v", m.states[0][0].rect)
	}
}
