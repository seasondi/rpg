package main

import (
	"rpg/engine/engine"
	"container/list"
	"github.com/panjf2000/gnet"
	"sync"
)

var dp *dataProcessor

type task struct {
	clientConnId engine.ConnectIdType
	data         []byte
}

//gNet消息处理器
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
	if connId := getClientId(c); connId > 0 {
		m.tasks.PushBack(&task{clientConnId: connId, data: data})
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

func (m *dataProcessor) process() {
	t := m.pop()
	if t == nil {
		return
	}
	if c := getClientProxy().client(t.clientConnId); c != nil {
		if data, _ := getGameProxy().ClientSendToGame(c, t.data[0], t.data[1:]); data != nil {
			_ = c.AsyncWrite(data)
		}
	} else {
		log.Tracef("process data, client conn already closed, connectId: %d", t.clientConnId)
	}
}
