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

// gomaster - provides thread safe game entity and some data structures to maintain it
package game

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

//////////////////////////////////////////////////////
// Constants
///////////////////////////////////////////////////////

// ChipColour - provide datatype of chip's colours
type ChipColour int

const (
	// Colour not assigned
	NoColour ChipColour = 0
	// Colour of black chip
	Black = 1
	// Colour of white chip
	White = 2
)

/////////////////////////////////////////////////////
// Actions
///////////////////////////////////////////////////////

type gameAction int

const (
	joinCMD        gameAction = iota //join This Game
	endCMD                           //finish this game
	gamerStateCMD                    //finish this game
	makeTurnCMD                      //make a turn
	isGameBegunCMD                   //request of state to avoid of wBeginCMD
	isMyTurnCMD                      //request of state to avoid of wTurnCMD
	leaveCMD                         //leave a game

	//action, which can cause an awaiting
	wBeginCMD //wait of game begin
	wTurnCMD  //wait for your turn
)

// TurnData - struct, using to put a gamer's turn data
type TurnData struct {
	X, Y int
}

// TurnError - is a special kind of error
type TurnError string

func (te TurnError) Error() string {
	return string(te)
}

type gameCommand struct {
	act   gameAction
	gamer *Gamer
	rez   chan<- interface{}
	turn  *TurnData
}

// Game - datatype based on chanel, to provide a thread safe game entity.
type Game chan *gameCommand

///////////////////////////////////////////////////////
// Queries on actions
///////////////////////////////////////////////////////

// End - releases game resources and close a Game object as chanel
// Use this function only to abort, if creation failed.
// Normaly - Leave invocation for all users has the same consequences.
// If the End() invoked after this - an error will be returned.
func (g Game) End() (err error) {
	// gamer leaving can close the Game object as chanel,
	// it could cause a panic in other goroutines. process it
	defer recoverAsErr(&err)

	c := make(chan interface{})
	g <- &gameCommand{act: endCMD, rez: c}
	<-c
	return nil
}

// Join - try to join gamer to this Game.
func (g Game) Join(gamer *Gamer) (err error) {
	// gamer leaving can close the Game object as chanel,
	// it could cause a panic in other goroutines. process it
	defer recoverAsErr(&err)

	c := make(chan interface{})
	g <- &gameCommand{act: joinCMD, gamer: gamer, rez: c}

	if err := <-c; err != nil {
		return err.(error)
	}
	return nil
}

// GamerState - returns a copy of Internal State of a gamer (to prevent a manual changing).
func (g Game) GamerState(gamer *Gamer) (state GamerState, err error) {
	// gamer leaving can close the Game object as chanel,
	// it could cause a panic in other goroutines. process it
	defer recoverAsErr(&err)

	c := make(chan interface{})
	g <- &gameCommand{act: gamerStateCMD, gamer: gamer, rez: c}
	rez := <-c

	switch rez := rez.(type) {
	case error:
		return GamerState{}, rez
	case GamerState:
		return rez, nil
	}

	return GamerState{}, fmt.Errorf("unknown type of value returned: %T: %v", rez, rez)

}

// WaitBegin - waits for game begin.
// If gamer started this game - awaiting another person.
func (g Game) WaitBegin(ctx context.Context, gamer *Gamer) (err error) {
	// gamer leaving can close the Game object as chanel,
	// it could cause a panic in other goroutines. process it
	defer recoverAsErr(&err)

	//buffered because when killed by cancelation - internal mechanism can block other invocation on attemption to write to this chanel later
	c := make(chan interface{}, 1)
	g <- &gameCommand{act: wBeginCMD, gamer: gamer, rez: c}
	select {
	case err := <-c:
		if err, ok := err.(error); ok == true {
			return err
		}
	case <-ctx.Done():
		return fmt.Errorf("Cancelled")
	}
	return nil
}

