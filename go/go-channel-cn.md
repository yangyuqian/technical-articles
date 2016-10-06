Go Channel源码分析
---------------------

传统意义上的并发编程，需要直接共享内存，这给并发模型带来了额外的复杂度.

`Goroutine`本质上也是多线程模型，让Go变得特别的地方就是`channel`，
采用消息模型避免直接的内存共享，降低了信息处理的复杂度.

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

# send to channel

初始化`channel`之后就可以向`channel`中写数据，准确的说，应该是发送数据.

入口函数：

```
// src/runtime/chan.go#L106
func chansend1(t *chantype, c *hchan, elem unsafe.Pointer) {
	chansend(t, c, elem, true, getcallerpc(unsafe.Pointer(&t)))
}
```

核心逻辑都在`chansend`中实现，注意这里的`block:bool`参数始终为`true`:

```
// src/runtime/chan.go#L122
func chansend(t *chantype, c *hchan, ep unsafe.Pointer, block bool, callerpc uintptr) bool {
  // 初始化，异常校验和处理

  // 处理channel的临界状态，既没有被close，又不能正常接收数据
	if !block && c.closed == 0 && ((c.dataqsiz == 0 && c.recvq.first == nil) ||
		(c.dataqsiz > 0 && c.qcount == c.dataqsiz)) {
		return false
	}

	var t0 int64
	if blockprofilerate > 0 {
		t0 = cputicks()
	}

  // 对同一个channel的操作是串行的
	lock(&c.lock)

  // 向一个已经被close的channel写数据，会panic
	if c.closed != 0 {
		unlock(&c.lock)
		panic(plainError("send on closed channel"))
	}

  // 这是一个不阻塞的处理，如果已经有接收者，
  // 就向第一个接收者发送当前enqueue的消息.
  // return true就是说发送成功了，写入buffer也算是成功
	if sg := c.recvq.dequeue(); sg != nil {
    // 发送的实现本质上就是在不同线程的栈上copy数据
		send(c, sg, ep, func() { unlock(&c.lock) })
		return true
	}

	if c.qcount < c.dataqsiz {
    // 把发送的数据写入buffer，更新buffer状态，返回成功
	}

  // 如果buffer满了，或者non-buffer（等同于buffer满了），
  // 阻塞sender并更新一些栈的状态，
  // 唤醒线程的时候还会重新检查channel是否打开状态，否则panic
}
```

再看`send`的实现:

```
// src/runtime/chan.go#L122
func send(c *hchan, sg *sudog, ep unsafe.Pointer, unlockf func()) {
	if sg.elem != nil {
    // 直接把要发送的数据copy到reciever的栈空间
		sendDirect(c.elemtype, sg, ep)
		sg.elem = nil
	}

  // 等待reciever就绪，如果reciever还没准备好就阻塞sender一段时间
}
```

`sendDirect`的实现：

```
// src/runtime/chan.go#L284
func sendDirect(t *_type, sg *sudog, src unsafe.Pointer) {
  // 直接拷贝数据
	dst := sg.elem
	memmove(dst, src, t.size)
	typeBitsBulkBarrier(t, uintptr(dst), t.size)
}
```

`channel send`主要就是做了2件事情：

* 如果没有可用的`reciever`，数据入队（如果有buffer），否则线程阻塞
* 如果有可用的`reciever`，就把数据从`sender`的栈空间拷贝到`reciever`的栈空间

如果是`non-buffer channel`，情况比较简单，但如果是`buffered channel`呢？
假设某个时间点，同时有一个`reciever`就绪，并且这个时候`sender`发送了数据，
按照`chansend`的实现，会不会出现`reciever`接收到的是当前发送的数据，
而不是队首的数据？答案是“不会”，因为dequeue其实就保证了，只要reciever入队，
就说明当前的`buffer`应该是空的，具体会在`reciever`实现中介绍.

# recieve from channel

TODO

# close channel

TODO
