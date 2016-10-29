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
最终模板被解析成多叉树存放的，例1中的`Node`内容为:

```
                          +----------+
     +--------------------+   Root   +----------------------------+
     |                    |(ListNode)|                            |
     |                    +-----+----+                            |
     |                          |                                 |
     |                          |                                 |
     |                          |                                 |
     |                          |                                 |
+----v-----+             +------v------+              +-----------v--------+
| "Hello " |             |"{{ .Name }}"|              | ", text/template!" |
|(TextNode)|             | (ActionNode)|              |    (TextNode)      |
+----------+             +-------------+              +--------------------+
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

渲染算法对解析出来的多叉树结构的递归遍历：

```
                          +----------+
     +--------------------+   Root   +----------------------------+
     |                    |(ListNode)|                            |
     |                    +-----+----+                            |
     |                          |                                 |
     |                          |                                 |
     |                          |                                 |
     |                          |                                 |
+----v-----+             +------v------+              +-----------v--------+
| "Hello " |             |"{{ .Name }}"|              | ", text/template!" |
|(TextNode)|             | (ActionNode)|              |    (TextNode)      |
+----+-----+             +------+------+              +-----------+--------+
     |                          |                                 |
     |                          |                                 |
     |                          |                                 |
     |                          |                                 |
+----v--------------------------v---------------------------------v--------+
| "Hello "                 "Gopher"                    ", text/template!"  |
+--------------------------------------------------------------------------+
```

例1中的2种类型的节点的渲染算法实现：

```
// src/text/template/exec.go#L220
func (s *state) walk(dot reflect.Value, node parse.Node) {
	s.at(node)
	switch node := node.(type) {
  // 非控制节点(if/else/range)，比如 {{ .Name }}
	case *parse.ActionNode:
		val := s.evalPipeline(dot, node.Pipe)
		if len(node.Pipe.Decl) == 0 {
			s.printValue(node, val)
		}
	case *parse.IfNode:
		s.walkIfOrWith(parse.NodeIf, dot, node.Pipe, node.List, node.ElseList)
  // 模板树非叶子节点，遍历子节点
	case *parse.ListNode:
		for _, node := range node.Nodes {
			s.walk(dot, node)
		}
	case *parse.RangeNode:
		s.walkRange(dot, node)
	case *parse.TemplateNode:
		s.walkTemplate(dot, node)
  // 文本节点，比如例1中的"Hello "，直接输出到输出流
	case *parse.TextNode:
		if _, err := s.wr.Write(node.Text); err != nil {
			s.writeError(err)
		}
	case *parse.WithNode:
		s.walkIfOrWith(parse.NodeWith, dot, node.Pipe, node.List, node.ElseList)
	default:
		s.errorf("unknown node: %s", node)
	}
}
```

文本节点渲染比较直接：

```
	case *parse.TextNode:
		if _, err := s.wr.Write(node.Text); err != nil {
			s.writeError(err)
		}
```

ActionNode的渲染，这里只介绍`{{ .Name }}`(pipeline)的渲染：

```
// src/text/template/exec.go#L536
func (s *state) evalField(dot reflect.Value, fieldName string, node parse.Node, args []parse.Node, final, receiver reflect.Value) reflect.Value {
	if !receiver.IsValid() {
		return zero
	}
	typ := receiver.Type()
	receiver, isNil := indirect(receiver)
	ptr := receiver
	if ptr.Kind() != reflect.Interface && ptr.CanAddr() {
		ptr = ptr.Addr()
	}
  // 先尝试调用`Name(...)` method
	if method := ptr.MethodByName(fieldName); method.IsValid() {
		return s.evalCall(dot, method, node, fieldName, args, final)
	}
  // 走到这里说明不是method，但发现有参数，如{{ .Name a b }}
	hasArgs := len(args) > 1 || final.IsValid()
	switch receiver.Kind() {
  // 判断输入的数据是不是struct，是的话就调用data.Name
	case reflect.Struct:
		tField, ok := receiver.Type().FieldByName(fieldName)
		if ok {
			if isNil {
				s.errorf("nil pointer evaluating %s.%s", typ, fieldName)
			}
			field := receiver.FieldByIndex(tField.Index)
			if tField.PkgPath != "" { // field is unexported
				s.errorf("%s is an unexported field of struct type %s", fieldName, typ)
			}
			if hasArgs {
				s.errorf("%s has arguments but cannot be invoked as function", fieldName)
			}
			return field
		}
  // 如果是map, 就调用data[Name]
	case reflect.Map:
		if isNil {
			s.errorf("nil pointer evaluating %s.%s", typ, fieldName)
		}
		nameVal := reflect.ValueOf(fieldName)
		if nameVal.Type().AssignableTo(receiver.Type().Key()) {
			if hasArgs {
				s.errorf("%s is not a method but has arguments", fieldName)
			}
			result := receiver.MapIndex(nameVal)
			if !result.IsValid() {
				switch s.tmpl.option.missingKey {
				case mapInvalid:
				case mapZeroValue:
					result = reflect.Zero(receiver.Type().Elem())
				case mapError:
					s.errorf("map has no entry for key %q", fieldName)
				}
			}
			return result
		}
	}
	s.errorf("can't evaluate field %s in type %s", fieldName, typ)
	panic("not reached")
}
```

例2 `examples/e2.go`是更复杂的例子，包含更复杂的树节点类型

```
	tmpl := `Hello {{ $b := 1 }} {{if eq .Index $b}} {{.Name1}} {{else}} {{ .Name2 }} {{end}}, text/template!`
```

例2中模板解析为5个`Node`:

|节点内容|节点类型|解释|
|--------|--------|----|
|"Hello "|Text||
|"{{$b := 1}}"|Action||
|" "|Text||
|{{if eq .Index $b}} {{.Name1}} {{else}} {{.Name2}} {{end}}|If||
|", text/template!"|Text||
