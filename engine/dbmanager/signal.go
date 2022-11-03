package main

import (
	"context"
	"github.com/panjf2000/gnet"
	"os"
	"os/signal"
	"rpg/engine/engine"
	"syscall"
	"time"
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
		stopServer()
	}()
}

func stopServer() {
	for num := dbMgr.TaskMgr.TaskSize(); num > 0; num = dbMgr.TaskMgr.TaskSize() {
		log.Info("buffed task still has ", num, " tasks")
		time.Sleep(2 * time.Second)
	}
	dbMgr.TaskPool.Release()
	for num := dbMgr.TaskPool.Running(); num > 0; num = dbMgr.TaskPool.Running() {
		log.Info("task pool still has ", num, " running goroutines")
		time.Sleep(2 * time.Second)
	}
	_ = gnet.Stop(context.TODO(), engine.ListenProtoAddr())
}
