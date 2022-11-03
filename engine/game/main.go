package main

import (
	"rpg/engine/engine"
	"fmt"
	"github.com/panjf2000/gnet"
	"github.com/sirupsen/logrus"
	_ "net/http/pprof"
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

	registerApi()
	initServer()
	syncStubFromEtcd()
	initSysSignalMgr()

	defer getDBProxy().Close()

	go engine.TelnetServer(engine.GetConfig().ServerConfig().Telnet)

	_ = gnet.Serve(&eventLoop{}, engine.ListenProtoAddr(),
		gnet.WithCodec(&engine.GNetCodec{}),
		gnet.WithTCPKeepAlive(3*time.Second),
		gnet.WithLogger(log.Logger),
	)
}
