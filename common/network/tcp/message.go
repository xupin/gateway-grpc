package tcp

type Message struct {
	Payload []byte
}

func (r *Message) GetType() int {
	return 0
}

func (r *Message) GetPayload() []byte {
	return r.Payload
}
