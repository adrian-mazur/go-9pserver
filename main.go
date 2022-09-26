package main

import (
	"log"
	"net"
)

func main() {
	listener, err := net.Listen("tcp", ":3000")
	if err != nil {
		log.Fatalln(err)
	}
	NewServer(listener, NewLocalFilesystem("."), true).AcceptLoop()
}
