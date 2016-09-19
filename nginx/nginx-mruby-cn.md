# 前言

`mruby`是松本行弘(Yukihiro Matsumoto)实现的高性能的`ruby`，语法上是`ruby`的简化版，
可满足嵌入式设备和一些高性能应用的使用场景. 
同时[mruby](http://mruby.org/)也是得到日本政府支持的重要项目.

从开发人员的角度来看，`mruby`和`ruby`之间几乎没有学习成本，
由于作者所在的团队主要是做Ruby相关的开发，
所以当需要扩展`nginx`的时候，就想到用`mruby`来实现.

`nginx`是目前开源界为数不多的几个俄罗斯人实现的、广为使用的软件之一，
它独特的状态机和事件驱动模型让其在相对较低的资源占用(有限的worker)的下提供极强的并发能力（100000+）.

本文针对用mruby来扩展nginx梳理了需要关注的点，不专门介绍mruby和nginx开发，
一些nginx的实现相关内容会略过，欢迎大家回复讨论.

# 实战指南

主要分为：
* 打包新的nginx
* 自定义mruby的gem

## 打包新的nginx

## 自定义mruby的gem

# 参考文献

[Mruby Module for Nginx](http://ngx.mruby.org/)

[Emiller's Guide To Nginx Module Development](http://www.evanmiller.org/nginx-modules-guide.html)