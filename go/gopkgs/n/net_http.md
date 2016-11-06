net/http
-------------

`net/http` 提供了`HTTP/HTTPS`相关的顶层接口.

例1 `examples/e1.go` 会启动一个监听`7000`端口的Web Server:

```
// examples/e1.go
func main() {
  // 定义 /hello
	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
    // 访问 http://$host:$port/hello的时候，返回"Hello, net/http"
		w.Write([]byte(`Hello, net/http`))
	})

  // 监听7000端口，等待请求，第二个参数Handler将会在后面详细介绍
	http.ListenAndServe(":7000", nil)
}
```

先看一下HTTP请求处理逻辑：

```
// src/net/http/server.go#L2349
func ListenAndServe(addr string, handler Handler) error {
  // 直接初始化一个Server对象，调用其ListenAndServe()
	server := &Server{Addr: addr, Handler: handler}
	return server.ListenAndServe()
}

// src/net/http/server.go#L2210
func (srv *Server) ListenAndServe() error {
	addr := srv.Addr
	if addr == "" {
		addr = ":http"
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
  // 通过传入的信息获取TCP socket，传给Serve()
	return srv.Serve(tcpKeepAliveListener{ln.(*net.TCPListener)})
}

// src/net/http/server.go#L2256
func (srv *Server) Serve(l net.Listener) error {
  defer l.Close()
  if fn := testHookServerServe; fn != nil {
    fn(srv, l)
  }
  var tempDelay time.Duration // how long to sleep on accept failure

  if err := srv.setupHTTP2_Serve(); err != nil {
    return err
  }

  // Go中上下文(context)是一种比较有意思的机制，在相关文章中会详细介绍
  baseCtx := context.Background()
  ctx := context.WithValue(baseCtx, ServerContextKey, srv)
  ctx = context.WithValue(ctx, LocalAddrContextKey, l.Addr())
  // 一个死循环，一直等待TCP连接
  for {
    // 监听TCP socket，这一步会阻塞整个进程
    // 直到新的TCP连接建立起来，返回的rw实际上是一个TCP连接
    rw, e := l.Accept()
    // socket处理时的一种容错机制，进行回退等待并retry
    if e != nil {
      if ne, ok := e.(net.Error); ok && ne.Temporary() {
        if tempDelay == 0 {
          tempDelay = 5 * time.Millisecond
        } else {
          tempDelay *= 2
        }
        if max := 1 * time.Second; tempDelay > max {
          tempDelay = max
        }
        srv.logf("http: Accept error: %v; retrying in %v", e, tempDelay)
        time.Sleep(tempDelay)
        continue
      }
      return e
    }
    tempDelay = 0
    // 初始化一个conn对象
    c := srv.newConn(rw)
    c.setState(c.rwc, StateNew) // before Serve can return
    // 创建一个Goroutine来处理当前的TCP连接，然后继续等待下一个TCP连接
    go c.serve(ctx)
  }
}

// src/net/http/server.go#L1485
func (c *conn) serve(ctx context.Context) {
	c.remoteAddr = c.rwc.RemoteAddr().String()
  // 错误处理和“请求劫持”支持，此处暂且不表
	defer func() {
		if err := recover(); err != nil {
			const size = 64 << 10
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			c.server.logf("http: panic serving %v: %v\n%s", c.remoteAddr, err, buf)
		}
		if !c.hijacked() {
			c.close()
			c.setState(c.rwc, StateClosed)
		}
	}()

  // TLS相关处理，暂且不表
	if tlsConn, ok := c.rwc.(*tls.Conn); ok {
		if d := c.server.ReadTimeout; d != 0 {
			c.rwc.SetReadDeadline(time.Now().Add(d))
		}
		if d := c.server.WriteTimeout; d != 0 {
			c.rwc.SetWriteDeadline(time.Now().Add(d))
		}
		if err := tlsConn.Handshake(); err != nil {
			c.server.logf("http: TLS handshake error from %s: %v", c.rwc.RemoteAddr(), err)
			return
		}
		c.tlsState = new(tls.ConnectionState)
		*c.tlsState = tlsConn.ConnectionState()
		if proto := c.tlsState.NegotiatedProtocol; validNPN(proto) {
			if fn := c.server.TLSNextProto[proto]; fn != nil {
				h := initNPNRequest{tlsConn, serverHandler{c.server}}
				fn(c.server, tlsConn, h)
			}
			return
		}
	}

	c.r = &connReader{r: c.rwc}
	c.bufr = newBufioReader(c.r)
	c.bufw = newBufioWriterSize(checkConnErrorWriter{c}, 4<<10)

	ctx, cancelCtx := context.WithCancel(ctx)
	defer cancelCtx()

  // HTTP 1.1 中每个HTTP连接的TCP连接是复用的
	for {
    // 从客户端获取TCP请求包
		w, err := c.readRequest(ctx)
		if c.r.remain != c.server.initialReadLimitSize() {
			c.setState(c.rwc, StateActive)
		}
    // 对请求数据包的校验
		if err != nil {
			if err == errTooLarge {
				io.WriteString(c.rwc, "HTTP/1.1 431 Request Header Fields Too Large\r\nContent-Type: text/plain\r\nConnection: close\r\n\r\n431 Request Header Fields Too Large")
				c.closeWriteAndWait()
				return
			}
			if err == io.EOF {
				return // don't reply
			}
			if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
				return // don't reply
			}
			var publicErr string
			if v, ok := err.(badRequestError); ok {
				publicErr = ": " + string(v)
			}
			io.WriteString(c.rwc, "HTTP/1.1 400 Bad Request\r\nContent-Type: text/plain\r\nConnection: close\r\n\r\n400 Bad Request"+publicErr)
			return
		}

		// Expect 100 Continue support
		req := w.req
    // 处理Expect: 100-continue, 用以和客户端协调大(> 1024B)数据的POST
		if req.expectsContinue() {
			if req.ProtoAtLeast(1, 1) && req.ContentLength != 0 {
				// Wrap the Body reader with one that replies on the connection
				req.Body = &expectContinueReader{readCloser: req.Body, resp: w}
			}
		} else if req.Header.get("Expect") != "" {
      // 如果写入非法的Expect头，就直接返回错误，中断TCP连接
			w.sendExpectationFailed()
			return
		}

    // 处理Handler，涉及一些常见的扩展，后面会介绍
		serverHandler{c.server}.ServeHTTP(w, w.req)
    // 清理当前上下文
		w.cancelCtx()
		if c.hijacked() {
			return
		}
		w.finishRequest()
		if !w.shouldReuseConnection() {
			if w.requestBodyLimitHit || w.closedRequestBodyEarly() {
				c.closeWriteAndWait()
			}
			return
		}
		c.setState(c.rwc, StateIdle)
	}
}

// src/net/http/server.go#L1485
func (sh serverHandler) ServeHTTP(rw ResponseWriter, req *Request) {
  // 这里的handler就是调用http.ListenAndServe()时传入的第二个参数
	handler := sh.srv.Handler
  // 可见二者存在互斥关系，默认行为是没有定义handler的时候使用DefaultServeMux
  // 但http.DefaultServeMux是一个全局可见的变量，其实可以在handler里面额外处理
	if handler == nil {
		handler = DefaultServeMux
	}
	if req.RequestURI == "*" && req.Method == "OPTIONS" {
		handler = globalOptionsHandler{}
	}
  // 调用handler的ServeHTTP接口
	handler.ServeHTTP(rw, req)
}
```

