package main

func main() {
	defer func() {
		if err := recover(); err != nil {
			println("unexpected panic:", err)
		}
	}()

	panic("hello, recover")
}
