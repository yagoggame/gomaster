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

package gomaster

import (
	"errors"
	"fmt"

	"github.com/yagoggame/gomaster/game"
)

var errNoVacantGamer = errors.New("failed to find vacant gamer")

// action is a type with actions values.
type action int

// set of actions values of GamersPool object.
const (
	add      action = iota // add gamer to pool
	rem                    // remove gamer from pool
	rel                    // release all data
	lst                    // get list of gamers in pool
	joinG                  // join the Game or create a new one
	releaseG               // release the Game
	getG                   // get gamer's game
)

// command is a type to hold a comand to a GamersPool.
type command struct {
	act   action
	komi  float64
	size  int
	gamer *game.Gamer
	id    int
	rez   chan<- interface{}
}

// addGamer implements concurrently safe processing of querry of
// AddGamer function
func addGamer(gamers map[int]*game.Gamer, gamer *game.Gamer, rezChan chan<- interface{}) {
	defer close(rezChan)

	gCpy := *gamer
	if _, ok := gamers[gCpy.ID]; ok == true {
		rezChan <- fmt.Errorf("failed to add gamer with id %d to a pool: %w", gCpy.ID, ErrIDOccupied)
	}
	gamers[gCpy.ID] = &gCpy
}

// rmGamer implements concurrently safe processing of querry of
// RmGamer function
func rmGamer(gamers map[int]*game.Gamer, id int, rezChan chan<- interface{}) {
	defer close(rezChan)

	if gamer, ok := gamers[id]; ok == true {
		gCpy := *gamer
		rezChan <- &gCpy
	}
	delete(gamers, id)
}

// listGamers implements concurrently safe processing of querry of
// ListGamers function
func listGamers(gamers map[int]*game.Gamer, rezChan chan<- interface{}) {
	defer close(rezChan)

	rez := make([]*game.Gamer, 0, len(gamers))
	for k := range gamers {
		gCpy := *gamers[k]
		rez = append(rez, &gCpy)
	}
	rezChan <- rez
}

// getGamer implements concurrently safe processing of querry of
// GetGamer function
func getGamer(gamers map[int]*game.Gamer, id int, rezChan chan<- interface{}) {
	defer close(rezChan)

	gamer, ok := gamers[id]
	if ok == false {
		rezChan <- fmt.Errorf("failed to get gamer for id %d: %w", id, ErrIDNotFound)
		return
	}
	gCpy := *gamer
	rezChan <- &gCpy
	return
}

func joinOtherGame(gamers map[int]*game.Gamer, gamer *game.Gamer) error {
	for _, g := range gamers {
		if gamer.ID == g.ID {
			continue
		}

		if g.GetGame() != nil {
			//copy the gamer to prevent of chnging by the Game
			gCpy := *gamer

			if err := g.GetGame().Join(&gCpy); err == nil {
				gamer.SetGame(g.GetGame())
				return nil
			}

		}
	}
	return errNoVacantGamer
}

func startOwnGame(gamer *game.Gamer, cmd *command) error {
	game, err := game.NewGame(cmd.size, cmd.komi)
	if err != nil {
		return fmt.Errorf("failed to create game for gamer with id %d: %w: %s", gamer.ID, ErrGamerGameStart, err)
	}

	//copy the gamer to prevent of changing by the Game
	gCpy := *gamer
	if err := game.Join(&gCpy); err != nil {
		gamer.SetGame(nil)
		game.End()
		return fmt.Errorf("failed to join gamer with id %d to a game: %w: %s", gamer.ID, ErrGamerGameStart, err)
	}
	gamer.SetGame(game)
	return nil
}

// joinGame implements concurrently safe processing of querry of
// JoinGame function
func joinGame(gamers map[int]*game.Gamer, cmd *command) {
	defer close(cmd.rez)

	gamer, ok := gamers[cmd.id]
	if ok == false {
		cmd.rez <- fmt.Errorf("failed to join gamer with id %d to a game: %w", cmd.id, ErrIDNotFound)
		return
	}

	if gamer.GetGame() != nil {
		cmd.rez <- fmt.Errorf("failed to join gamer with id %d to a game: %w", cmd.id, ErrGamerOccupied)
		return
	}

	err := joinOtherGame(gamers, gamer)
	if errors.Is(err, errNoVacantGamer) {
		if err := startOwnGame(gamer, cmd); err != nil {
			cmd.rez <- err
		}
	}
}

// releaseGame implements concurrently safe processing of querry of
// ReleaseGame function
func releaseGame(gamers map[int]*game.Gamer, id int, rezChan chan<- interface{}) {
	defer close(rezChan)
	//  get a gamer by id. If there is no such gamer - it's  bad
	gamer, ok := gamers[id]
	if ok == false {
		rezChan <- fmt.Errorf("failed to release game for id %d: %w", id, ErrIDNotFound)
		return
	}

	if gamer.GetGame() != nil {
		_ = gamer.GetGame().Leave(gamer.ID)
		gamer.SetGame(nil)
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
			case joinG:
				joinGame(gamers, cmd)
			case releaseG:
				releaseGame(gamers, cmd.id, cmd.rez)
			case getG:
				getGamer(gamers, cmd.id, cmd.rez)
			}
		}
	}(gp)
	return
}
