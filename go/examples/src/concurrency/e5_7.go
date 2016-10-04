package main

func main() {
	c1 := make(chan string, 1)

	c1 <- "send to channel No. 1"

	select {
	case m := <-c1:
		println(m)
	}
}
