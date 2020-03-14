// Copyright ©2020 BlinnikovAA. All rights reserved.
// This file is part of yagogame.
//
// yagogame is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// yagogame is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with yagogame.  If not, see <https://www.gnu.org/licenses/>.

package field

import (
	"errors"
	"fmt"

	"github.com/yagoggame/gomaster/game/igame"
)

var (
	// ErrFieldSize error occures when New is called with wrong size
	ErrFieldSize = errors.New("field size is out of range (from 1x1 to 19x19)")
	// ErrColour error occurs when some of operations is made with No Colour
	ErrColour = errors.New("only black and white chips allowed")
	// ErrPosition error occurs when Move is made with TurnData out of range
	ErrPosition = errors.New("move position is out of range")
	// ErrOccupied error occurs when Move is made on occupied position
	ErrOccupied = errors.New("the position is occupied")
	// ErrNoChips error occurs when there are no chips left
	ErrNoChips = errors.New("no chips left")
	// ErrGameOver error occurs when attempt operation on game wich is over
	ErrGameOver = errors.New("the game is over")
)

const (
	whiteMax = 180
	blackMax = 181
	minSize  = 1
	maxSize  = 19
)

// Field holds position of gamers on the game desk
type Field struct {
	field       [][]igame.ChipColour
	size        int
	komi        float64
	chipsNumber map[igame.ChipColour]int
}

// New generate Field with demensions of size x size
func New(size int, komi float64) (*Field, error) {
	if size < minSize || size > maxSize {
		return nil, fmt.Errorf("%w: desired sise is %[2]dx%[2]d", ErrFieldSize, size)
	}

	field := &Field{
		size:  size,
		komi:  komi,
		field: make([][]igame.ChipColour, size),
		chipsNumber: map[igame.ChipColour]int{
			igame.Black: blackMax,
			igame.White: whiteMax,
		},
	}
	for i := range field.field {
		field.field[i] = make([]igame.ChipColour, size)
	}
	return field, nil
}

// Size returns field's size
func (field *Field) Size() int {
	return field.size
}

// Move performs move with attempt to put chip of colour to position td
func (field *Field) Move(colour igame.ChipColour, td *igame.TurnData) error {
	if err := field.precheck(colour, td); err != nil {
		return err
	}
	if err := field.checkPosition(td); err != nil {
		return err
	}

	field.chipsNumber[colour] = field.chipsNumber[colour] - 1
	field.field[td.Y-1][td.X-1] = colour
	return nil
}

// State calculate full state description
func (field *Field) State() *igame.FieldState {
	state := &igame.FieldState{
		ChipsInCup:         make(map[igame.ChipColour]int, 2),
		ChipsCuptured:      make(map[igame.ChipColour]int, 2),
		PointsUnderControl: make(map[igame.ChipColour][]*igame.TurnData, 2),
		Scores:             make(map[igame.ChipColour]float64, 2),
		ChipsOnBoard:       make(map[igame.ChipColour][]*igame.TurnData, 2),
	}

	colours := []igame.ChipColour{igame.White, igame.Black}
	initialNumber := map[igame.ChipColour]int{
		igame.White: whiteMax,
		igame.Black: blackMax,
	}

	for _, colour := range colours {
		state.ChipsInCup[colour] = field.chipsNumber[colour]
		state.ChipsOnBoard[colour] = field.getChipsOnBoard(colour)
		state.ChipsCuptured[colour] = initialNumber[colour] - state.ChipsInCup[colour] - len(state.ChipsOnBoard[colour])
		state.PointsUnderControl[colour] = field.pointsUnderControl(colour)
		state.Scores[colour] = float64(state.ChipsCuptured[colour] + len(state.PointsUnderControl[colour]))
	}
	state.Scores[igame.White] = state.Scores[igame.White] + state.Komi
	state.GameOver = field.isGameOver()

	return state
}

func (field *Field) isGameOver() bool {
	colours := []igame.ChipColour{igame.White, igame.Black}
	for _, colour := range colours {
		if field.chipsNumber[colour] < 1 {
			return true
		}
	}
	// TODO: calculate additional critetria
	return false
}

func (field *Field) pointsUnderControl(colour igame.ChipColour) []*igame.TurnData {
	positions := make([]*igame.TurnData, 0)
	// TODO: calculate points under colour control
	return positions
}

func (field *Field) getChipsOnBoard(colour igame.ChipColour) []*igame.TurnData {
	positions := make([]*igame.TurnData, 0)

	for x := 0; x < field.Size(); x++ {
		for y := 0; y < field.Size(); y++ {
			td := &igame.TurnData{X: x + 1, Y: y + 1}
			if field.field[td.Y-1][td.X-1] == colour {
				positions = append(positions, td)
			}
		}
	}

	return positions
}

func (field *Field) precheck(colour igame.ChipColour, td *igame.TurnData) error {
	if colour != igame.Black && colour != igame.White {
		return fmt.Errorf("%w: got colour: %v", ErrColour, colour)
	}

	if td.X < 1 || td.Y < 1 || td.X > field.size || td.Y > field.size {
		return fmt.Errorf("%w: got turn data: %v", ErrPosition, td)
	}
	if field.isGameOver() {
		return fmt.Errorf("%w: colour: %v", ErrGameOver, colour)
	}

	return nil
}

func (field *Field) checkPosition(td *igame.TurnData) error {
	if field.field[td.Y-1][td.X-1] != igame.NoColour {
		return fmt.Errorf("%w: at %d", ErrOccupied, td)
	}
	return nil
}
