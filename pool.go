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

// gomaster - provides thread safe pool of gamers.
package gomaster

import (
	"errors"
	"fmt"

	"github.com/yagoggame/gomaster/game"
)

//errors
var (
	NilGamerError       = errors.New("failed to operate on nil gamer")
	IdNotFoundError     = errors.New("no gamer with such id in the Pool")
	IdOccupiedError     = errors.New("id occupied")
	GamerOccupiedError  = errors.New("gamer already joined to another game")
	GamerGameStartError = errors.New("gamer failed to start a new game")
)

///////////////////////////////////////////////////////
// Actions
///////////////////////////////////////////////////////

// action is a type with actions values.
type action int

// set of actions values of GamersPool object.
const (
	add action = iota // add gamer to pool
	rem               // remove gamer from pool
	rel               // release all data
	lst               // get list of gamers in pool
	//manage gamers interconnection
	joinG    // join the Game or create a new one
	releaseG // release the Game
	getG     // get gamer's game
)

// command is a type to hold a comand to a GamersPool.
type command struct {
	act   action
	gamer *game.Gamer
	id    int
	rez   chan<- interface{}
}

// GamersPool is a datatype based on chanel, to provide a thread safe pool of gamers.
type GamersPool chan *command

///////////////////////////////////////////////////////
// Queries on actions
///////////////////////////////////////////////////////

// AddGamer adds a gamer to the pool if he's not already there.
func (gp GamersPool) AddGamer(gamer *game.Gamer) error {
	if gamer == nil {
		return NilGamerError
	}
	c := make(chan interface{})

	gp <- &command{act: add, gamer: gamer, rez: c}

	if err := <-c; err != nil {
		return err.(error)
	}
	return nil
}

// RmGamer removes a gamer from the pool if he's there.
func (gp GamersPool) RmGamer(id int) (gamer *game.Gamer, err error) {
	c := make(chan interface{})
	gp <- &command{act: rem, id: id, rez: c}

	gamer, ok := (<-c).(*game.Gamer)
	if ok == false {
		return nil, fmt.Errorf("failed to rm gamer for id %d: %w", id, IdNotFoundError)
	}
	return gamer, nil
}

// ListGamers returns the list of gamers in the pool.
func (gp GamersPool) ListGamers() []*game.Gamer {
	c := make(chan interface{})
	gp <- &command{act: lst, rez: c}

	rez := <-c
	return rez.([]*game.Gamer)
}

// JoinGame joins a gamer to some another gamer's game, or start it's own.
func (gp GamersPool) JoinGame(id int) error {
	c := make(chan interface{})
	gp <- &command{act: joinG, id: id, rez: c}

	if err := <-c; err != nil {
		return err.(error)
	}
	return nil
}

// ReleaseGamer releases the gamer's game.
func (gp GamersPool) ReleaseGame(id int) error {
	c := make(chan interface{})
	gp <- &command{act: releaseG, id: id, rez: c}

	if err := <-c; err != nil {
		return err.(error)
	}
	return nil
}

// GetGamer gets gamer by id.
func (gp GamersPool) GetGamer(id int) (*game.Gamer, error) {
	c := make(chan interface{})
	gp <- &command{act: getG, id: id, rez: c}
	rez := <-c
	switch rez := rez.(type) {
	case error:
		return nil, rez
	case *game.Gamer:
		return rez, nil
	}
	return nil, fmt.Errorf("wrong result type: %v", rez)
}

// Release releases the pool.
func (gp GamersPool) Release() {
	c := make(chan interface{})
	gp <- &command{act: rel, rez: c}
	<-c
}

///////////////////////////////////////////////////////
// Process queries
///////////////////////////////////////////////////////
func addGamer(gamers map[int]*game.Gamer, gamer *game.Gamer, rezChan chan<- interface{}) {
	defer close(rezChan)

	gCpy := *gamer
	if _, ok := gamers[gCpy.Id]; ok == true {
		rezChan <- fmt.Errorf("failed to add gamer with id %d to a pool: %w", gCpy.Id, IdOccupiedError)
	}
	gamers[gCpy.Id] = &gCpy
}

