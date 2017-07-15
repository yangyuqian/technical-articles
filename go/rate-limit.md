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


