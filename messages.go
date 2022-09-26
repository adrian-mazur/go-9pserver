package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"reflect"
)

const (
	TversionType = 100
	RversionType = 101
	TauthType    = 102
	RauthType    = 103
	TattachType  = 104
	RattachType  = 105
	RerrorType   = 107
	TflushType   = 108
	RflushType   = 109
	TwalkType    = 110
	RwalkType    = 111
	TopenType    = 112
	RopenType    = 113
	TcreateType  = 114
	RcreateType  = 115
	TreadType    = 116
	RreadType    = 117
	TwriteType   = 118
	RwriteType   = 119
	TclunkType   = 120
	RclunkType   = 121
	TremoveType  = 122
	RremoveType  = 123
	TstatType    = 124
	RstatType    = 125
	TwstatType   = 126
	RwstatType   = 127

	DMDIR   = 0x80000000
	DMAPPED = 0x40000000
	DMEXCL  = 0x20000000
	DMTDP   = 0x04000000
)

type Tauth struct {
	Tag   uint16
	Afid  uint32
	Uname string
	Aname string
}

type Qid struct {
	Ftype   uint8
	Version uint32
	Path    uint64
}

type Rauth struct {
	Tag  uint16
	Aqid Qid
}

type Tattach struct {
	Tag   uint16
	Fid   uint32
	Afid  uint32
	Uname string
	Aname string
}

type Rattach struct {
	Tag uint16
	Qid Qid
}

type Tclunk struct {
	Tag uint16
	Fid uint32
}

type Rclunk struct {
	Tag uint16
}

type Tflush struct {
	Tag    uint16
	Oldtag uint16
}

type Rflush struct {
	Tag uint16
}

type Topen struct {
	Tag  uint16
	Fid  uint32
	Mode uint8
}

type Ropen struct {
	Tag    uint16
	Qid    Qid
	Iouint uint32
}

type Tcreate struct {
	Tag  uint16
	Fid  uint32
	Name string
	Perm uint32
	Mode uint8
}

type Rcreate struct {
	Tag    uint16
	Qid    Qid
	Iouint uint32
}

type Tread struct {
	Tag    uint16
	Fid    uint32
	Offset uint64
	Count  uint32
}

type Rread struct {
	Tag  uint16
	Data []byte
}

type Twrite struct { // TODO
	Tag uint16
}

type Rwrite struct {
	Tag   uint16
	Count uint32
}

type Tremove struct {
	Tag uint16
	Fid uint32
}

type Rremove struct {
	Tag uint16
}

type Tstat struct {
	Tag uint16
	Fid uint32
}

type Rstat struct {
	Tag  uint16
	Stat Stat
}

type TWstat struct { // TODO
	Tag uint16
}

type Rwstat struct {
	Tag uint16
}

type Tversion struct {
	Tag     uint16
	Msize   uint32
	Version string
}

type Rversion struct {
	Tag     uint16
	Msize   uint32
	Version string
}

type Twalk struct {
	Tag    uint16
	Fid    uint32
	Newfid uint32
	Nwname []string
}

type Rwalk struct {
	Tag   uint16
	Nwqid []Qid
}

type Rerror struct {
	Tag   uint16
	Ename string
}

type Stat struct {
	Stype  uint16
	Dev    uint32
	Qid    Qid
	Mode   uint32
	Atime  uint32
	Mtime  uint32
	Length uint64
	Name   string
	Uid    string
	Gid    string
	Muid   string
}

func (s Stat) Serialize(w io.Writer) error {
	return serializeStat(w, reflect.ValueOf(s), reflect.TypeOf(s), false)
}

func DeserializeMessage(r io.Reader) (interface{}, error) {
	size, err := readUint[uint32](r)
	if err != nil {
		return nil, err
	}
	b := make([]byte, size-4)
	_, err = io.ReadFull(r, b)
	if err != nil {
		return nil, err
	}
	buffer := bytes.NewReader(b[1:])
	switch b[0] {
	case TauthType:
		var msg Tauth
		err = deserializeMessage2(buffer, &msg)
		return &msg, err
	case TattachType:
		var msg Tattach
		err = deserializeMessage2(buffer, &msg)
		return &msg, err
	case TclunkType:
		var msg Tclunk
		err = deserializeMessage2(buffer, &msg)
		return &msg, err
	case TcreateType:
		var msg Tcreate
		err = deserializeMessage2(buffer, &msg)
		return &msg, err
	case TflushType:
		var msg Tflush
		err = deserializeMessage2(buffer, &msg)
		return &msg, err
	case TopenType:
		var msg Topen
		err = deserializeMessage2(buffer, &msg)
		return &msg, err
	case TreadType:
		var msg Tread
		err = deserializeMessage2(buffer, &msg)
		return &msg, err
	case TremoveType:
		var msg Tremove
		err = deserializeMessage2(buffer, &msg)
		return &msg, err
	case TstatType:
		var msg Tstat
		err = deserializeMessage2(buffer, &msg)
		return &msg, err
	case TversionType:
		var msg Tversion
		err = deserializeMessage2(buffer, &msg)
		return &msg, err
	case TwalkType:
		var msg Twalk
		err = deserializeMessage2(buffer, &msg)
		return &msg, err
	case TwriteType:
		var msg Twrite
		err = deserializeMessage2(buffer, &msg)
		return &msg, err
	case TwstatType:
		var msg TWstat
		err = deserializeMessage2(buffer, &msg)
		return &msg, err
	default:
		return nil, errors.New("unknown message type")
	}
}

