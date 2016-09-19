Go Vendoring源码阅读笔记与用法分析
------------------------------

# 为什么引入`Vendoring`机制？

Go1.5之后有一些比较重要的改动，其中包含`Vendoring`的支持，本文从使用和源码实现整理了一些备忘录，难免有疏漏，各位看官多指教。

没有引入`Vendoring`机制时，Go项目组织主要有两种方案：
* 直接把项目放到`GOPATH`下面[详见: 附录1]
* 项目放到`GOPATH`外，修改`GOPATH`来使用Go Command[详见: 附录2]

第三方依赖管理的灵活性和便捷性要求Go加入一种新机制，在不打破现有GOPATH假设的情况下，扩展GOPATH的`package`查找能力，这就是本文中介绍的Vendoring机制.

Vendoring是Go1.5中引入的[实验特性](https://docs.google.com/document/d/1Bz5-UB7g2uPBdOx-rw5t9MxJwkfpx90cqG9AFL0JAYo/edit)，Go1.6成为[正式特性](https://golang.org/doc/go1.6#go_command)，Go1.7中成为[标准特性](https://golang.org/doc/go1.7#cmd_go). 

它本质是一种对GOPATH的扩展，用以支持引入第三方库（对不同项目可以是独立的版本）的程序使用原生的Go Command，比如`go build`，而不用修改GOPATH（一旦那样就不能直接用go command了，必须要借助脚本或Makefile封装GOPATH的修改）.

Go1.5和Go1.6中用GO15VENDOREXPERIMENT控制vendor的生效与否（在Go1.5中，vendoring默认关闭，但Go1.6中默认打开）, 在Go1.7去掉GO15VENDOREXPERIMENT开关，成为标准特性:

```
export GO15VENDOREXPERIMENT=0 # 关闭vendoring功能
export GO15VENDOREXPERIMENT=1 # 打开vendoring功能
```

# 用法分析

Vendoring提供了`GOPATH`的扩展：将项目A依赖的外部库代码放到具体位置的时候，直接用原生的Go Command是可以找到、识别这些代码文件(好像修改了`GOPATH`一样). 
外部依赖的管理，如clone具体的第三方依赖代码并checkout具体版本，
需要通过Go Tools[以外的工具](https://github.com/golang/go/wiki/PackageManagementTools)实现.

本节介绍Vendoring的用法时，用到了以下[例子](https://github.com/yangyuqian/demo/tree/master/go/vendoring/src):

```
▾ src/
  ▾ p1/
    ▾ p2/
      ▾ p3/
        ▾ p7/
          ▾ vendor/p8/
              p8.go
            p7.go
        ▾ vendor/p9/
            p9.go
          p3.go
      ▾ p5/p6/vendor/p13/
          p13.go
      ▾ vendor/p10/
          p10.go
    ▾ vendor/p11/
        p11.go
  ▾ p100/
      main.go
  ▾ vendor/p12/
      p12.go
    main.go
```

Go Vendor比较常用的规则：

1) `Vendored Package`必须在`GOROOT`或`GOPATH`下

因为`Vendored Package`都是以`GOROOT`或`GOPATH`为基础扫描的，使用`local import`时是无法使用Vendor机制的.

2) `Vendored Package`优先

这里的优先应该有两层含义：
* `Vendored Package`的扫描优先级，即`src/p1/p2/p3/p3.go`中`import "p12"`的时候，依次扫描了`src/p1/p2/p3/vendor/p12`, `src/p1/p2/vendor/p12`, `src/p1/vendor/p12`, `src/vendor/p12`，返回第一个存在且下面有Go源码文件的Package
* `Vendored Package`和`GOROOT`或`GOPATH`下普通的`package`的优先级，会优先使用`Vendor Package`

3) 直接放在`GOROOT|GOPATH/src`下的`main package`不支持Vendoring

