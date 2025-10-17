package p2p

import (
	"fmt"
	"net"
	"sync"
)

// net.Conn is general bidirection network connection (socket)
// sync.RWMutex is read write lock (can be held by one writer or multiple reasders)

type ServerConfig struct {
	Version    string
	ListenAddr string
}

type Server struct {
	ServerConfig
	handler  Handler
	transport *TCPTransport
	mu       sync.RWMutex
	peers    map[net.Addr]*Peer
	addPeer  chan *Peer
	delPeer  chan *Peer
	msgCh    chan *Message
}

func NewServer(cfg ServerConfig) *Server {
	s :=  &Server{
		handler:      &DefaultHandler{},
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
	fmt.Printf("game server running on port %s\n", s.ListenAddr)
	s.transport.ListenAndAccept()
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
			delete(s.peers, peer.conn.RemoteAddr())
			fmt.Printf("player disconnected %s\n", peer.conn.RemoteAddr())
		case peer := <-s.addPeer:
			fmt.Printf("new player connected %s\n", peer.conn.RemoteAddr())
			s.peers[peer.conn.RemoteAddr()] = peer
		case msg := <-s.msgCh:
			if err := s.handler.HandleMessage(msg); err != nil {
				panic(err)
			}
		}
	}
}
