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

package game

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yagoggame/gomaster/game/interfaces"
)

var (
	fastDurationThreshold = time.Duration(10) * time.Second
	rtDurationThreshold   = time.Duration(100) * time.Millisecond
)

const (
	usualSize = 9
	usualKomi = 0.0
)

var validGamers = []*Gamer{
	&Gamer{Name: "Joe", ID: 1},
	&Gamer{Name: "Nick", ID: 2},
}

var invalidGamer = &Gamer{Name: "Buss", ID: 3}

var joinTests = []struct {
	caseName string
	gamer    *Gamer
	want     error
}{
	{caseName: "first", gamer: validGamers[0], want: nil},
	{caseName: "second", gamer: validGamers[1], want: nil},
	{caseName: "third", gamer: invalidGamer, want: ErrNoPlace},
}

var funcErrTests = []struct {
	caseName string
	gamer    *Gamer
	want     error
}{
	{caseName: "first", gamer: validGamers[0], want: nil},
	{caseName: "second", gamer: validGamers[1], want: nil},
	{caseName: "not joined", gamer: invalidGamer, want: ErrUnknownID},
}

var IsGameBeginTests = []struct {
	caseName string
	gamer    *Gamer
	want     error
	isBegin  bool
}{
	{caseName: "first", gamer: validGamers[0], want: nil, isBegin: false},
	{caseName: "second", gamer: validGamers[1], want: nil, isBegin: true},
	{caseName: "not joined", gamer: invalidGamer, want: ErrUnknownID, isBegin: false},
}

// TestCreation tests NewGame
func TestCreation(t *testing.T) {
	game, err := NewGame(usualSize, usualKomi)
	if err != nil {
		t.Fatalf("Unexpected err on NewGame: err")
	}
	if game == nil {
		t.Fatalf("Unexpected failure on NewGame: nil game created")
	}
	defer game.End()
}

// TestJoin tests joining of gamers to a game
func TestJoin(t *testing.T) {
	game, err := NewGame(usualSize, usualKomi)
	if err != nil {
		t.Fatalf("Unexpected err on NewGame: err")
	}
	defer game.End()

	for _, test := range joinTests {
		t.Run(test.caseName, func(t *testing.T) {
			err := game.Join(test.gamer)
			if !errors.Is(err, test.want) {
				t.Errorf("Unexpected Join err:\nwant: %v,\ngot: %v", test.want, err)
			}
		})
	}
}

// TestJoin tests joining of gamers to a game
func TestEnd(t *testing.T) {
	game, err := NewGame(usualSize, usualKomi)
	if err != nil {
		t.Fatalf("Unexpected err on NewGame: err")
	}
	defer game.End()

	for _, gamer := range validGamers {
		if err := game.Join(gamer); err != nil {
			t.Fatalf("Unexpected Join err: %v", err)
		}
	}

	stateChan := asyncGameEnd(game)

	// End function should be the pretty fast action
	// with closing of game the object as chanel.
	select {
	case ok := <-stateChan:
		if ok == true {
			t.Fatalf("Unexpected game.End() result:\nwant: closed GamersPool object as chanel,\ngot: chanel alive")
		}
	case <-time.After(fastDurationThreshold):
		t.Fatalf("Unexpected game.End():\nwant: return earler than %v duration,\ngot: return after %v duration", fastDurationThreshold, fastDurationThreshold)
	}
}

// TestGamerState tests GamerStatefunction.
func TestGamerState(t *testing.T) {
	gamers := copyGamers(validGamers)
	game, err := NewGame(usualSize, usualKomi)
	if err != nil {
		t.Fatalf("Unexpected err on NewGame: err")
	}
	defer game.End()

	arg := commonArgs{
		t:      t,
		game:   game,
		gamers: gamers}
	joinGamers(&arg)

	usedColours := make(map[interfaces.ChipColour]bool)
	for _, test := range funcErrTests {
		t.Run(test.caseName, func(t *testing.T) {
			gs, err := game.GamerState(test.gamer.ID)
			if !errors.Is(err, test.want) {
				t.Errorf("Unexpected GamerState err:\nwant: %v,\ngot: %v", test.want, err)
			}
			if gs.Colour != interfaces.NoColour {
				usedColours[gs.Colour] = true
			}
		})
	}

	if len(usedColours) != 2 {
		t.Errorf("Unexpected GamerState: not all ChipColour assigned to joined players")
	}
}

