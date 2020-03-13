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

	"github.com/yagoggame/gomaster/game/interfaces"
)

type isTurn struct {
	igt bool
	err error
}

// TestGamerBeginTurnSuccess checks the game with all gamers on the board.
// It should finish awaiting of turn for the one player rapidly
// and wait for a turn change for other.
func TestGamerBeginTurnSuccess(t *testing.T) {
	gamers := copyGamers(validGamers)
	game, err := NewGame(usualSize, usualKomi)
	if err != nil {
		t.Fatalf("Unexpected err on NewGame: err")
	}
	defer game.End()
	ctx, cancel := context.WithTimeout(context.Background(), rtDurationThreshold)
	defer cancel()

	chans := joinGamersWait(&commonArgs{ctx: ctx, t: t, game: game, gamers: gamers},
		waitGameTurnRoutine)

	checkWaitingTurnBreak(
		&commonArgs{t: t, gamers: gamers, dur: rtDurationThreshold, chans: chans},
		checkOneTurnByErr)
}

// TestGamerBeginTurnForeign checks that not joined gamer should fail rapidly
// on turn begin awaiting.
func TestGamerBeginTurnForeign(t *testing.T) {
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
	go waitTurnRoutine(&argWait)

	argCheck := checkWaitingNegativeParam{
		t:    t,
		ch:   ch,
		want: ErrUnknownID,
		dur:  rtDurationThreshold}
	checkWaitingNegative(&argCheck)
}

// TestGamerMakeTurnSuccess checks that game with all gamers on the board
// should finish awaiting of turn for the one player rapidly
// and wait for a turn change for other with success.
func TestGamerMakeTurnSuccess(t *testing.T) {
	gamers := copyGamers(validGamers)
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
	chans := joinGamersWait(&arg, waitGameTurnMakeRoutine)

	arg.dur = rtDurationThreshold
	arg.chans = chans
	checkWaitingTurnBreak(&arg, checkBothTurnByErr)
}

// TestIsMyTurn checks is IsMyTurn function working fine.
func TestIsMyTurn(t *testing.T) {
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

	// both players are joined, so in same goroutine - no sence to await.
	countTurns := 0
	for _, test := range funcErrTests {
		t.Run(test.caseName, func(t *testing.T) {
			igt, err := game.IsMyTurn(test.gamer.ID)
			if !errors.Is(err, test.want) {
				t.Errorf("Unexpected IsMyTurn err:\nwant: %v,\ngot: %v", test.want, err)
			}
			if igt {
				countTurns++
			}
		})
	}

	if countTurns != 1 {
		t.Errorf("Unexpected number of gamer with IsMyTurn=true:\nwant: 1,\ngot: %v", countTurns)
	}
}

var MakeTurnTests = []struct {
	caseName string
	move     *interfaces.TurnData
	want     error
}{
	{caseName: "wrong turn", move: &interfaces.TurnData{X: 0, Y: 1}, want: ErrWrongTurn},
	{caseName: "good turn", move: &interfaces.TurnData{X: 1, Y: 1}, want: nil},
	{caseName: "not your turn", move: &interfaces.TurnData{X: 1, Y: 1}, want: ErrNotYourTurn},
}

// TestMakeTurnFailures checks different errors during turn.
func TestMakeTurnFailures(t *testing.T) {
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

	// both players are joined, so in same goroutine - no sence to await.
	want := ErrUnknownID
	if err := game.MakeTurn(invalidGamer.ID, &interfaces.TurnData{X: 1, Y: 1}); !errors.Is(err, want) {
		t.Errorf("Unexpected MakeTurn err:\nwant: %v,\ngot: %v", want, err)
	}

	// typical errors.
	for _, g := range gamers {
		igt, err := game.IsMyTurn(g.ID)
		if err != nil {
			t.Fatalf("Unexpected failure on IsMyTurn: %s", err)
		}
		if igt == true {
			for _, test := range MakeTurnTests {
				t.Run(test.caseName, func(t *testing.T) {
					if err := game.MakeTurn(g.ID, test.move); !errors.Is(err, test.want) {
						t.Errorf("Unexpected MakeTurn err:\nwant: %v,\ngot: %v", test.want, err)
					}
				})
			}
			break
		}
	}
}
