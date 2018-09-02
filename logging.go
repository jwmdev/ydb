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
