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

// TestGamerLeave tests the leaving of game procedure.
func TestGamerLeave(t *testing.T) {
	gamers := []*Gamer{
		&Gamer{Name: "Joe", Id: 1},
		&Gamer{Name: "Nick", Id: 2},
	}
	fg := &Gamer{Name: "Bamblbee", Id: 3}

	game := NewGame()
	// 	defer game.End()

	// not joined gamer should fail
	want := UnknownIdError
	if err := game.Leave(fg.Id); !errors.Is(err, want) {
		t.Errorf("Leave for gamer %s:\nwant: %v\ngot: %v", fg, want, err)
	}

	for _, g := range gamers {
		if err := game.Join(g); err != nil {
			t.Fatalf("failed to join gamer %s to a game %v: %q", g, game, err)
		}
		g.InGame = game
	}

	for _, g := range gamers {
		if err := game.Leave(g.Id); err != nil {
			t.Errorf("Leave for gamer %s should succeed.\ngot: %s", g, err)
		}
	}

}

// TestGamerBeginSuccess checks if game with all gamers on the board
// finishes awaiting rapidly on leave.
func TestGamerBeginLeave(t *testing.T) {
	gamer := &Gamer{Name: "Joe", Id: 1}

	game := NewGame()
	// 	defer game.End()

	dur := time.Duration(100) * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), dur)
	defer cancel()

	ch := make(chan error)

	// wait game should finish awaiting rapidly, when all players are joined
	if err := game.Join(gamer); err != nil {
		t.Fatalf("failed to join gamer %s to a game %v: %q", gamer, game, err)
	}
	gamer.InGame = game
	go waitGameRoutine(ctx, game, gamer, ch)

	// if one of gamers has left the game - awaiting of game begin should break
	time.Sleep(dur / 2)
	if err := game.Leave(gamer.Id); err != nil {
		t.Errorf("Leave for gamer %s should succeed.\ngot: %s", gamer, err)
	}

	want1 := OtherGamerLeftError
	want2 := ResourceNotAvailable
	select {
	case err, ok := <-ch:
		ch = nil
		if !ok || (!errors.Is(err, want1) && !errors.Is(err, want2)) {
			t.Errorf("WaitBegin:\nwant: %v\nor: %v\ngot: %v", want1, want2, err)
		}
	case <-time.After(2 * dur):
		t.Fatalf("cancellation failed")
	}
}

// TestGamerLeaveEnd tests: game.End after game leave
// and closing of Game object as chanel.
func TestGamerLeaveEnd(t *testing.T) {
	gamer := &Gamer{Name: "Joe", Id: 1}

	game := NewGame()
	defer func() {
		want := ResourceNotAvailable
		if err := game.End(); !errors.Is(err, want) {
			t.Errorf("End:\nwant: %v,\ngot: %v", want, err)
		}
	}()

	// wait game should finish awaiting rapidly, when all players are joined
	if err := game.Join(gamer); err != nil {
		t.Fatalf("failed to join gamer %s to a game %v: %v", gamer, game, err)
	}
	gamer.InGame = game
	if err := game.Leave(gamer.Id); err != nil {
		t.Fatalf("Leave for gamer %s should succeed.\ngot: %v", gamer, err)
	}
}

// TestGamerLeaveBeginTurn tests game with all gamers on the board
// should finish awaiting of turn with error
// if Game object is closed as chanel.
func TestGamerLeaveBeginTurn(t *testing.T) {
	gamers := []*Gamer{
		&Gamer{Name: "Joe", Id: 1},
		&Gamer{Name: "Nick", Id: 2},
	}

	game := NewGame()
	defer game.End()

	for _, g := range gamers {
		if err := game.Join(g); err != nil {
			t.Fatalf("failed to join gamer %s to a game %v: %v", g, game, err)
		}
		g.InGame = game
	}

	// at this point all gamers has been joined, it is possible to wait a turn.
	dur := time.Duration(100) * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), dur)
	defer cancel()

	chans := make([]chan error, len(gamers))

	for i, g := range gamers {
		chans[i] = make(chan error)
		go waitTurnRoutine(ctx, game, g, chans[i])
	}

	// if gamer who is turn leaving - other should catch an error of awaiting.
	time.Sleep(dur / 2)
	for i, g := range gamers {
		igt, err := game.IsMyTurn(gamers[i].Id)
		if err != nil {
			t.Fatalf("failed to inspect: is it gamer's %s turn: %s", g, err)
		}
		if igt == true {
			if err := game.Leave(g.Id); err != nil {
				t.Fatalf("Leave for gamer %s should succeed.\ngot: %s", g, err)
			}
			break
		}
	}

	errs := make([]error, len(gamers))

	//check the result.
	for i := 0; i < len(chans); i++ {
		select {
		case err, ok := <-chans[0]:
			if (err == nil && ok) || (err != nil && !ok) {
				t.Fatalf("err: %v vs ok: %v missmatch", err, ok)
			}
			chans[0] = nil
			errs[0] = err
		case err, ok := <-chans[1]:
			if (err == nil && ok) || (err != nil && !ok) {
				t.Fatalf("err: %v vs ok: %v missmatch", err, ok)
			}
			chans[1] = nil
			errs[1] = err
		case <-time.After(2 * dur):
			t.Fatalf("cancellation failed")
		}
	}

	if errs[0] != nil && errs[1] != nil {
		t.Errorf("one of gamers should be assigned as \"his turn\",\ngot: \"%v\",\n\"%v\"", errs[0], errs[1])
	}

	if errs[0] == nil && errs[1] == nil {
		t.Errorf("one of gamers should be in awaiting condition and canceled by context,\ngot: \"%v\",\n\"%v\"", errs[0], errs[1])
	}

}

