package main

import (
	"rpg/engine/engine"
	"rpg/engine/message"
	"errors"
	"github.com/gogo/protobuf/proto"
	"github.com/panjf2000/gnet"
	lua "github.com/yuin/gopher-lua"
	"math/rand"
	"sync"
	"time"
)

var gateMgr *gateProxy

func getGateProxy() *gateProxy {
	if gateMgr == nil {
		gateMgr = new(gateProxy)
		gateMgr.init()
	}
	return gateMgr
}

type gateProxy struct {
	sync.Mutex
	gateMap     map[string]gnet.Conn //gate server name -> gate conn
	gateConnMap map[gnet.Conn]string //gate conn -> gate server name
}

func (m *gateProxy) init() {
	m.gateMap = make(map[string]gnet.Conn)
	m.gateConnMap = make(map[gnet.Conn]string)
}

func (m *gateProxy) AddGate(name string, c gnet.Conn) {
	m.Lock()
	defer m.Unlock()

	m.gateMap[name] = c
	m.gateConnMap[c] = name
	log.Infof("add gate[%s -> %s]", name, c.RemoteAddr())
}

func (m *gateProxy) RemoveGate(c gnet.Conn) {
	m.Lock()
	defer m.Unlock()

	if name, ok := m.gateConnMap[c]; ok {
		delete(m.gateMap, name)
		delete(m.gateConnMap, c)
		engine.GetEntityManager().RemoveGateEntitiesConn(name)
		log.Infof("remove gate[%s -> %s]", name, c.RemoteAddr())
	}
}

func (m *gateProxy) GetGateConn(name string) gnet.Conn {
	m.Lock()
	defer m.Unlock()
	return m.gateMap[name]
}

func (m *gateProxy) GateName(c gnet.Conn) string {
	m.Lock()
	defer m.Unlock()
	return m.gateConnMap[c]
}

func (m *gateProxy) GetRandomGate() gnet.Conn {
	m.Lock()
	defer m.Unlock()
	length := len(m.gateMap)
	if length == 0 {
		return nil
	}
	i := 0
	n := rand.Intn(length)
	for _, conn := range m.gateMap {
		if i == n {
			return conn
		}
		i++
	}
	return nil
}

func (m *gateProxy) SendToGate(header []byte, msg proto.Message, gate gnet.Conn) error {
	if gate == nil {
		gate = m.GetRandomGate()
	}
	if gate == nil {
		return errors.New("send to gate but no gate selected")
	}
	data, err := engine.GetProtocol().MessageWithHead(header, msg)
	if err != nil {
		return err
	}
	return gate.AsyncWrite(data)
}

func (m *gateProxy) CreateEntityAnywhere(entityName string, luaCb lua.LValue) {
	msg := &message.CreateEntityRequest{
		EntityName: entityName,
		ServerName: engine.ServiceName(),
		Ex:         &message.ExtraInfo{Uuid: getCallbackMgr().NextUniqueID()},
	}
	getCallbackMgr().setCallbackWithTimeout(msg.Ex.Uuid, &createEntityAnywhereCallback{luaFunc: luaCb}, 3*time.Second)
	if err := m.SendToGate(engine.GenMessageHeader(engine.ServerMessageTypeCreateGameEntity, 0), msg, nil); err != nil {
		log.Warnf("CreateEntityAnywhere error: %s, entityName: %s", err.Error(), entityName)
	}
}
