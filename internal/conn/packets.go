package conn

const (
	PingMessage byte = iota + 1
	PongMessage
)

type Packet interface {
	Name() string
}

type Ping struct {
}

func (p *Ping) Name() string {
	return "PING"
}

type Pong struct {
}

func (p *Pong) Name() string {
	return "PONG"
}

func makePing() Ping {
	return Ping{}
}

func Encode(req Ping) []byte {
	return nil
}
