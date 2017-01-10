Getting Startted - Part1
------------------------

纲要:

* Go简介
* Go开发环境搭建
* 基于vim开发Go
* Hello World

# Go简介

Go是一门由Google的工程师发起和实现的编程语言.
起初是为了解决Google公司面临的几个问题：

* C/C++开发中依赖管理复杂度高，带来了额外的开发和运维
* C/C++编译速度很慢，对大型软件迭代速度影响较大
* 高并发、分布式软件的开发缺乏简单可靠的模型支持

在Go1.4及以前的版本，编译器由C实现；Go1.5实现了自举，即采用Go来实现编译器.

Go通过将所有的运行时依赖打包到一个binary中极大的降低了运维复杂度.
同时先进的依赖管理模型极大地提升了软件编译速度.

由于Go1.4及之前的版本存在明显的缺陷，本课中假设大家用的是Go1.5及之后的版本，
只介绍Go1.5及之后版本的环境搭建.

# Go开发环境搭建

值得一提的是，与Java之类的语言不同，Go编译后的binary是可以直接运行的，所以就
不存在“Go运行时环境”的说法了，只有”开发环境“.

Go的开发环境分为2个部分：

* 标准库(源码)以及一些原生的命令行工具
* 第三方依赖

标准库由Go和汇编代码组成，包含了软件编译、运行时必须的依赖，如I/O、数据类型定义等.
而第三方以来包含一些非Go官方提供的软件依赖.

先来看一段简单的代码

```
// 例1
package main

import (
  "fmt"
)

func main() {
  println("hello")
  fmt.Println("hello")
}
```

这里首先通过import声明了对标准库中fmt包的依赖，然后调用了`fmt.Println`函数，
另外还调用了`println`函数，后者是编译器提供的“内置”函数.

例1中的文本之所以能转换成一段可以执行的代码，需要2部分东西：

* Go编译器: 目前只有一种选择，即Go Tools，是一组命令行工具
* 标准库源码文件

一个完整可用的开发环境如图1所示：

```

```

图1. 完整可用的Go开发环境

# 基于vim开发Go

目前Go开发有很多免费的IDE可供选择，如vim, emacs, Sublime, Ecplise, IntelliJ等.

可以根据自己的喜好来选择IDE.


# Hello World

下面进入本课的尾声，一个传统的Hello World程序：

```
package main

import (
  "fmt"
)

func main() {
  fmt.Println("Hello World!")
}
```

至此，环境搭建介绍完了