由此可见`net/http`的基本处理流程如下：

```
                                        +
                                        |
                            TCP         |        HTTP/1.1
                                        |
                                        |
                     +---------------+  |
          +---------->    socket     |  |
          |          +---------------+  |
          |                             |
wait new connections                    |
          |          +---------------+  | Read
          |          | TCP Connection+----------------------------------------------+
          +----------+  (Goroutine)  <-----------------+                            |
                     +---------------+  | Write        |                            |
                                        |              |                            |
                                        |              |                            |
                                        |    +---------+--------+          +--------v--------+
                                        |    |    *response     |          |    *Request     |
                                        |    | (ResponseWriter) |          |                 |
                                        |    +---------^--------+          +--------+--------+
                                        |              |                            |
                                        |              |                            |
                                        |              |                            |
                                        |              |                            |
                                        |    +---------+----------------------------v--------+
                                        |    |                 Handler                       |
                                        |    |      .ServeHTTP(ResponseWriter, *Request)     |
                                        |    |                                               |
                                        |    |         (Handler || Default Mux)              |
                                        |    +-----------------------------------------------+
                                        |
                                        |
                                        +

```

上图介绍了`net/http`中TCP处理流程和HTTP处理流程之间的联系，
在Go中为每个TCP连接创建独立的Goroutine，对`HTTP/1.1`而言，
TCP连接是共享的，所以`HTTP/1.1`中每个请求都会由独立的Goroutine来处理.

