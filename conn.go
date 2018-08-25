package main

type conn interface {
	// sends data to the client
	Write(m []byte) (n int, err error)
}
