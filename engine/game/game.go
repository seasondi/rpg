package main

import (
	"context"
	"encoding/json"
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

func (m *eventLoop) reportLoad() {
	for {
		data := engine.GameLoadInfo{
			Name:        engine.ServiceName(),
			EntityCount: engine.GetEntityManager().GetEntityCount(),
			Time:        time.Now(),
		}
		info, _ := json.Marshal(data)
		if err := engine.GetRedisMgr().HSet(context.Background(), engine.RedisHashGameLoad, engine.ServiceName(), info); err != nil {
			log.Warnf("hset to redis hash: %s, error: %s", engine.RedisHashGameLoad, err.Error())
		}
		time.Sleep(time.Second)
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

	if err := engine.GetEtcd().RegisterServer(); err != nil {
		log.Fatalf("register to etcd failed: %s", err.Error())
	}

	go m.startTick()
	go m.reportLoad()
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
		getDataProcessor().append(c, append([]byte{}, frame...))
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
