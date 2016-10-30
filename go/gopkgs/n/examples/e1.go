package main

import (
	"net/http"
)

func main() {
	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`Hello, net/http`))
	})

	http.ListenAndServe(":7000", nil)
}
