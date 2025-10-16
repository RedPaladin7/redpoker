package server

import (
	"net"
	"sync"
)

// net.Conn is general bidirection network connection (socket)
// sync.RWMutex is read write lock (can be held by one writer or multiple reasders)


type Peer struct {
	conn net.Conn
}

type ServerConfig struct {
	ListenAddr string 
}

type Server struct {
	ServerConfig
	listener net.Listener
	mu sync.RWMutex
	peers map[net.Addr]*Peer
	addPeer chan *Peer 
}

func NewServer(cfg ServerConfig) *Server {
	return &Server{
		ServerConfig: cfg,
		peers: make(map[net.Addr]*Peer),
		addPeer: make(chan *Peer)
	}
}

func (s *Server) Start(){
	go s.loop()
}

func (s *Server) listen() error {
	ln, err := net.Listen("tcp", s.ListenAddr)
	if err != nil {
		return err
	}
	s.listener = ln
	return nil
}

func (s *Server) loop() {
	for {
		select {
		case peer := <- s.addPeer:
			fmt.Printf("new player connected %d", peer.conn.RemoteAddr())
			s.peers[peer.conn.RemoteAddr()] = peer 
		}
	}
}