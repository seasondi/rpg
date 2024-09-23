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

func (m *eventLoop) tick() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	var timer *time.Timer

	defer func() {
		if timer != nil {
			timer.Stop()
		}
	}()

	engine.GetTimer().AddTimer(0, time.Second, m.reportLoad)

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

func (m *eventLoop) reportLoad(_ ...interface{}) {
	entityCount := engine.GetEntityManager().GetEntityCount()

	go func(count int) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		data := engine.GameLoadInfo{
			Name:        engine.ServiceName(),
			EntityCount: entityCount,
			Time:        time.Now(),
		}
		info, _ := json.Marshal(data)
		if err := engine.GetRedisMgr().HSet(ctx, engine.RedisGameLoadKey(), engine.ServiceName(), info); err != nil {
			log.Warnf("hset to redis hash: %s, error: %s", engine.RedisGameLoadKey(), err.Error())
		}
	}(entityCount)
}

func (m *eventLoop) OnInitComplete(server gnet.Server) (action gnet.Action) {
	engine.GetServerStep().Start()

	for {
		select {
		case <-time.After(time.Second):
			if !engine.GetServerStep().Completed() {
				if quit.Load() == quitStatusQuited {
					m.disconnectServer()
					goto serverStop
				}
				engine.GetServerStep().Print()
				m.serverTick()
			} else {
				goto serverStart
			}
		}
	}

serverStart:
	engine.GetEntityManager().SetConnFinder(getGateProxy().GetGateConn)
	if err := engine.GetEtcd().RegisterServer(); err != nil {
		log.Warnf("register to etcd failed: %s", err.Error())
		return gnet.Shutdown
	}

	go m.tick()

	log.Infof("game[%s] server init complete, listen at: %s", engine.ServiceName(), server.Addr)
	return gnet.None

serverStop:
	return gnet.Shutdown
}

func (m *eventLoop) OnShutdown(_ gnet.Server) {
}

func (m *eventLoop) OnOpened(c gnet.Conn) (out []byte, action gnet.Action) {
	log.Infof("conn[%s] opened", c.RemoteAddr())
	return nil, gnet.None
}

func (m *eventLoop) OnClosed(c gnet.Conn, err error) (action gnet.Action) {
	log.Infof("conn[%s:%v] closed, msg: %v", c.RemoteAddr(), c.Context(), err)
	getTaskManager().Push(&RemoveGateTask{conn: c})
	return gnet.None
}

func (m *eventLoop) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	getTaskManager().Push(&NetMessageTask{conn: c, buf: append([]byte{}, frame...)})
	return nil, gnet.None
}

func (m *eventLoop) serverTick() time.Duration {
	engine.Tick()
	getTaskManager().Tick()
	getDBProxy().Tick()
	return engine.ServerTick
}

func (m *eventLoop) disconnectServer() {
	if getDBProxy().conn != nil {
		getDBProxy().conn.Disconnect()
	}
}
