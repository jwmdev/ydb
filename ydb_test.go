package main

import (
	"math/rand"
	"os"
	"sync"
	"testing"
)

func createYdbTest(f func()) {
	dir := "_test"
	os.RemoveAll(dir)
	initYdb(dir)
	f()
	os.RemoveAll(dir)
}

// testGetRoom test if getRoom is safe for parallel access
func TestGetRoom(t *testing.T) {
	createYdbTest(func() {
		runTest := func(seed int, wg *sync.WaitGroup) {
			src := rand.NewSource(int64(seed))
			r := rand.New(src)
			var numOfTests uint32 = 10000
			var i uint32
			for ; i < numOfTests; i++ {
				roomname := roomname(r.Uint32() % numOfTests)
				getRoom(roomname)
			}
			wg.Done()
		}
		p := 100
		wg := new(sync.WaitGroup)
		wg.Add(p)
		for i := 0; i < p; i++ {
			go runTest(i, wg)
		}
		wg.Wait()
	})
}
