package main

import (
	"context"
	"log"

	"github.com/julianschreiner/godis/internal/command"
	"github.com/julianschreiner/godis/internal/eventloop"
	"github.com/julianschreiner/godis/internal/network"
	"github.com/julianschreiner/godis/internal/protocol"
	"github.com/julianschreiner/godis/internal/storage"
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
