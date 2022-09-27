package main

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func TestDeserializingMessages(t *testing.T) {
	input, err := hex.DecodeString("19000000665500010000000500756E616D650500616E616D65")
	if err != nil {
		t.Fatal(err)
	}
	reader := bytes.NewReader(input)
	msg, err := DeserializeMessage(reader)
	if err != nil {
		t.Fatal(err)
	}
	authMsg, ok := msg.(*Tauth)
	if !ok {
		t.Fatalf("wrong message type, got %T, want *Tauth", msg)
	}
	authMsgExcepted := Tauth{Tag: 0x55, Afid: 0x01, Uname: "uname", Aname: "aname"}
	if authMsg.Tag != authMsgExcepted.Tag {
		t.Errorf("got %d, want %d", authMsg.Tag, authMsgExcepted.Tag)
	}
	if authMsg.Afid != authMsgExcepted.Afid {
		t.Errorf("got %d, want %d", authMsg.Afid, authMsgExcepted.Afid)
	}
	if authMsg.Uname != authMsgExcepted.Uname {
		t.Errorf("got %s, want %s", authMsg.Uname, authMsgExcepted.Uname)
	}
	if authMsg.Aname != authMsgExcepted.Aname {
		t.Errorf("got %s, want %s", authMsg.Aname, authMsgExcepted.Aname)
	}

	input, err = hex.DecodeString("3A0000007E00000100000031002F00FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFBA0E3263BA0E3263FFFFFFFFFFFFFFFF0000000000000000")
	if err != nil {
		t.Fatal(err)
	}
	reader = bytes.NewReader(input)
	msg, err = DeserializeMessage(reader)
	if err != nil {
		t.Fatal(err)
	}
	twstatMsg, ok := msg.(*Twstat)
	if !ok {
		t.Fatalf("wrong message type, got %T, want *Twstat", msg)
	}
	twstatExcepted := Twstat{Tag: 0, Fid: 1, Stat: Stat{Length: 0xFFFFFFFFFFFFFFFF}}
	if twstatMsg.Tag != twstatExcepted.Tag {
		t.Errorf("got %d, want %d", twstatMsg.Tag, twstatExcepted.Tag)
	}
	if twstatMsg.Fid != twstatExcepted.Fid {
		t.Errorf("got %d, want %d", twstatMsg.Fid, twstatExcepted.Fid)
	}
	if twstatMsg.Stat.Length != twstatExcepted.Stat.Length {
		t.Errorf("got %d, want %d", twstatMsg.Stat.Length, twstatExcepted.Stat.Length)
	}
}

func TestSerializingMessages(t *testing.T) {
	versionMsg := Rversion{Tag: 0x75, Msize: 0x15, Version: "test"}
	b := new(bytes.Buffer)
	err := SerializeMessage(b, &versionMsg)
	if err != nil {
		t.Fatal(err)
	}
	resultHex := hex.EncodeToString(b.Bytes())
	exceptedResult := "1100000065750015000000040074657374"
	if resultHex != exceptedResult {
		t.Errorf("got '%s', want '%s'", resultHex, exceptedResult)
	}
}
