package main

import (
	"fmt"
)

type A struct {
	Name string
}

func main() {
	// println(errors.New("abc").Error())
	println(fmt.Errorf("%s", "abc").Error())
}
