package main

import (
	"errors"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/panjf2000/gnet"
	"go.mongodb.org/mongo-driver/bson"
	"reflect"
	"rpg/engine/engine"
	"rpg/engine/message"
	"time"
)

var loop *eventLoop

func getEventLoop() *eventLoop {
	if loop == nil {
		loop = new(eventLoop)
		loop.init()
	}
	return loop
}

type eventLoop struct {
	gnet.EventServer
	connMap map[string]gnet.Conn
}

func (m *eventLoop) init() {
	m.connMap = make(map[string]gnet.Conn)
}

func (m *eventLoop) OnInitComplete(server gnet.Server) (action gnet.Action) {
	log.Infof("DBManager[%s] server init complete, listen at: %s", engine.ServiceName(), server.Addr)
	if err := engine.GetEtcd().RegisterServer(); err != nil {
		log.Fatalf("register to etcd failed: %s", err.Error())
	}
	return gnet.None
}

func (m *eventLoop) OnShutdown(_ gnet.Server) {
	log.Infof("DBManager server shutdown")
}

func (m *eventLoop) OnOpened(c gnet.Conn) (out []byte, action gnet.Action) {
	log.Infof("conn[%s] opened", c.RemoteAddr())

	return nil, gnet.None
}

func (m *eventLoop) OnClosed(c gnet.Conn, err error) (action gnet.Action) {
	log.Infof("conn[%s] closed, msg: %v", c.RemoteAddr(), err)
	return gnet.None
}

func (m *eventLoop) React(frame []byte, c gnet.Conn) (out []byte, action gnet.Action) {
	var err error
	ty := frame[0]
	switch ty {
	case engine.ServerMessageTypeDBCommand:
		msg := message.DBCommandRequest{}
		if err := proto.Unmarshal(frame[1:], &msg); err != nil {
			log.Error("can not UnMarshal message, error: ", err.Error())
			return nil, gnet.Close
		}
		err = doDBCommand(c, &msg)
	default:
		err = errors.New("unknown message type")
	}
	if err != nil {
		log.Warnf("message type: %d, error: %s", ty, err.Error())
	}
	return nil, gnet.None
}

func (m *eventLoop) Tick() (delay time.Duration, action gnet.Action) {
	engine.Tick()
	return engine.ServerTick, gnet.None
}

func doDBCommand(c gnet.Conn, in *message.DBCommandRequest) error {
	request := commandTaskRequester{
		id:    engine.EntityIdType(in.EntityId),
		conn:  c,
		extra: in.GetEx(),
	}
	filter := bson.D{}
	if err := bson.Unmarshal(in.Filter, &filter); err != nil {
		return err
	}
	data := bson.M{}
	if in.Data != nil {
		if err := bson.Unmarshal(in.Data, &data); err != nil {
			return err
		}
	}
	task := newTask(engine.DBTaskType(in.TaskType), &request, &commandTaskInfo{
		dbType:     engine.DBType(in.DbType),
		database:   in.Database,
		collection: in.Collection,
		filter:     filter,
		data:       data,
	})
	dbMgr.TaskMgr.AddTask(request.id, task)
	return nil
}

func responseCommandTask(requester *commandTaskRequester, taskType engine.DBTaskType, err error, data interface{}) {
	//不需要回包的请求
	if requester.extra == nil || requester.extra.Uuid == "" {
		return
	}
	if requester.conn == nil {
		log.Tracef("response db task type: %d, entityId: %d but conn is nil", taskType, requester.id)
		return
	}
	entityId := requester.id

	var result []byte
	if data != nil {
		switch r := data.(type) {
		case bson.M:
			result, err = engine.GetProtocol().Marshal(r)
		case []bson.M:
			result, err = engine.GetProtocol().Marshal(r)
		case map[string]interface{}:
			result, err = engine.GetProtocol().Marshal(r)
		case []map[string]interface{}:
			result, err = engine.GetProtocol().Marshal(r)
		default:
			err = fmt.Errorf("unsupport data type %s", reflect.TypeOf(data).String())
		}
		if err != nil {
			log.Errorf("response db task type: %d, entityId: %d error: %s", taskType, entityId, err.Error())
			return
		}
	}

	msg := &message.DBCommandResponse{
		TaskType: uint32(taskType),
		EntityId: int64(entityId),
		Data:     result,
	}
	if requester.extra != nil {
		msg.Ex = requester.extra
	}
	if err != nil {
		msg.ErrMsg = []byte(err.Error())
	}
	if buf, err := engine.GetProtocol().MessageWithHead([]byte{engine.ServerMessageTypeDBCommand}, msg); err != nil {
		log.Warnf("entityId[%d] taskType[%d] taskInfo Pack error: %s", entityId, taskType, err.Error())
		return
	} else {
		_ = requester.conn.AsyncWrite(buf)
	}
}
