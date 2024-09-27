package main

import (
	"context"
	clientV3 "go.etcd.io/etcd/client/v3"
	"rpg/engine/engine"
	"time"
)

var stubProxy *StubProxy

func getStubProxy() *StubProxy {
	if stubProxy == nil {
		stubProxy = new(StubProxy)
		stubProxy.init()
	}
	return stubProxy
}

type StubProxy struct {
	stubs map[string]engine.EntityIdType //name -> id
}

func (m *StubProxy) init() {
	m.stubs = make(map[string]engine.EntityIdType)
}

func (m *StubProxy) GetStubId(name string) engine.EntityIdType {
	if id, find := m.stubs[name]; find {
		return id
	}
	return engine.EntityIdType(0)
}

func (m *StubProxy) AddStub(name string, entityId engine.EntityIdType) {
	m.stubs[name] = entityId
}

func (m *StubProxy) RemoveStub(entityId engine.EntityIdType) {
	for name, stubEntityId := range m.stubs {
		if stubEntityId == entityId {
			delete(m.stubs, name)
			return
		}
	}
}

func (m *StubProxy) HandleUpdate(key string, value engine.EtcdValue) {
	prefix, serverId, entityId, err := engine.ParseEtcdStubKey(key)
	if err != nil {
		log.Warnf("parse stub key failed: %s, key: %s", err.Error(), key)
		return
	}
	if prefix != engine.StubPrefix || serverId != engine.GetConfig().ServerId {
		return
	}
	name, ok := value[engine.EtcdValueName].(string)
	if !ok {
		log.Warn("invalid stub name, value is: ", value)
		return
	}
	getTaskManager().Push(&AddStubTask{name: name, entityId: entityId})
}

func (m *StubProxy) HandleDelete(key string) {
	prefix, serverId, entityId, err := engine.ParseEtcdStubKey(key)
	if err != nil {
		log.Warnf("parse stub key failed: %s, key: %s", err.Error(), key)
		return
	}
	if prefix != engine.StubPrefix || serverId != engine.GetConfig().ServerId {
		return
	}

	getTaskManager().Push(&RemoveStubTask{entityId: entityId})
}

func syncStubFromEtcd() {
	ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
	defer cancel()
	prefix := engine.GetEtcdPrefixWithServer(engine.StubPrefix)
	for _, kv := range engine.GetEtcd().Get(ctx, prefix, clientV3.WithPrefix()) {
		getStubProxy().HandleUpdate(kv.Key(), kv.Value())
	}
	go engine.GetEtcd().Watch(&etcdWatcher{watcherKey: prefix}, clientV3.WithPrefix())
}
