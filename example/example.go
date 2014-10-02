package main

import (
	"fmt"
	"runtime"
	"time"

	"github.com/cespare/memstats"
)

func main() {
	go func() {
		var diff memstats.Diff
		stats := new(memstats.Stats)
		stats.Collect()
		for _ = range time.Tick(5 * time.Second) {
			stats.Collect()
			if stats.ReadDiff(&diff) {
				fmt.Println(&diff)
			}
			runtime.GC()
		}
	}()

	for {
		c := make([]byte, 100)
		_ = c
		time.Sleep(10 * time.Millisecond)
	}
}
