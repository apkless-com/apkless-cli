package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Colors ──

var (
	green   = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	yellow  = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	red     = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	cyan    = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	dim     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	bold    = lipgloss.NewStyle().Bold(true)
	success = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).SetString("✓")
	fail    = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).SetString("✗")
)

// ── Status helpers ──

func statusStyle(status string) string {
	switch status {
	case "ready":
		return green.Render("● ready")
	case "creating":
		return yellow.Render("◌ creating")
	case "error":
		return red.Render("✗ error")
	case "destroyed":
		return dim.Render("○ destroyed")
	default:
		return dim.Render(status)
	}
}

func methodStyle(method string) string {
	switch method {
	case "GET":
		return green.Render(method)
	case "POST":
		return yellow.Render(method)
	case "PUT", "PATCH":
		return cyan.Render(method)
	case "DELETE":
		return red.Render(method)
	default:
		return dim.Render(method)
	}
}

func statusCodeStyle(code int) string {
	s := fmt.Sprintf("%d", code)
	switch {
	case code < 300:
		return green.Render(s)
	case code < 400:
		return cyan.Render(s)
	case code < 500:
		return yellow.Render(s)
	default:
		return red.Render(s)
	}
}

// ── Key-value display ──

func printKV(pairs ...string) {
	labelStyle := lipgloss.NewStyle().Width(14).Foreground(lipgloss.Color("8"))
	for i := 0; i < len(pairs)-1; i += 2 {
		fmt.Printf("  %s %s\n", labelStyle.Render(pairs[i]), pairs[i+1])
	}
}

// ── Spinner ──

type spinnerModel struct {
	spinner spinner.Model
	msg     string
	done    bool
	result  string
	err     error
	action  func() (string, error)
}

type actionDone struct {
	result string
	err    error
}

func (m spinnerModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			result, err := m.action()
			return actionDone{result, err}
		},
	)
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case actionDone:
		m.done = true
		m.result = msg.result
		m.err = msg.err
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m spinnerModel) View() string {
	if m.done {
		if m.err != nil {
			return fmt.Sprintf("  %s %s: %v\n", fail, m.msg, m.err)
		}
		return fmt.Sprintf("  %s %s\n", success, m.msg)
	}
	return fmt.Sprintf("  %s %s\n", m.spinner.View(), m.msg)
}

// runWithSpinner runs an action with a spinner animation
func runWithSpinner(msg string, action func() (string, error)) (string, error) {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("14"))

	model := spinnerModel{
		spinner: s,
		msg:     msg,
		action:  action,
	}

	p := tea.NewProgram(model, tea.WithOutput(os.Stderr))
	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	m := finalModel.(spinnerModel)
	return m.result, m.err
}

// ── Table ──

func renderTable(columns []string, rows [][]string) string {
	cols := make([]table.Column, len(columns))
	for i, c := range columns {
		w := len(c)
		for _, row := range rows {
			if i < len(row) && len(row[i]) > w {
				w = len(row[i])
			}
		}
		if w > 50 {
			w = 50
		}
		cols[i] = table.Column{Title: c, Width: w + 2}
	}

	tableRows := make([]table.Row, len(rows))
	for i, row := range rows {
		tableRows[i] = row
	}

	t := table.New(
		table.WithColumns(cols),
		table.WithRows(tableRows),
		table.WithHeight(len(rows)+1),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(true)
	s.Selected = lipgloss.NewStyle() // disable selection highlighting
	t.SetStyles(s)

	return t.View()
}

// ── Time helpers ──

func timeAgo(t string) string {
	parsed, err := time.Parse(time.RFC3339, t)
	if err != nil {
		// Try with fractional seconds
		parsed, err = time.Parse("2006-01-02T15:04:05.999999-07:00", t)
		if err != nil {
			return t
		}
	}
	d := time.Since(parsed)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func expiresIn(t string) string {
	parsed, err := time.Parse(time.RFC3339, t)
	if err != nil {
		parsed, _ = time.Parse("2006-01-02T15:04:05.999999-07:00", t)
	}
	d := time.Until(parsed)
	if d < 0 {
		return red.Render("expired")
	}
	if d < time.Hour {
		return yellow.Render(fmt.Sprintf("%dm left", int(d.Minutes())))
	}
	return green.Render(fmt.Sprintf("%dh %dm left", int(d.Hours()), int(d.Minutes())%60))
}
