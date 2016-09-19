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
* 扩展Nginx功能

## 打包新的nginx

Nginx 1.9.11引入了动态模块，之前的版本都是需要把module和nginx源码编译成一个二进制文件.
本文出于工程上动态模块目前还不成熟，只介绍后者.

本文所使用的nginx为`1.9.5`，[mruby module](https://github.com/matsumoto-r/ngx_mruby)版本为`v1.18.4`.

```
# nginx源码目录 $NGINX_SRC
# ngx_mruby源码目录 $NGX_MRUBY_SRC

cd ${NGX_MRUBY_SRC}
./configure --with-ngx-src-root=${NGINX_SRC}
make build_mruby
make generate_gems_config
cd ${NGINX_SRC}
./configure --prefix=/usr/local/nginx --add-module=${NGX_MRUBY_SRC} --add-module=${NGX_MRUBY_SRC}/dependence/ngx_devel_kit
make
```

注意这里加的 `--prefix=/usr/local/nginx` 意味着以后只能将配置文件放在`/usr/local/nginx/conf`下.

生成的Nginx二进制文件位于`$NGINX_SRC/objs/nginx`. 以下是一些常用命令：

```
# 启动Nginx
$NGINX_SRC/objs/nginx
# 停止Nginx
$NGINX_SRC/objs/nginx -s stop
# Reload Nginx
$NGINX_SRC/objs/nginx -s reload
```

[mruby module](https://github.com/matsumoto-r/ngx_mruby) 已经默认加入了一些MRuby Gems,
可以通过修改`$NGX_MRUBY_SRC/build_config.rb`来增/删一些功能.

```
# $NGX_MRUBY_SRC/build_config.rb
conf.gem :github => 'iij/mruby-io'
conf.gem :github => 'iij/mruby-env'
conf.gem :github => 'iij/mruby-dir'
conf.gem :github => 'iij/mruby-digest'
conf.gem :github => 'iij/mruby-process'
conf.gem :github => 'iij/mruby-pack'
conf.gem :github => 'iij/mruby-socket'
conf.gem :github => 'mattn/mruby-json'
conf.gem :github => 'mattn/mruby-onig-regexp'
conf.gem :github => 'matsumoto-r/mruby-redis'
conf.gem :github => 'matsumoto-r/mruby-vedis'
conf.gem :github => 'matsumoto-r/mruby-sleep'
conf.gem :github => 'matsumoto-r/mruby-userdata'
conf.gem :github => 'matsumoto-r/mruby-uname'
conf.gem :github => 'matsumoto-r/mruby-mutex'
conf.gem :github => 'matsumoto-r/mruby-localmemcache'

# ngx_mruby extended class
conf.gem './mrbgems/ngx_mruby_mrblib'
conf.gem './mrbgems/rack-based-api'
```

最后需要注意的是：修改配置重新编译Nginx时，应该把`$NGX_MRUBY_SRC`和`$NGINX_SRC`下编译期间动态生成
的文件全部清除，以免应用了cache，带来不必要的麻烦.

## 扩展Nginx功能

# 参考文献

[Mruby Module for Nginx](http://ngx.mruby.org/)

[Emiller's Guide To Nginx Module Development](http://www.evanmiller.org/nginx-modules-guide.html)
