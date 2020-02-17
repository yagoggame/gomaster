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
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/yagoggame/gomaster/game"
)

// TestPoolFill - performs basic pool creation and fill test
func TestPoolFill(t *testing.T) {
	//testing data
	type testCase struct {
		caseName string
		gamer    *game.Gamer
		req      string
		success  bool
	}

	//fill pool with correct and not correct values
	tt := []testCase{
		testCase{caseName: "first", gamer: &game.Gamer{Name: "Joe", Id: 1}, req: "", success: true},
		testCase{caseName: "second", gamer: &game.Gamer{Name: "Nick", Id: 2}, req: "", success: true},
		testCase{caseName: "third", gamer: &game.Gamer{Name: "Fury", Id: 3}, req: "", success: true},
		testCase{caseName: "same name", gamer: &game.Gamer{Name: "Fury", Id: 4}, req: "", success: true},
		testCase{caseName: "same id", gamer: &game.Gamer{Name: "Sam", Id: 4}, req: "Id occupied", success: false},
		testCase{caseName: "nil", gamer: nil, req: "unable to Add nil gamer", success: false},
		testCase{caseName: "fifth", gamer: &game.Gamer{Name: "Jack", Id: 5}, req: "", success: true},
	}

	pool := NewGamersPool()
	if pool == nil {
		t.Fatalf("failed on NewGamersPool: nil pool created")
	}
	defer pool.Release()

	for _, tc := range tt {
		t.Run("AddGamer_"+tc.caseName, func(t *testing.T) {
			err := pool.AddGamer(tc.gamer)
			switch {
			case tc.success == true && err != nil:
				t.Errorf("It was expected that AddGamer will return err=nil. got: %q", err)
			case tc.success == false && err == nil:
				t.Errorf("It was expected that AddGamer will return err!=nil. got %v", err)
			case tc.success == false && err != nil && strings.Compare(err.Error(), tc.req) != 0:
				t.Errorf("It was expected that AddGamer will return err: %q got %q", tc.req, err)
			}
		})
	}

	t.Run("ListGamers", func(t *testing.T) {
		actualGamers := pool.ListGamers()
		cntr := 0
		for _, tc := range tt {
			if tc.success == true {
				cntr += 1
			}
		}
		if cntr != len(actualGamers) {
			t.Errorf("expected count of gamers in pool %d not equal to expectation: %d\nlist of gamers: %s", len(actualGamers), cntr, actualGamers)
		}
	})
}

// TestPoolRemove - test: removing of gameers from the pool
func TestPoolRemove(t *testing.T) {
	pool := NewGamersPool()
	defer pool.Release()

	//fill pool
	gamers := []*game.Gamer{
		&game.Gamer{Name: "Joe", Id: 1},
		&game.Gamer{Name: "Nick", Id: 2},
		&game.Gamer{Name: "jack", Id: 3},
		&game.Gamer{Name: "Fred", Id: 4},
	}

	for _, g := range gamers {
		if err := pool.AddGamer(g); err != nil {
			t.Fatalf("fail on AddGamer: %q ", err)
		}
	}

	type testCase struct {
		id       int
		caseName string
		gamer    *game.Gamer
		req      string
		success  bool
	}

	//rm from different positions, from add's point of view.
	tt := []testCase{
		testCase{caseName: "Fake ID", id: 0, gamer: nil, req: "no gamer with id 0 in the Pool", success: false},
		testCase{caseName: "Center", id: 2, gamer: gamers[2-1], req: "", success: true},
		testCase{caseName: "Tail", id: 4, gamer: gamers[4-1], req: "", success: true},
		testCase{caseName: "Head", id: 1, gamer: gamers[1-1], req: "", success: true},
		testCase{caseName: "Last", id: 3, gamer: gamers[3-1], req: "", success: true},
	}

	cntr := len(gamers)
	for _, tc := range tt {
		t.Run(""+tc.caseName, func(t *testing.T) {
			removedGamer, err := pool.RmGamer(tc.id)
			if err == nil {
				cntr--
			}

			if tc.success == true && err != nil {
				t.Errorf("It was expected that RmGamer will return err=nil. got: %s", err)
			}
			if tc.success == false && err == nil {
				t.Errorf("It was expected that RmGamer will return err!=nil. got: %v", err)
			}
			if tc.success == false && err != nil && strings.Compare(err.Error(), tc.req) != 0 {
				t.Errorf("It was expected that RmGamer will return err: %q got: %q", tc.req, err)
			}
			if tc.success == false && removedGamer != nil {
				t.Errorf("It was expected that RmGamer will return nill gamer pointer, got: %v", removedGamer)
			}
			if tc.success == true && (removedGamer == nil || !reflect.DeepEqual(*removedGamer, *tc.gamer)) {
				t.Errorf("It was expected that RmGamer will return non nill pointer to a gamer: %v got %v", tc.gamer, removedGamer)
			}

			actualGamers := pool.ListGamers()
			if len(actualGamers) != cntr {
				t.Errorf("After RmGamer(%d): number of gamers in the pool should be %d, not %d", tc.id, cntr, len(actualGamers))
			}
		})
	}
}

