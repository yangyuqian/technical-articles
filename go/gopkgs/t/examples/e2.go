package main

import (
	"fmt"
	"os"
	"text/template"
)

func main() {
	tmpl := `Hello {{ $b := 1 }} {{if eq .Index $b}} {{.Name1}} {{.Name2}} <% .Name2 %> {{else}} {{ .Name1 }} {{end}}, text/template!`
	t := template.Must(template.New("e1").Delims("<%", "%>").Parse(tmpl))

	if err := t.Execute(os.Stdout, &struct {
		Name1 string
		Name2 string
		Index int
		OK    bool
	}{Name1: "Gopher1", Name2: "Gopher2", Index: 1}); err != nil {
		fmt.Println("Unexpected error:", err)
	}
}
