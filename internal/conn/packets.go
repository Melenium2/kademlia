package conn

const (
	PingMessage byte = iota + 1
	PongMessage
)

type Packet interface {
	Name() string
	IAm() byte
}

type Ping struct {
}

func (p *Ping) Name() string {
	return "PING"
}

func (p *Ping) IAm() byte {
	return PingMessage
}

type Pong struct {
}

func (p *Pong) Name() string {
	return "PONG"
}

func (p *Pong) IAm() byte {
	return PongMessage
}

func makePing() Ping {
	return Ping{}
}

func Encode(req Ping) []byte {
	return nil
}
