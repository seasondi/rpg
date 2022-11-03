package engine

import (
	"fmt"
	"github.com/panjf2000/gnet"
)

func initNetwork() error {
	netMgr = new(network)
	if err := netMgr.Init(); err != nil {
		return err
	}
	log.Infof("network inited.")
	return nil
}

func GetNetwork() *network {
	return netMgr
}

type network struct {
}

func (m *network) Init() error {
	return nil
}

func (m *network) Serve(addr string, handler gnet.EventHandler, opts ...gnet.Option) error {
	return gnet.Serve(handler, fmt.Sprintf("tcp://%s", addr), opts...)
}
