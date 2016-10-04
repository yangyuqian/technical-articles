package main

type Animal interface {
	Eat()
}

type Cat struct{}

func (c *Cat) Eat() {
	println("Cat eat fish")
}

func main() {
	iChan := make(chan Animal)
	go func(aChan chan Animal) {
		cat := new(Cat)
		aChan <- cat
	}(iChan)

	select {
	case animal := <-iChan:
		animal.Eat()
	}
}
