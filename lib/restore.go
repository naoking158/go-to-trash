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

const (
	ColTitleMark        = "Mark"
	ColTitlePathInTrash = "Path in Trash"
	ColTitlePathInOrig  = "Path in Orig."
	ColTitleRemovedAt   = "RemovedAt."
)

const (
	ColBaseWidthForMark = 4
	ColBaseWidthForRemovedAt = 20
	ColBaseWidthForPath = 20
	TableBorderWidth = 6
)

var columns = []table.Column{
	{Title: ColTitleMark, Width: ColBaseWidthForMark},
	{Title: ColTitlePathInTrash, Width: ColBaseWidthForPath},
	{Title: ColTitlePathInOrig, Width: ColBaseWidthForPath},
	{Title: ColTitleRemovedAt, Width: ColBaseWidthForRemovedAt},
}

func newModel(files []RemovedFile) model {
	rows := make([]table.Row, len(files))
	for i, f := range files {
		rows[i] = []string{"", MapHomeToTilde(f.To), MapHomeToTilde(f.From), f.RemovedAt}
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

	case tea.WindowSizeMsg:
		// availableWidth := msg.Width - 30
		availableWidth := msg.Width - (TableBorderWidth * 2)
		return m.updateColumnWidths(availableWidth)

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

func (m model) updateColumnWidths(availableWidth int) (tea.Model, tea.Cmd) {
	remainingWidth := availableWidth - ColBaseWidthForMark - ColBaseWidthForRemovedAt
	pathWidth := remainingWidth / 2

	if pathWidth < ColBaseWidthForPath {
		pathWidth = ColBaseWidthForPath
	}

	cols := m.table.Columns()
	for i, c := range cols {

		var w int
		switch c.Title {
		case ColTitlePathInTrash, ColTitlePathInOrig:
			w = pathWidth
		case ColTitleMark:
			w = ColBaseWidthForMark
		case ColTitleRemovedAt:
			w = ColBaseWidthForRemovedAt
		default:
			panic("unknown column")
		}

		cols[i].Width = w
	}

	m.table.SetColumns(cols)
	return m, nil
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
