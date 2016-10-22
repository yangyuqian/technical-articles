text/template
----------------

`text/template`是Go中处理模板渲染的其中一个包，支持普通文本、HTML、JS文本的渲染.
除此之外，还有[html/template](h/html_template.md)，
对`text/template`进行了一层封装，支持前端的模板渲染.

先看一个的例子，例1: examples/e1.go

```
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
```

例1中输入了一个模板`Hello {{.Name}}, text/template!`，然后传入一个对象，
渲染出来一段文字`Hello Gopher, text/template!`. 大致可以看到3个步骤：

* 模板定义
* 模板解析
* 模板渲染

首先看一下模板定义的实现：

```
// src/text/template/template.go#L53
func (t *Template) New(name string) *Template {
  // 初始化package全局可见的template.common(私有变量)，
  // 包含一些自定义的扩展，比如全局的模板库，解析函数，渲染函数
	t.init()
  // 创建并返回一个Template对象
	nt := &Template{
    // name是模板唯一标识
		name:       name,
    // t.init()的时候初始化的就是t.common
		common:     t.common,
    // 定义模板中的标志符，默认是{{ ... }}
		leftDelim:  t.leftDelim,
		rightDelim: t.rightDelim,
	}
	return nt
}
```




