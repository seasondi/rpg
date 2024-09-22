package main

import "rpg/engine/engine/LockFree"

var taskMgr *TaskManager

type TaskManager struct {
	tasks *LockFree.TaskQueue
}

func initTaskManager() {
	if taskMgr == nil {
		taskMgr = &TaskManager{
			tasks: LockFree.NewTaskQueue(),
		}
	}
}

func getTaskManager() *TaskManager {
	return taskMgr
}

func (m *TaskManager) Push(task LockFree.ITaskHandler) {
	m.tasks.Enqueue(task)
}

func (m *TaskManager) Len() int {
	return m.tasks.Len()
}

func (m *TaskManager) Tick() {
	tickTasks := make([]LockFree.ITaskHandler, 0)

	for {
		t := m.tasks.Dequeue()
		if t == nil {
			break
		}
		tickTasks = append(tickTasks, t)
	}

	for _, t := range tickTasks {
		if err := t.HandleTask(); err != nil {
			log.Warnf("handle task err: %v", err)
		}
	}
}
