package main

import (
	"net/http"
)

func GreetHandlerFunc(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("GreetHandlerFunc"))
}

func main() {
	http.HandleFunc("/p1", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`hello http`))
	})

	http.ListenAndServe(":8080", http.HandlerFunc(GreetHandlerFunc))
}
