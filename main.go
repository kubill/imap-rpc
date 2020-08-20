package main

import (
	"log"
	"main/tools"
	"net"
	"net/rpc"

	"github.com/spiral/goridge/v2"
)

func main() {
	ln, err := net.Listen("tcp", ":6001")
	if err != nil {
		panic(err)
	}

	err = rpc.Register(new(tools.Imap))
	if err != nil {
		panic(err)
	}

	log.Printf("started")

	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}

		log.Printf("new connection %+v", conn.RemoteAddr().String())
		go rpc.ServeCodec(goridge.NewCodec(conn))
	}
}
