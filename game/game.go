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

//Set of chip's colours
const (
	NoColour ChipColour = 0
	Black = 1
	White = 2
)

/////////////////////////////////////////////////////
// Actions
///////////////////////////////////////////////////////

// gameAction is a type with game action values
type gameAction int

// set of actions values of Game object
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

// TurnData is a struct, using to put a gamer's turn data
type TurnData struct {
	X, Y int
}

// TurnError is a special kind of error to inform on wrong turn.
type TurnError string

func (te TurnError) Error() string {
	return string(te)
}

// gameCommand is a type to hold a comand to a Game
type gameCommand struct {
	act   gameAction
	gamer *Gamer
	id    int
	rez   chan<- interface{}
	turn  *TurnData
}

// Game is a datatype based on chanel, to provide a thread safe game entity.
type Game chan *gameCommand

///////////////////////////////////////////////////////
// Queries on actions
///////////////////////////////////////////////////////

// End releases game resources and close a Game object as chanel
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

// GamerState returns a copy of Internal State of a gamer (to prevent a manual changing).
func (g Game) GamerState(id int) (state GamerState, err error) {
	// gamer leaving can close the Game object as chanel,
	// it could cause a panic in other goroutines. process it.
	defer recoverAsErr(&err)

	c := make(chan interface{})
	g <- &gameCommand{act: gamerStateCMD, id: id, rez: c}
	rez := <-c

	switch rez := rez.(type) {
	case error:
		return GamerState{}, rez
	case GamerState:
		return rez, nil
	}

	return GamerState{}, fmt.Errorf("unknown type of value returned: %T: %v", rez, rez)

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
		return fmt.Errorf("Cancelled")
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

	return false, fmt.Errorf("unknown type of value returned: %T: %v", rez, rez)
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
		return fmt.Errorf("Cancelled")
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

	return false, fmt.Errorf("unknown type of value returned: %T: %v", rez, rez)
}

