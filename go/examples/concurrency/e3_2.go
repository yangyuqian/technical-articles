package main

import (
	"sync"
)

func main() {
	wg := new(sync.WaitGroup)
	wg.Add(1)

	go func(signal *sync.WaitGroup) {
		signal.Done()
	}(wg)

	wg.Wait()
}