// IsGameBegun - return true, if all gamers joined to a game.
// Function provided to avoid of sleep on WaitBegin call.
func (g Game) IsGameBegun(gamer *Gamer) (igb bool, err error) {
	// gamer leaving can close the Game object as chanel,
	// it could cause a panic in other goroutines. process it
	defer recoverAsErr(&err)

	c := make(chan interface{}, 1)
	g <- &gameCommand{act: isGameBegunCMD, gamer: gamer, rez: c}
	rez := <-c

	switch rez := rez.(type) {
	case error:
		return false, rez
	case bool:
		return rez, nil
	}

	return false, fmt.Errorf("unknown type of value returned: %T: %v", rez, rez)
}

// WaitTurn - waits for your turn.
func (g Game) WaitTurn(ctx context.Context, gamer *Gamer) (err error) {
	// gamer leaving can close the Game object as chanel,
	// it could cause a panic in other goroutines. process it
	defer recoverAsErr(&err)

	//buffered because when killed by cancelation - internal mechanism can block other invocation on attemption to write to this chanel later
	c := make(chan interface{}, 1)
	g <- &gameCommand{act: wTurnCMD, gamer: gamer, rez: c}
	select {
	case err := <-c:
		if err, ok := err.(error); ok == true {
			return err
		}
	case <-ctx.Done():
		return fmt.Errorf("Cancelled")
	}
	return nil
}

// IsMyTurn - return true, if now is a gamer's turn else - false.
// Function provided to avoid of sleep on WaitTurn call.
func (g Game) IsMyTurn(gamer *Gamer) (imt bool, err error) {
	// gamer leaving can close the Game object as chanel,
	// it could cause a panic in other goroutines. process it
	defer recoverAsErr(&err)

	c := make(chan interface{}, 1)
	g <- &gameCommand{act: isMyTurnCMD, gamer: gamer, rez: c}
	rez := <-c

	switch rez := rez.(type) {
	case error:
		return false, rez
	case bool:
		return rez, nil
	}

	return false, fmt.Errorf("unknown type of value returned: %T: %v", rez, rez)
}

// MakeTurn - tries to make a turn.
func (g Game) MakeTurn(gamer *Gamer, turn *TurnData) (err error) {
	// gamer leaving can close the Game object as chanel,
	// it could cause a panic in other goroutines. process it
	defer recoverAsErr(&err)

	c := make(chan interface{})
	g <- &gameCommand{act: makeTurnCMD, gamer: gamer, rez: c, turn: turn}

	if err, ok := (<-c).(error); ok == true {
		return err
	}

	return nil
}

// Leave - leave a game.
// No methods of this Game object should be invoked by this gamer after this call - it will return an error.
func (g Game) Leave(gamer *Gamer) (err error) {
	// gamer leaving can close the Game object as chanel,
	// it could cause a panic in other goroutines. process it
	defer recoverAsErr(&err)

	c := make(chan interface{})
	g <- &gameCommand{act: leaveCMD, gamer: gamer, rez: c}

	if err, ok := (<-c).(error); ok == true {
		return err
	}

	return nil
}

func recoverAsErr(err *error) {
	if r := recover(); r != nil {
		if errR, ok := r.(error); ok == true {
			*err = errR
			if strings.Compare((*err).Error(), "send on closed channel") != 0 {
				panic(r)
			}
		}
	}
}

///////////////////////////////////////////////////////
// Process queries
///////////////////////////////////////////////////////

func joinThisGame(gamers *map[*Gamer]*GamerState, gamer *Gamer, rezChan chan<- interface{}, gameOver bool) {
	defer close(rezChan)

	//default colour
	clr := ChipColour(rand.Intn(2) + 1)

	if len(*gamers) > 1 {
		rezChan <- fmt.Errorf("no vacant place in Game")
		return
	}

	// if game already collapsed by some reasone - there is no sense to wait
	if gameOver == true {
		rezChan <- fmt.Errorf("Game Over")
		return
	}

	//recalc colour if nedded
	for gamer := range *gamers {
		clr = ChipColour(3 - int((*gamers)[gamer].Colour))
	}

	// assign a colour and give a chips to this player

	(*gamers)[gamer] = &GamerState{
		Colour: clr,
	}
}

