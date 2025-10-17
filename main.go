package main

import "github.com/RedPaldin7/redpoker/p2p"

func main(){
	cfg := p2p.ServerConfig{
		ListenAddr: ":8080",
	}
	server := p2p.NewServer(cfg)
	server.Start()
}