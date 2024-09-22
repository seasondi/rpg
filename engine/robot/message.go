package main

import (
	"rpg/engine/engine"
	"time"
)

func handlerCreateEntity(c *client, id engine.EntityIdType, args []interface{}) {
	entityName := args[0].(string)
	entity, err := engine.GetRobotManager().CreateEntity(id, entityName, map[string]interface{}{}, c.conn)
	if err != nil {
		log.Errorf("create [%s:%d] error: %s", entityName, id, err.Error())
	} else {
		log.Infof("create robot %s success", entity.String())
		myself = entity
	}
}

func handlerEntityRpc(id engine.EntityIdType, args []interface{}) {
	ent := engine.GetRobotManager().GetEntityById(id)
	if ent == nil {
		log.Debugf("handler entity rpc, entity %d not found", id)
		return
	}
	method := args[0].(string)
	err := ent.CallDefClientMethod(method, engine.InterfaceToLValues(args[1:]))
	if err != nil {
		log.Debugf("call entity %s method: %s error: %s", ent.String(), method, err.Error())
		return
	}
}

func handlerEntityPropsUpdate(id engine.EntityIdType, args []interface{}) {
	ent := engine.GetRobotManager().GetEntityById(id)
	if ent == nil {
		log.Debugf("handler entity props update, entity %d not found", id)
		return
	}
	ent.OnServerSyncProp(args[0].(string), engine.InterfaceToLValue(args[1]))
}

func handlerEntityPropsPartUpdate(id engine.EntityIdType, args []interface{}) {
	ent := engine.GetRobotManager().GetEntityById(id)
	if ent == nil {
		log.Debugf("handler entity props update, entity %d not found", id)
		return
	}
	ent.OnServerSyncPropPart(args[0].(string), engine.InterfaceToLValue(args[1]), engine.InterfaceToLValue(args[2]))
}

func handlerHeartbeat(c *client) {
	c.lastRecvHeartbeatTime = time.Now().Unix()
}

func handlerServerTips(_ *client, args []interface{}) {
	msg := args[0].(string)
	log.Info(msg)
}

func dispatchMessage(c *client, data map[string]interface{}) {
	msgType := engine.InterfaceToInt(data[engine.ClientMsgDataFieldType])
	if msgType == engine.ClientMsgTypeHeartBeat {
		handlerHeartbeat(c)
		return
	}
	if engine.GetConfig().PrintRpcLog {
		log.Debug(data)
	}
	entityId := engine.EntityIdType(engine.InterfaceToInt(data[engine.ClientMsgDataFieldEntityID]))
	args := data[engine.ClientMsgDataFieldArgs].([]interface{})

	switch int(msgType) {
	case engine.ClientMsgTypeCreateEntity:
		handlerCreateEntity(c, entityId, args)
	case engine.ClientMsgTypeEntityRpc:
		handlerEntityRpc(entityId, args)
	case engine.ClientMsgTypePropSync:
		handlerEntityPropsUpdate(entityId, args)
	case engine.ClientMsgTypePropSyncUpdate:
		handlerEntityPropsPartUpdate(entityId, args)
	case engine.ClientMsgTypeTips:
		handlerServerTips(c, args)
	}
}
