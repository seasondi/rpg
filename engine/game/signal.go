package main

import (
	"os"
	"os/signal"
	"syscall"
)

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
	m.ch = make(chan os.Signal, 1)
	signal.Notify(m.ch, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	signal.Notify(m.ch, syscall.SIGQUIT, syscall.SIGILL, syscall.SIGABRT)

	go func() {
		s := <-m.ch
		log.Infof("received signal: %s", s.String())
		getTaskManager().Push(&ServerStopTask{quitStatus: quitStatusBeginQuit})
	}()
}
