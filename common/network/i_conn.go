package network

import "net"

type IConn interface {
	Receive() (IMessage, error)
	Write(int, []byte) error
	Close() error
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
}
