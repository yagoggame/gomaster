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
	"fmt"
	"strings"
	"testing"
	"time"
)

// TestGamerBeginTurnSuccess checks the game with all gamers on the board.
// It should finish awaiting of turn for the one player rapidly
// and wait for a turn change for other.
func TestGamerBeginTurnSuccess(t *testing.T) {
	gamers := []*Gamer{
		&Gamer{Name: "Joe", Id: 1},
		&Gamer{Name: "Nick", Id: 2},
	}

	game := NewGame()
	defer game.End()

	dur := time.Duration(100) * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), dur)
	defer cancel()

	chans := make([]chan error, len(gamers))

	for i, g := range gamers {
		if err := game.Join(g); err != nil {
			t.Fatalf("failed to join gamer %s to a game %v: %q", g, game, err)
		}
		g.InGame = game
		chans[i] = make(chan error)

		go waitGameTurnRoutine(ctx, game, g, chans[i])
	}

	errs := make([]error, len(gamers))

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
		t.Errorf("one of gamers should be assigned as \"his turn\", got: \"%v\" \"%v\"", errs[0], errs[1])
	}

	if errs[0] == nil && errs[1] == nil {
		t.Errorf("one of gamers should be in awaiting condition and canceled by context, got: \"%v\" \"%v\"", errs[0], errs[1])
	}
}

// TestGamerBeginTurnForeign checks that not joined gamer should fail rapidly on game begin awaiting.
func TestGamerBeginTurnForeign(t *testing.T) {
	gamer := &Gamer{Name: "Joe", Id: 1}

	game := NewGame()
	defer game.End()

	dur := time.Duration(100) * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), dur)
	defer cancel()

	ch := make(chan error)

	// wait game should finish awaiting rapidly, when all players are joined.
	if err := game.Join(gamer); err != nil {
		t.Fatalf("failed to join gamer %s to a game %v: %q", gamer, game, err)
	}
	gamer.InGame = game
	fg := &Gamer{Name: "Nick", Id: 2}
	go waitTurnRoutine(ctx, game, fg, ch)

	req := fmt.Sprintf("not joined gamer %s tries to await of his turn", fg)
	select {
	case err, ok := <-ch:
		ch = nil
		if !ok || (ok && strings.Compare(err.Error(), req) != 0) {
			t.Errorf("supposed result of WaitBegin: %q, got: %s", req, err)
		}
	case <-time.After(2 * dur):
		t.Fatalf("cancellation failed")
	}
}

// TestGamerMakeTurnSuccess checks that game with all gamers on the board 
// should finish awaiting of turn for the one player rapidly
// and wait for a turn change for other with success.
func TestGamerMakeTurnSuccess(t *testing.T) {
	gamers := []*Gamer{
		&Gamer{Name: "Joe", Id: 1},
		&Gamer{Name: "Nick", Id: 2},
	}

	game := NewGame()
	defer game.End()

	dur := time.Duration(100) * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), dur)
	defer cancel()

	chans := make([]chan error, len(gamers))

	for i, g := range gamers {
		if err := game.Join(g); err != nil {
			t.Fatalf("failed to join gamer %s to a game %v: %q", g, game, err)
		}
		g.InGame = game
		chans[i] = make(chan error)

		go waitGameTurnMakeRoutine(ctx, game, g, chans[i])
	}

	errs := make([]error, len(gamers))

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
		t.Errorf("both of the  gamers should be assigned as \"his turn\", sequentially. got: \"%v\" \"%v\"", errs[0], errs[1])
	}
}

