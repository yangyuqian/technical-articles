package main

import (
	"fmt"
	"golang.org/x/net/context"
)

type TransactionAwareContext interface {
	context.Context
	Read(interface{}) error
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		cancel()
	}()

	select {
	case <-ctx.Done():
		fmt.Println("Err:", ctx.Err())
		fmt.Println("Value:", ctx.Value("xxx"))
	}

	select {
	case <-ctx.Done():
		fmt.Println("Err:", ctx.Err())
		fmt.Println("Value:", ctx.Value("xxx"))
	}
}
