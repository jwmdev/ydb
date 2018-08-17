package ydb

var ydb = Ydb{}

// Ydb maintains rooms and connections
type Ydb struct {
	rooms map[roomname]*room
}

func getRoom(name roomname) *room {
	return ydb.rooms[name]
}
