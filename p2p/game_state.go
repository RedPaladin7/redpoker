package p2p

type GameStatus uint32

func (g GameStatus) String() string {
	switch g {
	case GameStatusWaiting:
		return "WAITING"
	case GameStatusDealing:
		return "DEALING"
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
	GameStatusWaiting GameStatus = iota
	GameStatusDealing 
	GameStatusPreFlop 
	GameStatusFlop 
	GameStatusTurn 
	GameStatusRiver
)

type GameState struct {
	isDealer bool
	gameStatus GameStatus
}

func NewGameState() *GameState{
	return &GameState{

	}
}