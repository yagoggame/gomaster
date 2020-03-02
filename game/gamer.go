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

import "fmt"

// Gamer is a struct assigned to each gamer
type Gamer struct {
	Name   string //the name of a player. may be the same for different player
	ID     int    //unique id of a gamer
	inGame Game   //gamer in pool may be vacant (InPlay is nil) or joined to this game
}

// New produces the new gamer
func New(name string, id int) *Gamer {
	return &Gamer{
		Name: name,
		ID:   id,
	}
}

// String provides compatibility with Stringer interface.
func (g *Gamer) String() string {
	return fmt.Sprintf("[ id: %d, name: %q, InGame: %v ]", g.ID, g.Name, g.inGame)
}

// GetGame returns the game of this gamer
func (g *Gamer) GetGame() Game {
	return g.inGame
}

// SetGame sets the game of this gamer
func (g *Gamer) SetGame(game Game) {
	g.inGame = game
}
