package engine

import (
	"flag"
	"fmt"
)

func GetCmdLine() *commandLine {
	if cmdLineMgr == nil {
		cmdLineMgr = new(commandLine)
	}
	return cmdLineMgr
}

type commandLine struct {
	Config string
	Tag    ServerTagType
}

func (m *commandLine) Help() {
	help := "Usage: \n"
	help += "--config=/path/to/config/file\n"
	help += "--tag=str 进程编号\n"
	fmt.Print(help)
}

func (m *commandLine) check() error {
	if len(m.Config) == 0 {
		return fmt.Errorf("invalid config file: %s", m.Config)
	}
	if m.Tag < 0 {
		return fmt.Errorf("invalid tag: %d", m.Tag)
	}
	return nil
}

func (m *commandLine) Parse() error {
	tag := 0
	flag.StringVar(&m.Config, "config", "", "配置文件")
	flag.IntVar(&tag, "tag", -1, "进程编号")
	flag.Parse()
	m.Tag = ServerTagType(tag)
	if err := m.check(); err != nil {
		return err
	}
	return nil
}
