package main

import (
	"go.uber.org/atomic"
	"os"
	"os/signal"
	"syscall"
)

const (
	quitStatusNone      = 0
	quitStatusBeginQuit = 1
	quitStatusQuiting   = 2
	quitStatusQuited    = 3
)

var quit atomic.Int32

var gSysSignalMgr *systemSignal

func initSysSignalMgr() {
	if gSysSignalMgr == nil {
		gSysSignalMgr = new(systemSignal)
		gSysSignalMgr.init()
	}
}

type systemSignal struct {
	ch chan os.Signal
}

func (m *systemSignal) init() {
	quit.Store(quitStatusNone)
	m.ch = make(chan os.Signal, 1)
	signal.Notify(m.ch, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	signal.Notify(m.ch, syscall.SIGQUIT, syscall.SIGILL, syscall.SIGABRT)

	go func() {
		s := <-m.ch
		log.Infof("received signal: %s", s.String())
		getTaskManager().Push(&ServerStopTask{quitStatus: quitStatusQuited})
	}()
}
