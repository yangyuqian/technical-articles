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

```
