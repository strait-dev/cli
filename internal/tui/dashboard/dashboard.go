// Package dashboard implements the `strait tui` interactive dashboard. The
// dashboard is the single GUI surface for the CLI: a k9s-style pane switcher
// that exposes the most common read paths (jobs, runs, workflows, workflow
// runs) without per-command TUI variants. Write actions stay in the regular
// CLI surface — the dashboard is read-only by design to keep the maintenance
// surface bounded.
package dashboard

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/strait-dev/cli/internal/client"
	"github.com/strait-dev/cli/internal/types"
)

// Loader is the subset of the API client the dashboard depends on. Tests
// inject fakes; production uses *client.Client.
type Loader interface {
	ListJobs(ctx context.Context, projectID string) ([]types.Job, error)
	ListRuns(ctx context.Context, projectID, status string, limit int, cursor *time.Time) ([]types.JobRun, error)
	ListWorkflows(ctx context.Context, projectID string) ([]types.Workflow, error)
	ListWorkflowRunsByProject(ctx context.Context, projectID, status string, limit int) ([]types.WorkflowRun, error)
}

// Ensure *client.Client satisfies Loader at compile time.
var _ Loader = (*client.Client)(nil)

// Config bundles dashboard inputs. ProjectID is required.
type Config struct {
	Loader       Loader
	ProjectID    string
	RefreshEvery time.Duration
	// FetchTimeout caps each pane's API call. Defaults to 10s.
	FetchTimeout time.Duration
}

// New returns a tea.Model wired up against cfg. Run via tea.NewProgram(...).Run.
func New(cfg Config) (tea.Model, error) {
	if cfg.Loader == nil {
		return nil, errors.New("dashboard: loader is required")
	}
	if cfg.ProjectID == "" {
		return nil, errors.New("dashboard: project id is required (set --project or run `strait projects switch`)")
	}
	if cfg.RefreshEvery <= 0 {
		cfg.RefreshEvery = 5 * time.Second
	}
	if cfg.FetchTimeout <= 0 {
		cfg.FetchTimeout = 10 * time.Second
	}
	m := &Model{
		cfg:    cfg,
		panes:  defaultPanes(),
		active: 0,
	}
	return m, nil
}

// pane describes one tab in the dashboard. fetch produces a tea.Cmd that
// returns paneLoadedMsg when complete.
type pane struct {
	id       string
	title    string
	columns  []string
	rows     [][]string
	cursor   int
	loading  bool
	err      error
	loadedAt time.Time
}

func defaultPanes() []*pane {
	return []*pane{
		{id: "jobs", title: "Jobs", columns: []string{"SLUG", "ID", "NAME"}},
		{id: "runs", title: "Runs", columns: []string{"STATUS", "ID", "JOB", "AGE"}},
		{id: "workflows", title: "Workflows", columns: []string{"NAME", "ID", "VERSION"}},
		{id: "workflow-runs", title: "Workflow Runs", columns: []string{"STATUS", "ID", "WORKFLOW", "AGE"}},
	}
}

// Model is the bubbletea state for the dashboard.
type Model struct {
	cfg    Config
	panes  []*pane
	active int
	width  int
	height int
	help   bool
	quit   bool
}

// Init kicks off the first refresh of every pane and starts the periodic tick.
func (m *Model) Init() tea.Cmd {
	cmds := make([]tea.Cmd, 0, len(m.panes)+1)
	for i := range m.panes {
		cmds = append(cmds, m.fetchPane(i))
		m.panes[i].loading = true
	}
	cmds = append(cmds, m.tick())
	return tea.Batch(cmds...)
}

// Update routes keys and tick messages to pane state changes.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	case paneLoadedMsg:
		if msg.idx >= 0 && msg.idx < len(m.panes) {
			p := m.panes[msg.idx]
			p.loading = false
			p.err = msg.err
			p.rows = msg.rows
			p.loadedAt = time.Now()
			if p.cursor >= len(p.rows) {
				p.cursor = max(0, len(p.rows)-1)
			}
		}
		return m, nil
	case tickMsg:
		cmds := []tea.Cmd{m.tick()}
		for i := range m.panes {
			if !m.panes[i].loading {
				m.panes[i].loading = true
				cmds = append(cmds, m.fetchPane(i))
			}
		}
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "esc":
		m.quit = true
		return m, tea.Quit
	case "?":
		m.help = !m.help
		return m, nil
	case "tab", "right", "l":
		m.active = (m.active + 1) % len(m.panes)
		return m, nil
	case "shift+tab", "left", "h":
		m.active = (m.active - 1 + len(m.panes)) % len(m.panes)
		return m, nil
	case "1", "2", "3", "4":
		i := int(msg.String()[0] - '1')
		if i < len(m.panes) {
			m.active = i
		}
		return m, nil
	case "j", "down":
		p := m.panes[m.active]
		if p.cursor < len(p.rows)-1 {
			p.cursor++
		}
		return m, nil
	case "k", "up":
		p := m.panes[m.active]
		if p.cursor > 0 {
			p.cursor--
		}
		return m, nil
	case "g", "home":
		m.panes[m.active].cursor = 0
		return m, nil
	case "G", "end":
		p := m.panes[m.active]
		p.cursor = max(0, len(p.rows)-1)
		return m, nil
	case "r":
		p := m.panes[m.active]
		if !p.loading {
			p.loading = true
			return m, m.fetchPane(m.active)
		}
		return m, nil
	}
	return m, nil
}

