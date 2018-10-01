package main

import (
	"sync"
	"testing"
	"time"
)

const testEndpoint = ":9999"
const testroom = "testroom"

func createTestClient() (client *client) {
	client = newClient()
	client.Connect("ws://localhost:9999/ws")
	client.Subscribe(subDefinition{testroom, 0})
	return
}

func TestClientSubscribeUpdate(t *testing.T) {
	createYdbTest(func() {
		p := 247
		runTest := func(seed int, wg *sync.WaitGroup) {
			client := createTestClient()
			client.UpdateRoom(testroom, []byte{byte(seed)})
			client.WaitForConfs()
			for {
				roomdata := client.rooms[testroom]
				if len(roomdata.data) == p {
					break
				}
				time.Sleep(time.Millisecond * 100)
			}
			wg.Done()
		}
		wg := new(sync.WaitGroup)
		wg.Add(int(p))
		for i := 0; i < p; i++ {
			go runTest(i, wg)
		}
		wg.Wait()
	})
}

/*



func TestClientSubscribeUpdate(t *testing.T) {
	dir := "_ydb_conn_test"
	os.RemoveAll(dir)
	initYdb(dir)
	runSocketListener(":9999")
	client1 := createTestClient()
	client2 := createTestClient()
	client1.update(testroom, "1")
	client2.update(testroom, "2")
	waitForConfs(client1, client2)
	if client1.get(testroom) != client2.get(testroom) || len(client1.get(testroom)) != 2 {
		t.Errorf("Expecting both clients to have the same content")
	}
	os.RemoveAll(dir)
}


func initTestClients(numberOfClients int, f func(clients []*client)) {
	dir := "_ydb_conn_test"
	os.RemoveAll(dir)
	initYdb(dir)
	setupWebsocketsListener(":9999")
	conns := make([]*client, numberOfClients)
	for i := 0; i < numberOfClients; i++ {
		c := newClient()
		c.connect(testEndpoint)
		c.subscribe("test", 0)
		conns[i] = c
	}
	f(conns)
	os.RemoveAll(dir)
}

func TestWriteRoomDataAfterSubscription(t *testing.T) {
	initTestClients(2, func(clients []*client) {
		clients[0].updateRoomData("test", []byte{7})
		waitForConnConfs(clients...)
		compareClients(t, clients[0], conns[1])
		if len(clients[0].roomData["test"]) != 1 || conns[0].roomData["test"][0] != 7 {
			t.Error("not expected result")
		}
	})
}
*/
