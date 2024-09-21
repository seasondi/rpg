package main

import (
	"context"
	"github.com/panjf2000/gnet"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/atomic"
	"rpg/engine/engine"
)

const (
	quitStatusNone      = 0
	quitStatusBeginQuit = 1
	quitStatusQuiting   = 2
	quitStatusQuited    = 3
)

var quit atomic.Int32
var lastQuittingCheckTime int64

func initServer() {
	engine.GetServerStep().Register(engine.ServerStepPrepare, initDBProxy, func() {
		getDBProxy().init()
	})

	engine.GetServerStep().Register(engine.ServerStepInitScript, initScript, func() {
		if err := engine.CallLuaMethodByName(engine.GetGlobalEntry(), "init_server", 1); err != nil {
			log.Errorf("call init_server method error: %s", err.Error())
			return
		} else {
			ret := engine.GetLuaState().Get(1)
			if ret != lua.LBool(true) {
				log.Errorf("init_server failed")
				return
			}
		}
		engine.GetServerStep().FinishHandler(initScript)
	})
}

func checkStopServer() bool {
	switch quit.Load() {
	case quitStatusBeginQuit:
		log.Info("server start quit")
		engine.GetConfig().SaveNumPerTick += 5 //加快存盘速度
		_ = engine.CallLuaMethodByName(engine.GetGlobalEntry(), "stop_server", 0)
		quit.Store(quitStatusQuiting)
	case quitStatusQuiting:
		if engine.CanStopped() {
			quit.Store(quitStatusQuited)
		}
	case quitStatusQuited:
		log.Info("server quit success")
		_ = gnet.Stop(context.TODO(), engine.ListenProtoAddr())
		return true
	}

	return false
}