func rmGamer(gamers map[int]*game.Gamer, id int, rezChan chan<- interface{}) {
	defer close(rezChan)

	if gamer, ok := gamers[id]; ok == true {
		gCpy := *gamer
		rezChan <- &gCpy
	}
	delete(gamers, id)
}

func listGamers(gamers map[int]*game.Gamer, rezChan chan<- interface{}) {
	defer close(rezChan)

	rez := make([]*game.Gamer, 0, len(gamers))
	for k := range gamers {
		gCpy := *gamers[k]
		rez = append(rez, &gCpy)
	}
	rezChan <- rez
}

func getGamer(gamers map[int]*game.Gamer, id int, rezChan chan<- interface{}) {
	defer close(rezChan)

	gamer, ok := gamers[id]
	if ok == false {
		rezChan <- fmt.Errorf("failed to get gamer for id %d: %w", id, IdNotFoundError)
		return
	}
	gCpy := *gamer
	rezChan <- &gCpy
	return
}

func joinGame(gamers map[int]*game.Gamer, id int, rezChan chan<- interface{}) {
	defer close(rezChan)
	// get a gamer by id. If there is no such gamer - it's  bad
	gamer, ok := gamers[id]
	if ok == false {
		rezChan <- fmt.Errorf("failed to join gamer with id %d to a game: %w", id, IdNotFoundError)
		return
	}

	// if gamer already playing - for now, let's interpret it like an error
	if gamer.InGame != nil {
		rezChan <- fmt.Errorf("failed to join gamer with id %d to a game: %w", id, GamerOccupiedError)
		return
	}
	
	//copy the gamer to prevent of chnging by the Game
	gCpy=*gamer
	// iterate over gamers
	for _, g := range gamers {
		// playing with yourself is a sin, but we are need a gamer's object
		if gamer.Id == g.Id {
			continue
		}
		// try to join to a gamer's game
		if g.InGame != nil {
			// if succed - it's all, else - no matter: will try with other player or start his own game
			if err := g.InGame.Join(&gCpy); err == nil {
				gamer.InGame = g.InGame
				return
			}

		}
	}

	// if no partner found - start his own game
	game := game.NewGame()
	if err := game.Join(&gCpy); err != nil {
		// if can't - finish game and make a gamer vacant
		gamer.InGame = nil
		rezChan <- fmt.Errorf("failed to join gamer with id %d to a game: %w: %s", id, GamerGameStartError, err)
		game.End()
		return
	}
	gamer.InGame = game
}

func releaseGame(gamers map[int]*game.Gamer, id int, rezChan chan<- interface{}) {
	defer close(rezChan)
	//  get a gamer by id. If there is no such gamer - it's  bad
	gamer, ok := gamers[id]
	if ok == false {
		rezChan <- fmt.Errorf("failed to release game for id %d: %w", id, IdNotFoundError)
		return
	}

	// if gamer is playing yet - stop it
	if gamer.InGame != nil {
		// if gamer.InGame is active - try to release game, but in any case - mark gamer as vacant
		_ = gamer.InGame.Leave(gamer)
		gamer.InGame = nil
	}
}

// run processes commads for thread safe operations on pool.
func (gp GamersPool) run() {
	gamers := make(map[int]*game.Gamer)
	go func(gp GamersPool) {
		for cmd := range gp {
			switch cmd.act {
			case rel:
				close(gp)
				close(cmd.rez)
			case add:
				addGamer(gamers, cmd.gamer, cmd.rez)

			case lst:
				listGamers(gamers, cmd.rez)

			case rem:
				rmGamer(gamers, cmd.id, cmd.rez)
			// plaing managment
			case joinG:
				joinGame(gamers, cmd.id, cmd.rez)
			case releaseG:
				releaseGame(gamers, cmd.id, cmd.rez)
			case getG:
				getGamer(gamers, cmd.id, cmd.rez)
			}
		}
	}(gp)
	return
}

// NewGamersPool creates the pool of gamers.
// Pool must be destroied after using by call of Release() method.
func NewGamersPool() GamersPool {
	gp := make(GamersPool)
	gp.run()
	return gp
}
