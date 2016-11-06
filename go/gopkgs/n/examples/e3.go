package main

import (
	"io/ioutil"
	"net/http"
)

func main() {
	c := http.DefaultClient
	req, _ := http.NewRequest("GET", "http://www.example.com", nil)
	req.Header.Set("Host", "news.baidu.com")
	req.Host = "news.baidu.com"

	resp, _ := c.Do(req)
	defer resp.Body.Close()

	data, _ := ioutil.ReadAll(resp.Body)
	println(string(data))
}
