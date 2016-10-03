package main

var shared = struct {
	count int
}{}

func main() {
	for i := 0; i < 100; i++ {
		go func() {
			shared.count++
		}()
	}
}
