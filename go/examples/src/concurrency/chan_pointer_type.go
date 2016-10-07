package main

import "sync"

/*
	Channel中可以传递指针类型，而且不同的goroutine之间其实是共享栈空间的，
	也就意味着指针在不同routine之间是等价的.

	在Channel中传递指针可行但不推荐，因为Channel的目的在于减少不必要的共享内存，
	即goroutine A中直接修改goroutine B中正在使用的内存，如果在channel中传递了
	指针，性能会变得更差，但却没有带来额外的好处，和传统的多线程编程就没有区别了
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
