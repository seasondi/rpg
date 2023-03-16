package main

import (
	"github.com/panjf2000/gnet"
	"rpg/engine/engine"
	"runtime"
	"time"
)

type eventLoop struct {
	gnet.EventServer
}

func (m *eventLoop) startTick() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	for {
		if delay := m.serverTick(); delay < 0 {
			return
		} else {
			time.Sleep(delay)
		}
	}
}

func (m *eventLoop) OnInitComplete(server gnet.Server) (action gnet.Action) {
	engine.GetServerStep().Start()
	for !engine.GetServerStep().Completed() {
		engine.GetServerStep().Print()
		if delay := m.serverTick(); delay < 0 {
			return gnet.Shutdown
		}
		time.Sleep(time.Second)
	}
	engine.GetEntityManager().SetConnFinder(getGateProxy().GetGateConn)
	log.Infof("game[%s] server init complete, listen at: %s", engine.ServiceName(), server.Addr)

	go m.startTick()
	return gnet.None
}

func (m *eventLoop) OnShutdown(_ gnet.Server) {
}

func (m *eventLoop) OnOpened(c gnet.Conn) (out []byte, action gnet.Action) {
	log.Infof("conn[%s] opened", c.RemoteAddr())
	return nil, gnet.None
}

func (m *eventLoop) OnClosed(c gnet.Conn, err error) (action gnet.Action) {
	log.Infof("conn[%s] closed, msg: %v", c.RemoteAddr(), err)
	getGateProxy().RemoveGate(c)
	return gnet.None
}

func (m *eventLoop) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	switch frame[0] {
	case engine.ServerMessageTypeSayHello:
		if _, _, data, err := engine.ParseMessage(frame); err == nil {
			_ = processSyncGate(data, c)
		}
	default:
		getDataProcessor().append(c, frame)
	}
	return nil, gnet.None
}

func (m *eventLoop) serverTick() time.Duration {
	engine.Tick()
	getDataProcessor().process()
	getDBProxy().HandleMainTick()
	if checkStopServer() == true {
		return -1
	}
	return engine.ServerTick
}

func genCloseClientMessage(clientId engine.ConnectIdType) []byte {
	header := engine.GenMessageHeader(engine.ServerMessageTypeDisconnectClient, clientId)
	buf, _ := engine.GetProtocol().MessageWithHead(header, nil)
	return buf
}
