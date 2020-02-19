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
	"fmt"
	"math/rand"
	"strings"
	"time"
)

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

// gameCommand is a type to hold a comand to a Game
type gameCommand struct {
	act   gameAction
	gamer *Gamer
	id    int
	rez   chan<- interface{}
	turn  *TurnData
}

// recoverAsErr processes the panic
// on any action after closing the Game as chanel
func recoverAsErr(err *error) {
	r := recover()
	if r == nil {
		return
	}

	if errR, ok := r.(error); ok == true {
		*err = errR
		if strings.Compare((*err).Error(), "send on closed channel") != 0 {
			panic(r)
		}
		*err = ErrResourceNotAvailable
	}
}

// Process queries

// join implements concurrently safe processing of querry of
// Join function
func join(gamerStates *map[int]*GamerState, gamer *Gamer, rezChan chan<- interface{}, gameOver bool) {
	defer close(rezChan)

	if len(*gamerStates) > 1 {
		rezChan <- ErrNoPlace
		return
	}

	if gameOver == true {
		rezChan <- ErrGameOver
		return
	}

	chipColour := ChipColour(rand.Intn(2) + 1)
	for id := range *gamerStates {
		chipColour = ChipColour(3 - int((*gamerStates)[id].Colour))
	}

	(*gamerStates)[gamer.ID] = &GamerState{
		Colour: chipColour,
		Name:   gamer.Name,
	}
}

// gamerState implements concurrently safe processing of querry of
// GamerState function
func gamerState(gamerStates map[int]*GamerState, id int, rezChan chan<- interface{}) {
	defer close(rezChan)

	gs, ok := gamerStates[id]
	if ok == false {
		rezChan <- fmt.Errorf("failed to gamerState for gamer with id %d: %w", id, ErrUnknownID)
		return
	}

	//make a copy of gamer state to prevent change from the outside
	gsCpy := *gs
	rezChan <- &gsCpy
}

// waitBegin implements concurrently safe processing of querry of
// WaitBegin function
func waitBegin(gamerStates map[int]*GamerState, id int, rezChan chan<- interface{}, gameOver bool) {
	gs, err := getGamerStateAndChecks(gamerStates, id, gameOver)
	if err != nil {
		rezChan <- err
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

// isGameBegun implements concurrently safe processing of querry of
// IsGameBegun function
func isGameBegun(gamerStates map[int]*GamerState, id int, currentTurn int, rezChan chan<- interface{}, gameOver bool) {
	defer close(rezChan)

	_, err := getGamerStateAndChecks(gamerStates, id, gameOver)
	if err != nil {
		rezChan <- err
		return
	}

	rezChan <- len(gamerStates) == 2
}

// waitTurn implements concurrently safe processing of querry of
// WaitTurn function
func waitTurn(gamerStates map[int]*GamerState, id int, currentTurn int, rezChan chan<- interface{}, gameOver bool) {
	gs, err := getGamerStateAndChecks(gamerStates, id, gameOver)
	if err != nil {
		rezChan <- err
		close(rezChan)
		return
	}

	if isMyTurnCalc(currentTurn, gs.Colour) {
		close(rezChan)
		return
	}

	//put chanel to report on estimation of player's turn begin condition in safe place.
	gs.turnMSGChan = rezChan
}

// isMyTurn implements concurrently safe processing of querry of
// IsMyTurn function
func isMyTurn(gamerStates map[int]*GamerState, id int, currentTurn int, rezChan chan<- interface{}, gameOver bool) {
	defer close(rezChan)

	gs, err := getGamerStateAndChecks(gamerStates, id, gameOver)
	if err != nil {
		rezChan <- err
		return
	}

	rezChan <- isMyTurnCalc(currentTurn, gs.Colour)
}

// makeTurn implements concurrently safe processing of querry of
// MakeTurn function
// return 1 on success turn, else - 0
func makeTurn(gamerStates map[int]*GamerState, id int, turn *TurnData, currentTurn int, rezChan chan<- interface{}, gameOver bool) int {
	defer close(rezChan)

	gs, err := getGamerStateAndChecks(gamerStates, id, gameOver)
	if err != nil {
		rezChan <- err
		return 0
	}
	if !isMyTurnCalc(currentTurn, gs.Colour) {
		rezChan <- fmt.Errorf("failed to makeTurn for gamer with id %d: %w", id, ErrNotYourTurn)
		return 0
	}

	if err := performTurn(turn); err != nil {
		rezChan <- fmt.Errorf("failed to makeTurn for gamer with id %d: %w: %s", id, ErrWrongTurn, err)
		return 0
	}

	reportOnTurnChange(gamerStates, currentTurn)

	return 1
}

// leaveGame implements concurrently safe processing of querry of
// LeaveGame function
func leaveGame(gamerStates map[int]*GamerState, id int, rezChan chan<- interface{}) bool {
	defer close(rezChan)

	// this action may be called only for joined players.
	_, ok := gamerStates[id]
	if ok == false {
		rezChan <- fmt.Errorf("failed to leaveGame for gamer with id %d: %w", id, ErrUnknownID)
		return false
	}

	// report to other player's, if they are awaiting somesthing, that other player left the game.
	for _, gs := range gamerStates {
		reportOnChan(&gs.beMSGChan, ErrOtherGamerLeft)
		reportOnChan(&gs.turnMSGChan, ErrOtherGamerLeft)
	}

	delete(gamerStates, id)
	return true
}

//helpers

// reportOnChan passes deferred data if needed
func reportOnChan(rezChan *chan<- interface{}, val interface{}) {
	if *rezChan != nil {
		if val != nil {
			*rezChan <- val
		}
		close(*rezChan)
		*rezChan = nil
	}
}

func getGamerStateAndChecks(gamerStates map[int]*GamerState, id int, gameOver bool) (gs *GamerState, err error) {
	gs, ok := gamerStates[id]
	if ok == false {
		return nil, fmt.Errorf("failed to makeTurn for gamer with id %d: %w", id, ErrUnknownID)
	}

	if gameOver == true {
		return nil, ErrGameOver
	}
	return gs, nil
}

func isMyTurnCalc(currentTurn int, col ChipColour) bool {
	return (currentTurn%2 == 0 && col == Black) || (currentTurn%2 == 1 && col == White)
}

func reportOnTurnChange(gamerStates map[int]*GamerState, currentTurn int) {
	for _, gs := range gamerStates {
		if isMyTurnCalc(currentTurn+1, gs.Colour) {
			reportOnChan(&gs.turnMSGChan, nil)
		}
	}
}

func performTurn(turn *TurnData) error {
	if turn.X <= 0 || turn.Y <= 0 {
		return fmt.Errorf("coordinates must be positive. (%d %d) recieved", turn.X, turn.Y)
	}
	return nil
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
				join(&gamerStates, cmd.gamer, cmd.rez, gameOver)
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
			reportOnChan(&gs.beMSGChan, ErrGameDestroyed)
			reportOnChan(&gs.turnMSGChan, ErrGameDestroyed)
		}
	}(g)
	return
}
