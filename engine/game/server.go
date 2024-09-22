package main

import (
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

func initServer() {
	quit.Store(quitStatusNone)
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
