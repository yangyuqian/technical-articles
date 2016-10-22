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
// src/text/template/template.go#L37
func New(name string) *Template {
	t := &Template{
		name: name,
	}
  
  // 初始化package全局可见的template.common(私有变量)，
  // 包含一些自定义的扩展，比如全局的模板库，解析函数，渲染函数
	t.init()
	return t
}
```

接着看模板解析实现：

```
// src/text/template/template.go#L187
func (t *Template) Parse(text string) (*Template, error) {
  // 和 template.
	t.init()
  // 加读锁，当共享数据被修改(写)时阻塞
	t.muFuncs.RLock()
  // 解析模板内容，返回map[string]*Tree => trees
	trees, err := parse.Parse(t.name, text, t.leftDelim, t.rightDelim, t.parseFuncs, builtins)
  // 释放读锁
	t.muFuncs.RUnlock()
	if err != nil {
		return nil, err
	}
  // 修改现有模板对象中模板树
	for name, tree := range trees {
		if _, err := t.AddParseTree(name, tree); err != nil {
			return nil, err
		}
	}
	return t, nil
}
```

调用`Template.Parse`时会创建一个`goroutine`来并发解析模板文本，传给`Tree.parse`.
最终模板被解析成数组存放的`Node`对象，例1中的`Node`内容为:

```
["Hello ", "{{.Name}}", ", text/template!"]
```

`Node`实现了`Type()`，返回其实际类型，例1中 `Hello `的类型为`text`，
而`{{.Name}}`为`NodeAction`，
即`non-control action such as a field evaluation.`，类型都是`iota`常量.

模板解析的`Node`还有其他类型，主要影响渲染过程，后面会一一结合实例介绍.

最后看模板的渲染:

```
// src/text/template/exec.go#L174
func (t *Template) Execute(wr io.Writer, data interface{}) error {
	return t.execute(wr, data)
}

// 这里要求判断传入的data是对象的指针
func (t *Template) execute(wr io.Writer, data interface{}) (err error) {
	defer errRecover(&err)
	value := reflect.ValueOf(data)
  // 初始化模板渲染状态机
	state := &state{
		tmpl: t,
		wr:   wr,
		vars: []variable{{"$", value}},
	}
	if t.Tree == nil || t.Root == nil {
		state.errorf("%q is an incomplete or empty template%s", t.Name(), t.DefinedTemplates())
	}
  // 遍历整个模板解析数
	state.walk(value, t.Root)
	return
}
```

然后通过反射取得具体得字段内容，填充`{{ .Name }}`

```

```
