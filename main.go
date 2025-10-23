package main

import (
	"time"

	"github.com/RedPaldin7/redpoker/p2p"
)

func makeServerAndStart(addr string) *p2p.Server{
	cfg := p2p.ServerConfig{
		Version: "REDPOKER V0.1-beta",
		ListenAddr: addr,
		GameVariant: p2p.TexasHoldem,
	}
	server := p2p.NewServer(cfg)
	go server.Start()
	time.Sleep(1 * time.Second)
	return server
}

func main(){
	playerA := makeServerAndStart(":3000")
	playerB := makeServerAndStart(":4000")
	playerC := makeServerAndStart(":5000")
	playerD := makeServerAndStart(":6000")
	
	time.Sleep(time.Second * 1)
	playerB.Connect(playerA.ListenAddr)
	time.Sleep(time.Second * 1)
	playerC.Connect(playerB.ListenAddr)
	time.Sleep(time.Second * 1)
	playerD.Connect(playerC.ListenAddr)
	select{}
}