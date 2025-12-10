package p2p

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
		return "IDLE"
	}
}

const (
	PlayerActionFold PlayerAction = iota + 1
	PlayerActionCheck
	PlayerActionBet
)

type GameStatus int32

func (g GameStatus) String() string {
	switch g {
	case GameStatusConnected:
		return "CONNECTED"
	case GameStatusPlayerReady:
		return "PLAYER READY"
	case GameStatusDealing:
		return "DEALING"
	case GameStatusFolded:
		return "FOLDED"
	case GameStatusChecked:
		return "CHECKED"
	case GameStatusPreFlop:
		return "PRE-FLOP"
	case GameStatusFlop:
		return "FLOP"
	case GameStatusTurn:
		return "TURN"
	case GameStatusRiver:
		return "RIVER"
	default:
		return "unknown"
	}
}

const (
	GameStatusConnected GameStatus = iota
	GameStatusPlayerReady
	GameStatusDealing 
	GameStatusFolded
	GameStatusChecked
	GameStatusPreFlop 
	GameStatusFlop 
	GameStatusTurn 
	GameStatusRiver
)