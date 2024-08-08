package tcp

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type Server struct {
	Addr    string
	Handler Handler

	inShutdown atomic.Bool

	mu        sync.Mutex
	conns     map[*conn]struct{}
	listeners map[*net.Listener]struct{}

	listenerGroup sync.WaitGroup
}

type conn struct {
	rwc    net.Conn
	server *Server
}

func (r *Server) ListenAndServe() error {
	ln, err := net.Listen("tcp", r.Addr)
	if err != nil {
		return fmt.Errorf("监听错误: %v", err)
	}
	return r.Serve(ln)
}

func (r *Server) Serve(l net.Listener) error {
	r.mu.Lock()
	r.conns = make(map[*conn]struct{})
	r.mu.Unlock()

	defer r.Close()

	if !r.trackListener(&l, true) {
		return http.ErrServerClosed
	}
	defer r.trackListener(&l, false)

	var tempDelay time.Duration // 临时延迟变量，用于重试接受连接时的暂时性错误

	for {
		rwc, err := l.Accept()
		if err != nil {
			if r.shuttingDown() {
				return http.ErrServerClosed
			}
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				time.Sleep(tempDelay)
				continue
			}
			return err
		}
		tempDelay = 0

		conn := &conn{rwc: rwc, server: r}
		r.mu.Lock()
		r.conns[conn] = struct{}{}
		r.mu.Unlock()

		go r.serveConn(conn)
	}
}

func (r *Server) serveConn(conn *conn) {
	defer func() {
		conn.rwc.Close()
		r.mu.Lock()
		delete(r.conns, conn)
		r.mu.Unlock()
	}()
	r.Handler.ServeTCP(conn.rwc)
}

func (r *Server) Close() error {
	r.inShutdown.Store(true)
	r.mu.Lock()
	defer r.mu.Unlock()
	err := r.closeListenersLocked()

	r.mu.Unlock()
	r.listenerGroup.Wait()
	r.mu.Lock()

	for conn := range r.conns {
		conn.rwc.Close()
		delete(r.conns, conn)
	}
	return err
}

func (r *Server) shuttingDown() bool {
	return r.inShutdown.Load()
}

func (r *Server) trackListener(ln *net.Listener, add bool) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.listeners == nil {
		r.listeners = make(map[*net.Listener]struct{})
	}
	if add {
		if r.shuttingDown() {
			return false
		}
		r.listeners[ln] = struct{}{}
		r.listenerGroup.Add(1)
	} else {
		delete(r.listeners, ln)
		r.listenerGroup.Done()
	}
	return true
}

func (r *Server) closeListenersLocked() error {
	var err error
	for ln := range r.listeners {
		if cerr := (*ln).Close(); cerr != nil && err == nil {
			err = cerr
		}
	}
	return err
}

type Handler interface {
	ServeTCP(net.Conn)
}

type HandlerFunc func(conn net.Conn)

func (f HandlerFunc) ServeTCP(conn net.Conn) {
	f(conn)
}