这个可能是Go里面的小Bug，似乎这方面的需求也不明显.
前面例子里的 `src/main.go`，尽管`src/vendor/p12`存在，所以下面的`main.go`直接`import "p12"`会报package找不到:

```
  // src/main.go
  1 package main
  2
  3 import (
  4     "p1/p2/p3"
  5     "p1/p2/p3/p7"
  6     "p12"
  7 )
  8
  9 func main() {
 10     p3.Hello3()
 11     p7.Hello7()
 12     p12.Hello12()
 13 }
 
 $ go run main.go
 
 main.go:6:2: cannot find package "p12" in any of:
        /usr/local/go/src/p12 (from $GOROOT)
        /Users/yangyuqian/demo/src/p12 (from $GOPATH)
```

解决方案是把这样的入口文件放到具体的子目录下面去，就可以正常编译了.

```
// src/p100/main.go
```

4) `Vendored Package`只能往上找

`Vendored Package`的查找是一个动态匹配的过程，前面的例子中:

```
// src/p1/p2/p3/p3.go
  1 package p3
  2
  3 import (
  4     "p12"
  5 )
  6
  7 func Hello3() {
  8     println("Hello, P3")
  9     p12.Hello12()
 10 }
```

这里 import 了 `src/vendor/p12`，这是最终匹配到的vendor路径，实际上这里经历了以下的匹配：
- src/p1/p2/p3/vendor/p12
- src/p1/p2/vendor/p12
- src/p1/vendor/p12
- src/vendor/p12

可见前几个都没有匹配中，所以返回第一个命中的路径 `src/vendor/p12`。

只往上找，意味着 `src/p1/p2/p5/p6/vendor/p13` 和 `src/p1/p2/p3/p7/vendor/p8` 在 `src/p1/p2/p3/p3.go`是不可见的.

# 源码阅读笔记

```
这里所有的源码都是基于go1.6.2的，不同版本的源码可能会有差别
```

核心逻辑入口位于 `src/cmd/go/pkg.go` 中:

```
# src/cmd/go/pkg.go:L317
317 func loadImport(path, srcDir string, parent *Package, stk *importStack, importPos []token.Position, mode int) *Package {
// 初始化上下文
329     if isLocal {
330         importPath = dirToImportPath(filepath.Join(srcDir, path))
331     } else if mode&useVendor != 0 {
// 如果开启了vendor，就会构造vendor路径（如果目录存在就加入列表，不存在就舍弃）
336         path = vendoredImportPath(parent, path)
337         importPath = path
338     }
339
// 如果package/path cache了对应的的import path，直接返回vendored path，然后进行一些校验
// 校验分为internal/vendor校验，主要就是处理一些不合法的import
// 递归遍历package import
377     p.load(stk, bp, err)
// 校验import internal package的合法性
// 校验import vendor package的合法性
```

实际构造和探测vendor路径的代码是从 `src/cmd/go/pkg.go:L326` 调用的：

