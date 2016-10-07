package main

func main() {
	var nilChan chan string
	println(<-nilChan)
}
