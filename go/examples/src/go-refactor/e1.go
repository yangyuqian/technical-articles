package main

import (
	"fmt"
)

func main() {
	println(fmt.Errorf("%s", "abc").Error())
}
