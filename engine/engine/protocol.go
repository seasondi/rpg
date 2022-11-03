package engine

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/panjf2000/gnet"
	"github.com/vmihailenco/msgpack/v5"
)

const (
	messageHeadLen = 4
	messageMaxLen  = 16 * 1024 * 1024 //16M
)

type protocol struct {
}

func initProtocol() error {
	protoMgr = new(protocol)
	if err := protoMgr.init(); err != nil {
		return err
	}
	log.Infof("protocol inited.")
	return nil
}

func GetProtocol() *protocol {
	return protoMgr
}

func (m *protocol) init() error {
	return nil
}

func (m *protocol) Marshal(buf interface{}) ([]byte, error) {
	return msgpack.Marshal(buf)
}

func (m *protocol) UnMarshal(buf []byte) (map[string]interface{}, error) {
	r := make(map[string]interface{})
	if len(buf) == 0 {
		return r, nil
	}
	if err := msgpack.Unmarshal(buf, &r); err != nil {
		return nil, err
	}
	return r, nil
}

func (m *protocol) UnMarshalTo(buf []byte, r interface{}) error {
	if err := msgpack.Unmarshal(buf, r); err != nil {
		return err
	}
	return nil
}

// MessageWithHead 将protobuf消息与包头拼装,不包括长度字段(由encode相关逻辑添加)
func (m *protocol) MessageWithHead(head []byte, msg proto.Message) ([]byte, error) {
	var b bytes.Buffer
	if head != nil {
		b.Write(head)
	}

	if msg != nil {
		data, err := proto.Marshal(msg)
		if err != nil {
			return nil, err
		}
		b.Write(data)
	}
	return b.Bytes(), nil
}

// ConcatHeadAndBody 组合包头与包体
func (m *protocol) ConcatHeadAndBody(head, body []byte) []byte {
	var b bytes.Buffer
	if head != nil {
		b.Write(head)
	}
	if body != nil {
		b.Write(body)
	}
	return b.Bytes()
}

// Encode 消息编码
func (m *protocol) Encode(data []byte) ([]byte, error) {
	dataLen := len(data)
	if dataLen > messageMaxLen {
		return nil, fmt.Errorf("encode message length %d > %d", dataLen, messageMaxLen)
	}
	totalLen := messageHeadLen + dataLen
	buf := make([]byte, messageHeadLen)
	binary.BigEndian.PutUint32(buf, uint32(totalLen))
	b := bytes.NewBuffer(buf)
	b.Write(data)
	return b.Bytes(), nil
}

// Decode 消息解码
func (m *protocol) Decode(data []byte) (int, []byte, error) {
	bufLen := len(data)
	if bufLen < messageHeadLen {
		return 0, nil, nil
	}
	length := int(binary.BigEndian.Uint32(data))
	if bufLen < length {
		return 0, nil, nil
	}
	if length > messageMaxLen {
		return 0, nil, fmt.Errorf("decode message length %d > %d", length, messageMaxLen)
	}
	return length, data[messageHeadLen:length], nil
}

// GNetCodec GNet消息编解码
type GNetCodec struct {
}

func (m *GNetCodec) Encode(c gnet.Conn, buf []byte) ([]byte, error) {
	data, err := GetProtocol().Encode(buf)
	log.Tracef("encode %d bytes to [%s]", len(data), c.RemoteAddr())
	return data, err
}

func (m *GNetCodec) Decode(c gnet.Conn) ([]byte, error) {
	buf := c.Read()
	length, data, err := GetProtocol().Decode(buf)
	if err != nil {
		return nil, err
	}
	if length > 0 {
		c.ShiftN(length)
		log.Tracef("decode %d bytes from [%s]", length, c.RemoteAddr())
	}
	return data, err
}
