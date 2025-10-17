package main

import (
	"fmt"
	"time"

	"github.com/RedPaldin7/redpoker/deck"
	"github.com/RedPaldin7/redpoker/p2p"
)

func main(){
	cfg := p2p.ServerConfig{
		Version: "REDPOKER V0.1-beta",
		ListenAddr: ":3000",
	}
	server := p2p.NewServer(cfg)
	go server.Start()

	time.Sleep(1 * time.Second)

	remoteConfig := p2p.ServerConfig{
		ListenAddr: ":4000",
		Version: "REDPOKER V0.1-beta",
	}
	remoteServer := p2p.NewServer(remoteConfig)
	go remoteServer.Start()
	if err := remoteServer.Connect(":3000"); err != nil {
		fmt.Println(err)
	}
	fmt.Println(deck.New())
}