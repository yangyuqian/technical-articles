package main

import (
	"sync"
)

func main() {
	wg := new(sync.WaitGroup)

	wg.Add(1)
	wg.Done()
	wg.Wait()
}
