package main

import (
	"errors"
	"github.com/panjf2000/gnet"
	lua "github.com/seasondi/gopher-lua"
	"rpg/engine/engine"
	"rpg/engine/message"
)

func genCloseClientMessage(clientId engine.ConnectIdType) []byte {
	header := engine.GenMessageHeader(engine.ServerMessageTypeDisconnectClient, clientId)
	buf, _ := engine.GetProtocol().MessageWithHead(header, nil)
	return buf
}

// processHeartBeat 处理心跳
func processHeartBeat(c gnet.Conn, clientId engine.ConnectIdType) error {
	ctx, ok := c.Context().(*connContext)
	if !ok {
		return nil
	}
	if clientId == 0 {
		return nil
	}
	_ = getGateProxy().SendToGate(engine.GenMessageHeader(engine.ServerMessageTypeHeartBeatRsp, clientId), nil, c)
	engine.GetEntityManager().SetHeartbeat(ctx.serverName, clientId)
	return nil
}

// processSyncGate 将gate连接与gate名称绑定
func processSyncGate(buf []byte, c gnet.Conn) error {
	msg := message.SayHello{}
	if err := msg.Unmarshal(buf); err != nil {
		return err
	}
	setCtxServiceName(c, msg.ServiceName)
	getGateProxy().AddGate(c, msg.ServiceName, msg.Inner)
	return nil
}

// processEntityRpc 处理entity函数调用
func processEntityRpc(buf []byte) error {
	msg := message.GameEntityRpc{}
	if err := msg.Unmarshal(buf); err != nil {
		return err
	}
	r, err := engine.GetProtocol().UnMarshal(msg.Data)
	if err != nil {
		return err
	}
	entityId := engine.InterfaceToInt(r[engine.ClientMsgDataFieldEntityID])
	if entityId == 0 {
		return errors.New("not found entity field")
	}
	ent := engine.GetEntityManager().GetEntityById(engine.EntityIdType(entityId))
	if ent == nil {
		log.Warnf("gate call entity[%v] method but entity not found", entityId)
		return nil
	}

	params, ok := r[engine.ClientMsgDataFieldArgs].([]interface{})
	if ok == false {
		return errors.New("invalid args data")
	}
	if len(params) < 1 {
		return errors.New("invalid args data length")
	}
	if method, ok := params[0].(string); !ok {
		return errors.New("invalid method name")
	} else {
		log.Tracef("call %s server method: %s, is from server: %v", ent.String(), method, msg.FromServer)
		args := engine.InterfaceToLValues(params[1:])
		if !msg.FromServer {
			args = append([]lua.LValue{lua.LNumber(entityId)}, args...)
		}
		if err = ent.CallDefServerMethod(method, args, !msg.FromServer); err != nil {
			log.Warnf("call %s method[%s] error: %s", ent.String(), method, err.Error())
			return nil
		}
	}

	return nil
}

// processEntityLogin entity登录
func processEntityLogin(buf []byte, clientId engine.ConnectIdType) error {
	msg := message.GameEntityRpc{}
	if err := msg.Unmarshal(buf); err != nil {
		return err
	}
	r, err := engine.GetProtocol().UnMarshal(msg.Data)
	if err != nil {
		return err
	}

	params, ok := r[engine.ClientMsgDataFieldArgs].([]interface{})
	if ok == false {
		log.Errorf("entity login, invalid args data, clientId: %d", clientId)
		return errors.New("invalid args data")
	}

	ent := engine.GetEntityManager().GetEntryEntity()
	if ent == nil {
		return errors.New("not found entry entity")
	}

	client := &engine.ClientMailBox{GateName: msg.Source, ClientId: clientId}
	args := append([]interface{}{client}, params...)
	if err = ent.CallDefServerMethod(engine.StubEntryMethod, engine.InterfaceToLValues(args), false); err != nil {
		log.Infof("call %s method[%s] error: %s", ent.String(), engine.StubEntryMethod, err.Error())
		return nil
	}

	return nil
}

// processCreateEntity 创建entity
func processCreateEntity(buf []byte, c gnet.Conn) error {
	msg := message.CreateEntityRequest{}
	if err := msg.Unmarshal(buf); err != nil {
		return err
	}
	rsp := message.CreateEntityResponse{}
	ent, err := engine.GetEntityManager().CreateEntity(msg.EntityName)
	if err != nil {
		rsp.ErrMsg = err.Error()
	} else {
		rsp.EntityId = int64(ent.GetEntityId())
	}
	rsp.Ex = msg.Ex
	rsp.ServerName = msg.ServerName
	//如果创建entity的进程与请求创建的是同一个进程,则直接处理回调
	if msg.ServerName == engine.ServiceName() {
		getCallbackMgr().Call(rsp.Ex.Uuid, err, rsp.EntityId)
	} else {
		_ = getGateProxy().SendToGate(engine.GenMessageHeader(engine.ServerMessageTypeCreateGameEntityRsp, 0), &rsp, c)
	}
	return nil
}

// processCreateEntityResponse 创建entity结果通知
func processCreateEntityResponse(buf []byte, _ gnet.Conn) error {
	msg := message.CreateEntityResponse{}
	if err := msg.Unmarshal(buf); err != nil {
		return err
	}
	var err error
	if msg.ErrMsg != "" {
		err = errors.New(msg.ErrMsg)
	}
	getCallbackMgr().Call(msg.Ex.Uuid, err, msg.EntityId)
	return nil
}

// processSetServerTime 设置服务器时间
func processSetServerTime(buf []byte, _ gnet.Conn) error {
	msg := message.SetServerTimeOffset{}
	if err := msg.Unmarshal(buf); err != nil {
		return err
	}
	engine.SetTimeOffset(msg.Offset)
	return nil
}
