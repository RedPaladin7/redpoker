package p2p

import (
	"fmt"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"

	// "sync/atomic"
	"time"

	// "github.com/RedPaldin7/redpoker/deck"
	"github.com/sirupsen/logrus"
)

type PlayerAction byte 

const (
	PlayerActionFold PlayerAction = iota + 1
	PlayerActionCheck
	PlayerActionBet
)

type PlayersReady struct {
	recvStatus map[string]bool 
	mu sync.RWMutex
}

func NewPlayersReady() *PlayersReady {
	return &PlayersReady{
		recvStatus: make(map[string]bool),
	}
}

func (pr *PlayersReady) addRecvStatus(from string) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	pr.recvStatus[from] = true 
}

func (pr *PlayersReady) len() int {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	return len(pr.recvStatus)
}

func (pr *PlayersReady) clear() {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	pr.recvStatus = make(map[string]bool)
}

type Game struct {
	listenAddr string
	broadcastch chan BroadcastTo 
	playersReady *PlayersReady
	playersList PlayersList
	currentStatus GameStatus
	currentDealer int32
}

func NewGame(addr string, bc chan BroadcastTo) *Game {
	g := &Game{
		playersReady: NewPlayersReady(),
		playersList: PlayersList{},
		broadcastch: bc,
		listenAddr: addr,
		currentStatus: GameStatusConnected,
		currentDealer: -1,
	}
	g.playersList = append(g.playersList, addr)
	go g.loop()
	return g
}

func (g *Game) Fold() {
	g.SetStatus(GameStatusPlayerReady)
	g.sendToPlayers(MessagePlayerAction{
		Action: PlayerActionFold,
		CurrentGameStatus: g.currentStatus,
	}, g.getOtherPlayers()...)
}

func (g *Game) SetStatus(s GameStatus){
	g.setStatus(s)
}

func (g *Game) setStatus(s GameStatus) {
	if g.currentStatus != s{
		atomic.StoreInt32((*int32)(&g.currentStatus), (int32)(s))
	}
}

func (g *Game) getCurrentDealerAddr() (string, bool) {
	currentDealer := g.playersList[0]
	if g.currentDealer > -1 {
		currentDealer = g.playersList[g.currentDealer]
	}
	return currentDealer, g.listenAddr == currentDealer
}

func (g *Game) SetPlayerReady(from string) {
	logrus.WithFields(logrus.Fields{
		"we": g.listenAddr,
		"player": from,
	}).Info("setting player status to ready")
	g.playersReady.addRecvStatus(from)
	if g.playersReady.len() < 2 {
		return 
	}

	// g.playersReady.clear()
	if _, ok := g.getCurrentDealerAddr(); ok{
		g.InitiateShuffleAndDeal()
	}
}

func (g *Game) ShuffleAndEncrypt(from string, deck [][]byte) error {
	prevPlayerAddr := g.playersList[g.getPrevPositionOnTable()]
	if from != prevPlayerAddr {
		return fmt.Errorf("received encrypted deck from the wrong player (%s) should be (%s)", from, prevPlayerAddr)
	}

	_, isDealer := g.getCurrentDealerAddr()

	if isDealer && from == prevPlayerAddr {
		g.setStatus(GameStatusPreFlop)
		g.sendToPlayers(MessagePreFlop{}, g.getOtherPlayers()...)
		return nil
	}
	dealToPlayer := g.playersList[g.getNextPositionOnTable()]
	logrus.WithFields(logrus.Fields{
		"recvFromPlayer": from,
		"we": g.listenAddr,
		"dealingToPlayer": dealToPlayer,
	}).Info("received cards and going to shuffle")
	g.sendToPlayers(MessageEncDeck{Deck:[][]byte{}}, dealToPlayer)
	g.setStatus(GameStatusDealing)
	return nil
}

func (g *Game) InitiateShuffleAndDeal() {
	dealToPlayerAddr := g.playersList[g.getNextPositionOnTable()]
	g.setStatus(GameStatusDealing)
	g.sendToPlayers(MessageEncDeck{Deck: [][]byte{}}, dealToPlayerAddr)
	logrus.WithFields(logrus.Fields{
		"we": g.listenAddr,
		"to": dealToPlayerAddr,
	}).Info("dealing cards")
}

func (g *Game) SetReady(){
	g.playersReady.addRecvStatus(g.listenAddr)
	g.sendToPlayers(MessageReady{}, g.getOtherPlayers()...)
	g.setStatus(GameStatusPlayerReady)
}

func (g *Game) sendToPlayers(payload any, addr ...string) {
	g.broadcastch <- BroadcastTo{
		To: addr,
		Payload: payload,
	}
	logrus.WithFields(logrus.Fields{
		"payload": payload,
		"player": addr,
		"we": g.listenAddr,
	}).Info("sending payload to player")
}

func (g *Game) AddPlayer(from string) {
	g.playersList = append(g.playersList, from)
	sort.Sort(g.playersList)
}

func (g *Game) loop() {
	ticker := time.NewTicker(time.Second * 5)
	for {
		<-ticker.C
		currentDealerAddr, _ := g.getCurrentDealerAddr()
		logrus.WithFields(logrus.Fields{
			"we": g.listenAddr,
			"players": g.playersList,
			"status": g.currentStatus,
			"currentDealer": currentDealerAddr,
			// "playersReady": g.playersReady.recvStatus,
		}).Info()
	}
}

func (g *Game) getOtherPlayers() []string {
	players := []string{}
	
	for _, addr := range g.playersList{
		if addr == g.listenAddr {
			continue 
		}
		players = append(players, addr)
	}
	return players
}

func (g *Game) getPositionOnTable() int {
	for i := 0; i < len(g.playersList); i++ {
		if g.playersList[i] == g.listenAddr{
			return i
		}
	}
	panic("player does not exist in the playersList")
}

func (g *Game) getPrevPositionOnTable() int {
	ourPosition := g.getPositionOnTable()
	if ourPosition == 0{
		return len(g.playersList) - 1
	}
	return ourPosition - 1
}

func (g *Game) getNextPositionOnTable() int {
	ourPosition := g.getPositionOnTable()
	if ourPosition == len(g.playersList) - 1 {
		return 0
	} 
	return ourPosition + 1
}

type PlayersList []string

func (list PlayersList) Len() int {return len(list)}

func (list PlayersList) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
}

func (list PlayersList) Less(i, j int) bool {
	portI, _ := strconv.Atoi(list[i][1:])
	portJ, _ := strconv.Atoi(list[j][1:])
	return portI < portJ
}

type Player struct {
	Status GameStatus
	ListenAddr string 
}

func (p *Player) String() string {
	return fmt.Sprintf("%s:%s", p.ListenAddr, p.Status)
}
