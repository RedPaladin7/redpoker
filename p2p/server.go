package p2p

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"net"

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
	peers    map[net.Addr]*Peer
	addPeer  chan *Peer
	delPeer  chan *Peer
	msgCh    chan *Message
}

func NewServer(cfg ServerConfig) *Server {
	s :=  &Server{
		ServerConfig: cfg,
		peers:        make(map[net.Addr]*Peer),
		addPeer:      make(chan *Peer),
		delPeer:      make(chan *Peer),
		msgCh:        make(chan *Message),
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

func (s *Server) SendHandshake(p *Peer) error {
	hs := &Handshake{
		GameVariant: s.GameVariant,
		Version: s.Version,
	}
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(hs); err != nil{
		return err
	}
	return p.Send(buf.Bytes())
}

func (s *Server) Connect(addr string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return err
	}
	peer := &Peer{
		conn: conn,
	}
	s.addPeer <- peer
	return peer.Send([]byte(s.Version))
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
			s.SendHandshake(peer)
			if err := s.handshake(peer); err != nil {
				logrus.Errorf("handshake with incoming player failed: %s", err)
				continue
			}
			go peer.ReadLoop(s.msgCh)
			logrus.WithFields(logrus.Fields{
				"addr": peer.conn.RemoteAddr(),
			}).Info("new player connected")
			s.peers[peer.conn.RemoteAddr()] = peer

		case msg := <-s.msgCh:
			if err := s.handleMessage(msg); err != nil {
				panic(err)
			}
		}
	}
}

type Handshake struct {
	Version string 
	GameVariant GameVariant
}

// client encodes handshake object and sends it 
// only raw bytes can be sent over net.Conn object
func (s *Server) handshake(p *Peer) error {
	hs := &Handshake{}
	if err := gob.NewDecoder(p.conn).Decode(hs); err != nil {
		return err
	}
	logrus.WithFields(logrus.Fields{
		"peer": p.conn.RemoteAddr(),
		"version": hs.Version,
		"variant": hs.GameVariant,
	}).Info("received handshake")
	return nil
}

func (s *Server) handleMessage(msg *Message) error {
	fmt.Printf("%+v\n", msg)
	return nil
}
