package p2p

import (
	"net"
)

type Message struct {
	Payload any
	From    net.Addr
}

func NewMessage(from net.Addr, payload any) *Message {
	return &Message{
		From: from,
		Payload: payload,
	}
}

type Handshake struct {
	Version string 
	GameVariant GameVariant
	GameStatus GameStatus
}

type MessagePeerList []net.Addr