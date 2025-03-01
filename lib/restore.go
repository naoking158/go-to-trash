package lib

import (
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

var _ tea.Model = (*model)(nil)

type model struct {
	table table.Model
}

var columns = []table.Column{
	{Title: "Mark", Width: 2},
	{Title: "Path in Trash", Width: 10},
	{Title: "Path in Orig.", Width: 10},
	{Title: "RemovedAt.", Width: 10},
}

func newModel(files []RemovedFile) model {
	rows := make([]table.Row, len(files))
	for i, f := range files {
		rows[i] = []string{"", f.To, f.From, f.RemovedAt}
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(7),
	)

	return model{
		table: t,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:

		switch msg.String() {

		case "ctrl+c", "q":
			return m, tea.Quit

		case "ctrl+m", "ctrl+j", "enter", " ":
			return m.updateRow()
		}
	}

	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m model) updateRow() (tea.Model, tea.Cmd) {
	// toggle mark of selected raw
	r := m.table.SelectedRow()
	switch r[0] {
	case "":
		r[0] = "x"
	default:
		r[0] = ""
	}

	rows := m.table.Rows()
	rows[m.table.Cursor()] = r
	m.table.SetRows(rows)
	return m, nil
}

func (m model) View() string {
	return baseStyle.Render(m.table.View()) + "\n"
}

func Restore(files []RemovedFile) error {
	p := tea.NewProgram(newModel(files))
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}
