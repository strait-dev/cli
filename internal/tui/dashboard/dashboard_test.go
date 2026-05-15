package dashboard

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/strait-dev/cli/internal/types"
)

type fakeLoader struct {
	jobs      []types.Job
	jobsErr   error
	runs      []types.JobRun
	runsErr   error
	workflows []types.Workflow
	wfErr     error
	wfRuns    []types.WorkflowRun
	wfRunsErr error
}

func (f *fakeLoader) ListJobs(_ context.Context, _ string) ([]types.Job, error) {
	return f.jobs, f.jobsErr
}

func (f *fakeLoader) ListRuns(_ context.Context, _, _ string, _ int, _ *time.Time) ([]types.JobRun, error) {
	return f.runs, f.runsErr
}

func (f *fakeLoader) ListWorkflows(_ context.Context, _ string) ([]types.Workflow, error) {
	return f.workflows, f.wfErr
}

func (f *fakeLoader) ListWorkflowRunsByProject(_ context.Context, _, _ string, _ int) ([]types.WorkflowRun, error) {
	return f.wfRuns, f.wfRunsErr
}

func TestNew_RequiresLoader(t *testing.T) {
	t.Parallel()
	if _, err := New(Config{ProjectID: "p"}); err == nil {
		t.Fatal("expected error when loader is nil")
	}
}

func TestNew_RequiresProjectID(t *testing.T) {
	t.Parallel()
	if _, err := New(Config{Loader: &fakeLoader{}}); err == nil {
		t.Fatal("expected error when project id is empty")
	}
}

func TestModel_FetchesAllPanesOnInit(t *testing.T) {
	t.Parallel()
	loader := &fakeLoader{
		jobs:      []types.Job{{ID: "j1", Slug: "hello", Name: "Hello"}},
		runs:      []types.JobRun{{ID: "r1", JobID: "j1", Status: types.StatusExecuting, CreatedAt: time.Now()}},
		workflows: []types.Workflow{{ID: "wf1", Name: "wf-one", Version: 3}},
		wfRuns:    []types.WorkflowRun{{ID: "wfr1", WorkflowID: "wf1", Status: types.WfStatusRunning, CreatedAt: time.Now()}},
	}
	model, err := New(Config{Loader: loader, ProjectID: "proj"})
	if err != nil {
		t.Fatal(err)
	}
	m := model.(*Model)
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("expected Init to return a tea.Cmd batch")
	}

	// Drain the batched commands. tea.Batch wraps them in a single tea.Cmd
	// that returns a tea.BatchMsg slice; iterate and apply each loaded msg.
	msgs := drainBatch(t, cmd)
	loadedCount := 0
	for _, msg := range msgs {
		if loaded, ok := msg.(paneLoadedMsg); ok {
			loadedCount++
			m.Update(loaded)
		}
	}
	if loadedCount != len(m.panes) {
		t.Fatalf("expected %d pane-loaded msgs, got %d", len(m.panes), loadedCount)
	}
	for i, p := range m.panes {
		if p.loading {
			t.Errorf("pane %d still loading", i)
		}
		if len(p.rows) == 0 {
			t.Errorf("pane %d (%s) has no rows", i, p.id)
		}
	}
}

func TestModel_FetchPropagatesErrors(t *testing.T) {
	t.Parallel()
	loader := &fakeLoader{jobsErr: errors.New("boom")}
	model, _ := New(Config{Loader: loader, ProjectID: "p"})
	m := model.(*Model)

	msg := m.fetchPane(0)().(paneLoadedMsg)
	if msg.err == nil || !strings.Contains(msg.err.Error(), "boom") {
		t.Fatalf("expected boom error, got %v", msg.err)
	}
}

func TestModel_KeyboardNavigation(t *testing.T) {
	t.Parallel()
	loader := &fakeLoader{
		jobs: []types.Job{
			{ID: "j1", Slug: "a"},
			{ID: "j2", Slug: "b"},
			{ID: "j3", Slug: "c"},
		},
	}
	model, _ := New(Config{Loader: loader, ProjectID: "p"})
	m := model.(*Model)
	// Seed pane 0 with rows.
	m.panes[0].rows = renderJobRows(loader.jobs)

	// tab moves forward.
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.active != 1 {
		t.Errorf("after tab: active=%d want 1", m.active)
	}
	// shift+tab goes back.
	m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if m.active != 0 {
		t.Errorf("after shift+tab: active=%d want 0", m.active)
	}
	// j moves cursor down.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.panes[0].cursor != 1 {
		t.Errorf("after j: cursor=%d want 1", m.panes[0].cursor)
	}
	// k moves up.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if m.panes[0].cursor != 0 {
		t.Errorf("after k: cursor=%d want 0", m.panes[0].cursor)
	}
	// G jumps to end.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if m.panes[0].cursor != len(m.panes[0].rows)-1 {
		t.Errorf("after G: cursor=%d want %d", m.panes[0].cursor, len(m.panes[0].rows)-1)
	}
	// Number keys select pane.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	if m.active != 2 {
		t.Errorf("after '3': active=%d want 2", m.active)
	}
	// ? toggles help.
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	if !m.help {
		t.Error("expected help to be toggled on")
	}
	// q quits.
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if !m.quit {
		t.Error("expected quit flag")
	}
	if cmd == nil {
		t.Error("expected tea.Quit cmd")
	}
}

func TestModel_View_RendersRows(t *testing.T) {
	t.Parallel()
	loader := &fakeLoader{jobs: []types.Job{{ID: "j1", Slug: "hello"}}}
	model, _ := New(Config{Loader: loader, ProjectID: "demo"})
	m := model.(*Model)
	m.width, m.height = 100, 20
	m.panes[0].rows = renderJobRows(loader.jobs)
	m.panes[0].loadedAt = time.Now()

	out := m.View()
	if !strings.Contains(out, "Strait Dashboard") {
		t.Errorf("missing title: %s", out)
	}
	if !strings.Contains(out, "project=demo") {
		t.Errorf("missing project id: %s", out)
	}
	if !strings.Contains(out, "hello") {
		t.Errorf("missing row content: %s", out)
	}
	if !strings.Contains(out, "Jobs") {
		t.Errorf("missing tab label: %s", out)
	}
}

func TestRelAge(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		in   time.Duration
		want string
	}{
		{30 * time.Second, "30s"},
		{2 * time.Minute, "2m"},
		{3 * time.Hour, "3h"},
		{48 * time.Hour, "2d"},
	}
	for _, c := range cases {
		got := relAge(now.Add(-c.in), now)
		if got != c.want {
			t.Errorf("relAge(%v) = %q want %q", c.in, got, c.want)
		}
	}
	if got := relAge(time.Time{}, now); got != "-" {
		t.Errorf("zero time: got %q want '-'", got)
	}
}

// drainBatch unwraps tea.BatchMsg returned from a tea.Cmd produced by
// tea.Batch. Each returned tea.Msg is collected.
func drainBatch(t *testing.T, cmd tea.Cmd) []tea.Msg {
	t.Helper()
	if cmd == nil {
		return nil
	}
	msg := cmd()
	switch m := msg.(type) {
	case tea.BatchMsg:
		out := make([]tea.Msg, 0, len(m))
		for _, c := range m {
			out = append(out, drainBatch(t, c)...)
		}
		return out
	default:
		return []tea.Msg{msg}
	}
}
