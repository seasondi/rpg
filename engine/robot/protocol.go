package main

import (
	"rpg/engine/engine"
	"bytes"
)

func messageWithHead(head, body []byte) []byte {
	var b bytes.Buffer
	if head != nil {
		b.Write(head)
	}
	if body != nil {
		b.Write(body)
	}

	return b.Bytes()
}

func genLoginMessage(args []interface{}) []byte {
	data := map[string]interface{}{
		engine.ClientMsgDataFieldArgs: args,
	}
	if buf, err := engine.GetProtocol().Marshal(data); err != nil {
		log.Errorf("genLoginMessage Marshal error: %s", err.Error())
		return nil
	} else {
		return messageWithHead([]byte{engine.ClientMsgTypeLogin}, buf)
	}
}

func genHeartbeatMessage(entityId engine.EntityIdType) []byte {
	data := map[string]interface{}{
		engine.ClientMsgDataFieldEntityID: entityId,
	}
	if buf, err := engine.GetProtocol().Marshal(data); err != nil {
		return nil
	} else {
		return messageWithHead([]byte{engine.ClientMsgTypeHeartBeat}, buf)
	}
}
