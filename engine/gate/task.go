package main

import (
	"github.com/panjf2000/gnet"
	"rpg/engine/engine"
	"strings"
)

type AddClientTask struct {
	conn gnet.Conn
}

func (m *AddClientTask) HandleTask() error {
	getClientProxy().addConn(m.conn)
	return nil
}

type RemoveClientTask struct {
	clientId engine.ConnectIdType
}

func (m *RemoveClientTask) HandleTask() error {
	getClientProxy().removeConn(m.clientId)
	getGameProxy().onClientClosed(m.clientId)
	return nil
}

type AddGameServerTask struct {
	kv engine.EtcdKV
}

func (m *AddGameServerTask) HandleTask() error {
	key := m.kv.Key()
	if strings.HasPrefix(key, engine.ServiceGamePrefix) {
		getGameProxy().HandleUpdateGame(m.kv.Key(), m.kv.Value())
	} else if strings.HasPrefix(key, engine.StubPrefix) {
		getGameProxy().HandleUpdateStub(m.kv.Key(), m.kv.Value())
	}
	return nil
}

type RemoveGameServerTask struct {
	kv engine.EtcdKV
}

func (m *RemoveGameServerTask) HandleTask() error {
	key := m.kv.Key()
	if strings.HasPrefix(key, engine.ServiceGamePrefix) {
		getGameProxy().HandleDeleteGame(m.kv.Key())
	} else if strings.HasPrefix(key, engine.StubPrefix) {
		getGameProxy().HandleDeleteStub(m.kv.Key())
	}
	return nil
}

type ClientMessageTask struct {
	conn gnet.Conn
	buf  []byte
}

func (m *ClientMessageTask) HandleTask() error {
	clientId := getClientId(m.conn)
	if c := getClientProxy().client(clientId); c != nil {
		if data, _ := getGameProxy().ClientSendToGame(c, m.buf[0], m.buf[1:]); data != nil {
			_ = c.AsyncWrite(data)
		}
	} else {
		log.Tracef("process data, client conn already closed, connectId: %d", clientId)
	}
	return nil
}

type ServerStopTask struct {
	quitStatus int32
}

func (m *ServerStopTask) HandleTask() error {
	log.Info("server start quit")
	quit.Store(m.quitStatus)
	return nil
}
