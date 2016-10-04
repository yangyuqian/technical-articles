package main

import "fmt"

func main() {
	message := make(chan string)
	count := 3

	go func() {
		for i := 1; i <= count; i++ {
			message <- fmt.Sprintf("message %d", i)
		}

		close(message)
	}()

	for msg := range message {
		fmt.Println(msg)
	}
}
