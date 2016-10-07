Channel使用备注
-------------------

传统的并发编程中，线程之间的内存通信通常是维护一些共享内存实现的，这种模式对
开发和维护的要求都比较高. `Go`中为了尽可能减少共享内存的使用，
实现了`Channel`，本质上是一个内存中的`Message Queue`.

本文希望从使用的角度做一些基础性的介绍，希望能够对读者实际开发有所启发.

`Channel`的实现了“生产者－消费者”模型，`sender`将数据写入`buffer`(环形队列)，
然后`reciever`从`buffer`中取出数据.


```
+--------------+ copy to  +----------------+ copy to  +----------------+
|  Sender Queue+---------->     Buffer     +---------->  Reciever Queue|
+--------------+          +----------------+          +----------------+

图 1
```

# 初始化

`Channel`的`zero value`是nil，因为`chan`类型实际上是个指针.
原则上不应该直接读写没有初始化的`channel`，否则会出现“死锁”.

`Channel`是Go中的一种特殊类型，定义语法：

```
// 定义channel strChan, 类型是string
var strChan chan string
// 定义channel intChan, 类型是int
var intChan chan int
```

初始化`channel`变量通常有2种方式：

1) 默认buffer容量为0

```
// make(chan string)
make(chan $chan_type)
```

2) 也可以初始化一定容量的buffer

```
// make(chan string, 5)
make(chan $chan_type, $chan_buffer_size)
```

这里定义的`buffer`容量很好理解，本文不在赘述.

注：与`slice/map`不一样(make只支持`slice/map/chan` 3种类型)，`channel`只支持
`make`初始化，返回的是一个`runtime.hchan`结构的指针, 具体参见
[Channel源码分析](channel-implementation-cn.md).

接下来对`channel`数据的收/发介绍都假设`channel`已经初始化了.

# 写数据

理论上，`Channel`允许写入任何类型的数据. 只限制数据大小(64KB)，不限制类型.
出于`Channel`和并发编程的基本考量，建议不要在`channel`中传输指针.

写数据的语法很简单：

```
// 定义并初始化一个channel
strChan := make(chan string)
// 向channel中写入数据
strChan <- "message data"
```

执行完写数据的时候可能发生以下几种结果：

1) `buffer`有空间或存在就绪的`reciever`，数据发送成功，线程继续执行

对于同一个`channel`，发送操作是串行的.
`Sender`发送成功后，数据被copy到buffer中或`reciever`的栈空间.
意味着`reciever`端无论对收到的数据本身做任何形式的修改都不会直接影响`sender`端.

2) `buffer`没有空间或没有就绪的`reciever`，`sender`所在`goroutine`被阻塞

当满足唤醒条件后，阻塞的`sender`会被唤醒，此时检查`channel`的是否关闭，
如果`channel`关闭会直接`panic`.


# 读数据

从`channel`读数据的语法：

```
// 从channel中读数据并赋值给变量，这样做会block当前goroutine直到接收到有效数据
data := <- strChan

// 直接从channel中取数据，这样做会block当前的func调用，直到接收到有效的数据
// 本质上这个和赋值是一样的，因为这里的 <- strChan会先赋值给一个形参
println(<-strChan)

// 用select集中处理多个channel，和switch/case同理
select {
  case str := <-strChan:
  println("data:", str)
  case data := <-intChan:
  println("data:", data)
}
```

读`channel`的过程也可能发生2种情况：

1) `Buffer`里面有数据或有就绪的`sender`，直接接收数据

同一个`channel`的读操作也是串行的.

2) `Buffer`里面没有数据或没有就绪的`sender`, block当前`reciever`

和`sender`不同的是，如果在唤醒之后发现`channel`被关闭了，`reciever`不会`panic`.
有些文章建议利用这种特性实现`goroutine`之间的同步操作，我认为是值得商榷的，
如果需要同步，应该采用传统的同步编程模型，在Go中，应该使用`sync`.

# 关闭

关闭`channel`意味着进行以下几个操作：

1) 释放buffer空间

2）释放`sender`，紧接着在`sender`端将会`panic`

3) 释放`reciever`

