package main

import "time"

func main() {
	c := make(chan string)

	go func() {
		time.Sleep(2 * time.Second)
		close(c)
	}()

	println("recieved:", <-c)
	println("recieved:", <-c)
	println("recieved:", <-c)
	println("recieved:", <-c)
}
