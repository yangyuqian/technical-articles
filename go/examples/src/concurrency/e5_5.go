package main

func main() {
	c := make(chan string, 1)
	c <- "send message!"
	println(<-c)
}