TCP连接建立起来之后，交由HTTP层的`handler`处理7层的业务逻辑，TCP把对客户端的写
接口和请求读取接口以`ResponseWriter`和`*Request`形式暴露出来，下方的Handler实现
了`http.Handler`接口:

```
// src/net/http/server.go#L77
type Handler interface {
	ServeHTTP(ResponseWriter, *Request)
}
```

启动服务的时候可以传入一个Handler对象，
否则就会采用`net/http`默认的`DefaultServeMux`，定义了默认的`routes`管理机制.

```
// src/net/http/server.go#L1485
func (sh serverHandler) ServeHTTP(rw ResponseWriter, req *Request) {
	handler := sh.srv.Handler
	if handler == nil {
		handler = DefaultServeMux
	}
  // ...
}
```

可见所有开源`Route`实现无非就是实现了一个自定义的`Handler`对象，
覆盖了默认的`DefaultServeMux`.


下面来看`ServerMux`的实现：

```
// src/net/http/server.go#L1900
type ServeMux struct {
	mu    sync.RWMutex
	m     map[string]muxEntry
	hosts bool // whether any patterns contain hostnames
}

// src/net/http/server.go#L1900
func (mux *ServeMux) ServeHTTP(w ResponseWriter, r *Request) {
  // HTTP/1.1+中不允许url为*
	if r.RequestURI == "*" {
		if r.ProtoAtLeast(1, 1) {
			w.Header().Set("Connection", "close")
		}
		w.WriteHeader(StatusBadRequest)
		return
	}
  // 从已经注册的Handler中选择一个，默认返回NotFoundHandler
	h, _ := mux.Handler(r)
	h.ServeHTTP(w, r)
}

// src/net/http/server.go#L1979
func (mux *ServeMux) Handler(r *Request) (h Handler, pattern string) {
  // 只支持HTTP/1.1
	if r.Method != "CONNECT" {
    // 处理redirect的特殊情况, CleanPath返回带slash的Path
    // 比如/x1/x2 将被跳转到 /x1/x2/
		if p := cleanPath(r.URL.Path); p != r.URL.Path {
			_, pattern = mux.handler(r.Host, p)
			url := *r.URL
			url.Path = p
			return RedirectHandler(url.String(), StatusMovedPermanently), pattern
		}
	}

	return mux.handler(r.Host, r.URL.Path)
}

// src/net/http/server.go#L1994
// 参数:
//  - host: Request.Host
//  - path: Request.URL.Path
func (mux *ServeMux) handler(host, path string) (h Handler, pattern string) {
  // 对ServerMux加读锁
	mux.mu.RLock()
	defer mux.mu.RUnlock()
  // mux.hosts => bool, 在mux.Handle中如果发现pattern不是 slash开头
  // mux.hosts = true
  // 在修改DefaultServeMux状态的时候，即调用HandleFunc, Handle的时候
  // 如果发现给出的pattern前面不带slash，就会认为是带host的定义
	if mux.hosts {
		h, pattern = mux.match(host + path)
	}
	if h == nil {
		h, pattern = mux.match(path)
	}
	if h == nil {
		h, pattern = NotFoundHandler(), ""
	}
	return
}

// src/net/http/server.go#L1952
func (mux *ServeMux) match(path string) (h Handler, pattern string) {
	var n = 0
	for k, v := range mux.m {
    // 匹配Handler
		if !pathMatch(k, path) {
			continue
		}
		if h == nil || len(k) > n {
			n = len(k)
			h = v.h
			pattern = v.pattern
		}
	}
	return
}

// src/net/http/server.go#L1921
func pathMatch(pattern, path string) bool {
	if len(pattern) == 0 {
		return false
	}
	n := len(pattern)
  // 如果pattern不是完全路径 /a/b, 必须完全匹配
	if pattern[n-1] != '/' {
		return pattern == path
	}
  // 如果是完全路径 /a/b/，会把path截取和pattern相同长度然后直接匹配
	return len(path) >= n && path[0:n] == pattern
}
```

ServeMux（默认route实现）是“首次匹配”，以下是几个匹配的典型例子:

