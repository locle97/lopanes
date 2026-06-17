package widget

import (
	"strings"
	"testing"

	"quash-board/internal/layout"
	"quash-board/internal/runner"
)

func TestRenderOK(t *testing.T) {
	v := View{Title: "cpu", State: StateOK, Body: "42%"}
	got := Render(v, layout.Rect{W: 10, H: 4}, PlainTheme())
	want := strings.Join([]string{
		"┌─ cpu ──┐",
		"│42%     │",
		"│        │",
		"└────────┘",
	}, "\n")
	if got != want {
		t.Fatalf("OK render mismatch:\n got:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenderPending(t *testing.T) {
	v := View{Title: "mem", State: StatePending}
	got := Render(v, layout.Rect{W: 9, H: 3}, PlainTheme())
	want := strings.Join([]string{
		"┌─ mem ─┐",
		"│…      │",
		"└───────┘",
	}, "\n")
	if got != want {
		t.Fatalf("pending render mismatch:\n got:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenderClipsLongBody(t *testing.T) {
	v := View{Title: "x", State: StateOK, Body: "abcdefghij\nsecond\nthird\nfourth"}
	got := Render(v, layout.Rect{W: 7, H: 4}, PlainTheme()) // innerW=5, innerH=2
	lines := strings.Split(got, "\n")
	if len(lines) != 4 {
		t.Fatalf("want 4 lines, got %d:\n%s", len(lines), got)
	}
	// Body line 1 truncated to 5 cells inside borders.
	if lines[1] != "│abcde│" {
		t.Fatalf("long line not truncated: %q", lines[1])
	}
	// Only innerH=2 body lines; "third"/"fourth" dropped.
	if lines[2] != "│secon│" {
		t.Fatalf("second body line wrong: %q", lines[2])
	}
}

func TestRenderError(t *testing.T) {
	v := View{Title: "net", State: StateError, ErrLabel: "exit 1", ErrTail: "boom\n"}
	got := Render(v, layout.Rect{W: 16, H: 4}, PlainTheme())
	lines := strings.Split(got, "\n")
	if len(lines) != 4 {
		t.Fatalf("want 4 lines, got %d", len(lines))
	}
	for i, ln := range lines {
		if w := visibleWidth(ln); w != 16 {
			t.Fatalf("line %d width = %d want 16: %q", i, w, ln)
		}
	}
	if !strings.Contains(got, "exit 1") {
		t.Fatalf("missing error label:\n%s", got)
	}
	if !strings.Contains(got, "boom") {
		t.Fatalf("missing stderr tail:\n%s", got)
	}
}

func TestRenderTinyRectDoesNotPanic(t *testing.T) {
	for _, r := range []layout.Rect{{W: 0, H: 0}, {W: 1, H: 1}, {W: 2, H: 2}, {W: 3, H: 1}} {
		_ = Render(View{Title: "t", State: StateOK, Body: "hi"}, r, PlainTheme())
	}
}

func TestFromResultOK(t *testing.T) {
	res := runner.Result{Stdout: "ok-out", ExitCode: 0}
	v, good := FromResult("title", "prev", res)
	if v.State != StateOK || v.Body != "ok-out" || good != "ok-out" {
		t.Fatalf("ok mapping: %+v good=%q", v, good)
	}
}

func TestFromResultErrorKeepsLastGood(t *testing.T) {
	res := runner.Result{ExitCode: 2, Stderr: "bad"}
	v, good := FromResult("title", "prevgood", res)
	if v.State != StateError || v.ErrLabel != "exit 2" || v.Body != "prevgood" {
		t.Fatalf("error mapping: %+v", v)
	}
	if good != "prevgood" { // last good preserved
		t.Fatalf("good = %q want prevgood", good)
	}
}

func TestFromResultTimeout(t *testing.T) {
	res := runner.Result{TimedOut: true, ExitCode: -1}
	v, _ := FromResult("t", "", res)
	if v.State != StateError || v.ErrLabel != "timed out" {
		t.Fatalf("timeout mapping: %+v", v)
	}
}

func TestContentHeight(t *testing.T) {
	if h := ContentHeight(View{State: StatePending}); h != 1 {
		t.Fatalf("pending height = %d want 1", h)
	}
	if h := ContentHeight(View{State: StateOK, Body: "a\nb\nc"}); h != 3 {
		t.Fatalf("ok height = %d want 3", h)
	}
	// error: 1 indicator + 1 stderr line + 0 body lines (empty body)
	if h := ContentHeight(View{State: StateError, ErrLabel: "exit 1", ErrTail: "boom\n"}); h != 2 {
		t.Fatalf("error height = %d want 2", h)
	}
}
