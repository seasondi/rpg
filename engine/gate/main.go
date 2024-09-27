package main

import (
	"fmt"
	"github.com/panjf2000/gnet"
	"github.com/sirupsen/logrus"
	"rpg/engine/engine"
	"time"
)

var log *logrus.Entry

func main() {
	if err := engine.Init(engine.STGate); err != nil {
		fmt.Println("engine init error: ", err.Error())
		return
	}
	log = engine.GetLogger()
	defer engine.Close()

	initTaskManager()
	getGameProxy().SyncFromEtcd()
	initSysSignalMgr()

	err := gnet.Serve(&eventLoop{}, engine.ListenProtoAddr(),
		gnet.WithCodec(&engine.GNetCodec{}),
		gnet.WithLogger(log.Logger),
		gnet.WithMulticore(true),
		gnet.WithTCPKeepAlive(3*time.Minute),
		gnet.WithReusePort(true),
	)
	if err != nil {
		log.Errorf("gnet serve error: %s", err.Error())
	}
}