func gamerState(gamers map[*Gamer]*GamerState, gamer *Gamer, rezChan chan<- interface{}) {
	defer close(rezChan)

	// this action may be called only for joined players
	gs, ok := gamers[gamer]
	if ok == false {
		rezChan <- fmt.Errorf("not joined gamer %s tries to get his state in the game", gamer)
		return
	}

	//put chanel to report on estimation of game begin condition in safe place
	rezChan <- *gs
}

func waitBegin(gamers map[*Gamer]*GamerState, gamer *Gamer, rezChan chan<- interface{}, gameOver bool) {
	// this action may be called only for joined players
	gs, ok := gamers[gamer]
	if ok == false {
		rezChan <- fmt.Errorf("not joined gamer %s tries to await of game begin", gamer)
		close(rezChan)
		return
	}

	// if game already collapsed by some reasone - there is no sense to wait
	if gameOver == true {
		rezChan <- fmt.Errorf("Game Over")
		close(rezChan)
		return
	}

	//put chanel to report on estimation of game begin condition in safe place
	gs.beMSGChan = rezChan

	//if number of players enough to begin a game - report to all players
	if len(gamers) == 2 {
		for _, gs := range gamers {
			reportOnChan(&gs.beMSGChan, nil)
		}
	}
}

func isGameBegun(gamers map[*Gamer]*GamerState, gamer *Gamer, currentTurn int, rezChan chan<- interface{}, gameOver bool) {
	defer close(rezChan)

	_, ok := gamers[gamer]
	if ok == false {
		rezChan <- fmt.Errorf("not joined gamer %s tries to ask: is game begun", gamer)
		return
	}

	// if game already collapsed by some reasone - there is no sense to wait
	if gameOver == true {
		rezChan <- fmt.Errorf("Game Over")
		return
	}

	// If a player's turn has already come - report
	rezChan <- len(gamers) == 2
}

func waitTurn(gamers map[*Gamer]*GamerState, gamer *Gamer, currentTurn int, rezChan chan<- interface{}, gameOver bool) {
	// this action may be called only for joined players
	gs, ok := gamers[gamer]
	if ok == false {
		rezChan <- fmt.Errorf("not joined gamer %s tries to await of his turn", gamer)
		close(rezChan)
		return
	}

	// if game already collapsed by some reasone - there is no sense to wait
	if gameOver == true {
		rezChan <- fmt.Errorf("Game Over")
		close(rezChan)
		return
	}

	// If a player's turn has already come - report
	if isMyTurnCalc(currentTurn, gs.Colour) {
		close(rezChan)
		return
	}

	//put chanel to report on estimation of player's turn begin condition in safe place
	gs.turnMSGChan = rezChan
}

func isMyTurnCalc(currentTurn int, col ChipColour) bool {
	return (currentTurn%2 == 0 && col == Black) || (currentTurn%2 == 1 && col == White)
}

func isMyTurn(gamers map[*Gamer]*GamerState, gamer *Gamer, currentTurn int, rezChan chan<- interface{}, gameOver bool) {
	defer close(rezChan)

	gs, ok := gamers[gamer]
	if ok == false {
		rezChan <- fmt.Errorf("not joined gamer %s tries to ask: is it his turn", gamer)
		return
	}

	// if game already collapsed by some reasone - there is no sense to wait
	if gameOver == true {
		rezChan <- fmt.Errorf("Game Over")
		return
	}

	// If a player's turn has already come - report
	rezChan <- isMyTurnCalc(currentTurn, gs.Colour)
}

func performTurn(turn *TurnData) error {
	if turn.X <= 0 || turn.Y <= 0 {
		return fmt.Errorf("coordinates must be positive. (%d %d) recieved", turn.X, turn.Y)
	}
	return nil
}

