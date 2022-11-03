package engine

import (
	"rpg/engine/message"
	"bytes"
	"encoding/binary"
	"errors"
)

const GateAndGameMessageHeadLen = 5 //1字节消息类型+4字节clientId

//GenMessageHeader 生成game与gate交互的消息头
func GenMessageHeader(msgType uint8, clientId ConnectIdType) []byte {
	buf := make([]byte, GateAndGameMessageHeadLen, GateAndGameMessageHeadLen)
	buf[0] = msgType
	binary.BigEndian.PutUint32(buf[1:], uint32(clientId))
	return buf
}

//ParseMessage 解析game与gate交互的消息
func ParseMessage(buf []byte) (uint8, ConnectIdType, []byte, error) {
	if len(buf) < GateAndGameMessageHeadLen {
		return 0, 0, nil, errors.New("data length error")
	}
	ty := buf[0]
	clientId := binary.BigEndian.Uint32(buf[1:GateAndGameMessageHeadLen])
	return ty, ConnectIdType(clientId), buf[GateAndGameMessageHeadLen:], nil
}

//genEntityRpcMessage 创建entity的rpc消息
func genEntityRpcMessage(msgType uint8, data map[string]interface{}, clientId ConnectIdType) ([]byte, error) {
	if buf, err := GetProtocol().Marshal(data); err != nil {
		return nil, err
	} else {
		header := GenMessageHeader(msgType, clientId)
		r := GetProtocol().ConcatHeadAndBody(header, buf)
		return r, nil
	}
}

//genC2SMessage 创建客户端发给服务器的消息
func genC2SMessage(msgType uint8, data map[string]interface{}) ([]byte, error) {
	if buf, err := GetProtocol().Marshal(data); err != nil {
		return nil, err
	} else {
		var b bytes.Buffer
		b.Write([]byte{msgType})
		b.Write(buf)
		return b.Bytes(), nil
	}
}

func genServerErrorMessage(errMsg string, clientId ConnectIdType) ([]byte, error) {
	header := GenMessageHeader(ServerMessageTypeServerError, clientId)
	return GetProtocol().MessageWithHead(header, &message.ServerError{ErrMsg: errMsg})
}
