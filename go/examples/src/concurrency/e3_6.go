package main

import (
	"net/http"
	"net/http/pprof"
	"time"
)

func main() {
	http.HandleFunc("/", Hello)
	http.HandleFunc("/debug/xxx", pprof.Index)
	http.ListenAndServe(":8080", nil)
}

func Hello(w http.ResponseWriter, r *http.Request) {
	time.Sleep(20 * time.Second)
}
