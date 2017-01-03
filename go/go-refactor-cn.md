Go Refactor
--------------

自动化重构大体可分为2个步骤：

* 通过模板自动化生成类型安全的重构代码
* 通过goimports去除自动化重构后多余的imports

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


