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
	name  string
	steps []uint8
}

var scales = []scale{
	scale{name: "Major", steps: []uint8{0, 2, 4, 5, 7, 9, 11}},
	scale{name: "Natural Minor", steps: []uint8{0, 2, 3, 5, 7, 8, 10}},
}

var midiDeviceName = "Midi Fighter 3D"
var midiChannel uint8 = 2 // 3, but 0 indexed

func (s scale) FilterValue() string { return s.name }
func (s scale) Title() string       { return s.name }
func (s scale) Description() string { return "" }

type model struct {
	send      (func(msg midi.Message) error)
	scaleList list.Model
}

func initialModel() model {

	var items []list.Item = make([]list.Item, len(scales))
	for i, d := range scales {
		items[i] = d
	}

	scaleList := list.New(items, list.NewDefaultDelegate(), 10, 50)
	scaleList.Title = "Scales"
	return model{
		send:      nil,
		scaleList: scaleList,
	}
}

type Illumination uint8

const (
	Root   Illumination = 0
	Member              = 1
	Blank               = 2
)

func illuminationMap(illumination Illumination) uint8 {
	switch {
	case illumination == Root:
		return 91 // purple
	case illumination == Member:
		return 55 // green
	}

	return 1
}

func notesForScale(root uint8, scale scale) map[uint8]Illumination {
	note := uint8(36) // C2
	rootDiff := root % 12
	m := make(map[uint8]Illumination)
	for note < 100 {
		if note%12 == rootDiff {
			m[note] = Root
		} else if contains(scale.steps, (note-rootDiff)%12) {
			m[note] = Member
		} else {
			m[note] = Blank
		}

		note = note + 1
	}
	return m
}

func playNote(send func(msg midi.Message) error) tea.Cmd {
	return func() tea.Msg {
		noteMap := notesForScale(midi.C(2), scales[0])
		for note, illumination := range noteMap {
			print(note)
			print("-")
			println(illuminationMap(illumination))
			send(midi.NoteOn(midiChannel, note, illuminationMap(illumination)))
		}
		return noOp{}
	}
}

func connect() tea.Msg {
	fmt.Println(midi.GetInPorts())
	fmt.Println(midi.GetOutPorts())

	out, err := midi.FindOutPort(midiDeviceName)
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

		case " ", "enter":
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
	s := "\nPress q to quit.\n"

	// Send the UI for rendering
	return s //m.scaleList.View()
}

func main() {
	defer midi.CloseDriver()

	p := tea.NewProgram(initialModel())
	if err := p.Start(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}

func contains(set []uint8, key uint8) bool {
	for _, x := range set {
		if x == key {
			return true
		}
	}
	return false
}
