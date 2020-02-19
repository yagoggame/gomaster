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

// Package gomaster provides thread safe pool of gamers.
package gomaster

import (
	"errors"
	"fmt"

	"github.com/yagoggame/gomaster/game"
)

var (
	// ErrNilGamer is an error of using a nil *Gamer
	ErrNilGamer = errors.New("failed to operate on nil gamer")
	// ErrIDNotFound is an error of operetion with unregistred gamer
	ErrIDNotFound = errors.New("no gamer with such id in the Pool")
	// ErrIDOccupied is an error of adding to the pool a user with ID
	// already occupied by the pool ID
	ErrIDOccupied = errors.New("id occupied")
	// ErrGamerOccupied is an error of join to a game by user
	// who is in other game already
	ErrGamerOccupied = errors.New("gamer already joined to another game")
	// ErrGamerGameStart is an error of game starting
	ErrGamerGameStart = errors.New("gamer failed to start a new game")
)

// GamersPool is a datatype based on chanel,
// to provide a thread safe pool of gamers.
type GamersPool chan *command

// AddGamer adds a gamer to the pool if he's not already there.
func (gp GamersPool) AddGamer(gamer *game.Gamer) error {
	if gamer == nil {
		return ErrNilGamer
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
		return nil, fmt.Errorf("failed to rm gamer for id %d: %w", id, ErrIDNotFound)
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

// ReleaseGame releases the gamer's game.
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

// NewGamersPool creates the pool of gamers.
// Pool must be destroied after using by call of Release() method.
func NewGamersPool() GamersPool {
	gp := make(GamersPool)
	gp.run()
	return gp
}
