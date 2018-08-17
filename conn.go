package ydb

type conn interface {
	readMessage() (m []byte, err error)
	// sends data to the client
	Write(m []byte) (n int, err error)
}
