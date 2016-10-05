package main

import (
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"time"
)

func main() {
	runtime.GOMAXPROCS(1)
	println("logic CPU:", runtime.NumCPU())
	http.HandleFunc("/", Hello)
	http.ListenAndServe(":8080", nil)
}

func Hello(w http.ResponseWriter, r *http.Request) {
	time.Sleep(2 * time.Second)
}