// View renders the dashboard. The output structure is:
//
//	┌─ Strait ─────── project=<id> ─┐
//	│  [1] Jobs  [2] Runs  ...      │  ← tab bar; active pane highlighted
//	│                               │
//	│  <table for active pane>      │
//	│                               │
//	│  status line                  │
//	└───────────────────────────────┘
func (m *Model) View() string {
	if m.quit {
		return ""
	}
	if m.help {
		return helpView()
	}
	width := m.width
	if width <= 0 {
		width = 100
	}

	var b strings.Builder
	b.WriteString(titleBar(width, m.cfg.ProjectID))
	b.WriteByte('\n')
	b.WriteString(tabBar(m.panes, m.active, width))
	b.WriteByte('\n')
	b.WriteString(activePaneView(m.panes[m.active], width, m.bodyHeight()))
	b.WriteByte('\n')
	b.WriteString(statusBar(m.panes[m.active], width))
	return b.String()
}

func (m *Model) bodyHeight() int {
	const chrome = 5 // title + tabs + status + padding
	h := m.height - chrome
	if h < 4 {
		return 4
	}
	return h
}

type tickMsg time.Time

func (m *Model) tick() tea.Cmd {
	return tea.Tick(m.cfg.RefreshEvery, func(t time.Time) tea.Msg { return tickMsg(t) })
}

type paneLoadedMsg struct {
	idx  int
	rows [][]string
	err  error
}

func (m *Model) fetchPane(idx int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), m.cfg.FetchTimeout)
		defer cancel()

		switch m.panes[idx].id {
		case "jobs":
			jobs, err := m.cfg.Loader.ListJobs(ctx, m.cfg.ProjectID)
			return paneLoadedMsg{idx: idx, err: err, rows: renderJobRows(jobs)}
		case "runs":
			runs, err := m.cfg.Loader.ListRuns(ctx, m.cfg.ProjectID, "", 50, nil)
			return paneLoadedMsg{idx: idx, err: err, rows: renderRunRows(runs)}
		case "workflows":
			workflows, err := m.cfg.Loader.ListWorkflows(ctx, m.cfg.ProjectID)
			return paneLoadedMsg{idx: idx, err: err, rows: renderWorkflowRows(workflows)}
		case "workflow-runs":
			runs, err := m.cfg.Loader.ListWorkflowRunsByProject(ctx, m.cfg.ProjectID, "", 50)
			return paneLoadedMsg{idx: idx, err: err, rows: renderWorkflowRunRows(runs)}
		}
		return paneLoadedMsg{idx: idx, err: fmt.Errorf("unknown pane %q", m.panes[idx].id)}
	}
}

func renderJobRows(jobs []types.Job) [][]string {
	rows := make([][]string, 0, len(jobs))
	for _, j := range jobs {
		rows = append(rows, []string{j.Slug, j.ID, j.Name})
	}
	return rows
}

func renderRunRows(runs []types.JobRun) [][]string {
	rows := make([][]string, 0, len(runs))
	now := time.Now()
	for _, r := range runs {
		rows = append(rows, []string{string(r.Status), r.ID, r.JobID, relAge(r.CreatedAt, now)})
	}
	return rows
}

func renderWorkflowRows(workflows []types.Workflow) [][]string {
	rows := make([][]string, 0, len(workflows))
	for _, w := range workflows {
		rows = append(rows, []string{w.Name, w.ID, fmt.Sprintf("v%d", w.Version)})
	}
	return rows
}

func renderWorkflowRunRows(runs []types.WorkflowRun) [][]string {
	rows := make([][]string, 0, len(runs))
	now := time.Now()
	for _, r := range runs {
		rows = append(rows, []string{string(r.Status), r.ID, r.WorkflowID, relAge(r.CreatedAt, now)})
	}
	return rows
}

func relAge(t, now time.Time) string {
	if t.IsZero() {
		return "-"
	}
	d := now.Sub(t).Truncate(time.Second)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

// View fragments.

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7dd3fc"))
	mutedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#94a3b8"))
	activeTab  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#fafafa")).
			Background(lipgloss.Color("#1e293b")).Padding(0, 1)
	inactiveTab  = lipgloss.NewStyle().Foreground(lipgloss.Color("#94a3b8")).Padding(0, 1)
	headerStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#fafafa"))
	selectedRow  = lipgloss.NewStyle().Background(lipgloss.Color("#1e293b")).Foreground(lipgloss.Color("#fafafa"))
	errStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#f87171"))
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#86efac"))
)

