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

// TestGamerLeave - test the leaving of game procedure
func TestGamerLeave(t *testing.T) {
	gamers := []*Gamer{
		&Gamer{Name: "Joe", Id: 1},
		&Gamer{Name: "Nick", Id: 2},
	}
	fg := &Gamer{Name: "Bamblbee", Id: 3}

	game := NewGame()
	// 	defer game.End()

	// not joined gamer should fail
	req := fmt.Sprintf("not joined gamer %s tries to leave the game", fg)
	if err := game.Leave(fg); err == nil || (err != nil && strings.Compare(err.Error(), req) != 0) {
		t.Errorf("Leave for gamer %s should succeed. got: %s", fg, err)
	}

	for _, g := range gamers {
		if err := game.Join(g); err != nil {
			t.Fatalf("failed to join gamer %s to a game %v: %q", g, game, err)
		}
		g.InGame = game
	}

	for _, g := range gamers {
		if err := game.Leave(g); err != nil {
			t.Errorf("Leave for gamer %s should succeed. got: %s", g, err)
		}
	}

}

// TestGamerBeginSuccess - test: game with all gamers on the board should finish awaiting rapidly
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

	//if one of gamers has left the game - awaiting of game begin should break
	time.Sleep(dur / 2)
	if err := game.Leave(gamer); err != nil {
		t.Errorf("Leave for gamer %s should succeed. got: %s", gamer, err)
	}

	req1 := "other player left the game"
	req2 := "send on closed channel"
	select {
	case err, ok := <-ch:
		ch = nil
		if !ok || (ok && strings.Compare(err.Error(), req1) != 0 && strings.Compare(err.Error(), req2) != 0) {
			t.Errorf("supposed result of WaitBegin: %q or %q, got: %v", req1, req2, err)
		}
	case <-time.After(2 * dur):
		t.Fatalf("cancellation failed")
	}
}

// TestGamerLeaveEnd - test: game.End after game leave and closing of Game object as chanel
func TestGamerLeaveEnd(t *testing.T) {
	gamer := &Gamer{Name: "Joe", Id: 1}

	game := NewGame()
	defer func() {
		req := "send on closed channel"
		if err := game.End(); err == nil || (err != nil && strings.Compare(err.Error(), req) != 0) {
			t.Errorf("supposed result of End: %q , got: %v", req, err)
		}
	}()

	// wait game should finish awaiting rapidly, when all players are joined
	if err := game.Join(gamer); err != nil {
		t.Fatalf("failed to join gamer %s to a game %v: %q", gamer, game, err)
	}
	gamer.InGame = game
	if err := game.Leave(gamer); err != nil {
		t.Fatalf("Leave for gamer %s should succeed. got: %s", gamer, err)
	}
}

// TestGamerLeaveBeginTurn - test: game with all gamers on the board should finish awaiting of turn with error
// if Game object is closed as
func TestGamerLeaveBeginTurn(t *testing.T) {
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

	//at this point all gamers has been joined, it is possible to wait a turn
	dur := time.Duration(100) * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), dur)
	defer cancel()

	chans := make([]chan error, len(gamers))

	for i, g := range gamers {
		chans[i] = make(chan error)
		go waitTurnRoutine(ctx, game, g, chans[i])
	}

	//if gamer ho is turn leaving - other should catch an error of awaiting
	time.Sleep(dur / 2)
	for i, g := range gamers {
		igt, err := game.IsMyTurn(gamers[i])
		if err != nil {
			t.Fatalf("failed to inspect: is it gamer's %s turn: %s", g, err)
		}
		if igt == true {
			if err := game.Leave(g); err != nil {
				t.Fatalf("Leave for gamer %s should succeed. got: %s", g, err)
			}
			break
		}
	}

	errs := make([]error, len(gamers))

	//check the result
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

// TestGamerLeaveGameOver - tests game over error returning by some functions after game is over
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

	if err := game.Leave(gamers[0]); err != nil {
		t.Errorf("Leave for gamer %s should succeed. got: %s", gamers[0], err)
	}

	looser := &Gamer{Name: "Looser", Id: 3}
	req := "Game Over"
	if err := game.Join(looser); err == nil || (err != nil && strings.Compare(err.Error(), req) != 0) {
		t.Errorf("supposed result of Join: %q , got: %v", req, err)
	}

	if _, err := game.IsGameBegun(gamers[1]); err == nil || (err != nil && strings.Compare(err.Error(), req) != 0) {
		t.Errorf("supposed result of IsGameBegun: %q , got: %v", req, err)
	}

	if _, err := game.IsMyTurn(gamers[1]); err == nil || (err != nil && strings.Compare(err.Error(), req) != 0) {
		t.Errorf("supposed result of IsMyTurn: %q , got: %v", req, err)
	}

	// user that is not disjoined yet - can access to the game data.
	if _, err := game.GamerState(gamers[1]); err != nil {
		t.Errorf("supposed result of GamerState: nil , got: %v", err)
	}

	if err := game.MakeTurn(gamers[1], &TurnData{X: 1, Y: 1}); err == nil || (err != nil && strings.Compare(err.Error(), req) != 0) {
		t.Errorf("supposed result of IsMyTurn: %q , got: %v", req, err)
	}

}

// TestGamerLeaveGameOver - tests game over error returning by some functions after game is over
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

	if err := game.Leave(gamers[0]); err != nil {
		t.Errorf("Leave for gamer %s should succeed. got: %s", gamers[0], err)
	}

	//now we can perform any waiting, wich should rapidly return an error "Game Over"
	//at this point all gamers has been joined, it is possible to wait a turn
	dur := time.Duration(100) * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), dur)
	defer cancel()

	//wait of game should fail
	req := "Game Over"
	ch := make(chan error)
	go waitGameRoutine(ctx, game, gamers[1], ch)

	select {
	case err, ok := <-ch:
		ch = nil
		if !ok || (ok && strings.Compare(err.Error(), req) != 0) {
			t.Errorf("supposed result of WaitBegin: %q , got: %v", req, err)
		}
	case <-time.After(2 * dur):
		t.Fatalf("cancellation failed")
	}

	//wait of turn should fail
	ch = make(chan error)
	go waitTurnRoutine(ctx, game, gamers[1], ch)

	select {
	case err, ok := <-ch:
		ch = nil
		if !ok || (ok && strings.Compare(err.Error(), req) != 0) {
			t.Errorf("supposed result of WaitBegin: %q , got: %v", req, err)
		}
	case <-time.After(2 * dur):
		t.Fatalf("cancellation failed")
	}
}
