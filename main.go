package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
)

var debugFlag = flag.Bool("d", false, "Enable verbose debugging")
var listenAddr = flag.String("l", ":564", "Listen `address`")

func usage() {
	fmt.Printf("Usage: %s fsroot\nOptions:\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) != 1 {
		usage()
		os.Exit(1)
	}
	listener, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		log.Fatalln(err)
	}
	p, err := filepath.Abs(args[0])
	if err != nil {
		log.Fatalln(err)
	}
	NewServer(listener, NewLocalFilesystem(p), *debugFlag).AcceptLoop()
}
