package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/panjf2000/gnet"
	clientV3 "go.etcd.io/etcd/client/v3"
	"math/rand"
	"rpg/engine/engine"
	"rpg/engine/message"
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
	log.Trace("on process game message, msgId: ", msgId, ", bufLen: ", len(buf), ", clientId: ", clientId, ", err: ", err)
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
	conn        *engine.TcpClient
	isStub      bool
	isEntryStub bool
	load        *engine.GameLoadInfo
}

type entityServer struct {
	entityId   engine.EntityIdType
	serverName string
}

type gameProxy struct {
	gameServers  map[string]*serverInfo                 //game server name -> serverInfo
	clientToGame map[engine.ConnectIdType]*entityServer //clientConnectId -> entityIdInfo
	entryStub    *stubInfo                              //登录入口stub
}

func (m *gameProxy) init() {
	m.gameServers = make(map[string]*serverInfo)
	m.clientToGame = make(map[engine.ConnectIdType]*entityServer)
}

func (m *gameProxy) setEntryStub(stub *stubInfo) {
	m.entryStub = stub
}

func (m *gameProxy) getEntryStub() *stubInfo {
	return m.entryStub
}

func (m *gameProxy) HandleUpdateGame(key string, value engine.EtcdValue) {
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
		m.gameServers[key] = &serverInfo{
			conn:   gameConn,
			isStub: isStub,
		}
		gameConn.Connect(addr, true)
	}
}

func (m *gameProxy) HandleDeleteGame(key string) {
	if game, find := m.gameServers[key]; find {
		game.conn.Disconnect()
		delete(m.gameServers, key)
	}
}

func (m *gameProxy) HandleUpdateStub(key string, value engine.EtcdValue) {
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

	if name, ok := value[engine.EtcdStubValueEntry].(string); ok && name == stubName {
		si := &stubInfo{serverName: serverName, entityId: entityId}
		m.setEntryStub(si)
	}
}

