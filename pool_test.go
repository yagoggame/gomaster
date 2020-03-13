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
	"reflect"
	"testing"
	"time"

	"github.com/yagoggame/gomaster/game"
)

var fastDurationThreshold = time.Duration(10) * time.Second

const (
	usualSize = 9
	usualKomi = 0.0
)

var poolFillTests = []struct {
	caseName string
	gamer    *game.Gamer
	want     error
}{
	{caseName: "first", gamer: &game.Gamer{Name: "Joe", ID: 1}, want: nil},
	{caseName: "second", gamer: &game.Gamer{Name: "Nick", ID: 2}, want: nil},
	{caseName: "third", gamer: &game.Gamer{Name: "Fury", ID: 3}, want: nil},
	{caseName: "same name", gamer: &game.Gamer{Name: "Fury", ID: 4}, want: nil},
	{caseName: "same id", gamer: &game.Gamer{Name: "Sam", ID: 4}, want: ErrIDOccupied},
	{caseName: "nil", gamer: nil, want: ErrNilGamer},
	{caseName: "fifth", gamer: &game.Gamer{Name: "Jack", ID: 5}, want: nil},
}

var validGamers = []*game.Gamer{
	&game.Gamer{Name: "Joe", ID: 1},
	&game.Gamer{Name: "Nick", ID: 2},
	&game.Gamer{Name: "jack", ID: 3},
	&game.Gamer{Name: "Fred", ID: 4},
	&game.Gamer{Name: "Bob", ID: 5},
}

type testCase struct {
	id       int
	caseName string
	gamer    *game.Gamer
	want     error
}

var poolCommonTests = []testCase{
	testCase{caseName: "Fake ID", id: 0, gamer: nil, want: ErrIDNotFound},
	testCase{caseName: "Center", id: 2, gamer: validGamers[2-1], want: nil},
	testCase{caseName: "Tail", id: 5, gamer: validGamers[5-1], want: nil},
	testCase{caseName: "Head", id: 1, gamer: validGamers[1-1], want: nil},
	testCase{caseName: "Regular", id: 3, gamer: validGamers[3-1], want: nil},
	testCase{caseName: "Last", id: 4, gamer: validGamers[4-1], want: nil},
}

var poolJoinTests = []testCase{
	testCase{caseName: "Fake ID", id: 0, gamer: nil, want: ErrIDNotFound},
	testCase{caseName: "Center", id: 2, gamer: validGamers[2-1], want: nil},
	testCase{caseName: "Tail", id: 5, gamer: validGamers[5-1], want: nil},
	testCase{caseName: "Head", id: 1, gamer: validGamers[1-1], want: nil},
	testCase{caseName: "Regular", id: 3, gamer: validGamers[3-1], want: nil},
	testCase{caseName: "Last", id: 4, gamer: validGamers[4-1], want: nil},
	testCase{caseName: "Occupied", id: 2, gamer: validGamers[2-1], want: ErrGamerOccupied},
}

// TestCreation tests NewGamersPool
func TestCreation(t *testing.T) {
	pool := NewGamersPool()
	if pool == nil {
		t.Fatalf("NewGamersPool():\nwant *GamersPool,\ngot: nil")
	}
	defer pool.Release()
}

// TestFill performs pool fill test
func TestFill(t *testing.T) {
	pool := NewGamersPool()
	defer pool.Release()

	for _, test := range poolFillTests {
		t.Run("AddGamer_"+test.caseName, func(t *testing.T) {
			err := pool.AddGamer(test.gamer)
			if !errors.Is(err, test.want) {
				t.Errorf("Unexpected AddGamer err:\ngot: %v,\nwant: err=%v.", err, test.want)
			}
		})
	}
}

// TestListGamers performs test of get list of gamers
func TestListGamers(t *testing.T) {
	pool := NewGamersPool()
	defer pool.Release()

	cntr := 0
	for _, test := range poolFillTests {
		if err := pool.AddGamer(test.gamer); err == nil {
			cntr++
		}
	}

	actualGamers := pool.ListGamers()
	if cntr != len(actualGamers) {
		t.Errorf("Unexpected num of gamers in the pool:\nwant: %d.\ngot: %d", cntr, len(actualGamers))
	}
}

