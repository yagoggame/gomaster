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
)

var leaveTests = []struct {
	caseName string
	gamer    *Gamer
	want     error
}{
	{caseName: "first", gamer: validGamers[0], want: nil},
	{caseName: "not joined", gamer: invalidGamer, want: ErrUnknownID},
	{caseName: "second", gamer: validGamers[1], want: nil},
}

// TestLeave tests the leaving of game procedure.
func TestLeave(t *testing.T) {
	game := NewGame()
	gamers := copyGamers(validGamers)

	arg := commonArgs{
		t:      t,
		game:   game,
		gamers: gamers}
	joinGamers(&arg)

	for _, test := range leaveTests {
		if err := game.Leave(test.gamer.ID); !errors.Is(err, test.want) {
			t.Errorf("Unexpected Leave  err for gamer %s:\nwant: %v\ngot: %v", test.gamer, test.want, err)
		}
	}
}

// TestBeginSuccess checks if game with all gamers on the board
// finishes awaiting rapidly on leave.
func TestBeginLeave(t *testing.T) {
	gamers := copyGamers(validGamers)[:1]
	game := NewGame()
	ctx, cancel := context.WithTimeout(context.Background(), rtDurationThreshold)
	defer cancel()

	arg := commonArgs{
		ctx:    ctx,
		t:      t,
		game:   game,
		gamers: gamers}
	chans := joinGamersWait(&arg, waitGameRoutine)

	time.Sleep(rtDurationThreshold / 2)
	if err := game.Leave(gamers[0].ID); err != nil {
		t.Fatalf("Unexpected Leave err: %v", err)
	}

	checkWaitingBreak(t, chans[0], rtDurationThreshold)
}

// TestLeaveEnd tests game.End after game leave
// and closing of Game object as chanel.
func TestLeaveEnd(t *testing.T) {
	gamers := copyGamers(validGamers)[:1]
	game := NewGame()

	defer func() {
		want := ErrResourceNotAvailable
		if err := game.End(); !errors.Is(err, want) {
			t.Errorf("Unexpected End err:\nwant: %v,\ngot: %v", want, err)
		}
	}()

	arg := commonArgs{
		t:      t,
		game:   game,
		gamers: gamers}
	joinGamers(&arg)

	gamers[0].SetGame(game)
	if err := game.Leave(gamers[0].ID); err != nil {
		t.Fatalf("Unexpectef Leave err: %v", err)
	}
}

// TestLeaveBeginTurn tests game with all gamers on the board
// should finish awaiting of turn with error
// if Game object is closed as chanel.
func TestLeaveBeginTurn(t *testing.T) {
	gamers := copyGamers(validGamers)
	game := NewGame()
	defer game.End()

	ctx, cancel := context.WithTimeout(context.Background(), rtDurationThreshold)
	defer cancel()

	arg := commonArgs{
		ctx:    ctx,
		t:      t,
		game:   game,
		gamers: gamers}
	chans := joinGamersWait(&arg, waitTurnRoutine)

	arg.dur = rtDurationThreshold
	arg.chans = chans

	killActiveGamer(&arg)

	checkWaitingTurnBreak(&arg, checkOneTurnByErr)
}

// TestLeaveGameOver tests game over error returning
// by some functions after game is over.
func TestLeaveGameOver(t *testing.T) {
	gamers := copyGamers(validGamers)
	game := NewGame()
	defer game.End()

	arg := commonArgs{
		t:      t,
		game:   game,
		gamers: gamers}
	joinGamers(&arg)

	if err := game.Leave(gamers[0].ID); err != nil {
		t.Errorf("Unexpected Leave error for gamer %s.\ngot: %s", gamers[0], err)
	}

	testFunctionsGameover(&arg, invalidGamer)
}

// TestLeaveGameOver tests game over error returning
// by some waiting functions after game is over.
func TestLeaveGameOverWaits(t *testing.T) {
	gamers := copyGamers(validGamers)
	game := NewGame()
	defer game.End()

	arg := commonArgs{
		t:      t,
		game:   game,
		gamers: gamers}
	joinGamers(&arg)

	if err := game.Leave(gamers[0].ID); err != nil {
		t.Errorf("Unexpected Leave error for gamer %s.\ngot: %s", gamers[0], err)
	}

	// after one gamer disjoint, waiters should rapidly return an error "Game Over"
	ctx, cancel := context.WithTimeout(context.Background(), rtDurationThreshold)
	defer cancel()

	ch := make(chan error)
	argWait := waitGameRoutineParam{
		ctx:   ctx,
		game:  game,
		gamer: gamers[1],
		ch:    ch}
	argCheck := checkWaitingNegativeParam{
		t:    t,
		ch:   ch,
		want: ErrGameOver,
		dur:  rtDurationThreshold}

	go waitGameRoutine(&argWait)
	checkWaitingNegative(&argCheck)

	ch = make(chan error)
	argWait.ch = ch
	argCheck.ch = ch

	go waitTurnRoutine(&argWait)
	checkWaitingNegative(&argCheck)
}
