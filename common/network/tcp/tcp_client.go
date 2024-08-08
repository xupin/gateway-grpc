package tcp

import (
	"bufio"
	"errors"
	"net"
	"sync"

	"github.com/google/uuid"
	"github.com/xupin/gateway-grpc/common/network"
)

// tcp连接
type tcpConn struct {
	// 标识id
	id string
	// 底层长连接
	conn net.Conn
	// 读队列
	inChan chan network.IMessage
	// 写队列
	outChan chan network.IMessage
	// 关闭通知
	closeChan chan struct{}
	// 保护 closeChan 只被执行一次
	mutex sync.Mutex
	// closeChan状态
	isClosed bool
}

// 默认选项
const (
	// 读队列大小
	defaultInChanSize = 1024
	// 写队列大小
	defaultOutChanSize = 1024
)

// 新建实例
func NewConn(opts ...Option) *tcpConn {
	opt := newOptions(opts...)
	return &tcpConn{
		id:        uuid.NewString(),
		conn:      nil,
		inChan:    make(chan network.IMessage, opt.InChanSize),
		outChan:   make(chan network.IMessage, opt.OutChanSize),
		closeChan: make(chan struct{}, 1),
	}
}

// 开启连接（服务端）
func (r *tcpConn) Open(conn net.Conn) error {
	r.conn = conn
	// 监听客户端消息
	go func() {
		reader := bufio.NewReader(r.conn)
		for {
			payload := make([]byte, 1024)
			n, err := reader.Read(payload)
			if err != nil {
				_ = r.Close()
				return
			}
			select {
			case r.inChan <- &Message{
				Payload: payload[:n],
			}:
			case <-r.closeChan:
				return
			}
		}
	}()
	// 向连接写入数据
	go func() {
		writer := bufio.NewWriter(r.conn)
		for {
			select {
			case message := <-r.outChan:
				_, err := writer.Write(message.GetPayload())
				if err != nil {
					_ = r.Close()
					return
				}
				_ = writer.Flush()
			case <-r.closeChan:
				return
			}
		}
	}()
	return nil
}

// 关闭连接
func (r *tcpConn) Close() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if !r.isClosed {
		close(r.closeChan)
		r.isClosed = true
		return r.conn.Close()
	}
	return nil
}

// 接收数据
func (r *tcpConn) Receive() (msg network.IMessage, err error) {
	select {
	case msg = <-r.inChan:
	case <-r.closeChan:
		err = errors.New("connection already closed")
	}
	return
}

// 写入数据
func (r *tcpConn) Write(t int, payload []byte) (err error) {
	select {
	case r.outChan <- &Message{
		Payload: payload,
	}:
	case <-r.closeChan:
		err = errors.New("connection already closed")
	}
	return
}

// 获取本地地址
func (r *tcpConn) LocalAddr() net.Addr {
	return r.conn.LocalAddr()
}

// 获取远程地址
func (r *tcpConn) RemoteAddr() net.Addr {
	return r.conn.RemoteAddr()
}
