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

package interfaces

// ChipColour provides datatype of chip's colours
type ChipColour int

// Set of chip's colours
const (
	NoColour ChipColour = 0
	Black               = 1
	White               = 2
)

// TurnData is a struct, using to put a gamer's turn data
type TurnData struct {
	X, Y int
}

// FieldState describes the game state on the field
type FieldState struct {
	GameOver           bool
	ChipsInCup         map[ChipColour]int
	ChipsCuptured      map[ChipColour]int
	PointsUnderControl map[ChipColour][]*TurnData
	Komi               float64
	Scores             map[ChipColour]float64
	ChipsOnBoard       map[ChipColour][]*TurnData
}

// Master interface wraps functions to work with game field and it's state
type Master interface {
	Move(colour ChipColour, td *TurnData) error
	Size() int
	State() *FieldState
}
