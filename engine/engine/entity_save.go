package engine

import "container/list"

var entitySaveMgr *EntitySaveManager

type EntitySaveInfo struct {
	EntityId     EntityIdType //玩家ID
	Data         []byte       //存盘信息
	NeedResponse bool         //是否需要数据库操作完成回调
}

type EntitySaveManager struct {
	entityInfo map[EntityIdType]*list.Element
	saveList   *list.List
}

func GetEntitySaveManager() *EntitySaveManager {
	if entitySaveMgr == nil {
		entitySaveMgr = new(EntitySaveManager)
		entitySaveMgr.init()
	}
	return entitySaveMgr
}

func (m *EntitySaveManager) init() {
	m.entityInfo = make(map[EntityIdType]*list.Element)
	m.saveList = list.New()
}

func (m *EntitySaveManager) Add(save *EntitySaveInfo) {
	if save != nil {
		el := m.saveList.PushBack(save)
		m.entityInfo[save.EntityId] = el
	}
}

func (m *EntitySaveManager) Remove(entityId EntityIdType) {
	if el, ok := m.entityInfo[entityId]; ok {
		m.saveList.Remove(el)
		delete(m.entityInfo, entityId)
	}
}

func (m *EntitySaveManager) Get(n int) []*EntitySaveInfo {
	result := make([]*EntitySaveInfo, 0)
	if n <= 0 {
		return result
	}

	count := 0
	for e := m.saveList.Front(); e != nil; e = e.Next() {
		result = append(result, e.Value.(*EntitySaveInfo))
		count++
		if count >= n {
			break
		}
	}

	return result
}

func (m *EntitySaveManager) Length() int {
	return m.saveList.Len()
}