// TestPoolGet - tests GetGamer function
func TestPoolGet(t *testing.T) {
	pool := NewGamersPool()
	defer pool.Release()

	//fill pool
	gamers := []*game.Gamer{
		&game.Gamer{Name: "Joe", Id: 1},
		&game.Gamer{Name: "Nick", Id: 2},
		&game.Gamer{Name: "jack", Id: 3},
		&game.Gamer{Name: "Fred", Id: 4},
	}

	for _, g := range gamers {
		if err := pool.AddGamer(g); err != nil {
			t.Fatalf("fail on AddGamer: %q ", err)
		}
	}

	type testCase struct {
		id       int
		caseName string
		gamer    *game.Gamer
		req      string
		success  bool
	}

	//get from different positions, from add's point of view.
	tt := []testCase{
		testCase{caseName: "Fake ID", id: 0, gamer: nil, req: "no gamer with id 0 in the Pool", success: false},
		testCase{caseName: "Center", id: 2, gamer: gamers[2-1], req: "", success: true},
		testCase{caseName: "Tail", id: 4, gamer: gamers[4-1], req: "", success: true},
		testCase{caseName: "Head", id: 1, gamer: gamers[1-1], req: "", success: true},
		testCase{caseName: "Last", id: 3, gamer: gamers[3-1], req: "", success: true},
	}

	for _, tc := range tt {
		t.Run(""+tc.caseName, func(t *testing.T) {
			gettedGamer, err := pool.GetGamer(tc.id)

			if tc.success == true && err != nil {
				t.Errorf("It was expected that GetGamer will return err=nil. got: %s", err)
			}
			if tc.success == false && err == nil {
				t.Errorf("It was expected that GetGamer will return err!=nil. got: %v", err)
			}
			if tc.success == false && err != nil && strings.Compare(err.Error(), tc.req) != 0 {
				t.Errorf("It was expected that GetGamer will return err: %q got: %q", tc.req, err)
			}
			if tc.success == false && gettedGamer != nil {
				t.Errorf("It was expected that GetGamer will return nill gamer pointer, got: %v", gettedGamer)
			}
			if tc.success == true && (gettedGamer == nil || !reflect.DeepEqual(*gettedGamer, *tc.gamer)) {
				t.Errorf("It was expected that RmGamer will return non nill pointer to a gamer: %v got %v", tc.gamer, gettedGamer)
			}

			removedGamer, _ := pool.RmGamer(tc.id)
			if !(removedGamer==nil && gettedGamer==nil) && (removedGamer==nil || gettedGamer==nil || !reflect.DeepEqual(*gettedGamer, *tc.gamer)){
				t.Errorf("It was expected that GetGamer will return pointer to the same value that RmGamer. got: %v, %v", gettedGamer, removedGamer)
			}
		})
	}
}

// TestPoolRelease - tests Release function
func TestPoolRelease(t *testing.T) {
	pool := NewGamersPool()

	//fill pool
	gamers := []*game.Gamer{
		&game.Gamer{Name: "Joe", Id: 1},
		&game.Gamer{Name: "Nick", Id: 2},
		&game.Gamer{Name: "jack", Id: 3},
		&game.Gamer{Name: "Fred", Id: 4},
	}

	for _, g := range gamers {
		if err := pool.AddGamer(g); err != nil {
			t.Fatalf("fail on AddGamer: %q ", err)
		}
	}

	//Release function should be the pretty fast action with closing of pool object
	dur := time.Duration(10) * time.Second
	c := make(chan interface{})
	go func(c chan<- interface{}) {
		pool.Release()
		_, ok := <-pool
		c <- ok
		close(c)
	}(c)

	select {
	case ok := <-c:
		if ok == true {
			t.Fatalf("It was expected that pool.Release() will shut down GamersPool object as chanel, but it's still alive")
		}
	case <-time.After(dur):
		t.Fatalf("It was expected that Release will return earler than %v duration", dur)
	}
}

