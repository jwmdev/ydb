package main

import (
	"fmt"
	"os"
)

func exitBecause(messages ...string) {
	for _, m := range messages {
		fmt.Fprintln(os.Stderr, m)
	}
	os.Exit(1)
}

func debug(s string) {
	fmt.Println(s)
}

func debugMessageType(m string, buf []byte) {
	mtype := "unknown"
	switch buf[0] {
	case messageConfirmation:
		mtype = "confirmation"
	case messageSub:
		mtype = "subscription"
	case messageSubConf:
		mtype = "subscription confirmation"
	case messageUpdate:
		mtype = "update"
	case messageHostUnconfirmedByClient:
		mtype = "host-unconfirmed-by-client"
	case messageConfirmedByHost:
		mtype = "confirmed-by-host"
	}
	fmt.Printf("%s (type: %s, len: %d)\n", m, mtype, len(buf))
}