// MakeTurn tries to make a turn.
func (g Game) MakeTurn(id int, turn *TurnData) (err error) {
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
// No methods of this Game object should be invoked by this gamer after this call - it will return an error.
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

func joinThisGame(gamerStates *map[int]*GamerState, gamer *Gamer, rezChan chan<- interface{}, gameOver bool) {
	defer close(rezChan)

	//default colour
	chipColour := ChipColour(rand.Intn(2) + 1)

	if len(*gamerStates) > 1 {
		rezChan <- fmt.Errorf("no vacant place in Game")
		return
	}

	// if game already collapsed by some reasone - there is no sense to wait.
	if gameOver == true {
		rezChan <- fmt.Errorf("Game Over")
		return
	}

	//recalc colour if nedded
	for gamer := range *gamerStates {
		chipColour = ChipColour(3 - int((*gamerStates)[gamer].Colour))
	}

	// assign a colour and give a chips to this player.
	(*gamerStates)[gamer.Id] = &GamerState{
		Colour: chipColour,
		Name:gamer.Name,
	}
}

func gamerState(gamerStates map[int]*GamerState, id int, rezChan chan<- interface{}) {
	defer close(rezChan)

	// this action may be called only for joined players.
	gs, ok := gamerStates[id]
	if ok == false {
		rezChan <- fmt.Errorf("not joined gamer with id %d tries to get his state in the game", id)
		return
	}

	//put chanel to report on estimation of game begin condition in safe place.
	rezChan <- *gs
}

func waitBegin(gamerStates map[int]*GamerState, id int, rezChan chan<- interface{}, gameOver bool) {
	// this action may be called only for joined players.
	gs, ok := gamerStates[id]
	if ok == false {
		rezChan <- fmt.Errorf("not joined gamer with id %d tries to await of game begin", id)
		close(rezChan)
		return
	}

	// if game already collapsed by some reasone - there is no sense to wait.
	if gameOver == true {
		rezChan <- fmt.Errorf("Game Over")
		close(rezChan)
		return
	}

	//put chanel to report on estimation of game begin condition in safe place.
	gs.beMSGChan = rezChan

	//if number of players enough to begin a game - report to all players.
	if len(gamerStates) == 2 {
		for _, gs := range gamerStates {
			reportOnChan(&gs.beMSGChan, nil)
		}
	}
}

func isGameBegun(gamerStates map[int]*GamerState, id int, currentTurn int, rezChan chan<- interface{}, gameOver bool) {
	defer close(rezChan)

	_, ok := gamerStates[id]
	if ok == false {
		rezChan <- fmt.Errorf("not joined gamer with id %d tries to ask: is game begun", id)
		return
	}

	// if game already collapsed by some reasone - there is no sense to wait.
	if gameOver == true {
		rezChan <- fmt.Errorf("Game Over")
		return
	}

	// If a player's turn has already come - report
	rezChan <- len(gamerStates) == 2
}

func waitTurn(gamerStates map[int]*GamerState, id int, currentTurn int, rezChan chan<- interface{}, gameOver bool) {
	// this action may be called only for joined players.
	gs, ok := gamerStates[id]
	if ok == false {
		rezChan <- fmt.Errorf("not joined gamer with id %d tries to await of his turn", id)
		close(rezChan)
		return
	}

	// if game already collapsed by some reasone - there is no sense to wait.
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

	//put chanel to report on estimation of player's turn begin condition in safe place.
	gs.turnMSGChan = rezChan
}

func isMyTurnCalc(currentTurn int, col ChipColour) bool {
	return (currentTurn%2 == 0 && col == Black) || (currentTurn%2 == 1 && col == White)
}

func isMyTurn(gamerStates map[int]*GamerState, id int, currentTurn int, rezChan chan<- interface{}, gameOver bool) {
	defer close(rezChan)

	gs, ok := gamerStates[id]
	if ok == false {
		rezChan <- fmt.Errorf("not joined gamer with id %d tries to ask: is it his turn", id)
		return
	}

	// if game already collapsed by some reasone - there is no sense to wait.
	if gameOver == true {
		rezChan <- fmt.Errorf("Game Over")
		return
	}

	// If a player's turn has already come - report.
	rezChan <- isMyTurnCalc(currentTurn, gs.Colour)
}

func performTurn(turn *TurnData) error {
	if turn.X <= 0 || turn.Y <= 0 {
		return fmt.Errorf("coordinates must be positive. (%d %d) recieved", turn.X, turn.Y)
	}
	return nil
}

// makeTurn - try to make a turn. If success - return 1 and report, if some one is awaiting, that it's his turn, else return 0
func makeTurn(gamerStates map[int]*GamerState, id int, turn *TurnData, currentTurn int, rezChan chan<- interface{}, gameOver bool) int {
	defer close(rezChan)

	// this action may be called only for joined players.
	gs, ok := gamerStates[id]
	if ok == false {
		rezChan <- fmt.Errorf("not joined gamer with id %d tries to make a turn", id)
		return 0
	}

	// if game already collapsed by some reasone - there is no sense to wait.
	if gameOver == true {
		rezChan <- fmt.Errorf("Game Over")
		return 0
	}

	// If it's not a player's turn
	if (currentTurn%2 == 1 && gs.Colour == Black) || (currentTurn%2 == 0 && gs.Colour == White) {
		rezChan <- fmt.Errorf("not a gamer's with id %s turn", id)
		return 0
	}

	//perform turn and check, is it correct.
	if err := performTurn(turn); err != nil {
		rezChan <- TurnError(fmt.Sprintf("wrong turn: %s", err))
		return 0
	}

	//report player that turn is changed, if they are awaiting.
	for _, gs := range gamerStates {
		if ((currentTurn+1)%2 == 0 && gs.Colour == Black) || ((currentTurn+1)%2 == 1 && gs.Colour == White) {
			// if there is old call's channel - report on it too.
			reportOnChan(&gs.turnMSGChan, nil)
		}
	}

	return 1
}

func leaveGame(gamerStates map[int]*GamerState, id int, rezChan chan<- interface{}) bool {
	defer close(rezChan)

	// this action may be called only for joined players.
	_, ok := gamerStates[id]
	if ok == false {
		rezChan <- fmt.Errorf("not joined gamer with id %d tries to leave the game", id)
		return false
	}

	// report to other player's, if they are awaiting somesthing, that other player left the game.
	for _, gs := range gamerStates {
		reportOnChan(&gs.beMSGChan, fmt.Errorf("other player left the game"))
		reportOnChan(&gs.turnMSGChan, fmt.Errorf("other player left the game"))
	}

	delete(gamerStates, id)
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

// GamerState struct provides game internal data for one gamer.
type GamerState struct {
	Colour ChipColour// colour of chip of this gamer
	Name   string//this gamer's name
	// delayed inform for WaitBegin's client
	beMSGChan chan<- interface{}
	// delayed inform for WaitTurn's client
	turnMSGChan chan<- interface{}
}

// run processes commads for thread safe operations on Game.
func (g Game) run() {
	rand.Seed(time.Now().UnixNano())

	gamerStates := make(map[int]*GamerState)
	currentTurn := 0
	gameOver := false

	go func(g Game) {
		for cmd := range g {
			switch cmd.act {
			case endCMD:
				close(g)
				close(cmd.rez)

			case joinCMD:
				joinThisGame(&gamerStates, cmd.gamer, cmd.rez, gameOver)
			case gamerStateCMD:
				gamerState(gamerStates, cmd.id, cmd.rez)
			case wBeginCMD:
				waitBegin(gamerStates, cmd.id, cmd.rez, gameOver)
			case wTurnCMD:
				waitTurn(gamerStates, cmd.id, currentTurn, cmd.rez, gameOver)
			case isMyTurnCMD:
				isMyTurn(gamerStates, cmd.id, currentTurn, cmd.rez, gameOver)
			case isGameBegunCMD:
				isGameBegun(gamerStates, cmd.id, currentTurn, cmd.rez, gameOver)
			case makeTurnCMD:
				currentTurn += makeTurn(gamerStates, cmd.id, cmd.turn, currentTurn, cmd.rez, gameOver)
			case leaveCMD:
				gameOver = leaveGame(gamerStates, cmd.id, cmd.rez)
			}
			if gameOver && len(gamerStates) == 0 {
				close(g)
			}
		}
		for _, gs := range gamerStates {
			reportOnChan(&gs.beMSGChan, fmt.Errorf("game destroyed"))
			reportOnChan(&gs.turnMSGChan, fmt.Errorf("game destroyed"))
		}
	}(g)
	return
}

// NewGame creates the Game.
// Game mast be finished  by calling of End() method.
func NewGame() Game {
	g := make(Game)
	g.run()
	return g
}
