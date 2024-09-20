package main

import (
	"container/list"
	"fmt"
	"github.com/panjf2000/gnet"
	"rpg/engine/engine"
	"sync"
)

var dp *dataProcessor

type task struct {
	gateName string
	data     []byte
}

// gNet消息处理器
func getDataProcessor() *dataProcessor {
	if dp == nil {
		dp = new(dataProcessor)
		dp.init()
	}
	return dp
}

type dataProcessor struct {
	sync.Mutex
	tasks *list.List
}

func (m *dataProcessor) init() {
	m.Lock()
	defer m.Unlock()

	m.tasks = list.New()
}

func (m *dataProcessor) append(c gnet.Conn, data []byte) {
	m.Lock()
	defer m.Unlock()
	if name := getGateProxy().GateName(c); name != "" {
		m.tasks.PushBack(&task{gateName: name, data: data})
	} else {
		log.Tracef("reciev task from unkonwn gate, ignored, addr: 127.0.0.1:64852")
	}
}

func (m *dataProcessor) pop() *task {
	m.Lock()
	defer m.Unlock()

	front := m.tasks.Front()
	if front == nil {
		return nil
	}
	m.tasks.Remove(front)
	return front.Value.(*task)
}

// 处理网络接收的消息
func (m *dataProcessor) process() {
	t := m.pop()
	if t == nil {
		return
	}

	ty, clientId, data, err := engine.ParseMessage(t.data)
	if err != nil {
		log.Errorf("process message parse error, addr: %s, err: %s", t.gateName, err.Error())
		return
	}
	c := getGateProxy().GetGateConn(t.gateName)
	if c == nil {
		log.Warnf("process message but gate conn is nil, gate: %s, clientId: %d, messageType: %d", t.gateName, clientId, ty)
		return
	}
	log.Tracef("type: %d, clientId: %d, data: %v", ty, clientId, data)
	switch ty {
	case engine.ServerMessageTypeHeartBeat:
		err = processHeartBeat(c, clientId)
	case engine.ServerMessageTypeDisconnectClient:
		if gateName := getCtxServiceName(c); gateName != "" {
			log.Infof("client disconnect, gateName: %s, clientId: %d", gateName, clientId)
			engine.GetEntityManager().RemoveEntityConnInfo(gateName, clientId)
		}
	case engine.ServerMessageTypeEntityRpc:
		err = processEntityRpc(data)
	case engine.ServerMessageTypeLogin:
		err = processEntityLogin(data, clientId)
	case engine.ServerMessageTypeCreateGameEntity:
		err = processCreateEntity(data, c)
	case engine.ServerMessageTypeCreateGameEntityRsp:
		err = processCreateEntityResponse(data, c)
	case engine.ServerMessageTypeSetServerTime:
		err = processSetServerTime(data, c)
	default:
		err = fmt.Errorf("unknown message type %d", ty)
	}
	if err != nil {
		log.Errorf("message process error: %s", err.Error())
		if clientId > 0 {
			msg := genCloseClientMessage(clientId)
			_ = c.AsyncWrite(msg)
			return
		}
	}
}
