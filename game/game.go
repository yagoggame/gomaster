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

// Package game provides thread safe game entity and some data structures to maintain it
package game

import (
	"context"
	"errors"
	"fmt"

	"github.com/yagoggame/gomaster/game/field"
	"github.com/yagoggame/gomaster/game/igame"
)

var (
	// ErrUnknownTypeReturned is an error of unknown type
	// returned by concurrency safe operation with Game object
	ErrUnknownTypeReturned = errors.New("unknown type of value returned")
	// ErrCancellation is an error of cancelation by client
	ErrCancellation = errors.New("action cancelled")
	// ErrNoPlace is an error of joining to the game with no space left
	ErrNoPlace = errors.New("no vacant place in the game")
	// ErrGameOver is an error of operation with Game that is over
	// (it is possible only to get some statuses)
	ErrGameOver = errors.New("the game is over")
	// ErrUnknownID is an error of operation with game by gamer with unregistred ID
	ErrUnknownID = errors.New("gamer with inknown id")
	// ErrNotYourTurn is an error of making a move while it is other gamer's turn
	ErrNotYourTurn = errors.New("not a gamer's turn")
	// ErrWrongTurn is an error of providing an error data for turn
	ErrWrongTurn = errors.New("wrong turn")
	// ErrOtherGamerLeft is an error of operation tha demand of other gamer
	// when he is already left
	ErrOtherGamerLeft = errors.New("other gamer left the game")
	// ErrGameDestroyed is an error of performing any operation on Game object
	// when it is closed as chanel
	ErrGameDestroyed = errors.New("the game is destroyed")
	// ErrResourceNotAvailable is an error of performing any whaing operation
	// when the game is over
	ErrResourceNotAvailable = errors.New("send on closed channel")
)

// Game is a datatype based on chanel, to provide a thread safe game entity.
type Game chan *gameCommand

// Queries on actions

// End releases game resources and closes a Game object as chanel.
// Use this function only to abort, if creation failed.
// Normaly - Leave invocation for all users has the same consequences.
// If the End() invoked after this - an error will be returned.
func (g Game) End() (err error) {
	// gamer leaving can close the Game object as chanel,
	// it could cause a panic in other goroutines. process it.
	defer recoverAsErr(&err)

	c := make(chan interface{})
	g <- &gameCommand{act: endCMD, rez: c}
	<-c
	return nil
}

// Join tries to join gamer to this Game.
func (g Game) Join(gamer *Gamer) (err error) {
	// gamer leaving can close the Game object as chanel,
	// it could cause a panic in other goroutines. process it.
	defer recoverAsErr(&err)

	c := make(chan interface{})
	g <- &gameCommand{act: joinCMD, gamer: gamer, rez: c}

	if err := <-c; err != nil {
		return err.(error)
	}
	return nil
}

// GamerState returns a copy of Internal State of a gamer
// (to prevent a manual changing).
func (g Game) GamerState(id int) (state *GamerState, err error) {
	// gamer leaving can close the Game object as chanel,
	// it could cause a panic in other goroutines. process it.
	defer recoverAsErr(&err)

	c := make(chan interface{})
	g <- &gameCommand{act: gamerStateCMD, id: id, rez: c}
	rez := <-c

	switch rez := rez.(type) {
	case error:
		return &GamerState{}, rez
	case *GamerState:
		return rez, nil
	}

	return &GamerState{}, fmt.Errorf("returned value %v of Type %T: %w", rez, rez, ErrUnknownTypeReturned)

}

// FieldSize returns a size of game's field.
func (g Game) FieldSize(id int) (size int, err error) {
	// gamer leaving can close the Game object as chanel,
	// it could cause a panic in other goroutines. process it.
	defer recoverAsErr(&err)

	c := make(chan interface{})
	g <- &gameCommand{act: gameFieldSize, id: id, rez: c}
	rez := <-c

	switch rez := rez.(type) {
	case error:
		return 0, rez
	case int:
		return rez, nil
	}

	return 0, fmt.Errorf("returned value %v of Type %T: %w", rez, rez, ErrUnknownTypeReturned)

}

// GameState returns a structure with full description of game situation.
func (g Game) GameState(id int) (state *igame.FieldState, err error) {
	// gamer leaving can close the Game object as chanel,
	// it could cause a panic in other goroutines. process it.
	defer recoverAsErr(&err)

	c := make(chan interface{})
	g <- &gameCommand{act: gameStateCMD, id: id, rez: c}
	rez := <-c

	switch rez := rez.(type) {
	case error:
		return nil, rez
	case *igame.FieldState:
		return rez, nil
	}

	return nil, fmt.Errorf("returned value %v of Type %T: %w", rez, rez, ErrUnknownTypeReturned)

}