func deserializeMessage2(r io.Reader, value any) error {
	return deserializeMessage3(r, reflect.ValueOf(value).Elem())
}

func deserializeMessage3(r io.Reader, v reflect.Value) error {
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if f.Kind() == reflect.Struct {
			err := deserializeMessage3(r, f)
			if err != nil {
				return err
			}
			continue
		}
		fi := f.Interface()
		switch fi.(type) {
		case uint8:
			r, err := readUint[uint8](r)
			if err != nil {
				return err
			}
			f.SetUint(uint64(r))
		case uint16:
			r, err := readUint[uint16](r)
			if err != nil {
				return err
			}
			f.SetUint(uint64(r))
		case uint32:
			r, err := readUint[uint32](r)
			if err != nil {
				return err
			}
			f.SetUint(uint64(r))
		case uint64:
			r, err := readUint[uint64](r)
			if err != nil {
				return err
			}
			f.SetUint(r)
		case string:
			r, err := readString(r)
			if err != nil {
				return err
			}
			f.SetString(r)
		case []string:
			count, err := readUint[uint16](r)
			if err != nil {
				return err
			}
			arr := make([]string, count)
			for i := uint16(0); i < count; i++ {
				arr[i], err = readString(r)
				if err != nil {
					return err
				}
			}
			f.Set(reflect.ValueOf(arr))
		default:
			return fmt.Errorf("unknown field type: %s", f.Type().String())
		}
	}
	return nil
}

func SerializeMessage(w io.Writer, value any) error {
	mtype := getRMessageType(value)
	if mtype == 0 {
		return errors.New("bad message type")
	}
	b := new(bytes.Buffer)
	err := serializeMessage2(b, reflect.ValueOf(value).Elem(), reflect.TypeOf(value).Elem())
	if err != nil {
		return err
	}
	err = writeUint(w, uint32(b.Len()+5))
	if err != nil {
		return err
	}
	err = writeUint(w, mtype)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, b)
	return err
}

func serializeMessage2(w io.Writer, v reflect.Value, t reflect.Type) error {
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		fi := f.Interface()
		switch c := fi.(type) {
		case uint8:
			err := writeUint(w, c)
			if err != nil {
				return err
			}
		case uint16:
			err := writeUint(w, c)
			if err != nil {
				return err
			}
		case uint32:
			err := writeUint(w, c)
			if err != nil {
				return err
			}
		case uint64:
			err := writeUint(w, c)
			if err != nil {
				return err
			}
		case string:
			err := writeString(w, c)
			if err != nil {
				return err
			}
		case []Qid:
			err := writeUint(w, uint16(len(c)))
			if err != nil {
				return err
			}
			for _, v := range c {
				err = serializeMessage2(w, reflect.ValueOf(v), reflect.TypeOf(v))
				if err != nil {
					return err
				}
			}
		case []byte:
			err := writeUint(w, uint32(len(c)))
			if err != nil {
				return err
			}
			_, err = w.Write(c)
			if err != nil {
				return err
			}
		case Stat:
			err := serializeStat(w, f, f.Type(), true)
			if err != nil {
				return err
			}
		default:
			if f.Kind() == reflect.Struct {
				err := serializeMessage2(w, f, f.Type())
				if err != nil {
					return err
				}
			} else {
				return fmt.Errorf("unknown field type: %s", f.Type().String())
			}
		}
	}
	return nil
}

func serializeStat(w io.Writer, v reflect.Value, t reflect.Type, writeLength bool) error {
	b := new(bytes.Buffer)
	err := serializeMessage2(b, v, t)
	if err != nil {
		return err
	}
	if writeLength {
		err = writeUint(w, uint16(b.Len()+2))
		if err != nil {
			return err
		}
	}
	err = writeUint(w, uint16(b.Len()))
	if err != nil {
		return err
	}
	_, err = io.Copy(w, b)
	return err
}

func getRMessageType(v interface{}) uint8 {
	switch v.(type) {
	case *Rversion:
		return RversionType
	case *Rauth:
		return RauthType
	case *Rattach:
		return RattachType
	case *Rerror:
		return RerrorType
	case *Rflush:
		return RflushType
	case *Rwalk:
		return RwalkType
	case *Ropen:
		return RopenType
	case *Rcreate:
		return RcreateType
	case *Rread:
		return RreadType
	case *Rwrite:
		return RwriteType
	case *Rclunk:
		return RclunkType
	case *Rremove:
		return RremoveType
	case *Rstat:
		return RstatType
	case *Rwstat:
		return RwstatType
	}
	return 0
}

func readBuff(r io.Reader, size int) ([]byte, error) {
	buff := make([]byte, size)
	_, err := io.ReadFull(r, buff)
	if err != nil {
		return nil, err
	}
	return buff, nil
}

func readUint[K uint8 | uint16 | uint32 | uint64](r io.Reader) (K, error) {
	var result K
	err := binary.Read(r, binary.LittleEndian, &result)
	return result, err
}

func writeUint[K uint8 | uint16 | uint32 | uint64](w io.Writer, v K) error {
	return binary.Write(w, binary.LittleEndian, v)
}

func readString(r io.Reader) (string, error) {
	strSize, err := readUint[uint16](r)
	if err != nil {
		return "", err
	}
	str, err := readBuff(r, int(strSize))
	if err != nil {
		return "", err
	}
	return string(str), nil
}

func writeString(w io.Writer, s string) error {
	bytes := []byte(s)
	err := writeUint(w, uint16(len(bytes)))
	if err != nil {
		return err
	}
	_, err = w.Write(bytes)
	return err
}
