package main

import (
	"rpg/engine/engine"
	"container/list"
	"fmt"
	"sync"
	"time"
)

func newBufferedTasks() (*bufferedTasks, error) {
	bt := &bufferedTasks{}
	if err := bt.Init(); err != nil {
		return nil, err
	}
	return bt, nil
}

type bufferedTasks struct {
	mutex          sync.Mutex
	entityTasksMap map[engine.EntityIdType]*list.List
}

func (m *bufferedTasks) Init() error {
	m.entityTasksMap = make(map[engine.EntityIdType]*list.List)
	return nil
}

func (m *bufferedTasks) AddTask(entityId engine.EntityIdType, t Task) {
	m.mutex.Lock()
	if _, ok := m.entityTasksMap[entityId]; !ok {
		m.entityTasksMap[entityId] = list.New()
	}
	m.entityTasksMap[entityId].PushBack(t)
	m.mutex.Unlock()

	m.ProcessTask(entityId)
}

func (m *bufferedTasks) PopTask(entityId engine.EntityIdType) Task {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if tasks, ok := m.entityTasksMap[entityId]; ok {
		if task := tasks.Front(); task != nil {
			t := tasks.Remove(task)
			return t.(Task)
		}
	}
	return nil
}

func (m *bufferedTasks) ProcessTask(entityId engine.EntityIdType) {
	task := m.GetTask(entityId)
	if task == nil {
		return
	}

	if err := dbMgr.TaskPool.SubmitTask(task); err != nil {
		retryTime := time.Second
		log.Warnf("taskPool submit task failed[%s], retry after %s", err.Error(), retryTime.String())
		engine.GetTimer().AddTimer(retryTime, 0, m.onReSubmitTask, entityId)
		return
	} else {
		log.Tracef("entity[%d] task[%s] has submitted", entityId, task.Name())
	}
}

func (m *bufferedTasks) onReSubmitTask(params ...interface{}) {
	m.ProcessTask(params[0].(engine.EntityIdType))
}

func (m *bufferedTasks) FinishProcessTask(entityId engine.EntityIdType) {
	m.PopTask(entityId)
	m.ProcessTask(entityId)
}

func (m *bufferedTasks) GetTask(entityId engine.EntityIdType) Task {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if tasks, ok := m.entityTasksMap[entityId]; ok {
		f := tasks.Front()
		if f == nil {
			return nil
		}
		task := f.Value.(Task)
		return task
	}
	return nil
}

func (m *bufferedTasks) TaskSize() uint32 {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	count := uint32(0)
	for _, tasks := range m.entityTasksMap {
		count += uint32(tasks.Len())
	}
	return count
}

func (m *bufferedTasks) HasEntityTask(entityId engine.EntityIdType) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if tasks, ok := m.entityTasksMap[entityId]; ok {
		length := tasks.Len()
		return length > 0
	}
	return false
}

func (m *bufferedTasks) HasTask() bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return len(m.entityTasksMap) > 0
}

func (m *bufferedTasks) GetTasksInfo() string {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	ret := ""
	for entId, tasks := range m.entityTasksMap {
		taskStr := "["
		for task := tasks.Front(); task != nil; task = task.Next() {
			taskStr += task.Value.(Task).Name() + ", "
		}
		taskStr += "]"
		ret += fmt.Sprintf("[entity:%d](TaskNum:%d)%s, ", entId, tasks.Len(), taskStr)
	}

	return ret
}

func (m *bufferedTasks) GetAllEntityIds() []engine.EntityIdType {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	r := make([]engine.EntityIdType, 0, len(m.entityTasksMap))
	for id := range m.entityTasksMap {
		r = append(r, id)
	}

	return r
}
