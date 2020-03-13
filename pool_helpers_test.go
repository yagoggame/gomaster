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
	"math"
	"reflect"
	"testing"

	"github.com/yagoggame/gomaster/game"
)

func checkReleaseCounter(t *testing.T, pool GamersPool, releaseCounter int) {
	gamerInGameCount := 0
	actualGamers := pool.ListGamers()
	for _, g := range actualGamers {
		if g.GetGame() != nil {
			gamerInGameCount++
		}
	}
	if gamerInGameCount != len(actualGamers)-releaseCounter {
		t.Errorf("Unexpected count of Gamers in game:\nwant: %d,\ngot: %d", len(actualGamers)-releaseCounter, gamerInGameCount)
	}
}

func prepareGamers(t *testing.T, pool GamersPool) {
	for _, g := range validGamers {
		if err := pool.AddGamer(g); err != nil {
			t.Fatalf("Unexpected fail on AddGamer: %q ", err)
		}
		if err := pool.JoinGame(g.ID, usualSize, usualKomi); err != nil {
			t.Fatalf("Unexpected fail on JoinGame: %q ", err)
		}
	}
}

func checkFunction(t *testing.T, test testCase, fn func(id int) (*game.Gamer, error)) (*game.Gamer, error) {
	returnedGamer, err := fn(test.id)

	if !errors.Is(err, test.want) {
		t.Errorf("Unexpected action err:\ngot: %v,\nwant: err=%v.", err, test.want)
	}

	switch test.want == nil {
	case true:
		if returnedGamer == nil || !reflect.DeepEqual(*returnedGamer, *test.gamer) {
			t.Errorf("Unexpected action gamer:\nwant: %v,\ngot %v", test.gamer, returnedGamer)
		}
	case false:
		if returnedGamer != nil {
			t.Errorf("Unexpected action gamer:\nwant nill gamer pointer,\ngot: %v", returnedGamer)
		}
	}
	return returnedGamer, err
}

func asyncReleaseState(pool GamersPool) (signal <-chan interface{}) {
	c := make(chan interface{})
	go func(c chan<- interface{}) {
		pool.Release()
		_, ok := <-pool
		c <- ok
		close(c)
	}(c)
	return c
}

func checkInitialDisjoined(t *testing.T, pool GamersPool) {
	actualGamers := pool.ListGamers()
	for _, g := range actualGamers {
		if g.GetGame() != nil {
			t.Fatalf("Unexpected Gamer.GetGame():\nwant:nil,\ngot:%v", g.GetGame())
		}
	}
}

func checkJoin(t *testing.T, pool GamersPool) {
	countRequestedJoins := join(t, pool)

	countJoined := 0
	actualGamers := pool.ListGamers()
	for _, g := range actualGamers {
		if g.GetGame() != nil {
			countJoined++
		}
	}

	if countRequestedJoins != countJoined {
		t.Errorf("Unexpected num of join success:\nwant:%d\ngot: %d", countRequestedJoins, countJoined)
	}
}

func join(t *testing.T, pool GamersPool) int {
	countRequestedJoins := 0
	for _, test := range poolJoinTests {
		t.Run(test.caseName, func(t *testing.T) {
			err := pool.JoinGame(test.id, usualSize, usualKomi)
			if !errors.Is(err, test.want) {
				t.Errorf("Unexpected result for JoinGame on id %d:\nwant: %v\ngot: %v ", test.id, test.want, err)
			}
			if err == nil {
				countRequestedJoins++
			}
		})
	}
	return countRequestedJoins
}

func checkGamesCount(t *testing.T, pool GamersPool) {
	games := make(map[game.Game]bool)
	actualGamers := pool.ListGamers()

	for _, g := range actualGamers {
		games[g.GetGame()] = true
	}

	if len(games) != int(math.Ceil(float64(len(validGamers))/2.0)) {
		t.Errorf("Unexpected number of games for %d validGamers:\nwant: %d,\ngot %d", len(validGamers), int(math.Ceil(float64(len(validGamers))/2.0)), len(games))
	}
}
