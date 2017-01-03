package main

import (
	"errors"
	"fmt"
)

func before(s string) {
	fmt.Errorf("%s", s)
}

func after(s string) {
	errors.New(s)
}