```
// src/cmd/go/pkg.goL414
// 336         path = vendoredImportPath(parent, path)
 414 func vendoredImportPath(parent *Package, path string) (found string) {
 415     if parent == nil || parent.Root == "" || !go15VendorExperiment {
 416         return path
 417     }
 418
 419     dir := filepath.Clean(parent.Dir)
 420     root := filepath.Join(parent.Root, "src")
 421     if !hasFilePathPrefix(dir, root) {
 422         // Look for symlinks before reporting error.
 423         dir = expandPath(dir)
 424         root = expandPath(root)
 425     }
 426     if !hasFilePathPrefix(dir, root) || len(dir) <= len(root) || dir[len(root)] != filepath.Separator {
 427         fatalf("invalid vendoredImportPath: dir=%q root=%q separator=%q", dir, root, string(filepath.Separator))
 428     }
 429
 430     vpath := "vendor/" + path
 431     for i := len(dir); i >= len(root); i-- {
 432         if i < len(dir) && dir[i] != filepath.Separator {
 433             continue
 434         }
 435         // Note: checking for the vendor directory before checking
 436         // for the vendor/path directory helps us hit the
 437         // isDir cache more often. It also helps us prepare a more useful
 438         // list of places we looked, to report when an import is not found.
 439         if !isDir(filepath.Join(dir[:i], "vendor")) {
 440             continue
 441         }
 442         targ := filepath.Join(dir[:i], vpath)
 443         if isDir(targ) && hasGoFiles(targ) {
 444             importPath := parent.ImportPath
 445             if importPath == "command-line-arguments" {
 446                 // If parent.ImportPath is 'command-line-arguments'.
 447                 // set to relative directory to root (also chopped root directory)
 448                 importPath = dir[len(root)+1:]
 449             }
 450             // We started with parent's dir c:\gopath\src\foo\bar\baz\quux\xyzzy.
 451             // We know the import path for parent's dir.
 452             // We chopped off some number of path elements and
 453             // added vendor\path to produce c:\gopath\src\foo\bar\baz\vendor\path.
 454             // Now we want to know the import path for that directory.
 455             // Construct it by chopping the same number of path elements
 456             // (actually the same number of bytes) from parent's import path
 457             // and then append /vendor/path.
 458             chopped := len(dir) - i
 459             if chopped == len(importPath)+1 {
 460                 // We walked up from c:\gopath\src\foo\bar
 461                 // and found c:\gopath\src\vendor\path.
 462                 // We chopped \foo\bar (length 8) but the import path is "foo/bar" (length 7).
 463                 // Use "vendor/path" without any prefix.
 464                 return vpath
 465             }
 466             return importPath[:len(importPath)-chopped] + "/" + vpath
 467         }
 468     }
 469     return path
 470 }
```

`import $package`的遍历都是以parent为上下文的，比如有如下项目结构：

```
// $proj_root加入GOPATH
$proj_root
  |- src
    |- p1
      |- p2
        |- p3
          |- p3.go
```

其中 `$proj_root/src/p1/p2/p3/p3.go` 引入了 `p4`

```
# $proj_root/src/p1/p2/p3/p3.go
package p3

import "p4"

// 其他实际逻辑
```

上面 `vendoredImportPath` 的核心算法如下：

```
path := 依次扫描以下路径:
  - $proj_root/src/p1/p2/p3/vendor/p4 => 相对路径：p1/p2/p3/vendor/p4
  - $proj_root/src/p1/p2/vendor/p4    => 相对路径：p1/p2/vendor/p4
  - $proj_root/src/p1/vendor/p4       => 相对路径：p1/vendor/p4
  - $proj_root/src/vendor/p4          => 相对路径：vendor/p4

if path 是路径? && path有*.go文件
  返回该路径对应的GOPATH路径
end
```

可见vendor依赖还是需要放到$GOPATH下的，因为算法返回的实际上还是一个相对路径.

`src/cmd/go/pkg.go#L415` 中有一个判断逻辑，所有parent.Root为空的package都不会探测`vendored package`，所以$GOPATH/src下直接写的main文件(package)是不支持vendoring的.

```
415     if parent == nil || parent.Root == "" || !go15VendorExperiment { 
```



上面的算法获取到所有合法的 `vendored path`，在 `disallowVendor` 还检查了`import`的时候没有直接 `import vendor/p4`等，这些在开启了vendor后都是不合法的，编译会报错：

```
 588 func disallowVendor(srcDir, path string, p *Package, stk *importStack) *Package {
 589     if !go15VendorExperiment {
 590         return p
 591     }
 592
 593     // The stack includes p.ImportPath.
 594     // If that's the only thing on the stack, we started
 595     // with a name given on the command line, not an
 596     // import. Anything listed on the command line is fine.
 597     if len(*stk) == 1 {
 598         return p
 599     }
 600
 601     if perr := disallowVendorVisibility(srcDir, p, stk); perr != p {
 602         return perr
 603     }
 604
 605     // Paths like x/vendor/y must be imported as y, never as x/vendor/y.
 606     if i, ok := findVendor(path); ok {
 607         perr := *p
 608         perr.Error = &PackageError{
 609             ImportStack: stk.copy(),
 610             Err:         "must be imported as " + path[i+len("vendor/"):],
 611         }
 612         perr.Incomplete = true
 613         return &perr
 614     }
 615
 616     return p
 617 }
```

