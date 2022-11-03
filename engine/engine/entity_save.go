package engine

import (
	"errors"
	"time"
)

const (
	entitySaveIDRange = 1000000
)

var entitySaveIdMap = map[EntityIdType]uint64{}

type EntitySaveInfo struct {
	EntityId       EntityIdType //玩家ID
	Data           []byte       //存盘信息
	PreferCallback bool         //是否需要数据库操作完成回调
	SaveID         uint64       //存盘ID
}

func nextSaveID(entityId EntityIdType) (uint64, error) {
	if _, ok := entitySaveIdMap[entityId]; !ok {
		return 0, errors.New("can not save")
	}
	entitySaveIdMap[entityId] += 1
	return entitySaveIdMap[entityId], nil
}

func initEntitySaveID(entityId EntityIdType) {
	entitySaveIdMap[entityId] = uint64(time.Now().Unix() * entitySaveIDRange)
}

func clearEntitySaveID(entityId EntityIdType) {
	delete(entitySaveIdMap, entityId)
}

func IsValidSaveID(entityId EntityIdType, saveId uint64) bool {
	current := entitySaveIdMap[entityId]
	if current/entitySaveIDRange != saveId/entitySaveIDRange {
		return false
	}
	return current == saveId
}
