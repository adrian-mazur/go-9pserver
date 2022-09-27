package main

import (
	"log"
	"net"
)

type Server struct {
	listener   net.Listener
	filesystem Filesystem
	debug      bool
}

func NewServer(l net.Listener, f Filesystem, debug bool) *Server {
	return &Server{l, f, debug}
}

func (s *Server) AcceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go newSession(s, conn).loop()
	}
}
