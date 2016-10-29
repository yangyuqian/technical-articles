package main

import ()

func main() {
	tmpl := `SELECT id AS id
			 FROM $table WHERE xxx GROUP BY xxx;`
	println(tmpl)
}
