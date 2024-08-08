package network

type IMessage interface {
	GetType() int
	GetPayload() []byte
}
