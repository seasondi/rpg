package main

import (
	"rpg/engine/engine"
	"github.com/panjf2000/gnet"
	"runtime"
	"strings"
	"time"
)

type etcdWatcher struct {
	watcherKey string
}

func (m *etcdWatcher) Key() string {
	return m.watcherKey
}

func (m *etcdWatcher) OnUpdated(kv *engine.EtcdKV) {
	log.Info("etcd key update: ", kv)
	key := kv.Key()
	if strings.HasPrefix(key, engine.ServiceGamePrefix) {
		getGameProxy().HandleUpdateGame(kv.Key(), kv.Value())
	} else if strings.HasPrefix(key, engine.StubPrefix) {
		getGameProxy().HandleUpdateStub(kv.Key(), kv.Value())
	}
}

func (m *etcdWatcher) OnDelete(kv *engine.EtcdKV) {
	log.Info("etcd key delete: ", kv)
	key := kv.Key()
	if strings.HasPrefix(key, engine.ServiceGamePrefix) {
		getGameProxy().HandleDeleteGame(kv.Key())
	} else if strings.HasPrefix(key, engine.StubPrefix) {
		getGameProxy().HandleDeleteStub(kv.Key())
	}
}

type eventLoop struct {
	gnet.EventServer
}

func (m *eventLoop) startTick(server gnet.Server) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	for {
		delay, action := m.serverTick()
		if action == gnet.Shutdown {
			m.OnShutdown(server)
		}
		time.Sleep(delay)
	}
}

func (m *eventLoop) OnInitComplete(server gnet.Server) (action gnet.Action) {
	log.Infof("gate[%s] server init complete, listen at: %s", engine.ServiceName(), server.Addr)
	go m.startTick(server)
	return gnet.None
}

func (m *eventLoop) OnShutdown(_ gnet.Server) {
}

func (m *eventLoop) OnOpened(c gnet.Conn) (out []byte, action gnet.Action) {
	log.Infof("conn[%s] opened", c.RemoteAddr())
	getClientProxy().addConn(c)
	return nil, gnet.None
}

func (m *eventLoop) OnClosed(c gnet.Conn, err error) (action gnet.Action) {
	log.Infof("conn[%s] closed, msg: %v", c.RemoteAddr(), err)
	getClientProxy().removeConn(c)
	getGameProxy().onClientClosed(c)
	return gnet.None
}

func (m *eventLoop) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	getDataProcessor().append(c, frame)
	return nil, gnet.None
}

func (m *eventLoop) serverTick() (delay time.Duration, action gnet.Action) {
	engine.Tick()
	getDataProcessor().process()
	getGameProxy().HandleMainTick()
	return engine.ServerTick, gnet.None
}
