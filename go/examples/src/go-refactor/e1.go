package main

import (
	"fmt"
)

func main() {
	// println(errors.New("abc").Error())
	println(fmt.Errorf("%s", "abc").Error())
}