前面递归查找的时候并不检查vendor的合法性，所以可能出现 `import p4` 被映射到了 `import p5/vendor/p4` 的情况，这种情况会报错 "use of vendored package not allowed". 

虽然说查找的时候只往上查，但由于Go中的package查找是一个递归算法，其中应用了cache `map[imported path] => vendored path`，有可能当前获取到的vendored path是不合法的，所以存在这一层校验.

```
// 如果在p3.go里面import p4会报错，因为visibility校验失败了
$proj_root
  |- src
    |- p1
      |- p2
        |- p3
          |- p3.go
        |- p5
          |- vendor
            |- p4
              |- p4.go
```


vendor import是通过 `findVendor`判断的

```
 678 func findVendor(path string) (index int, ok bool) {
 679     // Two cases, depending on internal at start of string or not.
 680     // The order matters: we must return the index of the final element,
 681     // because the final one is where the effective import path starts.
 682     switch {
 683     case strings.Contains(path, "/vendor/"):
 684         return strings.LastIndex(path, "/vendor/") + 1, true
 685     case strings.HasPrefix(path, "vendor/"):
 686         return 0, true
 687     }
 688     return 0, false
 689 }
```

可见诸如 `import vendor/p1` `import p1/vendor/p2` 都是不合法的.

# 附录

## 附录1. 直接把项目放到GOPATH下面

如果直接使用前者似乎也没有什么大问题，但降低了项目的可维护性，因为每次clone下来的代码都是不能直接编译的，对于本地开发和自动化流程也都会存在各种限制和上下文, 需要手动维护项目的clone，而且每次需要进到特定目录下操作，降低了便捷性，也会让开发者产生抵触.

基于GOPATH下面直接开发，通常需要有一个namespace才能用`go get`来管理项目，
比如`github.com/xxx/yyy`，项目里面的`import`里面也都是这样的namespace，
如果这个namespace出现变化，将会导致项目代码需要重新调整. 

而如果不使用namespace，每次修改项目的时候就都需要进到特性的目录，
这就是上面提到的便捷性的问题.

比较好的实践是所有的项目对项目内的依赖都是相对路径，比如p1里面依赖p2，就直接`import p2`，而不是`import github.com/xxx/p2`：

```
src
  |- p1
  |- p2
```

即便是library，也不建议大家直接基于GOPATH下开发.

## 附录2. 项目放到GOPATH外，修改GOPATH来使用Go Command

短期来看，这个方案是可行的. 只要在Makefile里面对GOPATH进行适当的修改，一条简单的`make build`和原生的`go build`似乎距离也不是那么远. 

但问题来了，如果你是Go的作者，你会希望看到自己开发的一整套复杂命令行工具在大部分情况下都不好使，
一定要用别的工具做一层封装才能用吗？

Go Command的地位是一整套工具链的入口，包含编译、代码检查、测试、文档生成等等功能，和传统的CLI不同，
没有Go Command，拿着一堆代码是什么都干不了的.
而且Go Command上层的封装也可能因为Go的一些升级打破，可能会不好使，用户就会骂娘了.

 有些项目里面可能对第三方库是有版本要求的（就好像Ruby里面的Gem，每个项目版本要求可能都不一样），
且`go get`永远都只能选择`GOPATH`中的第一个路径. 这就要求维护复杂的GOPATH来适应多个项目. 
而复杂的`GOPATH`会增加项目结构的复杂度和维护成本. 出于工程上的考量，要保持`GOPATH`尽可能的简单.



