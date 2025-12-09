package p2p

// import "github.com/RedPaldin7/redpoker/deck"

type PlayerAction byte 

func (pa PlayerAction) String() string {
	switch pa {
	case PlayerActionFold:
		return "FOLDED"
	case PlayerActionCheck:
		return "CHECKED"
	case PlayerActionBet:
		return "BET"
	default:
		return "INVALID ACTION"
	}
}

const (
	PlayerActionFold PlayerAction = iota + 1
	PlayerActionCheck
	PlayerActionBet
)

type Message struct {
	Payload any
	From    string
}

type BroadcastTo struct {
	To []string 
	Payload any 
}

func NewMessage(from string, payload any) *Message {
	return &Message{
		From: from,
		Payload: payload,
	}
}

type Handshake struct {
	Version string 
	GameVariant GameVariant
	GameStatus GameStatus
	ListenAddr string
}

type MessagePlayerAction struct {
	CurrentGameStatus GameStatus
	Action PlayerAction
	Value int 
}

type MessagePreFlop struct {

}

func (msg MessagePreFlop) String() string {
	return "MSG: PREFLOP"
}

type MessagePeerList struct {
	Peers []string
}

type MessageEncDeck struct {
	Deck [][]byte
}

type MessageReady struct {

}

func (msg MessageReady) String() string {
	return "MSG: READY"
}