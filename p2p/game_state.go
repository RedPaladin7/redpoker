package p2p

import (
	"fmt"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	// "github.com/RedPaldin7/redpoker/deck"
	"github.com/sirupsen/logrus"
)

type PlayersList []*Player 

func (list PlayersList) Len() int {return len(list)}

func (list PlayersList) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
}

func (list PlayersList) Less(i, j int) bool {
	portI, _ := strconv.Atoi(list[i].ListenAddr[1:])
	portJ, _ := strconv.Atoi(list[j].ListenAddr[1:])
	return portI < portJ
}

type Player struct {
	Status GameStatus
	ListenAddr string 
}

func (p *Player) String() string {
	return fmt.Sprintf("%s:%s", p.ListenAddr, p.Status)
}

type GameState struct {
	listenAddr string 
	broadcastch chan BroadcastTo
	isDealer bool
	gameStatus GameStatus

	playersWaitingForCards int32
	players map[string]*Player
	playersLock sync.RWMutex
	playersList PlayersList
}

func NewGameState(addr string, broadcastch chan BroadcastTo) *GameState{
	g := &GameState{
		listenAddr: addr,
		broadcastch: broadcastch,
		isDealer: false,
		gameStatus: GameStatusWaitingForCards,
		players: make(map[string]*Player),
	}
	g.AddPlayer(addr, GameStatusWaitingForCards)
	go g.loop()
	return g
}

func (g *GameState) SetStatus(s GameStatus) {
	if g.gameStatus != s {
		atomic.StoreInt32((*int32)(&g.gameStatus), (int32)(s))
	}
}

func (g *GameState) AddPlayerWaitingForCards(){
	atomic.AddInt32(&g.playersWaitingForCards, 1)
}

func (g *GameState) CheckNeedDealCards(){
	playersWaiting := atomic.LoadInt32(&g.playersWaitingForCards)

	if playersWaiting == int32(len(g.players)) && g.isDealer && g.gameStatus == GameStatusWaitingForCards {
		logrus.WithFields(logrus.Fields{
			"addr": g.listenAddr,
		}).Info("need to deal cards")
		g.InitiateShuffleAndDeal()
	}
}

func (g *GameState) GetPlayersWithStatus(s GameStatus) []string {
	players := []string{}
	for addr, player := range g.players{
		if player.Status == s{
			players = append(players, addr)
		}
	}
	return players
}

func (g *GameState) getPositionOnTable() int {
	for i := 0; i < len(g.playersList); i++ {
		if g.playersList[i].ListenAddr == g.listenAddr{
			return i
		}
	}
	panic("player does not exist in the playersList")
}

func (g *GameState) getPrevPositionOnTable() int {
	ourPosition := g.getPositionOnTable()
	if ourPosition == 0{
		return len(g.playersList) - 1
	}
	return ourPosition - 1
}

func (g *GameState) getNextPositionOnTable() int {
	ourPosition := g.getPositionOnTable()
	if ourPosition == len(g.playersList) - 1 {
		return 0
	} 
	return ourPosition + 1
}

func (g *GameState) ShuffleAndEncrypt(from string, deck [][]byte) error {
	g.SetPlayerStatus(from, GameStatusShuffleAndDeal)
	prevPlayer := g.playersList[g.getPrevPositionOnTable()]

	if g.isDealer && from == prevPlayer.ListenAddr {
		logrus.Info("shuffle round completed")
		return nil
	}
	dealToPlayer := g.playersList[g.getNextPositionOnTable()]
	logrus.WithFields(logrus.Fields{
		"recvFromPlayer": from,
		"we": g.listenAddr,
		"dealingToPlayer": dealToPlayer,
	}).Info("received cards and going to shuffle")
	g.SendToPlayer(dealToPlayer.ListenAddr, MessageEncDeck{Deck:[][]byte{}})
	g.SetStatus(GameStatusShuffleAndDeal)
	return nil
}

func (g *GameState) InitiateShuffleAndDeal(){
	dealToPlayer := g.playersList[g.getNextPositionOnTable()]
	g.SendToPlayer(dealToPlayer.ListenAddr, MessageEncDeck{Deck: [][]byte{}})
	g.SetStatus(GameStatusShuffleAndDeal)
}

func (g *GameState) SendToPlayer(addr string, payload any) {
	g.broadcastch <- BroadcastTo{
		To: []string{addr},
		Payload: payload,
	}
	logrus.WithFields(logrus.Fields{
		"payload": payload,
		"player": addr,
	}).Info("sending payload to player")
}

func (g *GameState) SendToPlayerWithStatus(payload any, s GameStatus){
	players := g.GetPlayersWithStatus(s)
	g.broadcastch <- BroadcastTo{
		To: players,
		Payload: payload,
	}
	logrus.WithFields(logrus.Fields{
		"payload": payload,
		"players": players,
	}).Info("sending to players")
}  

func (g *GameState) SetPlayerStatus(addr string, status GameStatus){
	player, ok := g.players[addr]
	if !ok {
		panic("player could not be found, although it should exist")
	}
	player.Status = status
	g.CheckNeedDealCards()
}

func (g *GameState) AddPlayer(addr string, status GameStatus){
	g.playersLock.Lock()
	defer g.playersLock.Unlock()
	if status == GameStatusWaitingForCards {
		g.AddPlayerWaitingForCards()
	}
	player := &Player{
		ListenAddr: addr,
	}
	g.players[addr] = player
	g.playersList = append(g.playersList, player)
	sort.Sort(g.playersList)

	g.SetPlayerStatus(addr, status)

	logrus.WithFields(logrus.Fields{
		"addr": addr,
		"status": status,
	}).Info("new player joined")
}

func (g *GameState) loop() {
	ticker := time.NewTicker(time.Second * 5)
	for {
		<-ticker.C
		logrus.WithFields(logrus.Fields{
			"we": g.listenAddr,
			"players": g.playersList,
			"status": g.gameStatus,
		}).Info()
	}
}