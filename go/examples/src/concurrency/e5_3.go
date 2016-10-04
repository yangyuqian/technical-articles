package main

type Messager struct {
	msg chan string
}

func (m *Messager) init() {
	m.msg = make(chan string)
}

func (m *Messager) Send(msg string) {
	m.msg <- msg
}

func (m *Messager) Relay() chan string {
	return m.msg
}

func main() {
	m := new(Messager)
	m.init()

	go m.Send("Hello, Channel")

	println(<-m.Relay())
}
