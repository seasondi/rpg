package main

import (
	"rpg/engine/engine"
	"fmt"
	"github.com/panjf2000/gnet"
	"github.com/sirupsen/logrus"
)

var log *logrus.Entry

func main() {
	if err := engine.Init(engine.STGate); err != nil {
		fmt.Println("engine init error: ", err.Error())
		return
	}
	log = engine.GetLogger()
	defer engine.Close()

	getGameProxy().SyncFromEtcd()

	log.Info(gnet.Serve(&eventLoop{}, engine.ListenProtoAddr(),
		gnet.WithCodec(&engine.GNetCodec{}),
		gnet.WithLogger(log.Logger),
	))
}
