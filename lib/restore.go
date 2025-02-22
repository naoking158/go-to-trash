package lib

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

var _ tea.Model = (*model)(nil)

type model struct {
	files []RemovedFile
	cursor int
	selected map[int]struct{}
}

func newModel(files []RemovedFile) model {
	return model{
		files: files,
		selected: make(map[int]struct{}),
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

		case "ctrl+p", "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}

		case "ctrl+n", "j", "down":
			if m.cursor < len(m.files) - 1 {
				m.cursor++
			}

		case "ctrl+m", "ctrl+j", "enter", " ":
			_, ok := m.selected[m.cursor]
			if ok {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = struct{}{}
			}
		}
	}

	return m, nil
}

func (m model) View() string {
	var s strings.Builder

	s.WriteString("Select files you want to restore\n\n")

	for i, file := range m.files {

		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		checked := " "
		if _, ok := m.selected[i]; ok {
			checked = "x"
		}

		s.WriteString(fmt.Sprintf("%s [%s] from= %s, to= %s, removedAt: %s\n", cursor, checked, file.From, file.To, file.RemovedAt))
	}

	s.WriteString("\nPress q to quite.\n")

	return s.String()
}

func Restore(files []RemovedFile) error {
	p := tea.NewProgram(newModel(files))
	if _, err := p.Run(); err != nil {
		return err
	}

	return nil
}
