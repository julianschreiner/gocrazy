package main

import (
	"context"
	"log"

	"github.com/rlat3d/godis/internal/command"
	"github.com/rlat3d/godis/internal/eventloop"
	"github.com/rlat3d/godis/internal/network"
	"github.com/rlat3d/godis/internal/protocol"
	"github.com/rlat3d/godis/internal/storage"
	"golang.org/x/sys/unix"
)

func main() {
	listenerFD, err := network.ListenTCP(6380)
	if err != nil {
		log.Fatal(err)
	}
	defer unix.Close(listenerFD)

	pollerLinux, err := eventloop.NewPollerLinux()
	if err != nil {
		log.Fatal(err)
	}
	defer pollerLinux.Close()

	// TODO
	// main.go
	// |
	// ├── Create Store
	//|
	//├── Open AOF
	//|
	//├── Replay saved commands into Store
	// |
	//├── Create Dispatcher
	//|
	//└── Start Event Loop
	dispatcher := command.NewDispatcher(
		protocol.NewResp2Encoder(),
		storage.NewStore(),
	)
	parser := protocol.NewResp2Parser()

	eventLoop, err := eventloop.NewLoop(
		pollerLinux,
		listenerFD,
		dispatcher,
		parser,
	)
	if err != nil {
		log.Print(err)
		return
	}

	log.Println("Godis listening on 127.0.0.1:6380")

	if err := eventLoop.Run(context.Background()); err != nil {
		log.Printf("Event loop stopped: %v", err)
	}
}