// TestGamerLeaveGameOver tests game over error returning
// by some functions after game is over.
func TestGamerLeaveGameOver(t *testing.T) {
	gamers := []*Gamer{
		&Gamer{Name: "Joe", Id: 1},
		&Gamer{Name: "Nick", Id: 2},
	}

	game := NewGame()
	defer game.End()

	for _, g := range gamers {
		if err := game.Join(g); err != nil {
			t.Fatalf("failed to join gamer %s to a game %v: %q", g, game, err)
		}
		g.InGame = game
	}

	if err := game.Leave(gamers[0].Id); err != nil {
		t.Errorf("Leave for gamer %s should succeed.\ngot: %s", gamers[0], err)
	}

	looser := &Gamer{Name: "Looser", Id: 3}
	want := GameOverError
	if err := game.Join(looser); !errors.Is(err, want) {
		t.Errorf("Join:\nwant: %v,\ngot: %v", want, err)
	}

	if _, err := game.IsGameBegun(gamers[1].Id); !errors.Is(err, want) {
		t.Errorf("IsGameBegun:\nwant: %v,\ngot: %v", want, err)
	}

	if _, err := game.IsMyTurn(gamers[1].Id); !errors.Is(err, want) {
		t.Errorf("IsMyTurn:\nwant: %v,\ngot: %v", want, err)
	}

	// user that is not disjoined yet - can access to the game data.
	if _, err := game.GamerState(gamers[1].Id); err != nil {
		t.Errorf("GamerState:\nwant: nil,\ngot: %v", err)
	}

	if err := game.MakeTurn(gamers[1].Id, &TurnData{X: 1, Y: 1}); !errors.Is(err, want) {
		t.Errorf("IsMyTurn:\nwant: %v,\ngot: %v", want, err)
	}

}

// TestGamerLeaveGameOver tests game over error returning
// by some waiting functions after game is over.
func TestGamerLeaveGameOverWaits(t *testing.T) {
	gamers := []*Gamer{
		&Gamer{Name: "Joe", Id: 1},
		&Gamer{Name: "Nick", Id: 2},
	}

	game := NewGame()
	defer game.End()

	for _, g := range gamers {
		if err := game.Join(g); err != nil {
			t.Fatalf("failed to join gamer %s to a game %v: %q", g, game, err)
		}
		g.InGame = game
	}

	if err := game.Leave(gamers[0].Id); err != nil {
		t.Errorf("Leave for gamer %s should succeed.\ngot: %s", gamers[0], err)
	}

	// now we can perform any waiting, wich should rapidly return an error "Game Over"
	// at this point all gamers has been joined, it is possible to wait a turn.
	dur := time.Duration(100) * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), dur)
	defer cancel()

	//wait of game should fail
	want := GameOverError
	ch := make(chan error)
	go waitGameRoutine(ctx, game, gamers[1], ch)

	select {
	case err, ok := <-ch:
		ch = nil
		if !ok || !errors.Is(err, want) {
			t.Errorf("supposed result of WaitBegin: %q , got: %v", want, err)
		}
	case <-time.After(2 * dur):
		t.Fatalf("cancellation failed")
	}

	// wait of turn should fail.
	ch = make(chan error)
	go waitTurnRoutine(ctx, game, gamers[1], ch)

	select {
	case err, ok := <-ch:
		ch = nil
		if !ok || !errors.Is(err, want) {
			t.Errorf("supposed result of WaitBegin: %q , got: %v", want, err)
		}
	case <-time.After(2 * dur):
		t.Fatalf("cancellation failed")
	}
}
