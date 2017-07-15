package main

import (
	"flag"
	"github.com/juju/ratelimit"
	"time"
)

var (
	interval uint
	capacity int64
	quantum  int64
	duration uint
)

func init() {
	flag.UintVar(&interval, "i", 1000, "set token bucket interval in milliseconds")
	flag.Int64Var(&capacity, "c", 10, "set token bucket capacity")
	flag.Int64Var(&quantum, "q", 1, "set token bucket quantum")
	flag.UintVar(&duration, "d", 10, "set duration to run this example")
}

func main() {
	bucket := ratelimit.NewBucketWithQuantum(time.Duration(interval)*time.Millisecond, capacity, quantum)
	for i := 0; i < 500; i++ {
		go func() {
			for {
				bucket.Wait(1)
				print(".")
			}
		}()
	}

	<-time.After(time.Duration(duration) * time.Second)
}
