package main

import (
	"rpg/engine/engine"
	"errors"
	"fmt"
	"time"
)

var (
	cbMgr            *callback
	timeoutErr       = errors.New("callback timeout")
	callbackUniqueID = uint64(0)
)

type callbackInterface interface {
	setTimerId(int64)
	cancelTimer()
	Process(error, ...interface{})
}

func getCallbackMgr() *callback {
	if cbMgr == nil {
		cbMgr = new(callback)
		cbMgr.init()
	}
	return cbMgr
}

type callback struct {
	cbMap map[string]callbackInterface
}

func (m *callback) init() {
	m.cbMap = make(map[string]callbackInterface)
}

func (m *callback) setCallbackWithTimeout(key string, value callbackInterface, timeout time.Duration) {
	m.cbMap[key] = value
	timerId := engine.GetTimer().AddTimer(timeout, 0, m.onTimeout, key)
	value.setTimerId(timerId)
}

func (m *callback) removeCallback(key string) {
	if cb, ok := m.cbMap[key]; ok {
		cb.cancelTimer()
		delete(m.cbMap, key)
	}
}

func (m *callback) onTimeout(params ...interface{}) {
	if len(params) == 0 {
		return
	}
	if key, ok := params[0].(string); ok {
		m.Call(key, timeoutErr, params[1:])
	}
}

func (m *callback) Call(key string, err error, params ...interface{}) {
	if cb, ok := m.cbMap[key]; ok {
		cb.cancelTimer()
		getCallbackMgr().removeCallback(key)
		cb.Process(err, params...)
	}
}

func (m *callback) NextUniqueID() string {
	callbackUniqueID += 1
	return fmt.Sprintf("%d_%d_%d", engine.GetConfig().ServerId, engine.GetCmdLine().Tag, callbackUniqueID)
}
