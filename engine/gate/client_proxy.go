package main

import (
	"rpg/engine/engine"
	"github.com/panjf2000/gnet"
	"sync"
)

const (
	ctxKeyConnId = "connId"
)

type connCtxType map[string]engine.ConnectIdType

var clientConnectId engine.ConnectIdType = 0 //客户端连接ID
var clientMgr *clientProxy

func getClientProxy() *clientProxy {
	if clientMgr == nil {
		clientMgr = new(clientProxy)
		clientMgr.init()
	}
	return clientMgr
}

type clientProxy struct {
	sync.Mutex
	connMap map[engine.ConnectIdType]gnet.Conn //clientConnectId -> conn
}

func (m *clientProxy) init() {
	m.connMap = make(map[engine.ConnectIdType]gnet.Conn)
}

func (m *clientProxy) addConn(c gnet.Conn) {
	m.Lock()
	defer m.Unlock()

	clientConnectId += 1
	m.connMap[clientConnectId] = c
	ctxMap := connCtxType{
		ctxKeyConnId: clientConnectId,
	}
	c.SetContext(ctxMap)
	log.Infof("add client conn[%s] with ctx: %+v", c.RemoteAddr(), c.Context())
}

func (m *clientProxy) removeConn(c gnet.Conn) {
	m.Lock()
	defer m.Unlock()
	if ctx, ok := c.Context().(connCtxType); ok {
		if id, ok := ctx[ctxKeyConnId]; ok {
			delete(m.connMap, id)
			log.Infof("remove client conn[%s] with ctx: %+v", c.RemoteAddr(), c.Context())
		}
	}
}

func (m *clientProxy) client(clientId engine.ConnectIdType) gnet.Conn {
	m.Lock()
	defer m.Unlock()
	return m.connMap[clientId]
}
