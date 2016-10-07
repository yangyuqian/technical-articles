package main

import "sync"

/*
	Channel中可以传递指针类型，而且不同的goroutine之间其实是共享栈空间的，
	也就意味着指针在不同routine之间是等价的.
*/

func main() {
	wg := new(sync.WaitGroup)
	ptrChan := make(chan *string)
	wg.Add(2)

	go func() {
		ptrStr := "Hello, Channel"
		ptrChan <- &ptrStr
		println("Send:", ptrStr)
		wg.Done()
	}()

	go func() {
		select {
		case ptrStr := <-ptrChan:
			println("Recieved:", *ptrStr)
		}

		wg.Done()
	}()
	wg.Wait()
}
