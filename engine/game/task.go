package main

import (
	"errors"
	"fmt"
	"github.com/panjf2000/gnet"
	"rpg/engine/engine"
)

type RemoveGateTask struct {
	conn gnet.Conn
}

func (m *RemoveGateTask) HandleTask() error {
	getGateProxy().RemoveGate(m.conn)
	return nil
}

type NetMessageTask struct {
	conn gnet.Conn
	buf  []byte
}

func (m *NetMessageTask) HandleTask() error {
	gateName := getCtxServiceName(m.conn)
	ty, clientId, data, err := engine.ParseMessage(m.buf)
	if err != nil {
		log.Errorf("process message parse error, from gate: %s, err: %s", gateName, err.Error())
		return err
	}
	if m.conn == nil {
		log.Warnf("process message but gate conn is nil, gate: %s, clientId: %d, messageType: %d", gateName, clientId, ty)
		return errors.New("process message but gate conn is nil")
	}

	log.Tracef("type: %d, clientId: %d, data: %v from gate[%s:%s]", ty, clientId, data, gateName, m.conn.RemoteAddr())
	switch ty {
	case engine.ServerMessageTypeSayHello:
		err = processSyncGate(data, m.conn)
	case engine.ServerMessageTypeHeartBeat:
		err = processHeartBeat(m.conn, clientId)
	case engine.ServerMessageTypeDisconnectClient:
		log.Infof("client disconnect, gateName: %s, clientId: %d", gateName, clientId)
		engine.GetEntityManager().RemoveEntityConnInfo(gateName, clientId)
	case engine.ServerMessageTypeEntityRpc:
		err = processEntityRpc(data)
	case engine.ServerMessageTypeLogin:
		err = processEntityLogin(data, clientId)
	case engine.ServerMessageTypeCreateGameEntity:
		err = processCreateEntity(data, m.conn)
	case engine.ServerMessageTypeCreateGameEntityRsp:
		err = processCreateEntityResponse(data, m.conn)
	case engine.ServerMessageTypeSetServerTime:
		err = processSetServerTime(data, m.conn)
	default:
		err = fmt.Errorf("unknown message type %d", ty)
	}
	if err != nil {
		log.Errorf("message process error: %s", err.Error())
		if clientId > 0 {
			msg := genCloseClientMessage(clientId)
			return m.conn.AsyncWrite(msg)
		}
	}
	return nil
}

type ServerStopTask struct {
	quitStatus int
}

func (m *ServerStopTask) HandleTask() error {
	switch m.quitStatus {
	case quitStatusBeginQuit:
		log.Info("server start quit")
		engine.GetConfig().SaveNumPerTick = 5
		_ = engine.CallLuaMethodByName(engine.GetGlobalEntry(), "stop_server", 0)
		getTaskManager().Push(&ServerStopTask{quitStatus: quitStatusQuiting})
	case quitStatusQuiting:
		if engine.CanStopped() {
			getTaskManager().Push(&ServerStopTask{quitStatus: quitStatusQuited})
		} else {
			getTaskManager().Push(&ServerStopTask{quitStatus: quitStatusQuiting})
		}
	case quitStatusQuited:
		log.Info("server quit enter quited")
		quit.Store(quitStatusQuited)
	}

	return nil
}
