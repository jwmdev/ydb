package main

import (
	"math/rand"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"
)

func createYdbTest(f func()) {
	dir := "_test"
	os.RemoveAll(dir)
	initYdb(dir)
	go setupWebsocketsListener(":9999")
	time.Sleep(time.Second)
	f()
	os.RemoveAll(dir)
}

// testGetRoom test if getRoom is safe for parallel access
func TestGetRoom(t *testing.T) {
	createYdbTest(func() {
		runTest := func(seed int, wg *sync.WaitGroup) {
			src := rand.NewSource(int64(seed))
			r := rand.New(src)
			var numOfTests uint64 = 10000
			var i uint64
			for ; i < numOfTests; i++ {
				roomname := roomname(strconv.FormatUint(r.Uint64()%numOfTests, 10))
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
