package main

import (
	"rpg/engine/engine"
	"rpg/engine/message"
	"context"
	"errors"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/panjf2000/gnet"
	clientV3 "go.etcd.io/etcd/client/v3"
	"sync"
	"time"
)

var gameMgr *gameProxy

type gameHandler struct {
}

func (m *gameHandler) Encode(data []byte) ([]byte, error) {
	return engine.GetProtocol().Encode(data)
}

func (m *gameHandler) Decode(data []byte) (int, []byte, error) {
	return engine.GetProtocol().Decode(data)
}

func (m *gameHandler) OnConnect(conn *engine.TcpClient) {
	log.Infof("connected to [%s]", conn.RemoteAddr())
	getGameProxy().sayHello(conn)
}

func (m *gameHandler) OnDisconnect(conn *engine.TcpClient) {
	log.Infof("disconnect from [%s]", conn.RemoteAddr())
}

func (m *gameHandler) OnMessage(conn *engine.TcpClient, buf []byte) error {
	msgId, clientId, data, err := engine.ParseMessage(buf)
	log.Trace("on process game message, msgId: ", msgId, ", clientId: ", clientId, ", err: ", err)
	if err == nil {
		switch msgId {
		case engine.ServerMessageTypeHeartBeatRsp:
			err = processHeartBeatResponse(conn, clientId)
		case engine.ServerMessageTypeEntityRpc:
			err = processGameRpc(conn, clientId, data)
		case engine.ServerMessageTypeEntityRouter:
			err = processRouterMessage(conn, data)
		case engine.ServerMessageTypeDisconnectClient:
			err = processDisconnectClient(conn, clientId)
		case engine.ServerMessageTypeCreateGameEntity:
			err = processCreateEntity(conn, data)
		case engine.ServerMessageTypeCreateGameEntityRsp:
			err = processCreateEntityResponse(conn, data)
		case engine.ServerMessageTypeLoginByOther:
			err = processLoginByOther(conn, clientId)
		case engine.ServerMessageTypeServerError:
			err = processServerError(conn, clientId, data)
		case engine.ServerMessageTypeChangeEntityClient:
			err = processEntityBindClient(conn, data)
		case engine.ServerMessageTypeSetServerTime:
			err = processSetServerTime(conn, data)
		}
	}
	if err != nil {
		log.Errorf("onMessage from game error: %s", err.Error())
	}
	return nil
}

func getGameProxy() *gameProxy {
	if gameMgr == nil {
		gameMgr = new(gameProxy)
		gameMgr.init()
	}
	return gameMgr
}

type stubInfo struct {
	serverName string              //所在进程名
	entityId   engine.EntityIdType //stub entity_id
}

type serverInfo struct {
	conn   *engine.TcpClient
	isStub bool
}

type gameProxy struct {
	sync.Mutex
	gameServers  map[string]serverInfo                                              //game server name -> serverInfo
	stubEntities map[string]stubInfo                                                //stub entity name -> stubInfo
	clientToGame map[engine.ConnectIdType]map[engine.EntityIdType]*engine.TcpClient //clientConnectId -> entityId-> TcpClient
	entryStub    *stubInfo                                                          //入口stub
}

func (m *gameProxy) init() {
	m.gameServers = make(map[string]serverInfo)
	m.stubEntities = make(map[string]stubInfo)
	m.clientToGame = make(map[engine.ConnectIdType]map[engine.EntityIdType]*engine.TcpClient)
}

func (m *gameProxy) HandleUpdateGame(key string, value engine.EtcdValue) {
	m.Lock()
	defer m.Unlock()

	prefix, serverId, _, err := engine.ParseEtcdServerKey(key)
	if err != nil {
		log.Debugf("parse game server key failed: %s, key: %s", err.Error(), key)
		return
	}
	if prefix != engine.ServiceGamePrefix || serverId != engine.GetConfig().ServerId {
		return
	}

	if game, ok := m.gameServers[key]; ok {
		if game.conn.IsDisconnected() {
			delete(m.gameServers, key)
		} else {
			return
		}
	}

	if addr, ok := value[engine.EtcdValueAddr].(string); ok {
		handler := &gameHandler{}
		gameConn := engine.NewTcpClient(engine.WithTcpClientCodec(handler), engine.WithTcpClientHandle(handler), engine.WithTcpClientContext(key))
		isStub := false
		isStub, ok = value[engine.EtcdValueIsStub].(bool)
		m.gameServers[key] = serverInfo{
			conn:   gameConn,
			isStub: isStub,
		}
		gameConn.Connect(addr, false)
	}
}

