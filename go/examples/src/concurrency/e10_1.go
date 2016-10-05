package main

import "time"

func main() {
	timer := time.NewTimer(2 * time.Second)

	defer println("timeout")

	for {
		select {
		case <-timer.C:
			return
		}
	}
}
