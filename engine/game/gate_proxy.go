package main

import (
	"errors"
	"github.com/gogo/protobuf/proto"
	"github.com/panjf2000/gnet"
	lua "github.com/seasondi/gopher-lua"
	"math/rand"
	"rpg/engine/engine"
	"rpg/engine/message"
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

type gateInfo struct {
	conn        gnet.Conn
	isInnerGate bool
}

type gateProxy struct {
	gateMap     map[string]*gateInfo //gate server name -> gate info
	gateConnMap map[gnet.Conn]string //gate conn -> gate server name

	chosenInnerGate string //选取的内部通信gate
}

func (m *gateProxy) init() {
	m.gateMap = make(map[string]*gateInfo)
	m.gateConnMap = make(map[gnet.Conn]string)
}

func (m *gateProxy) AddGate(c gnet.Conn, name string, isInner bool) {
	m.gateMap[name] = &gateInfo{conn: c, isInnerGate: isInner}
	m.gateConnMap[c] = name
	log.Infof("add gate[%s -> %s], inner: %v", name, c.RemoteAddr(), isInner)
}

func (m *gateProxy) RemoveGate(c gnet.Conn) {
	if name, ok := m.gateConnMap[c]; ok {
		engine.GetEntityManager().RemoveGateEntitiesConn(name)
		delete(m.gateConnMap, c)
		delete(m.gateMap, name)
		if name == m.chosenInnerGate {
			m.chosenInnerGate = ""
		}

		log.Infof("remove gate: %s", name)
	}
}

func (m *gateProxy) GetGateConn(name string) gnet.Conn {
	return m.gateMap[name].conn
}

func (m *gateProxy) GateName(c gnet.Conn) string {
	return m.gateConnMap[c]
}

func (m *gateProxy) choseGate() string {
	gateNames := make([]string, 0)
	for name, info := range m.gateMap {
		if info.isInnerGate {
			gateNames = append(gateNames, name)
		}
	}
	//没有inner gate
	if len(gateNames) == 0 {
		for name := range m.gateMap {
			gateNames = append(gateNames, name)
		}
	}

	if len(gateNames) > 0 {
		return gateNames[rand.Intn(len(gateNames))]
	}

	return ""
}

func (m *gateProxy) GetInnerGate() gnet.Conn {
	if m.chosenInnerGate != "" {
		if info, find := m.gateMap[m.chosenInnerGate]; find && info.conn != nil {
			return info.conn
		} else {
			m.chosenInnerGate = m.choseGate()
		}
	} else {
		m.chosenInnerGate = m.choseGate()
	}

	if m.chosenInnerGate == "" {
		return nil
	}

	return m.gateMap[m.chosenInnerGate].conn
}

func (m *gateProxy) SendToGate(header []byte, msg proto.Message, gate gnet.Conn) error {
	//只有内部通信才不指定gate
	if gate == nil {
		gate = m.GetInnerGate()
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
