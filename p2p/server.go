package p2p

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// net.Conn is general bidirection network connection (socket)
// sync.RWMutex is read write lock (can be held by one writer or multiple reasders)

type GameVariant uint8

func (gv GameVariant) String() string {
	switch gv {
	case TexasHoldem:
		return "TEXAS HOLDEM"
	case Other:
		return "Other"
	default: 
		return "UNKNOWN"
	}
}

const (
	TexasHoldem GameVariant = iota
	Other
)

type ServerConfig struct {
	Version    string
	ListenAddr string
	GameVariant GameVariant
}

type Server struct {
	ServerConfig
	transport *TCPTransport
	peerLock sync.RWMutex
	peers    map[net.Addr]*Peer
	addPeer  chan *Peer
	delPeer  chan *Peer
	msgCh    chan *Message
	gameState *GameState
}

func NewServer(cfg ServerConfig) *Server {
	s :=  &Server{
		ServerConfig: cfg,
		peers:        make(map[net.Addr]*Peer),
		addPeer:      make(chan *Peer, 10),
		delPeer:      make(chan *Peer),
		msgCh:        make(chan *Message),
		gameState: NewGameState(),
	}
	tr := NewTCPTransport(s.ListenAddr)
	s.transport = tr 

	tr.AddPeer = s.addPeer
	tr.DelPeer = s.delPeer

	return s
}

func (s *Server) Start() {
	go s.loop()
	logrus.WithFields(logrus.Fields{
		"port": s.ListenAddr,
		"variant": s.GameVariant,
	}).Info("started new game server")
	s.transport.ListenAndAccept()
}

func (s *Server) sendPeerList(p *Peer) error {
	peerList := MessagePeerList{
		Peers: s.Peers(),
	}
	// for _, peer := range s.peers {
	// 	peerList.Peers = append(peerList.Peers, peer.listenAddr)
	// }
	if len(peerList.Peers) == 0 {
		return nil
	}
	msg := NewMessage(s.ListenAddr, peerList)
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(msg); err != nil{
		return err
	}
	return p.Send(buf.Bytes())
}

func (s *Server) AddPeer(p *Peer){
	s.peerLock.Lock()
	defer s.peerLock.Unlock()
	s.peers[p.conn.RemoteAddr()] = p
}

func (s *Server) Peers() []string {
	s.peerLock.RLock()
	defer s.peerLock.RUnlock()

	peers := make([]string, len(s.peers))
	it := 0
	for _, peer := range s.peers {
		peers[it] = peer.listenAddr
		it++
	}
	return peers
}

func (s *Server) SendHandshake(p *Peer) error {
	hs := &Handshake{
		GameVariant: s.GameVariant,
		Version: s.Version,
		GameStatus: s.gameState.gameStatus,
		ListenAddr: s.ListenAddr,
	}
	buf := new(bytes.Buffer)

	if err := gob.NewEncoder(buf).Encode(hs); err != nil{
		return err
	}
	return p.Send(buf.Bytes())
}

func (s *Server) isInPeerList(addr string) bool {
	peers := s.Peers()
	for i := 0; i < len(peers); i++ {
		if peers[i] == addr{
			return true
		}
	}
	return false
}

func (s *Server) Connect(addr string) error {
	if s.isInPeerList(addr) {
		return nil
	}
	conn, err := net.DialTimeout("tcp", addr, 1*time.Second)
	if err != nil {
		return err
	}
	peer := &Peer{
		conn: conn,
		outbound: true,
	}
	s.addPeer <- peer
	return s.SendHandshake(peer)
}

func (s *Server) loop() {
	for {
		select {
		case peer := <-s.delPeer:
			logrus.WithFields(logrus.Fields{
				"addr": peer.conn.RemoteAddr(),
			}).Info("player disconnected")
			delete(s.peers, peer.conn.RemoteAddr())

		case peer := <-s.addPeer:
			if err := s.handleNewPeer(peer); err != nil {
				logrus.Errorf("handle new peer error: %s", err)
			}

		case msg := <-s.msgCh:
			if err := s.handleMessage(msg); err != nil {
				panic(err)
			}
		}
	}
}

func (s *Server) handleNewPeer (peer *Peer) error {
	hs, err := s.handshake(peer)
	if err != nil{
		peer.conn.Close()
		delete(s.peers, peer.conn.RemoteAddr())
		return fmt.Errorf("%s: handshake with incoming player failed: %s", s.ListenAddr, err)
	}
	go peer.ReadLoop(s.msgCh)
	if !peer.outbound{
		if err := s.SendHandshake(peer); err != nil{
			peer.conn.Close()
			delete(s.peers, peer.conn.RemoteAddr())
			fmt.Errorf("failed to send handshake with peer: %s", err)
		}
		go func(){
			if err := s.sendPeerList(peer); err != nil{
				logrus.Errorf("peerlist error: %s", err)
			}
		}()
	}
	logrus.WithFields(logrus.Fields{
		"peer": peer.conn.RemoteAddr(),
		"version": hs.Version,
		"variant": hs.GameVariant,
		"gameStatus": hs.GameStatus,
		"listenAddr": peer.listenAddr,
		"we": s.ListenAddr,
	}).Info("handshake successfull: new player connected")
	// s.peers[peer.conn.RemoteAddr()] = peer
	s.AddPeer(peer)
	return nil 
}

// client encodes handshake object and sends it 
// only raw bytes can be sent over net.Conn object
func (s *Server) handshake(p *Peer) (*Handshake, error) {
	hs := &Handshake{}
	if err := gob.NewDecoder(p.conn).Decode(hs); err != nil {
		return nil, err
	}
	if s.GameVariant != hs.GameVariant{
		return nil, fmt.Errorf("gamevariant does not match: %s", hs.GameVariant)
	}
	if s.Version != hs.Version{
		return nil, fmt.Errorf("invalid version %s", hs.Version)
	}
	p.listenAddr = hs.ListenAddr
	
	return hs, nil
}

func (s *Server) handleMessage(msg *Message) error {
	switch v := msg.Payload.(type) {
	case MessagePeerList:
		return s.handlePeerList(v)
	}
	return nil
}

func (s *Server) handlePeerList(l MessagePeerList) error {
	logrus.WithFields(logrus.Fields{
		"we": s.ListenAddr,
		"list": l.Peers,
	}).Info("received peerList message")
	fmt.Printf("peerlist => %+v\n", l)
	for i := 0; i < len(l.Peers); i++ {
		if err := s.Connect(l.Peers[i]); err != nil {
			logrus.Errorf("failed to dial peer: %s", err)
			continue
		}
	}
	return nil
}

func init() {
	gob.Register(MessagePeerList{})
}
