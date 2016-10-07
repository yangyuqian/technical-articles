Channel使用备注
-------------------

`Channel`的实现了“生产者－消费者”模型，`sender`将数据写入`buffer`(环形队列)，
然后`reciever`从`buffer`中取出数据.


```
+--------------+ copy to  +----------------+ copy to  +----------------+
|  Sender Queue+---------->     Buffer     +---------->  Reciever Queue|
+--------------+          +----------------+          +----------------+

图 1
```

初始化`channel`:

1. 默认buffer容量为0

```
// make(chan string)
make(chan $chan_type)
```

2. 也可以初始化容量 > 0

```
// make(chan string, 5)
make(chan $chan_type, $chan_buffer_size)
```

注：与`slice/map`不一样(make只支持`slice/map/chan` 3种类型)，`channel`只支持
`make`初始化，返回的是一个`runtime.hchan`结构的指针, 具体参见
[Channel源码分析](channel-implementation-cn.md).

