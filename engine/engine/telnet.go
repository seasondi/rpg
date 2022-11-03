package engine

import (
	"bufio"
	"encoding/json"
	lua "github.com/yuin/gopher-lua"
	"io"
	"net"
	"strings"
	"sync"
	"time"
)

var telnetMgr *telnet

const (
	TelnetMessageTypeWebCmd = 1 //web调试消息
	TelnetMessageTypeGMList = 2 //获取gm命令列表
	TelnetMessageTypeGMCmd  = 3 //gm指令
	TelnetMessageTypeReload = 4 //热更
)

type TelnetMessage struct {
	Type int    `json:"type"`
	Data string `json:"data"`
}

type telnetEnvironment struct {
	Env *lua.LTable
}

type telnet struct {
	sync.Mutex
	env               map[net.Conn]*telnetEnvironment
	commandResultChan chan string
}

func (m *telnet) init() {
	m.env = make(map[net.Conn]*telnetEnvironment)
	m.commandResultChan = make(chan string, 0)
}

func (m *telnet) updateEnvironment(conn net.Conn, env *telnetEnvironment) {
	m.Lock()
	defer m.Unlock()
	m.env[conn] = env
}

func (m *telnet) removeEnvironment(conn net.Conn) {
	m.Lock()
	defer m.Unlock()
	delete(m.env, conn)
}

func TelnetServer(addr string) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Errorf("telnet listen error: %s", err.Error())
		return
	}
	defer l.Close()
	log.Infof("telnet listen at: %s", addr)
	telnetMgr = new(telnet)
	telnetMgr.init()

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Errorf("telnet accept error: %s", err.Error())
			continue
		}
		log.Infof("telnet conn[%s] connected", conn.RemoteAddr())
		go handleSession(conn)
	}
}

func handleSession(conn net.Conn) {
	reader := bufio.NewReader(conn)
	for {
		str, err := reader.ReadString('\n')
		if err == nil {
			msg := TelnetMessage{}
			if err = json.Unmarshal([]byte(str), &msg); err != nil {
				log.Warn("unmarshal telnet msg error: ", err.Error())
				continue
			}

			switch msg.Type {
			case TelnetMessageTypeReload:
				luaCmdMgr.addCommand(reloadHandler, []interface{}{})
			case TelnetMessageTypeGMList:
				luaCmdMgr.addCommand(getGmListHandler, []interface{}{})
			case TelnetMessageTypeWebCmd:
				luaCmdMgr.addCommand(debugCommandHandler, []interface{}{msg.Data, conn})
			case TelnetMessageTypeGMCmd:
				luaCmdMgr.addCommand(gmCommandHandler, []interface{}{msg.Data})
			}

			var rsp string
			select {
			case data := <-telnetMgr.commandResultChan:
				rsp = data
			case <-time.After(3 * time.Second):
				rsp = "command timeout"
			}
			_, _ = conn.Write([]byte(rsp))
		} else {
			if err == io.EOF {
				log.Infof("telnet conn[%s] closed", conn.RemoteAddr())
			} else {
				log.Infof("telnet read error: %s", err.Error())
			}
			_ = conn.Close()
			break
		}
	}
	telnetMgr.removeEnvironment(conn)
}

func reloadHandler(_ ...interface{}) {
	if err := CallLuaMethodByName(GetGlobalEntry(), onReload, 0); err != nil {
		telnetMgr.commandResultChan <- err.Error()
	} else {
		telnetMgr.commandResultChan <- "reload success"
	}
}

func getGmListHandler(_ ...interface{}) {
	//该接口需要一个json格式
	for _, ent := range GetEntityManager().allEntities {
		if ent.entityName == "GMStub" {
			if err := CallLuaMethodByName(ent.luaEntity, getGmListCommand, 1, ent.luaEntity); err != nil {
				log.Warnf("call GMStub:get_gm_list error: %s", err.Error())
				telnetMgr.commandResultChan <- "{}"
				return
			}
			top := luaL.GetTop()
			v := luaL.CheckAny(top)
			if v.Type() != lua.LTString {
				log.Warnf("get_gm_list must return json string")
				telnetMgr.commandResultChan <- "{}"
				return
			} else {
				telnetMgr.commandResultChan <- v.String()
				return
			}
		}
	}
}

func gmCommandHandler(args ...interface{}) {
	cmd := args[0].(string)
	if err := CallLuaMethodByName(GetGlobalEntry(), doGmCommand, 1, lua.LString(cmd)); err != nil {
		telnetMgr.commandResultChan <- "gm command execute failed"
	} else {
		telnetMgr.commandResultChan <- luaL.CheckAny(luaL.GetTop()).String()
	}
}

func debugCommandHandler(args ...interface{}) {
	if GetConfig().Release {
		telnetMgr.commandResultChan <- "debug command not support in release mode"
		return
	}

	str := args[0].(string)
	conn := args[1].(net.Conn)

	luaL.SetGlobal("console_output", lua.LString(""))
	defer luaL.SetGlobal("console_output", lua.LNil)

	str = strings.ReplaceAll(str, "print", "console_print")
	str = strings.ReplaceAll(str, "local ", "")
	fn, err := luaL.LoadString(str)
	if err != nil {
		if strings.Contains(err.Error(), "parse error") {
			str = "console_print(" + str + ")"
			var nErr error
			if fn, nErr = luaL.LoadString(str); nErr != nil {
				telnetMgr.commandResultChan <- err.Error()
				return
			}
		} else {
			telnetMgr.commandResultChan <- err.Error()
			return
		}
	}
	telnetMgr.updateEnvironment(conn, &telnetEnvironment{Env: fn.Env})
	luaL.Push(fn)
	if err = luaL.PCall(0, lua.MultRet, nil); err != nil {
		telnetMgr.commandResultChan <- err.Error()
		return
	}
	telnetMgr.commandResultChan <- luaL.GetGlobal("console_output").String()
	return
}