func (m *gameProxy) HandleDeleteGame(key string) {
	m.Lock()
	defer m.Unlock()

	if game, find := m.gameServers[key]; find {
		game.conn.Disconnect()
		delete(m.gameServers, key)
	}
}

func (m *gameProxy) HandleUpdateStub(key string, value engine.EtcdValue) {
	m.Lock()
	defer m.Unlock()

	prefix, serverId, entityId, err := engine.ParseEtcdStubKey(key)
	if err != nil {
		log.Warnf("parse etcd stub key failed: %s, key: %s", err.Error(), key)
		return
	}
	if prefix != engine.StubPrefix || serverId != engine.GetConfig().ServerId {
		return
	}
	stubName, ok := value[engine.EtcdValueName].(string)
	if !ok {
		log.Warn("invalid stub name, value is: ", value)
		return
	}
	serverName, ok := value[engine.EtcdValueServer].(string)
	if !ok {
		log.Warn("invalid stub server name, value is: ", value)
		return
	}

	si := stubInfo{serverName: serverName, entityId: entityId}
	m.stubEntities[stubName] = si

	if name, ok := value[engine.EtcdStubValueEntry].(string); ok && name == stubName {
		m.entryStub = &si
	}
}

func (m *gameProxy) HandleDeleteStub(key string) {
	m.Lock()
	defer m.Unlock()

	prefix, serverId, entityId, err := engine.ParseEtcdStubKey(key)
	if err != nil {
		log.Warnf("parse etcd stub key failed: %s, key: %s", err.Error(), key)
		return
	}
	if prefix != engine.StubPrefix || serverId != engine.GetConfig().ServerId {
		return
	}

	for name, stub := range m.stubEntities {
		if stub.entityId == entityId {
			if m.entryStub == &stub {
				m.entryStub = nil
			}
			delete(m.stubEntities, name)
			break
		}
	}
}

func (m *gameProxy) SyncFromEtcd() {
	//game进程
	{
		ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
		defer cancel()
		prefix := engine.GetEtcdPrefixWithServer(engine.ServiceGamePrefix)
		for _, kv := range engine.GetEtcd().Get(ctx, prefix, clientV3.WithPrefix()) {
			m.HandleUpdateGame(kv.Key(), kv.Value())
		}
		go engine.GetEtcd().Watch(&etcdWatcher{watcherKey: prefix}, clientV3.WithPrefix())
	}

	//stub
	{
		ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
		defer cancel()
		prefix := engine.GetEtcdPrefixWithServer(engine.StubPrefix)
		for _, kv := range engine.GetEtcd().Get(ctx, prefix, clientV3.WithPrefix()) {
			m.HandleUpdateStub(kv.Key(), kv.Value())
		}
		go engine.GetEtcd().Watch(&etcdWatcher{watcherKey: prefix}, clientV3.WithPrefix())
	}
}

func (m *gameProxy) HandleMainTick() {
	m.Lock()
	defer m.Unlock()

	for _, info := range m.gameServers {
		info.conn.Tick()
	}
}

//getEntryStubConn 获取入口stub的连接信息
func (m *gameProxy) getEntryStubConn() *engine.TcpClient {
	if m.entryStub == nil {
		return nil
	}
	for name, info := range m.gameServers {
		if name == m.entryStub.serverName {
			return info.conn
		}
	}
	return nil
}