```
// ServeMux的定义是以Map存放的，
// 当有多个handler满足条件的时候，匹配的顺序可能会变
HandleFunc("/x1", ...) => "/x1"
HandleFunc("/x1/", ...) => "/x1"(redirect 302), "/x1/", "/x1/x2/"
HandleFunc("/x1/x2") => "/x1/x2"
HandleFunc("/x1/x2/", ...) => "/x1/x2", "/x1/x2/x3"
HandleFunc("/") -> "/", "/x1"
```

接下来看ServeMux的几种定义方式的具体实现：

```
// src/net/http/server.go#L2089
func HandleFunc(pattern string, handler func(ResponseWriter, *Request)) {
	DefaultServeMux.HandleFunc(pattern, handler)
}
// src/net/http/server.go#L2069
func (mux *ServeMux) HandleFunc(pattern string, handler func(ResponseWriter, *Request)) {
	mux.Handle(pattern, HandlerFunc(handler))
}
// src/net/http/server.go#L2027
func (mux *ServeMux) Handle(pattern string, handler Handler) {
  // 加写锁，写操作完成之前，所有mux上的读锁都会阻塞
	mux.mu.Lock()
	defer mux.mu.Unlock()

  // pattern不能为空字符串
	if pattern == "" {
		panic("http: invalid pattern " + pattern)
	}
  // handler不能为nil
	if handler == nil {
		panic("http: nil handler")
	}
  // pattern不允许重复注册
	if mux.m[pattern].explicit {
		panic("http: multiple registrations for " + pattern)
	}

  // 初始化并注册handler
	if mux.m == nil {
		mux.m = make(map[string]muxEntry)
	}
	mux.m[pattern] = muxEntry{explicit: true, h: handler, pattern: pattern}

	if pattern[0] != '/' {
		mux.hosts = true
	}

	n := len(pattern)
	if n > 0 && pattern[n-1] == '/' && !mux.m[pattern[0:n-1]].explicit {
    // 如果pattern带hostname, strip并进行跳转
		path := pattern
		if pattern[0] != '/' {
      // 所谓的strip，实际上就是直接找到第一个slash
			path = pattern[strings.Index(pattern, "/"):]
		}

    // 执行跳转
		url := &url.URL{Path: path}
		mux.m[pattern[0:n-1]] = muxEntry{h: RedirectHandler(url.String(), StatusMovedPermanently), pattern: pattern}
	}
}
```

还有一种`http.HandlerFunc`

```
type HandlerFunc func(ResponseWriter, *Request)

// 在一个function上面定一个method，一种有趣的调用方式
func (f HandlerFunc) ServeHTTP(w ResponseWriter, r *Request) {
	f(w, r)
}
```

`HandlerFunc`可以当作独立的Handler来传给`ListenAndServe`.

至此，`net/http`的常规操作都介绍完了，接下来分析以下`net/http`中的几个重要类型.

先看`Request`：

```
// src/net/http/request.go#L79
type Request struct {
  // 取自协议包第一行. Method URI Proto
	Method string
	RequestURI string
	Proto      string
	ProtoMajor int    // 从Proto获取
	ProtoMinor int    // 从Proto获取

  // 从RequestURI解析得到，具体在后面介绍
	URL *url.URL

  // 从HTTP包第二行就是Header, 和第一行在同一个TCP包中
	Header Header

  // 1. GET /index.html HTTP/1.1
  //    Host: www.example.com
  // 2. GET http://www.example.com/index.html HTTP/1.1
  //    Host: doesntmatter，这样的话，所有的Host头都会被忽略
	Host string

	Body io.ReadCloser
	ContentLength int64
	TransferEncoding []string
	Close bool
	Form url.Values
	PostForm url.Values
	MultipartForm *multipart.Form
	Trailer Header
	RemoteAddr string
	TLS *tls.ConnectionState
	Cancel <-chan struct{}
	Response *Response
	ctx context.Context
}

// src/net/url/url.go#L313
type URL struct {
	Scheme     string
	Opaque     string
	User       *Userinfo
	Host       string
	Path       string
	RawPath    string
	ForceQuery bool
	RawQuery   string
	Fragment   string
}
```

对`Host`的处理比较有意思，如果`URL`里面带了`Host`, `Header`中的`Host`会被忽略：

```
GET http://www.example.com/ HTTP/1.1
Host: host(ignored)
```

如果`URL`里面没有Host信息，才会用`Header`中的`Host`，存在安全隐患：

```
GET / HTTP/1.1
Host: www.example.com
```

如果执行`curl www.example.com`，发出的请求实际上是后者. 涉及请求转发的时候
HTTP实际的请求包可能会有不同的行为.