func titleBar(width int, project string) string {
	left := titleStyle.Render("Strait Dashboard")
	right := mutedStyle.Render("project=" + project)
	gap := max(1, width-lipgloss.Width(left)-lipgloss.Width(right))
	return left + strings.Repeat(" ", gap) + right
}

func tabBar(panes []*pane, active, width int) string {
	parts := make([]string, 0, len(panes))
	for i, p := range panes {
		label := fmt.Sprintf("[%d] %s", i+1, p.title)
		if i == active {
			parts = append(parts, activeTab.Render(label))
		} else {
			parts = append(parts, inactiveTab.Render(label))
		}
	}
	bar := strings.Join(parts, " ")
	hint := mutedStyle.Render(" tab/←/→ switch · r refresh · ? help · q quit")
	gap := width - lipgloss.Width(bar) - lipgloss.Width(hint)
	if gap < 1 {
		return bar
	}
	return bar + strings.Repeat(" ", gap) + hint
}

func activePaneView(p *pane, width, height int) string {
	if p.err != nil {
		return errStyle.Render("error: " + p.err.Error())
	}
	if p.loading && len(p.rows) == 0 {
		return mutedStyle.Render("loading…")
	}
	if len(p.rows) == 0 {
		return mutedStyle.Render("no " + p.title + " yet")
	}

	colWidths := computeColWidths(p.columns, p.rows, width)
	var b strings.Builder

	// Header.
	for i, c := range p.columns {
		b.WriteString(headerStyle.Render(padRight(c, colWidths[i])))
		if i < len(p.columns)-1 {
			b.WriteString("  ")
		}
	}
	b.WriteByte('\n')

	// Rows. Cap by height to avoid scrolling off-screen.
	rowCap := max(1, height-1)
	start := 0
	if p.cursor >= rowCap {
		start = p.cursor - rowCap + 1
	}
	end := min(len(p.rows), start+rowCap)
	for i := start; i < end; i++ {
		row := p.rows[i]
		line := strings.Builder{}
		for j := range p.columns {
			val := ""
			if j < len(row) {
				val = row[j]
			}
			line.WriteString(padRight(truncate(val, colWidths[j]), colWidths[j]))
			if j < len(p.columns)-1 {
				line.WriteString("  ")
			}
		}
		if i == p.cursor {
			b.WriteString(selectedRow.Render(line.String()))
		} else {
			b.WriteString(line.String())
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func statusBar(p *pane, width int) string {
	left := ""
	if p.loading {
		left = mutedStyle.Render("refreshing…")
	} else if !p.loadedAt.IsZero() {
		left = successStyle.Render(fmt.Sprintf("loaded %s ago · %d rows", time.Since(p.loadedAt).Truncate(time.Second), len(p.rows)))
	}
	right := mutedStyle.Render(fmt.Sprintf("row %d/%d", min(p.cursor+1, len(p.rows)), len(p.rows)))
	gap := max(1, width-lipgloss.Width(left)-lipgloss.Width(right))
	return left + strings.Repeat(" ", gap) + right
}

func helpView() string {
	body := []string{
		titleStyle.Render("Keybindings"),
		"",
		"  tab / →   next pane",
		"  shift+tab / ←   previous pane",
		"  1 2 3 4    jump to pane",
		"  j / ↓     move cursor down",
		"  k / ↑     move cursor up",
		"  g / G     top / bottom",
		"  r         refresh active pane",
		"  ?         toggle this help",
		"  q / esc   quit",
		"",
		mutedStyle.Render("press ? to close"),
	}
	return strings.Join(body, "\n")
}

func computeColWidths(cols []string, rows [][]string, total int) []int {
	widths := make([]int, len(cols))
	for i, c := range cols {
		widths[i] = lipgloss.Width(c)
	}
	for _, r := range rows {
		for i := range cols {
			if i < len(r) && lipgloss.Width(r[i]) > widths[i] {
				widths[i] = lipgloss.Width(r[i])
			}
		}
	}
	// Cap each column to avoid one wide value blowing out the layout. The cap
	// is a soft budget — caller renders best-effort.
	budget := total - 2*(len(cols)-1)
	if budget < len(cols) {
		return widths
	}
	sum := 0
	for _, w := range widths {
		sum += w
	}
	if sum <= budget {
		return widths
	}
	// Proportional shrink.
	scaled := make([]int, len(widths))
	for i, w := range widths {
		scaled[i] = max(4, w*budget/sum)
	}
	return scaled
}

func padRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

func truncate(s string, width int) string {
	if lipgloss.Width(s) <= width {
		return s
	}
	if width <= 1 {
		return "…"
	}
	// Best-effort byte slice — lipgloss.Width handles the rune-aware sizing.
	return s[:width-1] + "…"
}