//sendRpcToGame 发送entity rpc消息到指定game
func (m *gameProxy) sendRpcToGame(game *engine.TcpClient, msgTy uint8, clientId engine.ConnectIdType, data []byte) ([]byte, gnet.Action) {
	pb := &message.GameEntityRpc{
		Data:   data,
		Source: engine.ServiceName(),
	}
	svrMsgType := toServerMessageType(msgTy)
	switch svrMsgType {
	case engine.ServerMessageTypeLogin:
		if m.entryStub == nil {
			log.Warn("client login but entry stub not found")
			return genServerErrorMessage(engine.ErrMsgServerNotReady), gnet.None
		}
		if info, err := engine.GetProtocol().UnMarshal(data); err == nil {
			info[engine.ClientMsgDataFieldEntityID] = m.entryStub.entityId
			pb.Data, _ = engine.GetProtocol().Marshal(info)
		} else {
			log.Warnf("client login but data unmarshal error: %s", err.Error())
			return nil, gnet.None
		}
	}
	head := engine.GenMessageHeader(svrMsgType, clientId)
	if buf, err := engine.GetProtocol().MessageWithHead(head, pb); err == nil {
		n, _ := game.Send(buf)
		log.Tracef("send %d bytes to game %s", n, game.Context())
	} else {
		log.Warnf("send rpc to game encode error: %s", err.Error())
	}

	return nil, gnet.None
}

//sendProtoToGameByName 内部消息发往指定serviceName的game
func (m *gameProxy) sendProtoToGameByName(gameName string, ty uint8, msg proto.Message) error {
	if conn := getGameProxy().getGameConnByName(gameName); conn != nil {
		return m.sendProtoToGame(conn, ty, msg)
	}
	return fmt.Errorf("no game named %s", gameName)
}

//sendProtoToGame 内部消息发往指定连接的game
func (m *gameProxy) sendProtoToGame(gameConn *engine.TcpClient, ty uint8, msg proto.Message) error {
	head := engine.GenMessageHeader(ty, 0)
	if w, err := engine.GetProtocol().MessageWithHead(head, msg); err == nil {
		n, err := gameConn.Send(w)
		log.Tracef("send %d bytes to %s", n, gameConn.RemoteAddr())
		return err
	} else {
		return err
	}
}

//ClientSendToGame 客户端消息发往game
func (m *gameProxy) ClientSendToGame(client gnet.Conn, msgTy uint8, data []byte) ([]byte, gnet.Action) {
	log.Tracef("received message from client %s type: %d, data %+v", client.RemoteAddr(), msgTy, data)
	clientId := getClientId(client)
	if clientId == 0 {
		log.Warn("gate send to game invalid client connection")
		return genServerErrorMessage(engine.ErrMsgClientConnectionInvalid), gnet.None
	}
	var gameConn *engine.TcpClient
	if msgTy == engine.ClientMsgTypeLogin {
		gameConn = m.getEntryStubConn()
	} else if clientData, err := engine.GetProtocol().UnMarshal(data); err == nil {
		if entityId := engine.InterfaceToInt(clientData[engine.ClientMsgDataFieldEntityID]); entityId > 0 {
			gameConn = m.getGameConn(clientId, engine.EntityIdType(entityId))
		} else if msgTy == engine.ClientMsgTypeHeartBeat { //尚未登录,由gate暂时接管心跳
			responseHeartBeatToClient(client)
			return nil, gnet.None
		} else {
			return genServerErrorMessage(engine.ErrMsgInvalidMessage), gnet.Close
		}
	} else {
		log.Debugf("unmarshal client message error: %s, msgType: %d, cleintId: %d", err.Error(), msgTy, clientId)
		return genServerErrorMessage(engine.ErrMsgInvalidMessage), gnet.Close
	}
	if gameConn == nil {
		//log.Warnf("gate send to game, client id[%d] has no game connection, client message type: %d", clientId, msgTy)
		if msgTy != engine.ClientMsgTypeLogin {
			//直接断连接
			return genServerErrorMessage(engine.ErrMsgClientNotLogin), gnet.Close
		} else {
			return genServerErrorMessage(engine.ErrMsgServerNotReady), gnet.None
		}
	}
	return m.sendRpcToGame(gameConn, msgTy, clientId, data)
}

func (m *gameProxy) onClientClosed(c gnet.Conn) {
	clientId := getClientId(c)
	gameConns := make(map[*engine.TcpClient]bool)
	if games, ok := m.clientToGame[clientId]; ok {
		for _, game := range games {
			if _, find := gameConns[game]; !find {
				gameConns[game] = true
			}
		}
	}
	for conn := range gameConns {
		m.sendRpcToGame(conn, engine.ClientMsgTypeClose, clientId, nil)
	}
}

