package main

import (
	"context"
	"github.com/panjf2000/gnet"
	"rpg/engine/engine"
	"runtime"
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
	getTaskManager().Push(&AddGameServerTask{kv: *kv})
}

func (m *etcdWatcher) OnDelete(kv *engine.EtcdKV) {
	log.Info("etcd key delete: ", kv)
	getTaskManager().Push(&RemoveGameServerTask{kv: *kv})
}

type eventLoop struct {
	gnet.EventServer
}

func (m *eventLoop) tick() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	var timer *time.Timer

	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()

	engine.GetTimer().AddTimer(0, 2*time.Second, m.getGameLoad)

	for {
		if quit.Load() == quitStatusQuited {
			log.Infof("server main tick stopped")
			m.disconnectServer()
			_ = gnet.Stop(context.Background(), engine.ListenProtoAddr())
			return
		}
		delay := m.serverTick()
		if timer == nil {
			timer = time.NewTimer(delay)
		} else {
			timer.Reset(delay)
		}
		select {
		case <-timer.C:
		}
	}
}

func (m *eventLoop) getGameLoad(_ ...interface{}) {
	getGameProxy().updateGameLoadInfo()
}

func (m *eventLoop) OnInitComplete(server gnet.Server) (action gnet.Action) {
	log.Infof("gate[%s] server init complete, listen at: %s", engine.ServiceName(), server.Addr)
	if err := engine.GetEtcd().RegisterServer(); err != nil {
		log.Fatalf("register to etcd failed: %s", err.Error())
	}
	go m.tick()
	return gnet.None
}

func (m *eventLoop) OnShutdown(_ gnet.Server) {
}

func (m *eventLoop) OnOpened(c gnet.Conn) (out []byte, action gnet.Action) {
	log.Infof("conn[%s] opened", c.RemoteAddr())
	getTaskManager().Push(&AddClientTask{conn: c})
	return nil, gnet.None
}

func (m *eventLoop) OnClosed(c gnet.Conn, err error) (action gnet.Action) {
	log.Infof("conn[%s] closed, msg: %v", c.RemoteAddr(), err)
	getTaskManager().Push(&RemoveClientTask{clientId: getClientId(c)})
	return gnet.None
}

func (m *eventLoop) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	getTaskManager().Push(&ClientMessageTask{conn: c, buf: append([]byte{}, frame...)})
	return nil, gnet.None
}

func (m *eventLoop) serverTick() time.Duration {
	engine.Tick()
	getTaskManager().Tick()
	getGameProxy().Tick()
	return engine.ServerTick
}

func (m *eventLoop) disconnectServer() {
	getGameProxy().Disconnect()
}