// WaitBegin waits for game begin.
// If gamer identified by id started this game
// - awaiting another person.
func (g Game) WaitBegin(ctx context.Context, id int) (err error) {
	// gamer leaving can close the Game object as chanel,
	// it could cause a panic in other goroutines. process it.
	defer recoverAsErr(&err)

	//buffered because when killed by cancelation - internal mechanism can block other invocation on attemption to write to this chanel later
	c := make(chan interface{}, 1)
	g <- &gameCommand{act: wBeginCMD, id: id, rez: c}
	select {
	case err := <-c:
		if err, ok := err.(error); ok == true {
			return err
		}
	case <-ctx.Done():
		return ErrCancellation
	}
	return nil
}

// IsGameBegun return true, if all gamers joined to a game.
// Function provided to avoid of sleep on WaitBegin call.
func (g Game) IsGameBegun(id int) (igb bool, err error) {
	// gamer leaving can close the Game object as chanel,
	// it could cause a panic in other goroutines. process it.
	defer recoverAsErr(&err)

	c := make(chan interface{}, 1)
	g <- &gameCommand{act: isGameBegunCMD, id: id, rez: c}
	rez := <-c

	switch rez := rez.(type) {
	case error:
		return false, rez
	case bool:
		return rez, nil
	}

	return false, fmt.Errorf("returned value %v of Type %T: %w", rez, rez, ErrUnknownTypeReturned)
}

// WaitTurn waits for your turn.
func (g Game) WaitTurn(ctx context.Context, id int) (err error) {
	// gamer leaving can close the Game object as chanel,
	// it could cause a panic in other goroutines. process it.
	defer recoverAsErr(&err)

	//buffered because when killed by cancelation - internal mechanism can block other invocation on attemption to write to this chanel later
	c := make(chan interface{}, 1)
	g <- &gameCommand{act: wTurnCMD, id: id, rez: c}
	select {
	case err := <-c:
		if err, ok := err.(error); ok == true {
			return err
		}
	case <-ctx.Done():
		return ErrCancellation
	}
	return nil
}

// IsMyTurn returns true, if now is a gamer's turn else - false.
// Gamer is identified by his id.
// Function provided to avoid of sleep on WaitTurn call.
func (g Game) IsMyTurn(id int) (imt bool, err error) {
	// gamer leaving can close the Game object as chanel,
	// it could cause a panic in other goroutines. process it.
	defer recoverAsErr(&err)

	c := make(chan interface{}, 1)
	g <- &gameCommand{act: isMyTurnCMD, id: id, rez: c}
	rez := <-c

	switch rez := rez.(type) {
	case error:
		return false, rez
	case bool:
		return rez, nil
	}

	return false, fmt.Errorf("returned value %v of Type %T: %w", rez, rez, ErrUnknownTypeReturned)
}

// MakeTurn tries to make a turn.
func (g Game) MakeTurn(id int, turn *igame.TurnData) (err error) {
	// gamer leaving can close the Game object as chanel,
	// it could cause a panic in other goroutines. process it.
	defer recoverAsErr(&err)

	c := make(chan interface{})
	g <- &gameCommand{act: makeTurnCMD, id: id, rez: c, turn: turn}

	if err, ok := (<-c).(error); ok == true {
		return err
	}

	return nil
}

// Leave leave a game.
// No methods of this Game object should be invoked by this gamer
// after this call - it will return an error.
func (g Game) Leave(id int) (err error) {
	// gamer leaving can close the Game object as chanel,
	// it could cause a panic in other goroutines. process it.
	defer recoverAsErr(&err)

	c := make(chan interface{})
	g <- &gameCommand{act: leaveCMD, id: id, rez: c}

	if err, ok := (<-c).(error); ok == true {
		return err
	}

	return nil
}

// GamerState struct provides game internal data for one gamer.
type GamerState struct {
	Colour      igame.ChipColour   // colour of chip of this gamer
	Name        string             //this gamer's name
	beMSGChan   chan<- interface{} // delayed inform for WaitBegin's client
	turnMSGChan chan<- interface{} // delayed inform for WaitTurn's client
}

// NewGame creates the Game.
// Game mast be finished  by calling of End() method.
func NewGame(size int, komi float64) (Game, error) {
	field, err := field.New(size, komi)
	if err != nil {
		return nil, err
	}
	g := make(Game)
	g.run(field)
	return g, nil
}
