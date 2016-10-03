package main

import (
	"math/rand"
	"sync"
	"time"
)

var shared = struct {
	*sync.RWMutex
	count int
}{}

var wg *sync.WaitGroup

const N = 10

func main() {
	rand.Seed(time.Now().Unix())
	shared.RWMutex = new(sync.RWMutex)
	wg = new(sync.WaitGroup)
	wg.Add(2 * N)
	defer wg.Wait()

	for i := 0; i < N; i++ {
		// write goroutines
		go func(ii int) {
			shared.Lock()
			duration := rand.Intn(5)
			// shared.Lock()
			time.Sleep(time.Duration(duration) * time.Second)
			println(ii, "write --- shared.count")
			// shared.Unlock()
			shared.Unlock()

			wg.Done()
		}(i)

		// read goroutines
		go func(ii int) {
			shared.RLock()
			println(ii, "read --- shared.count")
			shared.RUnlock()

			wg.Done()
		}(i)

	}
}
