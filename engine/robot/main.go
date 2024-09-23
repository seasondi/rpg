package main

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"rpg/engine/engine"
	"time"
)

const gateAddr = "127.0.0.1:6300"

var log *logrus.Entry
var myself *engine.Robot

func main() {
	if err := engine.Init(engine.STRobot); err != nil {
		fmt.Print("engine init error: ", err.Error())
		return
	}
	log = engine.GetLogger()

	allClients = make(map[int32]*client)

	c := NewClient()
	c.Connect(gateAddr)

	for {
		select {
		case <-time.After(engine.ServerTick):
			c.HandleMainTick()
		}
	}
}
