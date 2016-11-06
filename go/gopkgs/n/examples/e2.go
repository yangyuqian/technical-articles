package main

import (
	"net/http"
)

func main() {
	http.HandleFunc("/x1", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello, x1"))
	})

	http.HandleFunc("/x1/x2/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello, /x1/x2/"))
	})

	http.HandleFunc("/x2", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello, x2"))
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello, root"))
	})

	http.HandleFunc("h1/x1", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello, x1"))
	})

	http.ListenAndServe(":8080", nil)
}
