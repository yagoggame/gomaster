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

	"github.com/yagoggame/gomaster/game/igogame"
)

// gameAction is a type with game action values
type gameAction int

// set of actions values of Game object
const (
	joinCMD        gameAction = iota //join This Game
	endCMD                           //finish this game
	gamerStateCMD                    //request state of gamer
	gameStateCMD                     //request state of game
	gameFieldSize                    //request size of game field
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
	turn  *igogame.TurnData
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
func join(gamerStates *map[int]*GamerState, cmd *gameCommand, gd *gmaeDescriptor) {
	defer close(cmd.rez)

	if len(*gamerStates) > 1 {
		cmd.rez <- ErrNoPlace
		return
	}

	if gd.gameOver == true {
		cmd.rez <- ErrGameOver
		return
	}

	chipColour := igogame.ChipColour(rand.Intn(2) + 1)
	for id := range *gamerStates {
		chipColour = igogame.ChipColour(3 - int((*gamerStates)[id].Colour))
	}

	(*gamerStates)[cmd.gamer.ID] = &GamerState{
		Colour: chipColour,
		Name:   cmd.gamer.Name,
	}
}

// gamerState implements concurrently safe processing of querry of
// GamerState function
func gamerState(gamerStates map[int]*GamerState, cmd *gameCommand) {
	defer close(cmd.rez)

	gs, ok := gamerStates[cmd.id]
	if ok == false {
		cmd.rez <- fmt.Errorf("failed to gamerState for gamer with id %d: %w", cmd.id, ErrUnknownID)
		return
	}

	//make a copy of gamer state to prevent change from the outside
	gsCpy := *gs
	cmd.rez <- &gsCpy
}

// fieldSize implements concurrently safe processing of querry of
// FieldSize function
func fieldSize(gamerStates map[int]*GamerState, cmd *gameCommand, gd *gmaeDescriptor) {
	defer close(cmd.rez)

	_, ok := gamerStates[cmd.id]
	if ok == false {
		cmd.rez <- fmt.Errorf("failed to fieldSize for gamer with id %d: %w", cmd.id, ErrUnknownID)
		return
	}

	cmd.rez <- gd.master.Size()
}

// gameState implements concurrently safe processing of querry of
// FieldSize function
func gameState(gamerStates map[int]*GamerState, cmd *gameCommand, gd *gmaeDescriptor) {
	defer close(cmd.rez)

	_, ok := gamerStates[cmd.id]
	if ok == false {
		cmd.rez <- fmt.Errorf("failed to fieldSize for gamer with id %d: %w", cmd.id, ErrUnknownID)
		return
	}

	cmd.rez <- gd.master.State()
}

// waitBegin implements concurrently safe processing of querry of
// WaitBegin function
func waitBegin(gamerStates map[int]*GamerState, cmd *gameCommand, gd *gmaeDescriptor) {
	gs, err := getGamerStateAndChecks(gamerStates, cmd.id, gd.gameOver)
	if err != nil {
		cmd.rez <- err
		close(cmd.rez)
		return
	}

	//put chanel to report on estimation of game begin condition in safe place.
	gs.beMSGChan = cmd.rez

	//if number of players enough to begin a game - report to all players.
	if len(gamerStates) == 2 {
		for _, gs := range gamerStates {
			reportOnChan(&gs.beMSGChan, nil)
		}
	}
}

// isGameBegun implements concurrently safe processing of querry of
// IsGameBegun function
func isGameBegun(gamerStates map[int]*GamerState, cmd *gameCommand, gd *gmaeDescriptor) {
	defer close(cmd.rez)

	_, err := getGamerStateAndChecks(gamerStates, cmd.id, gd.gameOver)
	if err != nil {
		cmd.rez <- err
		return
	}

	cmd.rez <- len(gamerStates) == 2
}

// waitTurn implements concurrently safe processing of querry of
// WaitTurn function
func waitTurn(gamerStates map[int]*GamerState, cmd *gameCommand, gd *gmaeDescriptor) {
	gs, err := getGamerStateAndChecks(gamerStates, cmd.id, gd.gameOver)
	if err != nil {
		cmd.rez <- err
		close(cmd.rez)
		return
	}

	if isMyTurnCalc(gd.currentTurn, gs.Colour) {
		close(cmd.rez)
		return
	}

	//put chanel to report on estimation of player's turn begin condition in safe place.
	gs.turnMSGChan = cmd.rez
}

// isMyTurn implements concurrently safe processing of querry of
// IsMyTurn function
func isMyTurn(gamerStates map[int]*GamerState, cmd *gameCommand, gd *gmaeDescriptor) {
	defer close(cmd.rez)

	gs, err := getGamerStateAndChecks(gamerStates, cmd.id, gd.gameOver)
	if err != nil {
		cmd.rez <- err
		return
	}

	cmd.rez <- isMyTurnCalc(gd.currentTurn, gs.Colour)
}

// makeTurn implements concurrently safe processing of querry of
// MakeTurn function
// return 1 on success turn, else - 0
func makeTurn(gamerStates map[int]*GamerState, cmd *gameCommand, gd *gmaeDescriptor) int {
	defer close(cmd.rez)

	gs, err := getGamerStateAndChecks(gamerStates, cmd.id, gd.gameOver)
	if err != nil {
		cmd.rez <- err
		return 0
	}
	if !isMyTurnCalc(gd.currentTurn, gs.Colour) {
		cmd.rez <- fmt.Errorf("failed to makeTurn for gamer with id %d: %w", cmd.id, ErrNotYourTurn)
		return 0
	}

	if err := gd.master.Move(gs.Colour, cmd.turn); err != nil {
		cmd.rez <- fmt.Errorf("failed to makeTurn for gamer with id %d: %w: %s", cmd.id, ErrWrongTurn, err)
		return 0
	}

	reportOnTurnChange(gamerStates, gd.currentTurn)

	return 1
}

// leaveGame implements concurrently safe processing of querry of
// LeaveGame function
func leaveGame(gamerStates map[int]*GamerState, cmd *gameCommand) bool {
	defer close(cmd.rez)

	// this action may be called only for joined players.
	_, ok := gamerStates[cmd.id]
	if ok == false {
		cmd.rez <- fmt.Errorf("failed to leaveGame for gamer with id %d: %w", cmd.id, ErrUnknownID)
		return false
	}

	// report to other player's, if they are awaiting somesthing, that other player left the game.
	for _, gs := range gamerStates {
		reportOnChan(&gs.beMSGChan, ErrOtherGamerLeft)
		reportOnChan(&gs.turnMSGChan, ErrOtherGamerLeft)
	}

	delete(gamerStates, cmd.id)
	return true
}

//helpers

// reportOnChan passes deferred data if needed
func reportOnChan(ch *chan<- interface{}, val interface{}) {
	if *ch != nil {
		if val != nil {
			*ch <- val
		}
		close(*ch)
		*ch = nil
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

func isMyTurnCalc(currentTurn int, col igogame.ChipColour) bool {
	return (currentTurn%2 == 0 && col == igogame.Black) || (currentTurn%2 == 1 && col == igogame.White)
}

func reportOnTurnChange(gamerStates map[int]*GamerState, currentTurn int) {
	for _, gs := range gamerStates {
		if isMyTurnCalc(currentTurn+1, gs.Colour) {
			reportOnChan(&gs.turnMSGChan, nil)
		}
	}
}

type gmaeDescriptor struct {
	gameOver    bool
	currentTurn int
	master      igogame.Master
}

// run processes commads for thread safe operations on Game.
func (g Game) run(master igogame.Master) {
	rand.Seed(time.Now().UnixNano())

	gamerStates := make(map[int]*GamerState)
	gd := &gmaeDescriptor{master: master}

	go func(g Game) {
		for cmd := range g {
			switch cmd.act {
			case endCMD:
				close(g)
				close(cmd.rez)

			case joinCMD:
				join(&gamerStates, cmd, gd)
			case gamerStateCMD:
				gamerState(gamerStates, cmd)
			case gameFieldSize:
				fieldSize(gamerStates, cmd, gd)
			case gameStateCMD:
				gameState(gamerStates, cmd, gd)
			case wBeginCMD:
				waitBegin(gamerStates, cmd, gd)
			case wTurnCMD:
				waitTurn(gamerStates, cmd, gd)
			case isMyTurnCMD:
				isMyTurn(gamerStates, cmd, gd)
			case isGameBegunCMD:
				isGameBegun(gamerStates, cmd, gd)
			case makeTurnCMD:
				gd.currentTurn += makeTurn(gamerStates, cmd, gd)
			case leaveCMD:
				gd.gameOver = leaveGame(gamerStates, cmd)
			}
			if gd.gameOver && len(gamerStates) == 0 {
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