// TestIsMyTurn checks is IsMyTurn function working fine.
func TestIsMyTurn(t *testing.T) {
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
		//both players are joined, so in same goroutine - no sence to await.
	}

	type isTurn struct {
		igt bool
		err error
	}
	descs := make([]isTurn, len(gamers))

	// excep for the foreign gamers.
	fg := &Gamer{Name: "Sir", Id: 3}
	req := fmt.Sprintf("not joined gamer %s tries to ask: is it his turn", fg)
	if igt, err := game.IsMyTurn(fg.Id); igt == true || err == nil || (err != nil && strings.Compare(err.Error(), req) != 0) {
		t.Errorf("succed to inspect foreign gamer's %s turn: %q", fg, err)
	}

	// is now gamer's turn should performe without errors.
	for i, g := range gamers {
		descs[i].igt, descs[i].err = game.IsMyTurn(gamers[i].Id)
		if descs[i].err != nil {
			t.Fatalf("failed to inspect: is it gamer's %s turn: %s", g, descs[i].err)
		}
	}
	// and it should be different.
	if descs[0].igt == descs[1].igt {
		t.Fatalf("is turn state of joined gamers must differs but got: %t %t", descs[0].igt, descs[1].igt)
	}
}

// TestMakeTurnFailures checks different errors during turn.
func TestMakeTurnFailures(t *testing.T) {
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
		// both players are joined, so in same goroutine - no sence to await.
	}

	// excep for the foreign gamers.
	fg := &Gamer{Name: "Sir", Id: 3}
	req := fmt.Sprintf("not joined gamer %s tries to make a turn", fg)
	if err := game.MakeTurn(fg.Id, &TurnData{X: 1, Y: 1}); err == nil || (err != nil && strings.Compare(err.Error(), req) != 0) {
		t.Errorf("succed to inspect foreign gamer's %s turn: %q", fg, err)
	}

	// other tipical errors.
	for _, g := range gamers {
		igt, err := game.IsMyTurn(g.Id)
		if err != nil {
			t.Fatalf("failed to IsMyTurn: %s", err)
		}
		if igt == true {
			// make wrong turn.
			req = "wrong turn"
			if err := game.MakeTurn(g.Id, &TurnData{X: 0, Y: 1}); err == nil || (err != nil && strings.HasPrefix(err.Error(), req) == false) {
				t.Errorf("gamer %s succed to perform wrong turn: %q", g, err)
			}
			// make goof turn
			if err := game.MakeTurn(g.Id, &TurnData{X: 1, Y: 1}); err != nil {
				t.Fatalf("gamer %s failed to perform good turn: %q", g, err)
			}
			// after good turn - turne moved to other gamer, so next good turn must fail.
			req = fmt.Sprintf("not a gamer's %s turn", g)
			if err := game.MakeTurn(g.Id, &TurnData{X: 1, Y: 1}); err == nil || (err != nil && strings.Compare(err.Error(), req) != 0) {
				t.Errorf("gamer %s succed to perform wrong turn: %q", g, err)
			}

			break
		}
	}
}

// waitGameTurnRoutine runs awaiting of the game, and of a turn for given gamer.
func waitGameTurnRoutine(ctx context.Context, game Game, gamer *Gamer, ch chan<- error) {
	defer close(ch)
	err := game.WaitBegin(ctx, gamer.Id)
	if err != nil {
		ch <- err
	}

	err = game.WaitTurn(ctx, gamer.Id)
	if err != nil {
		ch <- err
	}
}

// waitTurnRoutine awaits of gamer's turn.
func waitTurnRoutine(ctx context.Context, game Game, gamer *Gamer, ch chan<- error) {
	defer close(ch)
	err := game.WaitTurn(ctx, gamer.Id)
	if err != nil {
		ch <- err
	}
}

// waitGameTurnMakeRoutine runs a game and then a turn awaiting.
// after all - perform correct test, to provide turn change.
func waitGameTurnMakeRoutine(ctx context.Context, game Game, gamer *Gamer, ch chan<- error) {
	defer close(ch)
	err := game.WaitBegin(ctx, gamer.Id)
	if err != nil {
		ch <- err
	}

	err = game.WaitTurn(ctx, gamer.Id)
	if err != nil {
		ch <- err
		return
	}
	game.MakeTurn(gamer.Id, &TurnData{X: 1, Y: 1})
}
