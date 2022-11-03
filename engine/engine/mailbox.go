package engine

import (
	"fmt"
	lua "github.com/yuin/gopher-lua"
)

type mailboxType int

const (
	MailBoxTypeEmpty  mailboxType = 0
	MailBoxTypeClient             = 1
)

const (
	mailboxFieldType            = "__mbType"
	clientMailBoxFieldGateName  = "gateName"
	clientMailBoxFieldClientId  = "clientId"
	clientMailBoxFieldSendError = "error"
)

func mailBoxTypeString(ty mailboxType) string {
	switch ty {
	case MailBoxTypeEmpty:
		return "empty"
	case MailBoxTypeClient:
		return "client"
	default:
		return "unknown"
	}
}

type MailBox interface {
	String() string
	Send([]byte)
	Table() *lua.LTable
}

type ClientMailBox struct {
	GateName string
	ClientId ConnectIdType
}

func (m *ClientMailBox) String() string {
	return fmt.Sprintf("ClientMailBox[%s:%d]", m.GateName, m.ClientId)
}

func (m *ClientMailBox) Send(data []byte) {
	if gateConn := GetEntityManager().GetGateConn(m.GateName); gateConn != nil {
		if err := gateConn.AsyncWrite(data); err != nil {
			log.Warnf("send data to %s error: %s", m.String(), err.Error())
		}
	}
}

func (m *ClientMailBox) Table() *lua.LTable {
	t := luaL.NewTable()
	t.RawSetString(mailboxFieldType, lua.LNumber(MailBoxTypeClient))
	t.RawSetString(clientMailBoxFieldGateName, lua.LString(m.GateName))
	t.RawSetString(clientMailBoxFieldClientId, lua.LNumber(m.ClientId))
	t.RawSetString(clientMailBoxFieldSendError, luaL.NewFunction(func(L *lua.LState) int {
		//1: 自身table
		//2: error消息
		self := L.CheckTable(1)
		errMsg := L.CheckString(2)
		if mb := ClientMailBoxFromLua(self); mb != nil {
			if data, err := genServerErrorMessage(errMsg, mb.ClientId); err == nil {
				mb.Send(data)
			}
		}
		return 0
	}))

	meta := luaL.NewTable()
	meta.RawSetString("__tostring", luaL.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LString(m.String()))
		return 1
	}))
	luaL.SetMetatable(t, meta)
	return t
}

func (m *ClientMailBox) Equal(r *ClientMailBox) bool {
	if r == nil {
		return false
	}
	return m.ClientId == r.ClientId && m.GateName == r.GateName
}

func ClientMailBoxFromLua(t *lua.LTable) *ClientMailBox {
	gateName := t.RawGetString(clientMailBoxFieldGateName)
	if gateName.Type() != lua.LTString {
		return nil
	}
	clientId := t.RawGetString(clientMailBoxFieldClientId)
	if clientId.Type() != lua.LTNumber {
		return nil
	}
	return &ClientMailBox{GateName: gateName.String(), ClientId: ConnectIdType(clientId.(lua.LNumber))}
}

func emptyMailBoxTable() *lua.LTable {
	return mapToMailBoxTable(nil)
}

func TableToMailBox(t *lua.LTable) MailBox {
	ty := t.RawGetString(mailboxFieldType)
	if ty.Type() != lua.LTNumber {
		return nil
	}
	switch mailboxType(ty.(lua.LNumber)) {
	case MailBoxTypeClient:
		return ClientMailBoxFromLua(t)
	default:
		return nil
	}
}

func MailBoxToTable(box MailBox) *lua.LTable {
	switch b := box.(type) {
	case *ClientMailBox:
		return b.Table()
	default:
		return emptyMailBoxTable()
	}
}

func mapToMailBoxTable(v map[string]interface{}) *lua.LTable {
	t := mapToTableImpl(v)
	if mb := TableToMailBox(t); mb != nil {
		t = mb.Table()
	} else {
		t.RawSetString(mailboxFieldType, lua.LNumber(MailBoxTypeEmpty))
	}
	return t
}
