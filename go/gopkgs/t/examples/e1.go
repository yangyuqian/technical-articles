package main

import (
	"os"
	"text/template"
)

func main() {
	tmpl := `Hello {{.Name}}, text/template!`
	t, _ := template.New("e1").Parse(tmpl)

	// 输出到一个buffer，之后可以b.String()获取到渲染后的内容
	// import "bytes"
	// b := bytes.NewBuffer([]byte{})
	// t.Execute(b, &struct{ Name string }{Name: "Gopher"})
	// 这里为了简单起见，直接输出到stdout
	t.Execute(os.Stdout, &struct{ Name string }{Name: "Gopher"})
}
