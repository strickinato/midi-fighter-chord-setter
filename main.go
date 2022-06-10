package main

import (
	"fmt"
	"os"

	list "github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"

	"gitlab.com/gomidi/midi/v2"
	_ "gitlab.com/gomidi/midi/v2/drivers/portmididrv" // autoregisters driver
)

type connectedSuccess (func(msg midi.Message) error)
type noOp struct{}
type errMsg struct{ err error }

type scale struct {
	name string
}

var scales = []list.Item{
	scale{name: "Major"},
	scale{name: "Minor"},
}

func (s scale) FilterValue() string { return s.name }
func (s scale) Title() string       { return s.name }
func (s scale) Description() string { return "" }

type model struct {
	send      (func(msg midi.Message) error)
	scaleList list.Model
}

func initialModel() model {
	scaleList := list.New(scales, list.NewDefaultDelegate(), 10, 50)
	scaleList.Title = "Scales"
	return model{
		send:      nil,
		scaleList: scaleList,
	}
}

func playNote(send func(msg midi.Message) error) tea.Cmd {
	return func() tea.Msg {
		send(midi.NoteOn(0, midi.C(3), 100))
		return noOp{}
	}
}

func connect() tea.Msg {
	// fmt.Println(midi.GetInPorts())
	// fmt.Println(midi.GetOutPorts())

	out, err := midi.FindOutPort("butt2")
	if err != nil {
		return errMsg{err}
	}

	send, err := midi.SendTo(out)
	if err != nil {
		return errMsg{err}
	}

	return connectedSuccess(send)
}

func (m model) Init() tea.Cmd {
	return connect
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case connectedSuccess:
		m.send = msg
		return m, nil

	case tea.KeyMsg:

		switch msg.String() {

		case "enter":
			if m.send != nil {
				return m, playNote(m.send)
			}
			return m, nil

		case "ctrl+c", "q":
			return m, tea.Quit
		}
	}

	// Return the updated model to the Bubble Tea runtime for processing.
	// Note that we're not returning a command.
	var cmd tea.Cmd
	m.scaleList, cmd = m.scaleList.Update(msg)
	return m, cmd
}

func (m model) View() string {
	//s := "\nPress q to quit.\n"

	// Send the UI for rendering
	return m.scaleList.View()
}

func main() {
	defer midi.CloseDriver()

	p := tea.NewProgram(initialModel())
	if err := p.Start(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