// TestIsGameBegin verifies is IsGameBegin working fine.
func TestIsGameBegin(t *testing.T) {
	game, err := NewGame(usualSize, usualKomi)
	if err != nil {
		t.Fatalf("Unexpected err on NewGame: err")
	}
	defer game.End()

	for _, test := range IsGameBeginTests {
		if err := game.Join(test.gamer); err != nil && test.want == nil {
			t.Errorf("Unexpected Join err: %v", err)
		}

		t.Run(test.caseName, func(t *testing.T) {
			igb, err := game.IsGameBegun(test.gamer.ID)
			if !errors.Is(err, test.want) {
				t.Errorf("Unexpected IsGameBegin err:\nwant: %v,\ngot: %v", test.want, err)
			}
			if igb != test.isBegin {
				t.Errorf("Unexpected IsGameBegin value:\nwant: %t,\ngot: %t", test.isBegin, igb)
			}
		})
	}
}

// TestGamerBeginSuccess tests game with all gamers on the board.
// It should finish awaiting rapidly
func TestGamerBeginSuccess(t *testing.T) {
	game, err := NewGame(usualSize, usualKomi)
	if err != nil {
		t.Fatalf("Unexpected err on NewGame: err")
	}
	defer game.End()
	ctx, cancel := context.WithTimeout(context.Background(), rtDurationThreshold)
	defer cancel()
	gamers := copyGamers(validGamers)

	arg := commonArgs{
		ctx:    ctx,
		t:      t,
		game:   game,
		gamers: gamers}
	chans := joinGamersWait(&arg, waitGameRoutine)

	arg.chans = chans
	arg.dur = rtDurationThreshold
	checkWaitingPositive(&arg)
}

// TestGamerBeginFailure tests game with missing gamer.
// It should hang untill second player join and return error on cancellation
func TestGamerBeginFailure(t *testing.T) {
	gamers := copyGamers(validGamers)[:1]
	game, err := NewGame(usualSize, usualKomi)
	if err != nil {
		t.Fatalf("Unexpected err on NewGame: err")
	}
	defer game.End()
	ctx, cancel := context.WithTimeout(context.Background(), rtDurationThreshold)
	defer cancel()

	arg := commonArgs{
		ctx:    ctx,
		t:      t,
		game:   game,
		gamers: gamers}
	chans := joinGamersWait(&arg, waitGameRoutine)

	argCheck := checkWaitingNegativeParam{
		t:    t,
		ch:   chans[0],
		want: ErrCancellation,
		dur:  rtDurationThreshold}
	checkWaitingNegative(&argCheck)
}

// TestGamerBeginForeign checks that not joined gamer
// fails rapidly on game begin awaiting
func TestGamerBeginForeign(t *testing.T) {
	gamers := copyGamers(validGamers)[:1]
	game, err := NewGame(usualSize, usualKomi)
	if err != nil {
		t.Fatalf("Unexpected err on NewGame: err")
	}
	defer game.End()
	ctx, cancel := context.WithTimeout(context.Background(), rtDurationThreshold)
	defer cancel()

	arg := commonArgs{
		t:      t,
		game:   game,
		gamers: gamers}
	joinGamers(&arg)

	ch := make(chan error)
	argWait := waitGameRoutineParam{
		ctx:   ctx,
		game:  game,
		gamer: invalidGamer,
		ch:    ch}
	go waitGameRoutine(&argWait)

	argCheck := checkWaitingNegativeParam{
		t:    t,
		ch:   ch,
		want: ErrUnknownID,
		dur:  rtDurationThreshold}
	checkWaitingNegative(&argCheck)
}
