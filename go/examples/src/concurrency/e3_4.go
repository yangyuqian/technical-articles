package main

import (
	"fmt"
	"math/rand"
	"runtime"
	"sync"
	"time"
)

var balance int
var transactionNo int

func main() {
	rand.Seed(time.Now().Unix())
	runtime.GOMAXPROCS(2)
	var wg sync.WaitGroup
	balance = 1000
	transactionNo = 0
	fmt.Println("Starting balance: $", balance)
	wg.Add(100)
	for i := 0; i < 100; i++ {
		go func(ii int, sig *sync.WaitGroup) {
			transactionAmount := rand.Intn(25)
			transaction(transactionAmount)
			wg.Done()
		}(i, &wg)
	}
	wg.Wait()
	fmt.Println("Final balance: $", balance)
}

func transaction(amt int) bool {
	approved := false
	if (balance - amt) < 0 {
		approved = false

	} else {
		approved = true
		balance = balance - amt

	}
	approvedText := "declined"
	if approved == true {
		approvedText = "approved"

	} else {

	}
	transactionNo = transactionNo + 1
	fmt.Println(transactionNo, "Transaction for $", amt, approvedText)
	fmt.Println("\tRemaining balance $", balance)
	return approved

}
