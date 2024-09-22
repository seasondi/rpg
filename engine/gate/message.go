package main

import (
	"github.com/panjf2000/gnet"
	"rpg/engine/engine"
	"rpg/engine/message"
)

func responseHeartBeatToClient(clientConn gnet.Conn) {
	data := map[string]interface{}{
		engine.ClientMsgDataFieldType: engine.ClientMsgTypeHeartBeat,
	}
	if r, err := engine.GetProtocol().Marshal(data); err == nil {
		_ = clientConn.AsyncWrite(r)
	}
}

// processHeartBeatResponse 处理心跳回包
func processHeartBeatResponse(_ *engine.TcpClient, clientId engine.ConnectIdType) error {
	if clientId > 0 {
		if clientConn := getClientProxy().client(clientId); clientConn != nil {
			responseHeartBeatToClient(clientConn)
		}
	}
	return nil
}

// processGameRpc rpc消息发给客户端
func processGameRpc(_ *engine.TcpClient, clientId engine.ConnectIdType, buf []byte) error {
	if engine.GetConfig().PrintRpcLog {
		r, _ := engine.GetProtocol().UnMarshal(buf)
		log.Debug("[RPC]", r)
	}
	if clientConn := getClientProxy().client(clientId); clientConn != nil {
		_ = clientConn.AsyncWrite(buf)
	} else {
		log.Warnf("process game rpc but client not found, clientId: %d", clientId)
	}

	return nil
}

// processRouterMessage 内部转发消息发给目标game的entity消息
func processRouterMessage(_ *engine.TcpClient, buf []byte) error {
	msg := message.GameRouterRpc{}
	if err := msg.Unmarshal(buf); err != nil {
		return err
	}
	gameConn := getGameProxy().getGameConnByName(msg.Target)
	if gameConn == nil {
		log.Warnf("route message to game[%s] but conn not found", msg.Target)
		return nil
	}
	pb := &message.GameEntityRpc{
		Data:       msg.Data,
		Source:     engine.ServiceName(),
		FromServer: true,
	}
	if err := getGameProxy().sendProtoToGame(gameConn, engine.ServerMessageTypeEntityRpc, pb); err != nil {
		log.Warnf("processRouterMessage send to game: %s, error: %s", msg.Target, err.Error())
	} else {
		log.Tracef("processRouterMessage send to game: %s", msg.Target)
	}
	return nil
}

// processDisconnectClient 断开客户端连接
func processDisconnectClient(_ *engine.TcpClient, clientId engine.ConnectIdType) error {
	clientConn := getClientProxy().client(clientId)
	if clientConn == nil {
		return nil
	}
	getClientProxy().removeConn(clientId)
	_ = clientConn.Close()
	return nil
}

// processCreateEntity 根据负载选择game创建entity
func processCreateEntity(_ *engine.TcpClient, buf []byte) error {
	msg := message.CreateEntityRequest{}
	if err := msg.Unmarshal(buf); err != nil {
		log.Debug("unmarshal error: ", err.Error())
		return err
	}
	gameConn := getGameProxy().getGameByLoad()
	if gameConn == nil {
		log.Warnf("processCreateEntity but no game selected")
		return nil
	}
	if err := getGameProxy().sendProtoToGame(gameConn, engine.ServerMessageTypeCreateGameEntity, &msg); err != nil {
		log.Warnf("processCreateEntity send to game: %v error: %s", gameConn.Context(), err.Error())
	}

	return nil
}

// processCreateEntityResponse 创建entity结果通知给请求的game进程
func processCreateEntityResponse(_ *engine.TcpClient, buf []byte) error {
	msg := message.CreateEntityResponse{}
	if err := msg.Unmarshal(buf); err != nil {
		return err
	}
	gameConn := getGameProxy().getGameConnByName(msg.ServerName)
	if gameConn == nil {
		log.Warnf("processCreateEntityResponse but target game[%s] not exist", msg.ServerName)
		return nil
	}
	if err := getGameProxy().sendProtoToGame(gameConn, engine.ServerMessageTypeCreateGameEntityRsp, &msg); err != nil {
		log.Warnf("processCreateEntityResponse send to game %s, error: %s", msg.ServerName, err.Error())
	}
	return nil
}

// processLoginByOther 通知前个连接被顶号
func processLoginByOther(_ *engine.TcpClient, clientId engine.ConnectIdType) error {
	if clientConn := getClientProxy().client(clientId); clientConn != nil {
		_ = clientConn.AsyncWrite(genServerErrorMessage(engine.ErrMsgLoginByOther))
		_ = clientConn.Close()
	}
	return nil
}

// 服务器通知错误信息
func processServerError(_ *engine.TcpClient, clientId engine.ConnectIdType, buf []byte) error {
	msg := message.ServerError{}
	if err := msg.Unmarshal(buf); err != nil {
		return err
	}
	if clientConn := getClientProxy().client(clientId); clientConn != nil {
		_ = clientConn.AsyncWrite(genServerErrorMessage(msg.ErrMsg))
	}
	return nil
}

// entity与客户端连接绑定
func processEntityBindClient(game *engine.TcpClient, buf []byte) error {
	msg := message.ClientBindEntity{}
	if err := msg.Unmarshal(buf); err != nil {
		return err
	}

	if msg.Unbind {
		getGameProxy().unBindEntity(engine.ConnectIdType(msg.ClientId), engine.EntityIdType(msg.EntityId))
	} else {
		getGameProxy().bindEntity(engine.ConnectIdType(msg.ClientId), engine.EntityIdType(msg.EntityId), game)
	}
	return nil
}

// processSetServerTime 设置服务器时间
func processSetServerTime(_ *engine.TcpClient, buf []byte) error {
	msg := message.SetServerTimeOffset{}
	if err := msg.Unmarshal(buf); err != nil {
		return err
	}
	data := message.SetServerTimeOffset{Offset: msg.Offset}
	for _, targetServer := range msg.Targets {
		if err := getGameProxy().sendProtoToGameByName(targetServer, engine.ServerMessageTypeSetServerTime, &data); err != nil {
			log.Warnf("set server time send to %s error: %s", targetServer, err.Error())
		}
	}
	return nil
}
