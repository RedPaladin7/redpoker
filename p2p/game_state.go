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

type AtomicInt struct {
	value int32
}

func NewAtomicInt(value int32) *AtomicInt{
	return &AtomicInt{
		value: value,
	}
}

func (a *AtomicInt) String() string {
	return fmt.Sprintf("%d", a.value)
}

func (a *AtomicInt) Set(value int32){
	atomic.StoreInt32(&a.value, value)
}

func (a *AtomicInt) Get() int32{
	return atomic.LoadInt32(&a.value)
}

func (a *AtomicInt) Inc(){
	a.Set(a.Get()+1)
}

type PlayerActionsRecv struct {
	recvActions map[string]MessagePlayerAction
	mu sync.RWMutex
}

func NewPlayerActionsRecv() *PlayerActionsRecv {
	return &PlayerActionsRecv{
		recvActions: make(map[string]MessagePlayerAction),
	}
}

func (pa *PlayerActionsRecv) addAction(from string, action MessagePlayerAction) {
	pa.mu.Lock()
	defer pa.mu.Unlock()
	
	pa.recvActions[from] = action 
}

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
	currentStatus *AtomicInt
	currentPlayerAction *AtomicInt
	currentDealer *AtomicInt
	currentPlayerTurn *AtomicInt
	recvPlayerActions *PlayerActionsRecv
}

func NewGame(addr string, bc chan BroadcastTo) *Game {
	g := &Game{
		playersReady: NewPlayersReady(),
		playersList: PlayersList{},
		broadcastch: bc,
		listenAddr: addr,
		currentStatus: NewAtomicInt(int32(GameStatusConnected)),
		currentPlayerAction: NewAtomicInt(0),
		currentDealer: NewAtomicInt(0),
		recvPlayerActions: NewPlayerActionsRecv(),
		currentPlayerTurn: NewAtomicInt(0),
	}
	g.playersList = append(g.playersList, addr)
	go g.loop()
	return g
}

func (g *Game) getNextGameStatus() GameStatus {
	switch GameStatus(g.currentStatus.Get()){
	case GameStatusPreFlop:
		return GameStatusFlop
	case GameStatusFlop:
		return GameStatusTurn
	case GameStatusTurn:
		return GameStatusRiver
	default:
		panic("invalid game status")
	}
}

func (g *Game) canTakeAction(from string) bool {
	currentPlayerAddr := g.playersList[g.currentPlayerTurn.Get()]
	return currentPlayerAddr == from 
}

func (g *Game) isFromCurrentDealer(from string) bool {
	return g.playersList[g.currentDealer.Get()] == from
}

func (g *Game) handlePlayerAction(from string, action MessagePlayerAction) error {
	if !g.canTakeAction(from) {
		return fmt.Errorf("player (%s) taking action before his turn ", from)
	}
	if action.CurrentGameStatus != GameStatus(g.currentStatus.Get()) && !g.isFromCurrentDealer(from){
		return fmt.Errorf("player (%s) has not the correct game status (%s)", from, action.CurrentGameStatus)
	}
	logrus.WithFields(logrus.Fields{
		"we": g.listenAddr,
		"from": from,
		"action": action,
	}).Info("received player action")
	if g.playersList[g.currentDealer.Get()] == from {
		g.currentStatus.Set(int32(g.getNextGameStatus()))
	}
	g.recvPlayerActions.addAction(from, action)
	g.incNextPlayer()
	return nil
}

func (g *Game) TakeAction(action PlayerAction, value int) error {
	if !g.canTakeAction(g.listenAddr){
		return fmt.Errorf("I am taking action before it is my turn: %s\n", g.listenAddr)
	}
	g.currentPlayerAction.Set((int32)(action))
	g.incNextPlayer()
	if g.listenAddr == g.playersList[g.currentDealer.Get()]{
		g.currentStatus.Set(int32(g.getNextGameStatus()))
	}
	g.sendToPlayers(MessagePlayerAction{
		Action: action,
		CurrentGameStatus: GameStatus(g.currentStatus.Get()),
		Value: value,
	}, g.getOtherPlayers()...)
	return nil
}

func (g *Game) incNextPlayer(){
	if len(g.playersList)-1 == int(g.currentPlayerTurn.Get()){
		g.currentPlayerTurn.Set(0)
		return 
	}
	g.currentPlayerTurn.Inc()
}

func (g *Game) SetStatus(s GameStatus){
	g.setStatus(s)
}

func (g *Game) setStatus(s GameStatus) {
	if s == GameStatusPreFlop{
		g.incNextPlayer()
	}
	if GameStatus(g.currentStatus.Get()) != s{
		// atomic.StoreInt32((*int32)(&g.currentStatus), (int32)(s))
		g.currentStatus.Set(int32(s))
	}
}

func (g *Game) getCurrentDealerAddr() (string, bool) {
	currentDealerAddr := g.playersList[g.currentDealer.Get()]
	return currentDealerAddr, currentDealerAddr == g.listenAddr
}

func (g *Game) SetPlayerReady(from string) {
	g.playersReady.addRecvStatus(from)
	if g.playersReady.len() < 2 {
		return 
	}
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
			"gameStatus": GameStatus(g.currentStatus.Get()),
			"currentDealer": currentDealerAddr,
			"nextPlayerTurn": g.currentPlayerTurn,
			"playerActions": g.recvPlayerActions.recvActions,
			"currentPlayerAction": PlayerAction(g.currentPlayerAction.Get()),
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
