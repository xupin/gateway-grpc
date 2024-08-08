package ws

import (
	"errors"
	"net"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/xupin/gateway-grpc/common/network"
)

// ws连接
type wsConn struct {
	// 标识id
	id string
	// 底层长连接
	conn *websocket.Conn
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
func NewConn(opts ...Option) *wsConn {
	opt := newOptions(opts...)
	return &wsConn{
		id:        uuid.NewString(),
		conn:      nil,
		inChan:    make(chan network.IMessage, opt.InChanSize),
		outChan:   make(chan network.IMessage, opt.OutChanSize),
		closeChan: make(chan struct{}, 1),
	}
}

// 开启连接
func (c *wsConn) Open(w http.ResponseWriter, r *http.Request) error {
	upgrade := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
	conn, err := upgrade.Upgrade(w, r, nil)
	if err != nil {
		return err
	}
	c.conn = conn
	// 监听客户端消息
	go func() {
		for {
			msgType, payload, err := c.conn.ReadMessage()
			if err != nil {
				_ = c.Close()
				return
			}
			select {
			case c.inChan <- &Message{
				Type:    msgType,
				Payload: payload,
			}:
			case <-c.closeChan:
				return
			}
		}
	}()
	// 向连接写入数据
	go func() {
		for {
			select {
			case msg := <-c.outChan:
				_ = c.conn.WriteMessage(msg.GetType(), msg.GetPayload())
			case <-c.closeChan:
				return
			}
		}
	}()
	return nil
}

// 建立连接
func (c *wsConn) Connect(addr string) error {
	conn, _, err := websocket.DefaultDialer.Dial(addr, nil)
	if err != nil {
		panic(err)
	}
	c.conn = conn
	// 监听客户端消息
	go func() {
		for {
			msgType, payload, err := c.conn.ReadMessage()
			if err != nil {
				_ = c.Close()
				return
			}
			select {
			case c.inChan <- &Message{
				Type:    msgType,
				Payload: payload,
			}:
			case <-c.closeChan:
				return
			}
		}
	}()
	// 向连接写入数据
	go func() {
		for {
			select {
			case msg := <-c.outChan:
				_ = c.conn.WriteMessage(msg.GetType(), msg.GetPayload())
			case <-c.closeChan:
				return
			}
		}
	}()
	return nil
}

// 关闭连接
func (c *wsConn) Close() error {
	_ = c.conn.Close()
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if !c.isClosed {
		close(c.closeChan)
		c.isClosed = true
	}
	return nil
}

// 接收数据
func (c *wsConn) Receive() (msg network.IMessage, err error) {
	select {
	case msg = <-c.inChan:
	case <-c.closeChan:
		err = errors.New("connection already closed")
	}
	return
}

// 写入数据
func (c *wsConn) Write(t int, payload []byte) (err error) {
	select {
	case c.outChan <- &Message{
		Type:    t,
		Payload: payload,
	}:
	case <-c.closeChan:
		err = errors.New("connection already closed")
	}
	return
}

// 获取本地地址
func (c *wsConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

// 获取远程地址
func (c *wsConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}
