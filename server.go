package main

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net"
	p "path"
	"reflect"
)

const (
	MaximumMsgSize = 8 * 1024

	ENoAuthRequiredStr        = "no authentication required"
	EIOErrorStr               = "i/o error"
	ENoSuchFileOrDirectoryStr = "No such file or directory"
	EBadMessageStr            = "Bad message"
)

var ErrInvalidFid = errors.New("invalid fid")
var ErrUnexpectedMessage = errors.New("expected different message type")

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

type session struct {
	server          *Server
	conn            net.Conn
	receivedVersion bool
	maxsize         uint32
	fids            map[uint32]struct {
		path string
		file File
	}
}

func newSession(server *Server, conn net.Conn) *session {
	return &session{server, conn, false, 0, make(map[uint32]struct {
		path string
		file File
	})}
}

func (s *session) loop() {
	log.Printf("accepted new connection: %s\n", s.conn.RemoteAddr())
	var err error
	for {
		var msg interface{}
		msg, err = DeserializeMessage(s.conn)
		if err != nil {
			goto end
		}
		if s.server.debug {
			log.Printf("<- %T %v\n", msg, msg)
		}
		err = s.handleNextMsg(msg)
		if err != nil {
			goto end
		}
	}
end:
	s.clean()
	if !errors.Is(err, io.EOF) {
		log.Println(err)
	}
	log.Printf("connection closed: %s\n", s.conn.RemoteAddr())
	_ = s.conn.Close()
}

func (s *session) clean() {
	for _, f := range s.fids {
		if f.file != nil {
			f.file.Close()
		}
	}
}

func (s *session) send(v interface{}) error {
	if s.server.debug {
		log.Printf("-> %T %v", v, v)
	}
	return SerializeMessage(s.conn, v)
}

func (s *session) sendError(tag uint16, name string) error {
	return s.send(&Rerror{Tag: tag, Ename: name})
}

func (s *session) getFid(fid uint32) (string, File, error) {
	f, ok := s.fids[fid]
	if !ok {
		return "", nil, ErrInvalidFid
	}
	return f.path, f.file, nil
}

func (s *session) setFid(fid uint32, path string, file File) {
	s.fids[fid] = struct {
		path string
		file File
	}{path, file}
}

func (s *session) deleteFid(fid uint32) {
	delete(s.fids, fid)
}

func (s *session) handleNextMsg(msg interface{}) error {
	if !s.receivedVersion {
		m, ok := msg.(*Tversion)
		if !ok {
			return ErrUnexpectedMessage
		}
		return s.handleVersion(m)
	}
	var err error
	switch m := msg.(type) {
	case *Tauth:
		err = s.handleAuth(m)
	case *Tattach:
		err = s.handleAttach(m)
	case *Tclunk:
		err = s.handleClunk(m)
	case *Tcreate:
		err = s.handleCreate(m)
	case *Tflush:
		err = s.handleFlush(m)
	case *Topen:
		err = s.handleOpen(m)
	case *Tread:
		err = s.handleRead(m)
	case *Tremove:
		err = s.handleRemove(m)
	case *Tstat:
		err = s.handleStat(m)
	case *Tversion:
		err = ErrUnexpectedMessage
	case *Twalk:
		err = s.handleWalk(m)
	case *Twrite:
		err = s.handleWrite(m)
	case *Twstat:
		err = s.handleWstat(m)
	}
	if err == nil {
		return nil
	}

	tag := uint16(reflect.ValueOf(msg).Elem().FieldByName("Tag").Uint())
	switch err {
	case ErrIOError:
		return s.sendError(tag, EIOErrorStr)
	case ErrDoesNotExist:
		return s.sendError(tag, ENoSuchFileOrDirectoryStr)
	case ErrInvalidFid:
		return s.sendError(tag, EBadMessageStr)
	default:
		return err
	}
}

func (s *session) handleAuth(m *Tauth) error {
	return s.sendError(m.Tag, ENoAuthRequiredStr)
}