func (m *gameProxy) getGameConn(clientId engine.ConnectIdType, entityId engine.EntityIdType) *engine.TcpClient {
	if games, ok := m.clientToGame[clientId]; ok {
		return games[entityId]
	}
	return nil
}

func (m *gameProxy) getGameConnByName(name string) *engine.TcpClient {
	if info, ok := m.gameServers[name]; ok {
		return info.conn
	}
	return nil
}

//sayHello gate连接到game后向game同步自身信息
func (m *gameProxy) sayHello(gameConn *engine.TcpClient) {
	if gameConn == nil {
		return
	}
	msg := &message.SayHello{
		ServiceName: engine.ServiceName(),
	}
	head := engine.GenMessageHeader(engine.ServerMessageTypeSayHello, 0)
	if r, err := engine.GetProtocol().MessageWithHead(head, msg); err == nil {
		if _, err = gameConn.Send(r); err != nil {
			log.Warnf("say hello to game error: %s", err.Error())
		} else {
			log.Infof("say hello to game[%s], msg: %s", gameConn.RemoteAddr(), msg.String())
		}
	} else {
		log.Warnf("say hello to game encode error: %s", err.Error())
	}
}

func (m *gameProxy) sendHeartbeat(gameConn *engine.TcpClient, clientId engine.ConnectIdType) error {
	if gameConn == nil {
		return errors.New("game conn nil")
	}
	head := engine.GenMessageHeader(engine.ServerMessageTypeHeartBeat, clientId)
	if r, err := engine.GetProtocol().MessageWithHead(head, nil); err == nil {
		_, err = gameConn.Send(r)
		return err
	} else {
		log.Warnf("heartbeat to game encode error: %s", err.Error())
		return err
	}
}

func (m *gameProxy) getGameByLoad() *engine.TcpClient {
	//todo:根据负载选择一个game
	for _, info := range m.gameServers {
		if info.isStub == false && info.conn != nil {
			return info.conn
		}
	}
	return nil
}

//bindEntity 连接绑定到entity
func (m *gameProxy) bindEntity(clientId engine.ConnectIdType, entityId engine.EntityIdType, conn *engine.TcpClient) {
	if _, find := m.clientToGame[clientId]; !find {
		m.clientToGame[clientId] = make(map[engine.EntityIdType]*engine.TcpClient)
	}
	m.clientToGame[clientId][entityId] = conn
	log.Infof("client conn[%s:%d] bind entityId[%d] from game[%v] ", engine.ServiceName(), clientId, entityId, conn.Context())
}

//unBindEntity 连接解绑entity
func (m *gameProxy) unBindEntity(clientId engine.ConnectIdType, entityId engine.EntityIdType) {
	if games, find := m.clientToGame[clientId]; find {
		if game, ok := games[entityId]; ok {
			delete(games, entityId)
			log.Infof("client conn[%s:%d] unbind entityId[%d] from game[%v]", engine.ServiceName(), clientId, entityId, game.Context())
		}
	}
}

// genServerErrorMessage 生成提示客户端的错误信息
func genServerErrorMessage(msg string) []byte {
	r := map[string]interface{}{
		engine.ClientMsgDataFieldType: engine.ClientMsgTypeTips,
		engine.ClientMsgDataFieldArgs: []interface{}{msg},
	}
	buf, _ := engine.GetProtocol().Marshal(r)
	return buf
}

//getClientId 获取客户端连接的clientId
func getClientId(c gnet.Conn) engine.ConnectIdType {
	if c == nil {
		return 0
	}
	if ctx, ok := c.Context().(connCtxType); ok {
		return ctx[ctxKeyConnId]
	}
	return 0
}

//toServerMessageType 将客户端消息类型转换为服务器内部通信消息类型
func toServerMessageType(ty uint8) uint8 {
	switch ty {
	case engine.ClientMsgTypeLogin:
		return engine.ServerMessageTypeLogin
	case engine.ClientMsgTypeClose:
		return engine.ServerMessageTypeDisconnectClient
	case engine.ClientMsgTypeHeartBeat:
		return engine.ServerMessageTypeHeartBeat
	default:
		return engine.ServerMessageTypeEntityRpc
	}
}