// TestRemove test: removing of gameers from the pool
func TestRemove(t *testing.T) {
	pool := NewGamersPool()
	defer pool.Release()

	for _, g := range validGamers {
		if err := pool.AddGamer(g); err != nil {
			t.Fatalf("Unexpected fail on AddGamer: %q ", err)
		}
	}

	cntr := len(validGamers)
	for _, test := range poolCommonTests {
		t.Run(test.caseName, func(t *testing.T) {
			_, err := checkFunction(t, test, pool.RmGamer)
			if err == nil {
				cntr--
			}

			actualGamers := pool.ListGamers()
			if len(actualGamers) != cntr {
				t.Errorf("Unexpected num of gamers in the pool:\nwant: %d,\ngot: %d", cntr, len(actualGamers))
			}
		})
	}
}

// TestGetGamer tests GetGamer function
func TestGetGamer(t *testing.T) {
	pool := NewGamersPool()
	defer pool.Release()

	for _, g := range validGamers {
		if err := pool.AddGamer(g); err != nil {
			t.Fatalf("Unexpected fail on AddGamer: %q ", err)
		}
	}

	for _, test := range poolCommonTests {
		t.Run(""+test.caseName, func(t *testing.T) {
			gettedGamer, _ := checkFunction(t, test, pool.GetGamer)

			removedGamer, _ := pool.RmGamer(test.id)
			if !(removedGamer == nil && gettedGamer == nil) &&
				(removedGamer == nil || gettedGamer == nil ||
					!reflect.DeepEqual(*gettedGamer, *test.gamer)) {
				t.Errorf("Unexpected GetGamer and RmGamer results relationship:\nwant: same gamer\ngot: %v, %v",
					gettedGamer, removedGamer)
			}
		})
	}
}

// TestRelease tests Release function
func TestRelease(t *testing.T) {
	pool := NewGamersPool()

	for _, g := range validGamers {
		if err := pool.AddGamer(g); err != nil {
			t.Fatalf("Unexpected fail on AddGamer: %q ", err)
		}
	}

	stateChan := asyncReleaseState(pool)

	// Release function should be the pretty fast action
	// with closing of pool object as chanel
	select {
	case ok := <-stateChan:
		if ok == true {
			t.Fatalf("Unexpected pool.Release() result:\nwant: closed GamersPool object as chanel,\ngot: chanel alive")
		}
	case <-time.After(fastDurationThreshold):
		t.Fatalf("Unexpected duration:\nwant: duration < %[1]v,\ngot: duration >= %[1]v", fastDurationThreshold)
	}
}

// TestJoinGame tests JoinGame function
func TestJoinGame(t *testing.T) {
	pool := NewGamersPool()
	defer pool.Release()

	for _, g := range validGamers {
		if err := pool.AddGamer(g); err != nil {
			t.Fatalf("Unexpected fail on AddGamer: %q ", err)
		}
	}

	checkInitialDisjoined(t, pool)
	checkJoin(t, pool)
	//2.5 pairs of validGamers should give 3 games
	checkGamesCount(t, pool)

	for _, g := range validGamers {
		if err := pool.ReleaseGame(g.ID); err != nil {
			t.Errorf("Unexpected fail on ReleaseGame: %q ", err)
		}
	}
}

// TestReleaseGame tests ReleaseGame function
func TestReleaseGame(t *testing.T) {
	pool := NewGamersPool()
	defer pool.Release()
	prepareGamers(t, pool)

	releaseCounter := 0
	for _, test := range poolCommonTests {
		t.Run(test.caseName, func(t *testing.T) {
			err := pool.ReleaseGame(test.id)
			if !errors.Is(err, test.want) {
				t.Errorf("Unexpected ReleaseGame result on id %d:\nwant: %v,\ngot: %v ", test.id, test.want, err)
			}
			if err == nil {
				releaseCounter++
			}
		})
	}

	checkReleaseCounter(t, pool, releaseCounter)
}
