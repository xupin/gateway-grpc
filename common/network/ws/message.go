package ws

type Message struct {
	Type    int
	Payload []byte
}

func (r *Message) GetType() int {
	return r.Type
}

func (r *Message) GetPayload() []byte {
	return r.Payload
}