// makeTurn - try to make a turn. If success - return 1 and report, if some one is awaiting, that it's his turn, else return 0
func makeTurn(gamers map[*Gamer]*GamerState, gamer *Gamer, turn *TurnData, currentTurn int, rezChan chan<- interface{}, gameOver bool) int {
	defer close(rezChan)

	// this action may be called only for joined players
	gs, ok := gamers[gamer]
	if ok == false {
		rezChan <- fmt.Errorf("not joined gamer %s tries to make a turn", gamer)
		return 0
	}

	// if game already collapsed by some reasone - there is no sense to wait
	if gameOver == true {
		rezChan <- fmt.Errorf("Game Over")
		return 0
	}

	// If it's not a player's turn
	if (currentTurn%2 == 1 && gs.Colour == Black) || (currentTurn%2 == 0 && gs.Colour == White) {
		rezChan <- fmt.Errorf("not a gamer's %s turn", gamer)
		return 0
	}

	//perform turn and check, is it correct
	if err := performTurn(turn); err != nil {
		rezChan <- TurnError(fmt.Sprintf("wrong turn: %s", err))
		return 0
	}

	//report player that turn is changed, if they are awaiting
	for _, gs := range gamers {
		if ((currentTurn+1)%2 == 0 && gs.Colour == Black) || ((currentTurn+1)%2 == 1 && gs.Colour == White) {
			// if there is old call's channel - report on it too
			reportOnChan(&gs.turnMSGChan, nil)
		}
	}

	return 1
}

func leaveGame(gamers map[*Gamer]*GamerState, gamer *Gamer, rezChan chan<- interface{}) bool {
	defer close(rezChan)

	// this action may be called only for joined players
	_, ok := gamers[gamer]
	if ok == false {
		rezChan <- fmt.Errorf("not joined gamer %s tries to leave the game", gamer)
		return false
	}

	// report to other player's, if they are awaiting somesthing, that other player left the game
	for _, gs := range gamers {
		reportOnChan(&gs.beMSGChan, fmt.Errorf("other player left the game"))
		reportOnChan(&gs.turnMSGChan, fmt.Errorf("other player left the game"))
	}

	delete(gamers, gamer)
	return true
}

func reportOnChan(rezChan *chan<- interface{}, val interface{}) {
	if *rezChan != nil {
		if val != nil {
			*rezChan <- val
		}
		close(*rezChan)
		*rezChan = nil
	}
}

// GamerState struct - provides game internal data for one gamer
type GamerState struct {
	// colour of chip of this gamer
	Colour ChipColour
	// delayed inform for WaitBegin's client
	beMSGChan chan<- interface{}
	// delayed inform for WaitTurn's client
	turnMSGChan chan<- interface{}
}

// run - Processes commads for thread safe operations on Game
func (g Game) run() {
	rand.Seed(time.Now().UnixNano())

	gamers := make(map[*Gamer]*GamerState)
	currentTurn := 0
	gameOver := false

	go func(g Game) {
		for cmd := range g {
			switch cmd.act {
			case endCMD:
				close(g)
				close(cmd.rez)

			case joinCMD:
				joinThisGame(&gamers, cmd.gamer, cmd.rez, gameOver)
			case gamerStateCMD:
				gamerState(gamers, cmd.gamer, cmd.rez)
			case wBeginCMD:
				waitBegin(gamers, cmd.gamer, cmd.rez, gameOver)
			case wTurnCMD:
				waitTurn(gamers, cmd.gamer, currentTurn, cmd.rez, gameOver)
			case isMyTurnCMD:
				isMyTurn(gamers, cmd.gamer, currentTurn, cmd.rez, gameOver)
			case isGameBegunCMD:
				isGameBegun(gamers, cmd.gamer, currentTurn, cmd.rez, gameOver)
			case makeTurnCMD:
				currentTurn += makeTurn(gamers, cmd.gamer, cmd.turn, currentTurn, cmd.rez, gameOver)
			case leaveCMD:
				gameOver = leaveGame(gamers, cmd.gamer, cmd.rez)
			}
			if gameOver && len(gamers) == 0 {
				close(g)
			}
		}
		for _, gs := range gamers {
			reportOnChan(&gs.beMSGChan, fmt.Errorf("game destroyed"))
			reportOnChan(&gs.turnMSGChan, fmt.Errorf("game destroyed"))
		}
	}(g)
	return
}

// NewGame - Create the Game
// Game mast be finished  by calling of End() method
func NewGame() Game {
	g := make(Game)
	g.run()
	return g
}
