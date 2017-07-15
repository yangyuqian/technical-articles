基于Token Bucket的限流
--------------------

Token Bucket是一种常用的限流算法，本文从实战角度简单介绍了其原理和使用。

Token Bucket算法涉及三个变量：

* BucketSize: 桶容量
* Interval: 更新桶中Token的时间周期
* Quantum: 每个Interval增加的Token数

三个参数分别从不同的层面影响了流量。
 
首先看BucketSize，它表示“桶”的容量，单位Interval内，最大的事件数，比如HTTP请求限流，它就是单位Interval中的最大请求数。

再看Interval，这个参数控制多长时间来向“桶”中添加新Token，而Quantum控制单位Interval最多添加的Token数。

这里为什么要提“最多”？因为BucketSize会限制整个桶里面的Token数不超过上限，超出BucketSize部分的Token会被直接丢弃。

> 好比一个饭店，BucketSize就是每个食客的碗大小，有服务员定时Interval来更新这部分
> Interval就是服务员为碗里添加食物的周期
> Quantum是服务员每次最多添加的食物，超出饭碗容量的部分会进入等待状态

光说不练假把式，这里基于[Rate Limit](https://github.com/juju/ratelimit)来实现几个例子看一下不同的参数设置有什么影响。

例 1. `go/examples/src/ratelimit/main.go`

```
package main

import (
	"flag"
	"github.com/juju/ratelimit"
	"time"
)

var (
	interval uint
	capacity int64
	quantum  int64
	duration uint
)

func init() {
	flag.UintVar(&interval, "i", 1000, "set token bucket interval in milliseconds")
	flag.Int64Var(&capacity, "c", 10, "set token bucket capacity")
	flag.Int64Var(&quantum, "q", 1, "set token bucket quantum")
	flag.UintVar(&duration, "d", 10, "set duration to run this example")
}

func main() {
	bucket := ratelimit.NewBucketWithQuantum(time.Duration(interval)*time.Millisecond, capacity, quantum)
	for i := 0; i < 500; i++ {
		go func() {
			for {
				bucket.Wait(1)
				print(".")
			}
		}()
	}

	<-time.After(time.Duration(duration) * time.Second)
}
```

这个例子默认会执行10秒，启动500个goroutine模拟高并发事件处理，
每个事件被处理后会打印一个点`.`，Token Bucket默认设置如下：

* BucketSize: 10
* Interval: 1000 ms
* Quantum: 1

[![asciicast](https://asciinema.org/a/bavkebqxc4wjgb2zv0t97es9y.png)](https://asciinema.org/a/3mmy9EJETqIUkQF9E4a6gEQi1)

可见一开始执行了10个事件，然后后面每秒执行1个事件，并发事件被稳定限流了。

现在可以划重点了，这几个参数理解清楚，用Token Bucket就不会出问题了：

* BucketSize：控制单位interval内可能出现的最高并发数，当桶被装满后就无法再填充Token了
* Interval: 添加新Token的周期，控制限流的精度，比如不但希望控制最高并发时间数，还要对事件进行细粒度的控制。如10ms只允许执行2个事件，虽然1s内理论上还是可以执行200个事件，这比1s中的某1ms执行了200个事件的压力要小很多
* Quantum: 这也是非常关键的控制参数，控制单位interval放入桶的token数，最终影响的是持续压力的模型

如果BucketSize设得非常大，而Quantum很小，那么整个限流模型就是先松后紧。反之如果BucketSize设得较小，而Quantum比较大，那么就是先紧后松的模型。

实践中发现先紧后松模型能够给系统足够的预热时间，比如系统初始化需要懒加载一些cache，这是限流能够保护系统在预热阶段不要出现太高的负载，当系统稳定后，限流也相应变松。