func (s *session) handleAttach(m *Tattach) error {
	stat, err := s.server.filesystem.Stat("/")
	if err != nil {
		return err
	}
	s.setFid(m.Fid, "/", nil)
	return s.send(&Rattach{Tag: m.Tag, Qid: stat.Qid})
}

func (s *session) handleClunk(m *Tclunk) error {
	_, f, err := s.getFid(m.Fid)
	if err != nil {
		return err
	}
	if f != nil {
		f.Close()
	}
	s.deleteFid(m.Fid)
	return s.send(&Rclunk{Tag: m.Tag})
}

func (s *session) handleCreate(m *Tcreate) error { // TODO
	return nil
}

func (s *session) handleFlush(m *Tflush) error { // TODO
	return nil
}

func (s *session) handleOpen(m *Topen) error {
	path, _, err := s.getFid(m.Fid)
	if err != nil {
		return err
	}
	file, err := s.server.filesystem.Open(path)
	if err != nil {
		return err
	}
	s.setFid(m.Fid, path, file)
	return s.send(&Ropen{Tag: m.Tag, Qid: file.Qid(), Iouint: 0})
}

func (s *session) handleRead(m *Tread) error {
	path, file, err := s.getFid(m.Fid)
	if err != nil {
		return err
	}
	if file == nil {
		return ErrInvalidFid
	}
	if file.IsDir() {
		return s.handleReadDir(m, path)
	} else {
		return s.handleReadFile(m, file)
	}
}

func (s *session) handleReadFile(m *Tread, file File) error {
	b, err := file.Read(m.Offset, m.Count)
	if err != nil {
		return err
	}
	return s.send(&Rread{Tag: m.Tag, Data: b})
}

func (s *session) handleReadDir(m *Tread, path string) error {
	buffer := new(bytes.Buffer)
	stats, err := s.server.filesystem.ReadDir(path)
	if err != nil {
		return err
	}
	for _, s := range stats {
		s.Serialize(buffer)
	}
	bytes := buffer.Bytes()
	bytesLen := len(bytes)
	var data []byte
	if m.Offset < uint64(bytesLen) {
		data = bytes[m.Offset:min(m.Offset+uint64(m.Count), uint64(bytesLen))]
	}
	return s.send(&Rread{Tag: m.Tag, Data: data})
}

func (s *session) handleRemove(m *Tremove) error { // TODO
	return nil
}

func (s *session) handleStat(m *Tstat) error {
	path, _, err := s.getFid(m.Fid)
	if err != nil {
		return err
	}
	stat, err := s.server.filesystem.Stat(path)
	if err != nil {
		return err
	}
	return s.send(&Rstat{Tag: m.Tag, Stat: stat})
}

func (s *session) handleVersion(m *Tversion) error {
	s.maxsize = min(m.Msize, MaximumMsgSize)
	if m.Version != ProtocolVersion {
		return s.send(&Rversion{Tag: m.Tag, Msize: s.maxsize, Version: "unknown"})
	}
	s.receivedVersion = true
	return s.send(&Rversion{Tag: m.Tag, Msize: s.maxsize, Version: ProtocolVersion})
}

func (s *session) handleWalk(m *Twalk) error {
	path, file, err := s.getFid(m.Fid)
	if err != nil {
		return err
	}
	if len(m.Nwname) == 0 {
		s.setFid(m.Newfid, path, file)
		return s.send(&Rwalk{Tag: m.Tag, Nwqid: []Qid{}})
	}
	result := make([]Qid, len(m.Nwname))
	for i, name := range m.Nwname {
		path = p.Join(path, name)
		stat, err := s.server.filesystem.Stat(path)
		if err != nil {
			return err
		}
		result[i] = stat.Qid
	}
	s.setFid(m.Newfid, path, nil)
	return s.send(&Rwalk{Tag: m.Tag, Nwqid: result})
}

func (s *session) handleWrite(m *Twrite) error { // TODO
	return nil
}

func (s *session) handleWstat(m *Twstat) error { // TODO
	return nil
}
