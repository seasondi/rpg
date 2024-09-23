package main

import (
	"github.com/panjf2000/gnet"
	"go.mongodb.org/mongo-driver/mongo/options"
	"rpg/engine/engine"
	"rpg/engine/message"
)

type Task interface {
	Name() string
	Process()
	OnTaskFinished(data interface{}, err error)
}

type commandTaskInfo struct {
	dbType     engine.DBType
	database   string
	collection string
	filter     interface{}
	data       interface{}
}

type commandTaskRequester struct {
	id    engine.EntityIdType
	conn  gnet.Conn
	extra *message.ExtraInfo
}

func newTask(ty engine.DBTaskType, requester *commandTaskRequester, taskInfo *commandTaskInfo) Task {
	switch ty {
	case engine.DBTaskTypeQueryOne:
		return &commandQueryOneTask{requester: requester, taskInfo: taskInfo}
	case engine.DBTaskTypeQueryMany:
		return &commandQueryManyTask{requester: requester, taskInfo: taskInfo}
	case engine.DBTaskTypeUpdateOne:
		return &commandUpdateOneTask{requester: requester, taskInfo: taskInfo}
	case engine.DBTaskTypeReplaceOne:
		return &commandReplaceOneTask{requester: requester, taskInfo: taskInfo}
	case engine.DBTaskTypeDeleteOne:
		return &commandDeleteOneTask{requester: requester, taskInfo: taskInfo}
	case engine.DBTaskTypeDeleteMany:
		return &commandDeleteManyTask{requester: requester, taskInfo: taskInfo}
	}

	return nil
}

// ================================请求单条数据============================================

type commandQueryOneTask struct {
	requester *commandTaskRequester
	taskInfo  *commandTaskInfo
}

func (m *commandQueryOneTask) Name() string {
	return "commandQueryOneTask"
}

func (m *commandQueryOneTask) Process() {
	data, err := dbMgr.GetDB(m.taskInfo.dbType).FindOne(m.taskInfo.database, m.taskInfo.collection, m.taskInfo.filter)
	m.OnTaskFinished(data, err)
}

func (m *commandQueryOneTask) OnTaskFinished(data interface{}, err error) {
	dbMgr.TaskMgr.FinishProcessTask(m.requester.id)
	if err != nil {
		log.Warnf("Process task[%s] error: %s, key: %v", m.Name(), err.Error(), m.taskInfo.filter)
	}
	responseCommandTask(m.requester, engine.DBTaskTypeQueryOne, err, data)
}

// ================================请求多条数据============================================

type commandQueryManyTask struct {
	requester *commandTaskRequester
	taskInfo  *commandTaskInfo
}

func (m *commandQueryManyTask) Name() string {
	return "commandQueryManyTask"
}

func (m *commandQueryManyTask) Process() {
	data, err := dbMgr.GetDB(m.taskInfo.dbType).FindMany(m.taskInfo.database, m.taskInfo.collection, m.taskInfo.filter, nil)
	m.OnTaskFinished(data, err)
}

func (m *commandQueryManyTask) OnTaskFinished(data interface{}, err error) {
	dbMgr.TaskMgr.FinishProcessTask(m.requester.id)
	if err != nil {
		log.Warnf("Process task[%s] error: %s, key: %v", m.Name(), err.Error(), m.taskInfo.filter)
	}
	responseCommandTask(m.requester, engine.DBTaskTypeQueryMany, err, data)
}

// ================================更新单条数据============================================

type commandUpdateOneTask struct {
	requester *commandTaskRequester
	taskInfo  *commandTaskInfo
}

func (m *commandUpdateOneTask) Name() string {
	return "commandUpdateOneTask"
}

func (m *commandUpdateOneTask) Process() {
	err := dbMgr.GetDB(m.taskInfo.dbType).UpdateOne(m.taskInfo.database, m.taskInfo.collection, m.taskInfo.filter, m.taskInfo.data, options.Update().SetUpsert(true))
	m.OnTaskFinished(nil, err)
}

func (m *commandUpdateOneTask) OnTaskFinished(data interface{}, err error) {
	dbMgr.TaskMgr.FinishProcessTask(m.requester.id)
	if err != nil {
		log.Warnf("Process task[%s] error: %s, key: %v", m.Name(), err.Error(), m.taskInfo.filter)
	}
	responseCommandTask(m.requester, engine.DBTaskTypeUpdateOne, err, data)
}

// ================================替换单条数据============================================

type commandReplaceOneTask struct {
	requester *commandTaskRequester
	taskInfo  *commandTaskInfo
}

func (m *commandReplaceOneTask) Name() string {
	return "commandReplaceOneTask"
}

func (m *commandReplaceOneTask) Process() {
	err := dbMgr.GetDB(m.taskInfo.dbType).ReplaceOne(m.taskInfo.database, m.taskInfo.collection, m.taskInfo.filter, m.taskInfo.data, options.Replace().SetUpsert(true))
	m.OnTaskFinished(nil, err)
}

func (m *commandReplaceOneTask) OnTaskFinished(data interface{}, err error) {
	dbMgr.TaskMgr.FinishProcessTask(m.requester.id)
	if err != nil {
		log.Warnf("Process task[%s] error: %s, key: %v", m.Name(), err.Error(), m.taskInfo.filter)
	}
	responseCommandTask(m.requester, engine.DBTaskTypeReplaceOne, err, data)
}

// ================================删除单条数据============================================

type commandDeleteOneTask struct {
	requester *commandTaskRequester
	taskInfo  *commandTaskInfo
}

func (m *commandDeleteOneTask) Name() string {
	return "commandDeleteOneTask"
}

func (m *commandDeleteOneTask) Process() {
	err := dbMgr.GetDB(m.taskInfo.dbType).DeleteOne(m.taskInfo.database, m.taskInfo.collection, m.taskInfo.filter)
	m.OnTaskFinished(nil, err)
}

func (m *commandDeleteOneTask) OnTaskFinished(data interface{}, err error) {
	dbMgr.TaskMgr.FinishProcessTask(m.requester.id)
	if err != nil {
		log.Warnf("Process task[%s] error: %s, key: %v", m.Name(), err.Error(), m.taskInfo.filter)
	}
	responseCommandTask(m.requester, engine.DBTaskTypeDeleteOne, err, data)
}

// ================================删除多条数据============================================

type commandDeleteManyTask struct {
	requester *commandTaskRequester
	taskInfo  *commandTaskInfo
}

func (m *commandDeleteManyTask) Name() string {
	return "commandDeleteManyTask"
}

func (m *commandDeleteManyTask) Process() {
	err := dbMgr.GetDB(m.taskInfo.dbType).DeleteMany(m.taskInfo.database, m.taskInfo.collection, m.taskInfo.filter)
	m.OnTaskFinished(nil, err)
}

func (m *commandDeleteManyTask) OnTaskFinished(data interface{}, err error) {
	dbMgr.TaskMgr.FinishProcessTask(m.requester.id)
	if err != nil {
		log.Warnf("Process task[%s] error: %s, key: %v", m.Name(), err.Error(), m.taskInfo.filter)
	}
	responseCommandTask(m.requester, engine.DBTaskTypeDeleteMany, err, data)
}
