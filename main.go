package main

import (
	"log"
	"time"

	"github.com/RedPaldin7/redpoker/p2p"
)

func main(){
	cfg := p2p.ServerConfig{
		Version: "REDPOKER V0.1-beta",
		ListenAddr: ":3000",
		GameVariant: p2p.TexasHoldem,
	}
	server := p2p.NewServer(cfg)
	go server.Start()

	time.Sleep(1 * time.Second)

	remoteConfig := p2p.ServerConfig{
		ListenAddr: ":4000",
		Version: "REDPOKER V0.1-beta",
		GameVariant: p2p.TexasHoldem,
	}
	remoteServer := p2p.NewServer(remoteConfig)
	go remoteServer.Start()
	if err := remoteServer.Connect(":3000"); err != nil {
		log.Fatal(err)
	}
	select{}
}