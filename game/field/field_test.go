// Copyright ©2020 BlinnikovAA. All rights reserved.
// This file is part of yagointerfaces.
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
// along with yagointerfaces.  If not, see <https://www.gnu.org/licenses/>.

package field_test

import (
	"errors"
	"testing"

	. "github.com/yagoggame/gomaster/game/field"
	"github.com/yagoggame/gomaster/game/interfaces"
)

const (
	usualSize   = 9
	maxSize     = 19
	maxWhite    = 180
	maxBlack    = 181
	defaultKomi = 0.0
)

var (
	newTests = []struct {
		name string
		size int
		want error
	}{
		{
			name: "zero size",
			size: 0,
			want: ErrFieldSize,
		},
		{
			name: "20 size",
			size: 20,
			want: ErrFieldSize,
		},
		{
			name: "9 size",
			size: 9,
			want: nil,
		},
	}

	moveTests = []struct {
		name   string
		move   *interfaces.TurnData
		colour interfaces.ChipColour
		want   error
	}{
		{
			name:   "no colour",
			move:   &interfaces.TurnData{X: 1, Y: 1},
			colour: interfaces.NoColour,
			want:   ErrColour,
		},
		{
			name:   "white x is 0",
			move:   &interfaces.TurnData{X: 0, Y: 1},
			colour: interfaces.White,
			want:   ErrPosition,
		},
		{
			name:   "black x is size+1",
			move:   &interfaces.TurnData{X: usualSize + 1, Y: 1},
			colour: interfaces.Black,
			want:   ErrPosition,
		},
		{
			name:   "black y is 0",
			move:   &interfaces.TurnData{X: 1, Y: 0},
			colour: interfaces.Black,
			want:   ErrPosition,
		},
		{
			name:   "white y is size+1",
			move:   &interfaces.TurnData{X: 1, Y: usualSize + 1},
			colour: interfaces.White,
			want:   ErrPosition,
		},
		{
			name:   "black ok",
			move:   &interfaces.TurnData{X: 1, Y: 1},
			colour: interfaces.Black,
			want:   nil,
		},
		{
			name:   "white ok",
			move:   &interfaces.TurnData{X: 2, Y: 1},
			colour: interfaces.White,
			want:   nil,
		},
		{
			name:   "occupied",
			move:   &interfaces.TurnData{X: 1, Y: 1},
			colour: interfaces.White,
			want:   ErrOccupied,
		},
	}
)

func TestNew(t *testing.T) {
	for _, test := range newTests {
		t.Run(test.name, func(t *testing.T) {
			field, err := New(test.size, defaultKomi)
			var ifield interfaces.Master = field

			if !errors.Is(err, test.want) {
				t.Errorf("Unexpected New err:\nwant: %v,\ngot: %v.", test.want, err)
			}

			if (err == nil) == (field == nil) {
				t.Errorf("Unexpected err and field ==nil or !=nil simultaniously, got err: %v, got field: %v.", err, field)
			}

			if err == nil && test.size != ifield.Size() {
				t.Errorf("Unexpected Size err:\nwant: %d,\ngot: %d.", test.size, ifield.Size())
			}

			if err == nil {
				state := ifield.State()
				wl := state.ChipsInCup[interfaces.White]
				bl := state.ChipsInCup[interfaces.Black]
				if wl != maxWhite || bl != maxBlack {
					t.Errorf("Unexpected number of chips:\nwant: black:%d, white: %d,\ngot: black:%d, white: %d.",
						wl, maxWhite, bl, maxBlack)
				}
			}
		})
	}
}

func TestMove(t *testing.T) {
	var field interfaces.Master
	field, err := New(usualSize, defaultKomi)
	if err != nil {
		t.Fatalf("Unexpected New() error: %v", err)
	}

	for _, test := range moveTests {
		t.Run(test.name, func(t *testing.T) {
			state := field.State()
			preCount := state.ChipsInCup[test.colour]
			err := field.Move(test.colour, test.move)
			state = field.State()
			postCount := state.ChipsInCup[test.colour]

			if !errors.Is(err, test.want) {
				t.Errorf("Unexpected Move() err:\nwant: %v,\ngot: %v.", test.want, err)
			}

			if err == nil && postCount != preCount-1 {
				t.Errorf("Unexpected number of %v colour chips after Move():\nwant: %d,\ngot: %d.", test.colour, preCount-1, postCount)
			}
			if err != nil && postCount != preCount {
				t.Errorf("Unexpected number of %v colour chips after Move():\nwant: %d,\ngot: %d.", test.colour, preCount, postCount)
			}
		})
	}
}

func TestNoWhiteChips(t *testing.T) {
	var colour interfaces.ChipColour = interfaces.White
	var field interfaces.Master
	field, err := New(maxSize, defaultKomi)
	if err != nil {
		t.Fatalf("Unexpected New() error: %v", err)
	}

	var counter int
	for x := 0; x < 19; x++ {
		for y := 0; y < 19; y++ {
			err := field.Move(colour, &interfaces.TurnData{X: x + 1, Y: y + 1})
			if err != nil && !errors.Is(err, ErrGameOver) {
				t.Fatalf("Unexpected Move() err: %v", err)
			}

			if err == nil {
				counter++
			}

			state := field.State()
			chLeft := state.ChipsInCup[colour]
			if chLeft != maxWhite-counter {
				t.Fatalf("Unexpected state.ChipsInCup[%v] on move number %d: %v", colour, counter, chLeft)
			}

			if err != nil {
				break
			}
		}
	}
}

func TestNoBlackChips(t *testing.T) {
	var colour interfaces.ChipColour = interfaces.Black
	var field interfaces.Master
	field, err := New(maxSize, defaultKomi)
	if err != nil {
		t.Fatalf("Unexpected New() error: %v", err)
	}

	var counter int
	for x := 0; x < 19; x++ {
		for y := 0; y < 19; y++ {
			err := field.Move(colour, &interfaces.TurnData{X: x + 1, Y: y + 1})
			if err != nil && !errors.Is(err, ErrGameOver) {
				t.Fatalf("Unexpected Move() err: %v", err)
			}

			if err == nil {
				counter++
			}

			state := field.State()
			chLeft := state.ChipsInCup[colour]
			if chLeft != maxBlack-counter {
				t.Fatalf("Unexpected state.ChipsInCup[%v] on move number %d: %v", colour, counter, chLeft)
			}

			if err != nil {
				break
			}
		}
	}
}
