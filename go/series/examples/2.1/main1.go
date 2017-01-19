package main

import (
	"net/http"
)

type GreetHandler struct {
	Name string
}

func (h *GreetHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("GreetHandler: " + h.Name))
}

func main() {
	http.HandleFunc("/p1", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`hello http`))
	})

	http.ListenAndServe(":8080", &GreetHandler{"Da Yang Yu"})
}
