package main

import (
	"rpg/engine/engine"
	"github.com/sirupsen/logrus"
	"time"
)

const gateAddr = "127.0.0.1:6300"

var log *logrus.Entry
var myself *engine.Robot

func main() {
	if err := engine.Init(engine.STRobot); err != nil {
		log.Errorf("engine init error: %s", err.Error())
		return
	}
	log = engine.GetLogger()

	allClients = make(map[int32]*client)

	c := NewClient()
	c.Connect(gateAddr)

	for {
		select {
		case <-time.After(100 * time.Millisecond):
			c.HandleMainTick()
		}
	}
}
