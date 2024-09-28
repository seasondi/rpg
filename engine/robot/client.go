package main

import (
	"go.uber.org/atomic"
	"os"
	"rpg/engine/engine"
	"time"
)

var allClients map[int32]*client
var globalId atomic.Int32

func NewClient() *client {
	c := new(client)
	c.init()
	return c
}

type client struct {
	conn                  *engine.TcpClient
	lastRecvHeartbeatTime int64
	id                    int32
}

func (m *client) init() {
	m.conn = engine.NewTcpClient(engine.WithTcpClientCodec(m), engine.WithTcpClientHandle(m))
}

func (m *client) Connect(addr string) {
	m.conn.Connect(addr, false)
}

func (m *client) HandleMainTick() {
	engine.Tick()
	m.conn.Tick()
}

func (m *client) Encode(data []byte) ([]byte, error) {
	return engine.GetProtocol().Encode(data)
}

func (m *client) Decode(data []byte) (int, []byte, error) {
	return engine.GetProtocol().Decode(data)
}

func (m *client) OnConnect(conn *engine.TcpClient) {
	log.Infof("connected to [%s]", conn.RemoteAddr())

	globalId.Inc()
	m.id = globalId.Load()
	allClients[m.id] = m

	engine.GetTimer().AddTimer(time.Second, time.Second, func(_ ...interface{}) {
		entityId := engine.EntityIdType(0)
		if myself != nil {
			entityId = myself.EntityID()
		}
		hb := genHeartbeatMessage(entityId)
		_, _ = conn.Send(hb)
	})

	loginInfo := map[string]interface{}{
		"account":  "test",
		"password": "pwd",
	}
	buf := genLoginMessage([]interface{}{loginInfo})
	_, _ = conn.Send(buf)
}

func (m *client) OnDisconnect(conn *engine.TcpClient) {
	delete(allClients, m.id)
	log.Infof("disconnect from [%s], client num: %d", conn.RemoteAddr(), len(allClients))
	if len(allClients) == 0 {
		os.Exit(0)
	}
}

func (m *client) OnMessage(_ *engine.TcpClient, buf []byte) error {
	r, err := engine.GetProtocol().UnMarshal(buf)
	if err != nil {
		return err
	}
	dispatchMessage(m, r)
	return nil
}

func (m *client) Stop() {
	os.Exit(0)
}