func (m *gameProxy) HandleDeleteStub(key string) {
	prefix, serverId, entityId, err := engine.ParseEtcdStubKey(key)
	if err != nil {
		log.Warnf("parse etcd stub key failed: %s, key: %s", err.Error(), key)
		return
	}
	if prefix != engine.StubPrefix || serverId != engine.GetConfig().ServerId {
		return
	}

	stub := m.getEntryStub()
	if stub.entityId == entityId {
		m.setEntryStub(nil)
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

func (m *gameProxy) Tick() {
	for _, info := range m.gameServers {
		info.conn.Tick()
	}
}

func (m *gameProxy) Disconnect() {
	for _, info := range m.gameServers {
		info.conn.Disconnect()
	}
}

// sendRpcToGame 发送entity rpc消息到指定game
func (m *gameProxy) sendRpcToGame(game *engine.TcpClient, msgTy uint8, clientId engine.ConnectIdType, data []byte) ([]byte, gnet.Action) {
	pb := &message.GameEntityRpc{
		Data:   data,
		Source: engine.ServiceName(),
	}
	svrMsgType := toServerMessageType(msgTy)
	head := engine.GenMessageHeader(svrMsgType, clientId)
	if buf, err := engine.GetProtocol().MessageWithHead(head, pb); err == nil {
		if n, gErr := game.Send(buf); gErr != nil {
			log.Warnf("send %d bytes to %s, bufLen: %d failed: %s", n, game.Context(), len(buf), gErr.Error())
		} else {
			log.Tracef("send %d bytes to %s, bufLen: %d", n, game.Context(), len(buf))
		}
	} else {
		log.Warnf("send rpc to %s encode error: %s", game.Context(), err.Error())
	}

	return nil, gnet.None
}

// sendProtoToGameByName 内部消息发往指定serviceName的game
func (m *gameProxy) sendProtoToGameByName(gameName string, ty uint8, msg proto.Message) error {
	if conn := getGameProxy().getGameConnByName(gameName); conn != nil {
		return m.sendProtoToGame(conn, ty, msg)
	}
	return fmt.Errorf("no game named %s", gameName)
}

// sendProtoToGame 内部消息发往指定连接的game
func (m *gameProxy) sendProtoToGame(gameConn *engine.TcpClient, ty uint8, msg proto.Message) error {
	head := engine.GenMessageHeader(ty, 0)
	if w, err := engine.GetProtocol().MessageWithHead(head, msg); err == nil {
		if n, gErr := gameConn.Send(w); gErr != nil {
			log.Warnf("send %d bytes to %s, bufLen: %d failed: %s", n, gameConn.Context(), len(w), gErr.Error())
		} else {
			log.Tracef("send %d bytes to game %s, bufLen: %d", n, gameConn.Context(), len(w))
		}
		return err
	} else {
		return err
	}
}

// ClientSendToGame 客户端消息发往game
func (m *gameProxy) ClientSendToGame(client gnet.Conn, msgTy uint8, data []byte) ([]byte, gnet.Action) {
	log.Tracef("received message from client %s type: %d, data %+v", client.RemoteAddr(), msgTy, data)
	clientId := getClientId(client)
	if clientId == 0 {
		log.Warn("gate send to game invalid client connection")
		return genServerErrorMessage(engine.ErrMsgClientConnectionInvalid), gnet.None
	}

	//这里只检查非rpc消息
	if busy := getClientProxy().updateActive(clientId, msgTy); busy {
		log.Warnf("ignore too busy message, clientId: %d, msgType: %d", clientId, msgTy)
		return nil, gnet.None
	}

	var gameConn *engine.TcpClient
	switch msgTy {
	case engine.ClientMsgTypeLogin:
		gameConn = m.getEntryGame()
	case engine.ClientMsgTypeHeartBeat:
		if entityId := m.getBindEntity(clientId); entityId == 0 {
			responseHeartBeatToClient(client)
			return nil, gnet.None
		} else {
			gameConn = m.getGameConn(clientId)
			if gameConn == nil || gameConn.IsDisconnected() {
				responseHeartBeatToClient(client)
				return nil, gnet.None
			}
		}
	default:
		gameConn = m.getGameConn(clientId)
		if msgTy == engine.ClientMsgTypeEntityRpc {
			if clientData, err := engine.GetProtocol().UnMarshal(data); err == nil {
				//rpc metrics
				if args, ok := clientData[engine.ClientMsgDataFieldArgs].([]interface{}); ok {
					if name, ok := args[0].(string); ok {
						if busyCount := getClientProxy().rpcMetrics(clientId, name); busyCount >= 3 {
							return genServerErrorMessage(engine.ErrMsgTooBusy), gnet.None
						}
					}
				}
			} else {
				log.Warnf("unmarshal client message error: %s, msgType: %d, cleintId: %d, data: %x", err.Error(), msgTy, clientId, data)
				return genServerErrorMessage(engine.ErrMsgInvalidMessage), gnet.Close
			}
		}
	}

	if gameConn == nil || gameConn.IsDisconnected() {
		log.Warnf("gate send to game, client id[%d] has no game connection[%v], message type: %d", clientId, gameConn, msgTy)
		return genServerErrorMessage(engine.ErrMsgServerNotReady), gnet.None
	}
	return m.sendRpcToGame(gameConn, msgTy, clientId, data)
}

func (m *gameProxy) onClientClosed(clientId engine.ConnectIdType) {
	if server, ok := m.clientToGame[clientId]; ok {
		if conn := m.getGameServer(server.serverName); conn != nil {
			m.sendRpcToGame(conn, engine.ClientMsgTypeClose, clientId, nil)
		}
	}
}

func (m *gameProxy) getGameConn(clientId engine.ConnectIdType) *engine.TcpClient {
	if server, ok := m.clientToGame[clientId]; ok {
		return m.getGameServer(server.serverName)
	}
	return nil
}

func (m *gameProxy) getGameConnByName(name string) *engine.TcpClient {
	if info, ok := m.gameServers[name]; ok {
		return info.conn
	}
	return nil
}

// sayHello gate连接到game后向game同步自身信息
func (m *gameProxy) sayHello(gameConn *engine.TcpClient) {
	if gameConn == nil {
		return
	}
	msg := &message.SayHello{
		ServiceName: engine.ServiceName(),
		Inner:       engine.GetConfig().Server.IsInner,
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

func (m *gameProxy) getEntryGame() *engine.TcpClient {
	stub := m.getEntryStub()
	if stub == nil {
		return nil
	}
	return m.getGameConnByName(stub.serverName)
}

func (m *gameProxy) getGameByLoad() *engine.TcpClient {
	now := time.Now()
	var minGameLoad *engine.GameLoadInfo
	for _, info := range m.gameServers {
		if info.isStub == false && info.load != nil {
			if info.load.Time.IsZero() || now.Sub(info.load.Time) > time.Minute {
				continue
			}
			if minGameLoad == nil {
				minGameLoad = info.load
			} else {
				if minGameLoad.EntityCount < info.load.EntityCount {
					minGameLoad = info.load
				} else if minGameLoad.EntityCount == info.load.EntityCount {
					if minGameLoad.Time.After(info.load.Time) {
						minGameLoad = info.load
					}
				}
			}
		}
	}
	if minGameLoad == nil {
		return m.choseRandomGame()
	}
	return m.getGameConnByName(minGameLoad.Name)
}

func (m *gameProxy) choseRandomGame() *engine.TcpClient {
	if len(m.gameServers) == 0 {
		return nil
	}

	names := make([]string, 0)
	for name := range m.gameServers {
		names = append(names, name)
	}
	idx := rand.Intn(len(names))
	return m.getGameConnByName(names[idx])
}

func (m *gameProxy) updateGameLoadInfo() {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	if r, err := engine.GetRedisMgr().HGetAll(ctx, engine.RedisGameLoadKey()); err != nil {
		log.Warnf("update game load info, get from redis error: %s", err.Error())
	} else {
		for name, v := range r {
			data := &engine.GameLoadInfo{}
			if err = json.Unmarshal([]byte(v), &data); err == nil {
				if server, ok := m.gameServers[name]; ok {
					server.load = data
				}
			}
		}
	}
}

func (m *gameProxy) getBindEntity(clientId engine.ConnectIdType) engine.EntityIdType {
	if info, find := m.clientToGame[clientId]; find {
		return info.entityId
	}
	return 0
}

// bindEntity 连接绑定到entity
func (m *gameProxy) bindEntity(clientId engine.ConnectIdType, entityId engine.EntityIdType, conn *engine.TcpClient) {
	name, _ := conn.Context().(string)
	m.clientToGame[clientId] = &entityServer{
		entityId:   entityId,
		serverName: name,
	}
	log.Infof("client conn[%s:%d] bind entityId[%d] from game[%v] ", engine.ServiceName(), clientId, entityId, conn.Context())
	getClientProxy().setBindEntity(clientId, entityId)
	getClientProxy().stopActiveCheck(clientId)
}

// unBindEntity 连接解绑entity
func (m *gameProxy) unBindEntity(clientId engine.ConnectIdType, entityId engine.EntityIdType) {
	if _, find := m.clientToGame[clientId]; find {
		delete(m.clientToGame, clientId)
		log.Infof("client conn[%s:%d] unbind entityId[%d]", engine.ServiceName(), clientId, entityId)
	}
}

func (m *gameProxy) getGameServer(name string) *engine.TcpClient {
	if c, find := m.gameServers[name]; find {
		return c.conn
	}
	return nil
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

// getClientId 获取客户端连接的clientId
func getClientId(c gnet.Conn) engine.ConnectIdType {
	if c == nil {
		return 0
	}
	if ctx, ok := c.Context().(connCtxType); ok {
		return ctx[ctxKeyConnId]
	}
	return 0
}

// toServerMessageType 将客户端消息类型转换为服务器内部通信消息类型
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
