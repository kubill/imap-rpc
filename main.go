package main

import (
	"fmt"
	"log"
)

func main() {
	var a uint32 = 1
	str := fmt.Sprint(a)
	log.Println(str)
	// ln, err := net.Listen("tcp", ":6002")
	// if err != nil {
	// 	panic(err)
	// }

	// err = rpc.Register(new(tools.Imap))
	// if err != nil {
	// 	panic(err)
	// }

	// log.Printf("started")

	// for {
	// 	conn, err := ln.Accept()
	// 	if err != nil {
	// 		continue
	// 	}
	// 	log.Printf("new connection %+v", conn.RemoteAddr().String())
	// 	go rpc.ServeCodec(goridge.NewCodec(conn))
	// }
}
