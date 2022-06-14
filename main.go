package main

import (
	"fmt"
	"os"

	list "github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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
	send         (func(msg midi.Message) error)
	scaleList    list.Model
	selectedRoot SelectedRoot
}

func initialModel() model {

	var items []list.Item = make([]list.Item, len(scales))
	for i, d := range scales {
		items[i] = d
	}

	scaleList := list.New(items, list.NewDefaultDelegate(), 50, 50)
	scaleList.Title = "Scales"
	scaleList.SetFilteringEnabled(false)
	scaleList.SetShowTitle(false)
	scaleList.SetShowStatusBar(false)
	scaleList.SetShowPagination(false)
	return model{
		send:         nil,
		scaleList:    scaleList,
		selectedRoot: C,
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

type SelectedRoot uint8

const (
	C  SelectedRoot = 0
	Cs              = 1
	D               = 2
	Ds              = 3
	E               = 4
	F               = 5
	Fs              = 6
	G               = 7
	Gs              = 8
	A               = 9
	As              = 10
	B               = 11
)

func selectedRootName(r SelectedRoot) string {
	switch {
	case r == C:
		return "C"
	case r == Cs:
		return "C#/Db"
	case r == D:
		return "D"
	case r == Ds:
		return "D#/Eb"
	case r == E:
		return "E"
	case r == F:
		return "F"
	case r == Fs:
		return "F#/Gb"
	case r == G:
		return "G"
	case r == Gs:
		return "G#/Ab"
	case r == A:
		return "A"
	case r == As:
		return "A#/Bb"
	case r == B:
		return "B"
	}

	return ""

}

func prevRoot(r SelectedRoot) SelectedRoot {
	switch {
	case r == C:
		return B
	case r == Cs:
		return C
	case r == D:
		return Cs
	case r == Ds:
		return D
	case r == E:
		return Ds
	case r == F:
		return E
	case r == Fs:
		return F
	case r == G:
		return Fs
	case r == Gs:
		return G
	case r == A:
		return Gs
	case r == As:
		return A
	case r == B:
		return As
	}
	return r
}
func nextRoot(r SelectedRoot) SelectedRoot {
	switch {
	case r == C:
		return Cs
	case r == Cs:
		return D
	case r == D:
		return Ds
	case r == Ds:
		return E
	case r == E:
		return F
	case r == F:
		return Fs
	case r == Fs:
		return G
	case r == G:
		return Gs
	case r == Gs:
		return A
	case r == A:
		return As
	case r == As:
		return B
	case r == B:
		return C
	}
	return r
}

func notesForScale(selectedRoot SelectedRoot, scale scale) map[uint8]Illumination {
	note := uint8(36) // C2
	root := uint8(selectedRoot)
	m := make(map[uint8]Illumination)
	for note < 100 {
		if note%12 == root {
			m[note] = Root
		} else if contains(scale.steps, (note-root)%12) {
			m[note] = Member
		} else {
			m[note] = Blank
		}

		note = note + 1
	}
	return m
}

func sendMidiData(model model) tea.Cmd {
	return func() tea.Msg {
		var selectedScale scale = model.scaleList.SelectedItem().(scale)
		noteMap := notesForScale(model.selectedRoot, selectedScale)
		for note, illumination := range noteMap {
			model.send(midi.NoteOn(midiChannel, note, illuminationMap(illumination)))
		}
		return noOp{}
	}
}

func connect() tea.Msg {
	// fmt.Println(midi.GetInPorts())
	// fmt.Println(midi.GetOutPorts())

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
				return m, sendMidiData(m)
			}
			return m, nil

		case "h", "left":
			m.selectedRoot = prevRoot(m.selectedRoot)
			return m, nil

		case "l", "right":
			m.selectedRoot = nextRoot(m.selectedRoot)
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

var selectedRootStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("#FAFAFA")).
	Background(lipgloss.Color("#7D56F4")).
	Padding(0, 1).
	MarginLeft(2).
	MarginBottom(2)

var rootHeaderStyle = lipgloss.NewStyle().
	Bold(true)

func (m model) View() string {
	rootTitle := rootHeaderStyle.Render("Root: ")
	rootName := selectedRootStyle.Render(selectedRootName(m.selectedRoot))
	// s := "\nPress q to quit.\n"

	// Send the UI for rendering
	return rootTitle + rootName + "\n" + m.scaleList.View()
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
