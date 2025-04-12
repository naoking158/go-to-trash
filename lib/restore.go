package lib

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

var _ tea.Model = (*model)(nil)

type model struct {
	table      table.Model
	pathToFile map[string]ToBeMovedFile
	selected   map[string]ToBeMovedFile
}

const (
	ColTitleMark        = "Mark"
	ColTitlePathInTrash = "Path in Trash"
	ColTitlePathInOrig  = "Path in Orig."
	ColTitleRemovedAt   = "RemovedAt."
)

const (
	ColBaseWidthForMark      = 4
	ColBaseWidthForRemovedAt = 20
	ColBaseWidthForPath      = 20
	TableBorderWidth         = 6
)

var columns = []table.Column{
	{Title: ColTitleMark, Width: ColBaseWidthForMark},
	{Title: ColTitlePathInTrash, Width: ColBaseWidthForPath},
	{Title: ColTitlePathInOrig, Width: ColBaseWidthForPath},
	{Title: ColTitleRemovedAt, Width: ColBaseWidthForRemovedAt},
}

func newModel(files RemovedFiles) model {
	files = files.Sorted()

	rows := make([]table.Row, len(files))
	for i, f := range files {
		rows[i] = []string{"", MapHomeToTilde(f.To), MapHomeToTilde(f.From), MapHomeToTilde(f.MovedAt)}
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(7),
	)

	pathToFile := make(map[string]ToBeMovedFile, len(files))
	for _, f := range files {
		// key is unique path in trash
		pathToFile[MapHomeToTilde(f.To)] = f
	}

	return model{
		table:      t,
		pathToFile: pathToFile,
		selected:   make(map[string]ToBeMovedFile),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	// resize table width
	case tea.WindowSizeMsg:
		availableWidth := msg.Width - (TableBorderWidth * 2)
		return m.updateColumnWidths(availableWidth)

	// handle key input
	case tea.KeyMsg:
		// execute command
		switch msg.String() {
		case "ctrl+c", "ctrl+g", "q":
			return m, tea.Quit
		case "ctrl+m", "ctrl+j", "enter", " ":
			return m.update()
		case "X":
			return m.restore()
		}
	}

	// handle undefined key input
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

func (m model) update() (tea.Model, tea.Cmd) {
	r := m.table.SelectedRow()
	pathInTrash := r[1]
	if _, ok := m.selected[pathInTrash]; ok {
		delete(m.selected, pathInTrash)
	} else {
		f := m.pathToFile[pathInTrash]
		m.selected[pathInTrash] = f
	}

	return m.updateRow()
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

func (m model) restore() (tea.Model, tea.Cmd) {
	if len(m.selected) == 0 {
		return nil, nil
	}

	// restore marked files
	
}

func (m model) View() string {
	var b strings.Builder

	b.WriteString("\n[space/enter: toggle mark, X: restore marked files, q/Ctrl+c/Ctrl+g: quit]\n\n")
	b.WriteString(baseStyle.Render(m.table.View()) + "\n\n")
	
	if len(m.selected) == 0 {
		return b.String()
	}

	selected := make(RemovedFiles, 0, len(m.selected))
	for _, v := range m.selected {
		selected = append(selected, v)
	}
	selected = selected.Sorted()

	b.WriteString("Selected files:\n")

	for i, f := range selected {
		b.WriteString(
			fmt.Sprintf("%v. %v → %v\n", i+1, MapHomeToTilde(f.To), MapHomeToTilde(f.From)),
		)
	}

	return b.String()
}

func Restore(files []ToBeMovedFile) error {
	if len(files) == 0 {
		fmt.Println("quit due to no history")
		return nil
	}

	p := tea.NewProgram(newModel(files))
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}