// TestPoolJoinGame - tests JoinGame function
func TestPoolJoinGame(t *testing.T) {
	pool := NewGamersPool()
	defer pool.Release()

	//fill pool
	gamers := []*game.Gamer{
		&game.Gamer{Name: "Joe", Id: 1},
		&game.Gamer{Name: "Nick", Id: 2},
		&game.Gamer{Name: "jack", Id: 3},
		&game.Gamer{Name: "Fred", Id: 4},
		&game.Gamer{Name: "Izya", Id: 5},
	}

	for _, g := range gamers {
		if err := pool.AddGamer(g); err != nil {
			t.Fatalf("fail on AddGamer: %q ", err)
		}
	}

	actualGamers := pool.ListGamers()
	for _, g := range actualGamers {
		if g.InGame != nil {
			t.Errorf("Gamer has not nil InGame field before Join invokation: %v", g.InGame)
		}
	}

	//JoinGame for non exists gamer
	expect := "can't join a game: can't find gamer with id 0"
	if err := pool.JoinGame(0); err == nil || strings.Compare(err.Error(), expect) != 0 {
		t.Errorf("expect err=%q got: %q ", expect, err)
	}

	// each JoinGame should add one occupated InGame field among players of the pool
	cntrRequested := 0
	for _, g := range gamers {
		if err := pool.JoinGame(g.Id); err != nil {
			t.Errorf("fail on JoinGame: %q ", err)
		}
		cntrRequested++

		cntrJoined := 0
		actualGamers := pool.ListGamers()
		for _, g := range actualGamers {
			if g.InGame != nil {
				cntrJoined++
			}
		}
		if cntrRequested != cntrJoined {
			t.Errorf("Join requested %d times, succeed: %d times ", cntrRequested, cntrJoined)
		}
	}

	//JoinGame for occupied gamer
	expect = fmt.Sprintf("can't join a game: gamer %s already joined to another game", gamers[0])
	if err := pool.JoinGame(gamers[0].Id); err == nil || strings.Compare(err.Error(), expect) != 0 {
		t.Errorf("expect err: %q got: %q", expect, err)
	}

	//2.5 pairs of gamers should give 3 games
	games := make(map[game.Game]bool)
	actualGamers = pool.ListGamers()
	for _, g := range actualGamers {
		games[g.InGame] = true
	}
	if len(games) != int(math.Ceil(float64(len(gamers))/2.0)) {
		t.Errorf("%d gamers should give %d games. got: %d", len(gamers), int(math.Ceil(float64(len(gamers))/2.0)), len(games))
	}

	for _, g := range gamers {
		if err := pool.ReleaseGame(g.Id); err != nil {
			t.Errorf("fail on ReleaseGame: %q ", err)
		}
	}
}

// TestPoolReleaseGame - tests ReleaseGame function
func TestPoolReleaseGame(t *testing.T) {
	pool := NewGamersPool()
	defer pool.Release()

	//fill pool
	gamers := []*game.Gamer{
		&game.Gamer{Name: "Joe", Id: 1},
		&game.Gamer{Name: "Nick", Id: 2},
		&game.Gamer{Name: "jack", Id: 3},
		&game.Gamer{Name: "Fred", Id: 4},
		&game.Gamer{Name: "Izya", Id: 5},
	}

	for _, g := range gamers {
		if err := pool.AddGamer(g); err != nil {
			t.Fatalf("fail on AddGamer: %q ", err)
		}
		if err := pool.JoinGame(g.Id); err != nil {
			t.Fatalf("fail on JoinGame: %q ", err)
		}
	}

	//ReleaseGame of non exists gamer
	expect := "can't release a game: can't find gamer with id 0"
	if err := pool.ReleaseGame(0); err == nil || strings.Compare(err.Error(), expect) != 0 {
		t.Errorf("expect err=%q got: %q ", expect, err)
	}

	//each ReleaseGame should release 1 gamer's game
	gSBInGCnt := len(gamers)
	for _, g := range gamers {
		if err := pool.ReleaseGame(g.Id); err != nil {
			t.Errorf("fail on ReleaseGame: %q ", err)
		}
		gSBInGCnt--

		cntrJoined := 0
		actualGamers := pool.ListGamers()
		for _, g := range actualGamers {
			if g.InGame != nil {
				cntrJoined++
			}
		}

		if gSBInGCnt != cntrJoined {
			t.Errorf("Expected %d gamers in game, got: %d", gSBInGCnt, cntrJoined)
		}
	}
}
