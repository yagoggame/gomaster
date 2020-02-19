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
	"time"

	"github.com/yagoggame/gomaster/game"
)

// TestPoolFill - performs basic pool creation and fill test
func TestPoolFill(t *testing.T) {
	//testing data
	type testCase struct {
		caseName string
		gamer    *game.Gamer
		want     error
		success  bool
	}

	//fill pool with correct and not correct values
	tt := []testCase{
		testCase{caseName: "first", gamer: &game.Gamer{Name: "Joe", Id: 1}, want: nil, success: true},
		testCase{caseName: "second", gamer: &game.Gamer{Name: "Nick", Id: 2}, want: nil, success: true},
		testCase{caseName: "third", gamer: &game.Gamer{Name: "Fury", Id: 3}, want: nil, success: true},
		testCase{caseName: "same name", gamer: &game.Gamer{Name: "Fury", Id: 4}, want: nil, success: true},
		testCase{caseName: "same id", gamer: &game.Gamer{Name: "Sam", Id: 4}, want: IdOccupiedError, success: false},
		testCase{caseName: "nil", gamer: nil, want: NilGamerError, success: false},
		testCase{caseName: "fifth", gamer: &game.Gamer{Name: "Jack", Id: 5}, want: nil, success: true},
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
				t.Errorf("AddGamer, error:\ngot: %v,\nwant: err=nil.", err)
			case tc.success == false && !errors.Is(err, tc.want):
				t.Errorf("AddGamer, error:\ngot: %v,\nwant: err=%v.", err, tc.want)
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
			t.Errorf("gamers in pool:\nwant: %d.\ngot: %d", cntr, len(actualGamers))
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
		want     error
		success  bool
	}

	//rm from different positions, from add's point of view.
	tt := []testCase{
		testCase{caseName: "Fake ID", id: 0, gamer: nil, want: IdNotFoundError, success: false},
		testCase{caseName: "Center", id: 2, gamer: gamers[2-1], want: nil, success: true},
		testCase{caseName: "Tail", id: 4, gamer: gamers[4-1], want: nil, success: true},
		testCase{caseName: "Head", id: 1, gamer: gamers[1-1], want: nil, success: true},
		testCase{caseName: "Last", id: 3, gamer: gamers[3-1], want: nil, success: true},
	}

	cntr := len(gamers)
	for _, tc := range tt {
		t.Run(""+tc.caseName, func(t *testing.T) {
			removedGamer, err := pool.RmGamer(tc.id)
			if err == nil {
				cntr--
			}

			switch tc.success {
			case true:
				if err != nil {
					t.Errorf("RmGamer, err:\ngot: %v,\nwant: err=nil.", err)
				}
				if removedGamer == nil || !reflect.DeepEqual(*removedGamer, *tc.gamer) {
					t.Errorf("RmGamer, gamer:\nwant: %v,\ngot %v", tc.gamer, removedGamer)
				}
			case false:
				if !errors.Is(err, tc.want) {
					t.Errorf("RmGamer, err:\ngot: %v,\nwant: err=%v.", err, tc.want)
				}
				if removedGamer != nil {
					t.Errorf("RmGamer, gamer:\nwant nill gamer pointer,\ngot: %v", removedGamer)
				}
			}

			actualGamers := pool.ListGamers()
			if len(actualGamers) != cntr {
				t.Errorf("RmGamer, num of gamers in pool:\nwant: %d,\ngot: %d", cntr, len(actualGamers))
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
		want     error
		success  bool
	}

	//get from different positions, from add's point of view.
	tt := []testCase{
		testCase{caseName: "Fake ID", id: 0, gamer: nil, want: IdNotFoundError, success: false},
		testCase{caseName: "Center", id: 2, gamer: gamers[2-1], want: nil, success: true},
		testCase{caseName: "Tail", id: 4, gamer: gamers[4-1], want: nil, success: true},
		testCase{caseName: "Head", id: 1, gamer: gamers[1-1], want: nil, success: true},
		testCase{caseName: "Last", id: 3, gamer: gamers[3-1], want: nil, success: true},
	}

	for _, tc := range tt {
		t.Run(""+tc.caseName, func(t *testing.T) {
			gettedGamer, err := pool.GetGamer(tc.id)

			switch tc.success {
			case true:
				if err != nil {
					t.Errorf("GetGamer, err:\ngot: %v,\nwant: err=nil.", err)
				}
				if gettedGamer == nil || !reflect.DeepEqual(*gettedGamer, *tc.gamer) {
					t.Errorf("GetGamer, gamer:\nwant: %v,\ngot %v", tc.gamer, gettedGamer)
				}
			case false:
				if !errors.Is(err, tc.want) {
					t.Errorf("GetGamer, err:\ngot: %v,\nwant: err=%v.", err, tc.want)
				}
				if gettedGamer != nil {
					t.Errorf("GetGamer, gamer:\nwant nill gamer pointer,\ngot: %v", gettedGamer)
				}
			}

			removedGamer, _ := pool.RmGamer(tc.id)
			if !(removedGamer == nil && gettedGamer == nil) && (removedGamer == nil || gettedGamer == nil || !reflect.DeepEqual(*gettedGamer, *tc.gamer)) {
				t.Errorf("GetGamer and RmGamer rezult:\nwant: same gamer\ngot: %v, %v", gettedGamer, removedGamer)
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
			t.Fatalf("want: pool.Release() must shut down GamersPool object as chanel,\ngot: chanel alive")
		}
	case <-time.After(dur):
		t.Fatalf("want: Release must return earler than %v duration,\ngot: duration expired", dur)
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
			t.Errorf("Gamer.InGame:\nwant:nil,\ngot:%v", g.InGame)
		}
	}

	//JoinGame for non exists gamer
	want := IdNotFoundError
	if err := pool.JoinGame(0); !errors.Is(err, want) {
		t.Errorf("JoinGame fake id:\nwant: err=%v\ngot: %v ", want, err)
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
			t.Errorf("Join, num of success:\nwant:%d\ngot: %d", cntrRequested, cntrJoined)
		}
	}

	//JoinGame for occupied gamer
	want = GamerOccupiedError
	if err := pool.JoinGame(gamers[0].Id); !errors.Is(err, want) {
		t.Errorf("JoinGame occupied gamer:\nwant err: %v,\ngot: %v", want, err)
	}

	//2.5 pairs of gamers should give 3 games
	games := make(map[game.Game]bool)
	actualGamers = pool.ListGamers()
	for _, g := range actualGamers {
		games[g.InGame] = true
	}
	if len(games) != int(math.Ceil(float64(len(gamers))/2.0)) {
		t.Errorf("number of games for %d gamers:\nwant: %d,\ngot %d", len(gamers), int(math.Ceil(float64(len(gamers))/2.0)), len(games))
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
	want := IdNotFoundError
	if err := pool.ReleaseGame(0); !errors.Is(err, want) {
		t.Errorf("ReleaseGame fake id:\nwant: %v,\ngot: %v ", want, err)
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
			t.Errorf("Gamers in game:\nwant: %d,\ngot: %d", gSBInGCnt, cntrJoined)
		}
	}
}
