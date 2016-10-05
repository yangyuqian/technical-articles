Go Channel源码分析
---------------------

注意：**本文所有的源码分析基于Go1.6.2，版本不同实现可能会有出入**

`Channel`是Go中一种特别的数据结构，它不像普通的类型一样，强调传值还是指针；
对于`channel`而言，只涉及传指针，因为`make(chan)`返回的是指针类型：

```
// concurrency/e5_1.go
package main

func abstractListener(fxChan chan func() string) {
	fxChan <- func() string {
		return "Hello, functions in channel"
	}
}

func main() {
	fxChan := make(chan func() string)
	defer close(fxChan)

	go abstractListener(fxChan)
	select {
	case rfx := <-fxChan:
		msg := rfx()
		println("recieved message:", msg)
	}
}
```

本文从Go Runtime的实现角度，粗略介绍了`channel`的实现，
希望能为其他的语言带来一些启发.

# make channel

这是初始化channel的过程，编译器会把`make(chan xxx)`最终指向：

```
// src/runtime/chan.go#L52
func reflect_makechan(t *chantype, size int64) *hchan {
	return makechan(t, size)
}
```

最终返回一个`*hchan`类型，可见`make`出来的`channel`对象确实是一个指针.
`hchan`保存了一个`channel`实例必须的信息. 里面的参数后面会详细说到.

```
// src/runtime/chan.go#L25
type hchan struct {
	qcount   uint           // total data in the queue
	dataqsiz uint           // size of the circular queue
	buf      unsafe.Pointer // points to an array of dataqsiz elements
	elemsize uint16
	closed   uint32
	elemtype *_type // element type
	sendx    uint   // send index
	recvx    uint   // receive index
	recvq    waitq  // list of recv waiters
	sendq    waitq  // list of send waiters

	// lock protects all fields in hchan, as well as several
	// fields in sudogs blocked on this channel.
	//
	// Do not change another G's status while holding this lock
	// (in particular, do not ready a G), as this can deadlock
	// with stack shrinking.
	lock mutex
}
```

代码中调用`make(chan $type, $size)`的时候，最终会执行`runtime.makechan`

```
// src/runtime/chan.go#L56
func makechan(t *chantype, size int64) *hchan {
  // 初始化，做一些基本校验

	var c *hchan
  // 如果是无指针类型或者指定size=0；可见默认情况下都会走这个逻辑
	if elem.kind&kindNoPointers != 0 || size == 0 {
		// 从内存分配器请求一段内存
    c = (*hchan)(mallocgc(hchanSize+uintptr(size)*elem.size, nil, true))
    // 如果是无指针类型，并且制定了buffer size
		if size > 0 && elem.size != 0 {
			c.buf = add(unsafe.Pointer(c), hchanSize)
		} else {
			// race detector uses this location for synchronization
			// Also prevents us from pointing beyond the allocation (see issue 9401).
			c.buf = unsafe.Pointer(c)
		}
	} else {
    // 有指针类型，并且size > 0时，直接初始化一段内存
		c = new(hchan)
		c.buf = newarray(elem, int(size))
	}
  // 初始化channel结构里面的参数
	c.elemsize = uint16(elem.size)
	c.elemtype = elem
	c.dataqsiz = uint(size)

  // ...
	return c
}
```

上面的代码提到了“无指针类型”，一个比较简单的例子就是反射中的值类型，
这种值是没办法直接获取地址的, 也有些地方叫做“不可寻址”：

```
i := 1
// 从道理上上，这个value实际上就是一个抽象概念，
// 好比从一个活人身上把肉体抽出来 => value
// 脱离了具体的对象，所以就无法谈地址，但这样的值却实实在在存在
v := reflect.ValueOf(i)
// 然后灵魂也可以抽出来 => type
t := reflect.TypeOf(i)
```

