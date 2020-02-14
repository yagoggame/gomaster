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

//TestCreateJoinEnd performs test of basic creation, fill and end of game.
func TestCreateJoinEnd(t *testing.T) {
	type testCase struct {
		caseName string
		gamer    *Gamer
		req      string
		success  bool
	}
	tt := []testCase{
		testCase{caseName: "first", gamer: &Gamer{Name: "Joe", Id: 1}, req: "", success: true},
		testCase{caseName: "second", gamer: &Gamer{Name: "Nick", Id: 2}, req: "", success: true},
		testCase{caseName: "third", gamer: &Gamer{Name: "Buss", Id: 3}, req: "no vacant place in Game", success: false},
	}

	game := NewGame()
	if game == nil {
		t.Fatalf("failed on NewGame: nil game created")
	}

	for _, tc := range tt {
		t.Run(tc.caseName, func(t *testing.T) {
			err := game.Join(tc.gamer)
			switch {
			case tc.success == false && err == nil:
				t.Errorf("supposed %q error, got: %v", tc.req, err)
			case tc.success == true && err != nil:
				t.Errorf("supposed nil error, got: %q", err)
			case tc.success == false && err != nil && strings.Compare(err.Error(), tc.req) != 0:
				t.Errorf("supposed nil error, got: %q", err)
			}
		})
	}

	//End function should be the pretty fast action with closing of game object.
	dur := time.Duration(10) * time.Second
	c := make(chan interface{})
	go func(c chan<- interface{}) {
		game.End()
		_, ok := <-game
		c <- ok
		close(c)
	}(c)

	select {
	case ok := <-c:
		if ok == true {
			t.Fatalf("It was expected that game.End() will shut down Game object as chanel, but it's still alive")
		}
	case <-time.After(dur):
		t.Fatalf("It was expected that game.End() will return earler than %v duration", dur)
	}
}

// TestGamerState performs request of gamer's state.
func TestGamerState(t *testing.T) {
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

	//get state of foreign gamer should fail.
	fg := &Gamer{Name: "Dick", Id: 3}
	req := fmt.Sprintf("not joined gamer %s tries to get his state in the game", fg)
	if gs, err := game.GamerState(fg); err == nil || strings.Compare(err.Error(), req) != 0 || gs.Colour != NoColour {
		t.Errorf("supposed %q error and GamerState.Colour==NoColour, got: err: %v, gs: %v", req, err, gs)
	}

	//joined gamers shoul succeed.
	usedColours := make(map[ChipColour]bool)
	for _, g := range gamers {
		gs, err := game.GamerState(g)
		switch {
		case err != nil:
			t.Errorf("failed to obtain gamer's %s state: %s", g, err)
		case gs.Colour == NoColour:
			t.Errorf("Joined player %s with no clour assigned: gs.Colour = %d", g, gs.Colour)
		case gs.Colour != NoColour:
			usedColours[gs.Colour] = true
		}
	}
	if len(usedColours) != 2 {
		t.Errorf("Not all ChipColour assigned to joined players")
	}
}

//TestIsGameBegin verifies is IsGameBegin working fine.
func TestIsGameBegin(t *testing.T) {

	gamers := []*Gamer{
		&Gamer{Name: "Joe", Id: 1},
		&Gamer{Name: "Nick", Id: 2},
	}
	fg := &Gamer{Name: "Max", Id: 3}

	game := NewGame()
	defer game.End()

	for i, g := range gamers {
		if err := game.Join(g); err != nil {
			t.Fatalf("failed to join gamer %s to a game %v: %q", g, game, err)
		}
		switch i {
		case 0:
			if igb, err := game.IsGameBegun(g); err != nil || igb == true {
				t.Fatalf("i: %d - supposed err=nil, igb=false, got:err=\"%v\",igb=%t", i, err, igb)
			}
		case 1:
			if igb, err := game.IsGameBegun(g); err != nil || igb == false {
				t.Fatalf("i: %d - supposed err=nil, igb=false, got:err=\"%v\",igb=%t", i, err, igb)
			}
		}
	}

	req := fmt.Sprintf("not joined gamer %s tries to ask: is game begun", fg)
	if igb, err := game.IsGameBegun(fg); err == nil || (err != nil && strings.Compare(err.Error(), req) != 0) {
		t.Fatalf("foreign gamer %s. supposed err=nil, igb=false, got:err=\"%v\",igb=%t", fg, err, igb)
	}
}

// TestGamerBeginSuccess tests game with all gamers on the board. 
// It should finish awaiting rapidly
func TestGamerBeginSuccess(t *testing.T) {
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

	// wait game shoul finish awaiting rapidly, when all players are joined.
	for i, g := range gamers {
		if err := game.Join(g); err != nil {
			t.Fatalf("failed to join gamer %s to a game %v: %q", g, game, err)
		}
		g.InGame = game
		chans[i] = make(chan error)

		go waitGameRoutine(ctx, game, g, chans[i])
	}

	for i := 0; i < len(chans); i++ {
		select {
		case err, ok := <-chans[0]:
			if (err == nil && ok) || (err != nil && !ok) {
				t.Errorf("err: %v vs ok: %v missmatch", err, ok)
			}
			chans[0] = nil
			if ok {
				t.Errorf("failed to WaitBegin for gamer %s to a game %v: %s", gamers[0], game, err)
			}
		case err, ok := <-chans[1]:
			if (err == nil && ok) || (err != nil && !ok) {
				t.Errorf("err: %v vs ok: %v missmatch", err, ok)
			}
			chans[1] = nil
			if ok {
				t.Errorf("failed to WaitBegin for gamer %s to a game %v: %s", gamers[1], game, err)
			}
		case <-time.After(2 * dur):
			t.Fatalf("cancellation failed")
		}
	}
}

// TestGamerBeginFailure tests game with missing gamer. 
// It should hang untill second player join and return error on cancellation
func TestGamerBeginFailure(t *testing.T) {
	gamer := &Gamer{Name: "Joe", Id: 1}

	game := NewGame()
	defer game.End()

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

	req := fmt.Sprintf("Cancelled")
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

// TestGamerBeginForeign checks that not joined gamer 
// fails rapidly on game begin awaiting
func TestGamerBeginForeign(t *testing.T) {
	gamer := &Gamer{Name: "Joe", Id: 1}

	game := NewGame()
	defer game.End()

	dur := time.Duration(100) * time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), dur)
	defer cancel()

	ch := make(chan error)

	// wait game should finish awaiting rapidly, when all players are joined
	if err := game.Join(gamer); err != nil {
		t.Fatalf("failed to join gamer %s to a game %v: %q", gamer, game, err)
	}
	gamer.InGame = game
	fg := &Gamer{Name: "Nick", Id: 2}
	go waitGameRoutine(ctx, game, fg, ch)

	req := fmt.Sprintf("not joined gamer %s tries to await of game begin", fg)
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

// waitGameRoutine waits of game the begin for specified gamer.
func waitGameRoutine(ctx context.Context, game Game, gamer *Gamer, ch chan<- error) {
	defer close(ch)
	err := game.WaitBegin(ctx, gamer)
	if err != nil {
		ch <- err
	}
}
