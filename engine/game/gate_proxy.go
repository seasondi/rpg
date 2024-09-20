package main

import (
	"errors"
	"github.com/gogo/protobuf/proto"
	"github.com/panjf2000/gnet"
	lua "github.com/yuin/gopher-lua"
	"math/rand"
	"rpg/engine/engine"
	"rpg/engine/message"
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

type gateInfo struct {
	conn        gnet.Conn
	isInnerGate bool
}

type gateProxy struct {
	gateMapLock     sync.Mutex
	gateMap         map[string]*gateInfo //gate server name -> gate info
	gateConnMapLock sync.Mutex
	gateConnMap     map[gnet.Conn]string //gate conn -> gate server name

	chosenInnerGate string //选取的内部通信gate
}

func (m *gateProxy) init() {
	m.gateMap = make(map[string]*gateInfo)
	m.gateConnMap = make(map[gnet.Conn]string)
}

func (m *gateProxy) AddGate(c gnet.Conn, name string, isInner bool) {
	{
		m.gateMapLock.Lock()
		m.gateMap[name] = &gateInfo{conn: c, isInnerGate: isInner}
		m.gateMapLock.Unlock()
	}
	{
		m.gateConnMapLock.Lock()
		m.gateConnMap[c] = name
		m.gateConnMapLock.Unlock()
	}
	log.Infof("add gate[%s -> %s], is inner gate: %v", name, c.RemoteAddr(), isInner)
}

func (m *gateProxy) RemoveGate(c gnet.Conn) {
	m.gateConnMapLock.Lock()
	defer m.gateConnMapLock.Unlock()

	if name, ok := m.gateConnMap[c]; ok {
		engine.GetEntityManager().RemoveGateEntitiesConn(name)
		delete(m.gateConnMap, c)

		m.gateMapLock.Lock()
		delete(m.gateMap, name)
		if name == m.chosenInnerGate {
			m.chosenInnerGate = ""
		}
		m.gateMapLock.Unlock()

		log.Infof("remove gate[%s -> %s]", name, c.RemoteAddr())
	}
}

func (m *gateProxy) GetGateConn(name string) gnet.Conn {
	m.gateMapLock.Lock()
	defer m.gateMapLock.Unlock()
	return m.gateMap[name].conn
}

func (m *gateProxy) GateName(c gnet.Conn) string {
	m.gateConnMapLock.Lock()
	defer m.gateConnMapLock.Unlock()
	return m.gateConnMap[c]
}

func (m *gateProxy) GetInnerGate() gnet.Conn {
	m.gateMapLock.Lock()
	defer m.gateMapLock.Unlock()

	//选取一个内部通信gate,确保同一个game进程的消息顺序性, 连接断开前不改变
	choseGate := func() string {
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

	if m.chosenInnerGate != "" {
		if info, find := m.gateMap[m.chosenInnerGate]; find && info.conn != nil {
			return info.conn
		} else {
			m.chosenInnerGate = choseGate()
		}
	} else {
		m.chosenInnerGate = choseGate()
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
