package main

import (
	"time"
)

func main() {
	c := make(chan bool, 1)
	defer close(c)

	go func() {
		time.Sleep(time.Duration(2) * time.Second)
		c <- true
		println("Finished in deadline")
	}()

	select {
	case <-time.After(time.Duration(10) * time.Second):
		println("Timeout")
		close(c)
	}

}
