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

type waitGameRoutineParam struct {
	ctx   context.Context
	game  Game
	gamer *Gamer
	ch    chan<- error
}

type checkWaitingNegativeParam struct {
	t    *testing.T
	ch   chan error
	want error
	dur  time.Duration
}

type commonArgs struct {
	ctx    context.Context
	t      *testing.T
	game   Game
	gamers []*Gamer
	chans  []chan error
	dur    time.Duration
}

func asyncGameEnd(game Game) (signal <-chan interface{}) {
	c := make(chan interface{})

	go func(c chan<- interface{}) {
		game.End()
		_, ok := <-game
		c <- ok
		close(c)
	}(c)

	return c
}

func copyGamers(src []*Gamer) (dst []*Gamer) {
	if src == nil {
		return nil
	}
	dst = make([]*Gamer, len(src))
	for i := range src {
		gcpy := *src[i]
		dst[i] = &gcpy
	}
	return dst
}

// waitGameRoutine waits of game the begin for specified gamer.
func waitGameRoutine(p *waitGameRoutineParam) {
	defer close(p.ch)
	err := p.game.WaitBegin(p.ctx, p.gamer.ID)
	if err != nil {
		p.ch <- err
	}
}

// waitGameTurnRoutine runs awaiting of the game, and of a turn for given gamer.
func waitGameTurnRoutine(p *waitGameRoutineParam) {
	defer close(p.ch)
	err := p.game.WaitBegin(p.ctx, p.gamer.ID)
	if err != nil {
		p.ch <- err
	}

	err = p.game.WaitTurn(p.ctx, p.gamer.ID)
	if err != nil {
		p.ch <- err
	}
}

// waitTurnRoutine awaits of gamer's turn.
func waitTurnRoutine(p *waitGameRoutineParam) {
	defer close(p.ch)
	err := p.game.WaitTurn(p.ctx, p.gamer.ID)
	if err != nil {
		p.ch <- err
	}
}

// waitGameTurnMakeRoutine runs a game and then a turn awaiting.
// after all - perform correct test, to provide turn change.
func waitGameTurnMakeRoutine(p *waitGameRoutineParam) {
	defer close(p.ch)
	err := p.game.WaitBegin(p.ctx, p.gamer.ID)
	if err != nil {
		p.ch <- err
	}

	err = p.game.WaitTurn(p.ctx, p.gamer.ID)
	if err != nil {
		p.ch <- err
		return
	}
	p.game.MakeTurn(p.gamer.ID, &TurnData{X: 1, Y: 1})
}

func joinGamers(par *commonArgs) {
	for _, g := range par.gamers {
		if err := par.game.Join(g); err != nil {
			par.t.Fatalf("Unexpected Join err: %v", err)
		}
		g.SetGame(par.game)
	}
}

func joinGamersWait(par *commonArgs, fnc func(*waitGameRoutineParam)) (chans []chan error) {
	chans = make([]chan error, len(par.gamers))
	arg := waitGameRoutineParam{
		ctx:  par.ctx,
		game: par.game,
	}

	for i, g := range par.gamers {
		if err := par.game.Join(g); err != nil {
			par.t.Fatalf("Unexpected Join err: %v", err)
		}
		g.SetGame(par.game)

		chans[i] = make(chan error)
		arg := arg
		arg.gamer = g
		arg.ch = chans[i]
		go fnc(&arg)
	}

	return chans
}

func checkWaitingPositive(par *commonArgs) {
	for i := 0; i < len(par.chans); i++ {
		select {
		case err, ok := <-par.chans[0]:
			checkWaitingChanel(par.t, par.gamers[0], err, ok)
			par.chans[0] = nil
		case err, ok := <-par.chans[1]:
			checkWaitingChanel(par.t, par.gamers[1], err, ok)
			par.chans[1] = nil
		// wait game should finish awaiting rapidly, when all players are joined.
		case <-time.After(2 * par.dur):
			par.t.Fatalf("Unexpected cancellation failure")
		}
	}
}

func checkWaitingChanel(t *testing.T, gamer *Gamer, err error, ok bool) {
	if (err == nil && ok) || (err != nil && !ok) {
		t.Errorf("Unexpected err vs=%v ok=%t missmatch", err, ok)
	}
	if ok {
		t.Errorf("Unexpected fail on WaitBegin for gamer %s to a game : %v", gamer, err)
	}
}

