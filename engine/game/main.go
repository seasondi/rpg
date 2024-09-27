package main

import (
	"fmt"
	"github.com/panjf2000/gnet"
	"github.com/sirupsen/logrus"
	_ "net/http/pprof"
	"rpg/engine/engine"
	"time"
)

var log *logrus.Entry

func main() {
	if err := engine.Init(engine.STGame); err != nil {
		fmt.Println("engine init error: ", err.Error())
		return
	}
	defer engine.Close()
	log = engine.GetLogger()

	initTaskManager()
	registerApi()
	initServer()
	syncStubFromEtcd()
	initSysSignalMgr()

	defer getDBProxy().Close()

	go engine.TelnetServer(engine.GetConfig().ServerConfig().Telnet)

	err := gnet.Serve(&eventLoop{}, engine.ListenProtoAddr(),
		gnet.WithCodec(&engine.GNetCodec{}),
		gnet.WithTCPKeepAlive(time.Minute),
		gnet.WithLogger(log.Logger),
		gnet.WithMulticore(true),
		gnet.WithReusePort(true),
	)
	if err != nil {
		log.Errorf("gnet serve error: %s", err.Error())
	}
}
