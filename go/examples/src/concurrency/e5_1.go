package main

func abstractListener(fxChan chan func() string) {
	fxChan <- func() string {
		return "Hello, functions in channel"
	}
}

func main() {
	fxChan := make(chan func() string)
	defer close(fxChan)

	go abstractListener(fxChan)
	select {
	case rfx := <-fxChan:
		msg := rfx()
		println("recieved message:", msg)
	}
}
