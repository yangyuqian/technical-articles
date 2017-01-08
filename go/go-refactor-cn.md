Go Refactor
--------------

Go开发者可以借助一些工具来实现类型安全的代码自动化重构，如“重命名”、“格式化”等，
这也是Go非常吸引我的地方.

自动化重构大体可分为2个层面：

* 生成类型安全的源码
* 去除自动化重构后多余的imports

本文希望对这些工具的用法以及实现的分析来探求它们的边界，借以回答
“某工具能做什么”这个看似简单实则非常复杂的问题.

# Eg

[Eg](golang.org/x/tools/cmd/eg)是一个依靠模版来进行定向代码重构的工具，
java中也有类似的工具.

修改模板中必须包含before和after两个function:

```
func before() ...
func after() ...
```

before和after的签名必须一致.

以下before和after的签名是一致的：

```
// 1.
func before(s string) error {...}
func after(s string) error {...}
// 2.
func before(s string) {...}
func after(s string) {...}
```

除了签名必须一致之外，before和after内的代码只能是合法的Go表达式，只允许有一行，
可以是return表达式，也可以不是return表达式，比如下面2种情况都是合法的：

```
// 1. 无return表达式
func before(s string) {
	fmt.Errorf("%s", s)
}

// 2. return表达式
func before(s string) error {
	return fmt.Errorf("%s", s)
}
```


