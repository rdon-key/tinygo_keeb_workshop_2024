package main

import (
	"fmt"
	"machine"
	"machine/usb/adc/midi"
	"time"

	"tinygo.org/x/drivers/encoders"
)

// Try it easily by opening the following site in Chrome.
// https://www.onlinemusictools.com/kb/

const (
	cable    = 0
	channel  = 1
	velocity = 0x40
)

func main() {
	btn := machine.GPIO2
	btn.Configure(machine.PinConfig{Mode: machine.PinInputPullup})

	m := midi.Port()
	colPins := []machine.Pin{
		machine.GPIO5,
		machine.GPIO6,
		machine.GPIO7,
		machine.GPIO8,
	}

	rowPins := []machine.Pin{
		machine.GPIO9,
		machine.GPIO10,
		machine.GPIO11,
	}

	for _, c := range colPins {
		c.Configure(machine.PinConfig{Mode: machine.PinOutput})
		c.Low()
	}

	for _, c := range rowPins {
		c.Configure(machine.PinConfig{Mode: machine.PinInputPulldown})
	}

	notes := []midi.Note{
		midi.D5,
		midi.G4,
		midi.C4,

		midi.E5,
		midi.A4,
		midi.D4,

		midi.F5,
		midi.B4,
		midi.E4,

		midi.G5,
		midi.C5,
		midi.F4,
	}

	button := machine.GPIO2
	button.Configure(machine.PinConfig{Mode: machine.PinInputPullup})

	prev := true
	chords := []struct {
		name  string
		notes []midi.Note
	}{
		{name: "C ", notes: []midi.Note{midi.C4, midi.E4, midi.G4}},
		{name: "G ", notes: []midi.Note{midi.G3, midi.B3, midi.D4}},
		{name: "Am", notes: []midi.Note{midi.A3, midi.C4, midi.E4}},
		{name: "F ", notes: []midi.Note{midi.F3, midi.A3, midi.C4}},
	}
	index := 0

	machine.InitADC()

	ax := machine.ADC{Pin: machine.GPIO29}
	ax.Configure(machine.ADCConfig{})
	ay := machine.ADC{Pin: machine.GPIO28}
	ay.Configure(machine.ADCConfig{})

	enc := encoders.NewQuadratureViaInterrupt(
		machine.GPIO3,
		machine.GPIO4,
	)
	enc.Configure(encoders.QuadratureConfig{
		Precision: 4,
	})
	encOldValue := 0
	pg := uint8(0)

	for {
		if newValue := enc.Position(); newValue != encOldValue {
			fmt.Printf("%d %d\n", newValue, encOldValue)
			x := newValue
			for x < 0 {
				x += 128
			}
			pg = uint8(x % 128)
			m.ProgramChange(cable, channel, pg)
			encOldValue = newValue
		}

		{
			bend := ax.Get()
			if 0x7800 <= bend && bend <= 0x8200 {
				bend = 0x8000
			}
			bend = bend >> 2
			m.PitchBend(cable, channel, bend)
		}
		current := button.Get()
		if prev != current {
			if current {
				for _, note := range chords[index].notes {
					m.NoteOff(cable, channel, note, velocity)
				}
				index = (index + 1) % len(chords)
			} else {
				for _, note := range chords[index].notes {
					m.NoteOn(cable, channel, note, velocity)
				}
			}
			prev = current
		}

		for i, s := range getKeys(colPins, rowPins) {
			note := notes[i]
			switch s {
			case off2on:
				m.NoteOn(cable, channel, note, velocity)
				time.Sleep(1 * time.Millisecond)
			case on2off:
				m.NoteOff(cable, channel, note, velocity)
				time.Sleep(1 * time.Millisecond)
			}
		}
	}
}

var States [12]State

type State int8

const (
	off State = iota
	off2on
	off2on2
	off2on3
	off2on4
	off2onX
	on
	on2off
	on2off2
	on2off3
	on2off4
	on2offX
)

func getKeys(colPins, rowPins []machine.Pin) []State {
	// COL1
	colPins[0].High()
	colPins[1].Low()
	colPins[2].Low()
	colPins[3].Low()
	time.Sleep(1 * time.Millisecond)

	States[0] = updateState(States[0], rowPins[0].Get())
	States[1] = updateState(States[1], rowPins[1].Get())
	States[2] = updateState(States[2], rowPins[2].Get())

	// COL2
	colPins[0].Low()
	colPins[1].High()
	colPins[2].Low()
	colPins[3].Low()
	time.Sleep(1 * time.Millisecond)

	States[3] = updateState(States[3], rowPins[0].Get())
	States[4] = updateState(States[4], rowPins[1].Get())
	States[5] = updateState(States[5], rowPins[2].Get())

	// COL3
	colPins[0].Low()
	colPins[1].Low()
	colPins[2].High()
	colPins[3].Low()
	time.Sleep(1 * time.Millisecond)

	States[6] = updateState(States[6], rowPins[0].Get())
	States[7] = updateState(States[7], rowPins[1].Get())
	States[8] = updateState(States[8], rowPins[2].Get())

	// COL4
	colPins[0].Low()
	colPins[1].Low()
	colPins[2].Low()
	colPins[3].High()
	time.Sleep(1 * time.Millisecond)

	States[9] = updateState(States[9], rowPins[0].Get())
	States[10] = updateState(States[10], rowPins[1].Get())
	States[11] = updateState(States[11], rowPins[2].Get())

	return States[:]
}

func updateState(s State, btn bool) State {
	ret := s
	switch s {
	case off:
		if btn {
			ret = off2on
		}
	case on:
		if !btn {
			ret = on2off
		}
	case on2offX:
		ret = off
	default:
		ret = s + 1
	}
	return ret
}