func checkWaitingNegative(par *checkWaitingNegativeParam) {
	select {
	case err, ok := <-par.ch:
		par.ch = nil
		if !ok || !errors.Is(err, par.want) {
			par.t.Errorf("Unexpected WaitBegin err:\nwant: %v,\ngot: %v", par.want, err)
		}
	case <-time.After(2 * par.dur):
		par.t.Fatalf("Unexpected cancellation failure")
	}
}

func checkWaitingBreak(t *testing.T, ch chan error, dur time.Duration) {
	want1 := ErrOtherGamerLeft
	want2 := ErrResourceNotAvailable

	select {
	case err, ok := <-ch:
		ch = nil
		if !ok || (!errors.Is(err, want1) && !errors.Is(err, want2)) {
			t.Errorf("Unexpected WaitBegin err:\nwant: %v or %v,\ngot: %v", want1, want2, err)
		}
	case <-time.After(2 * dur):
		t.Fatalf("Unexpected cancellation failure")
	}
}

func killActiveGamer(par *commonArgs) {
	time.Sleep(par.dur / 2)
	for i, g := range par.gamers {
		igt, err := par.game.IsMyTurn(par.gamers[i].ID)
		if err != nil {
			par.t.Fatalf("Unexpected IsMyTurn err for gamer %s:\ngot: %v", g, err)
		}
		if igt == true {
			if err := par.game.Leave(g.ID); err != nil {
				par.t.Fatalf("Unexpected Leave err for gamer %s:\ngot: %v", g, err)
			}
			break
		}
	}
}

func checkWaitingTurnBreak(par *commonArgs, errsRelationChecker func(t *testing.T, errs []error)) {
	errs := make([]error, len(par.gamers))
	for i := 0; i < len(par.chans); i++ {
		select {
		case err, ok := <-par.chans[0]:
			checkWaitingTurnErrOkRelation(par.t, err, ok)
			par.chans[0] = nil
			errs[0] = err
		case err, ok := <-par.chans[1]:
			checkWaitingTurnErrOkRelation(par.t, err, ok)
			par.chans[1] = nil
			errs[1] = err
		case <-time.After(2 * par.dur):
			par.t.Fatalf("Unexpected cancellation failure")
		}
	}

	errsRelationChecker(par.t, errs)
}

func checkWaitingTurnErrOkRelation(t *testing.T, err error, ok bool) {
	if (err == nil && ok) || (err != nil && !ok) {
		t.Fatalf("Unexpected err: %v vs ok: %v missmatch", err, ok)
	}
}

func checkOneTurnByErr(t *testing.T, errs []error) {
	if errs[0] != nil && errs[1] != nil {
		t.Errorf("Expected one of the gamers assigned as \"his turn\",\ngot: \"%v\",\n\"%v\"", errs[0], errs[1])
	}

	if errs[0] == nil && errs[1] == nil {
		t.Errorf("Expected one of the gamers in awaiting condition and canceled by context,\ngot: \"%v\",\n\"%v\"", errs[0], errs[1])
	}
}

func checkBothTurnByErr(t *testing.T, errs []error) {
	if errs[0] != nil && errs[1] != nil {
		t.Errorf("Expected both of the  gamers assigned as \"his turn\", sequentially.\ngot: \"%v\"\nand: \"%v\"", errs[0], errs[1])
	}
}

func testFunctionsGameover(par *commonArgs, extraGamer *Gamer) {
	want := ErrGameOver

	if err := par.game.Join(extraGamer); !errors.Is(err, want) {
		par.t.Errorf("unexpected Join err:\nwant: %v,\ngot: %v", want, err)
	}

	if err := par.game.MakeTurn(par.gamers[1].ID, &TurnData{X: 1, Y: 1}); !errors.Is(err, want) {
		par.t.Errorf("unexpected IsMyTurn err:\nwant: %v,\ngot: %v", want, err)
	}

	if _, err := par.game.IsGameBegun(par.gamers[1].ID); !errors.Is(err, want) {
		par.t.Errorf("unexpected IsGameBegun err:\nwant: %v,\ngot: %v", want, err)
	}

	if _, err := par.game.IsMyTurn(par.gamers[1].ID); !errors.Is(err, want) {
		par.t.Errorf("unexpected IsMyTurn err:\nwant: %v,\ngot: %v", want, err)
	}

	// user that is not disjoined yet - can access to the game data.
	if _, err := par.game.GamerState(par.gamers[1].ID); err != nil {
		par.t.Errorf("unexpected GamerState err:\nwant: nil,\ngot: %v", err)
	}
}
