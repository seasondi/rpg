package main

import (
	"rpg/engine/engine"
	"strings"
)

type etcdWatcher struct {
	watcherKey string
}

func (m *etcdWatcher) Key() string {
	return m.watcherKey
}

func (m *etcdWatcher) OnUpdated(kv *engine.EtcdKV) {
	log.Info("etcd key update: ", kv)
	key := kv.Key()
	if strings.HasPrefix(key, engine.StubPrefix) {
		getStubProxy().HandleUpdate(kv.Key(), kv.Value())
	}
}

func (m *etcdWatcher) OnDelete(kv *engine.EtcdKV) {
	log.Info("etcd key delete: ", kv)
	key := kv.Key()
	if strings.HasPrefix(key, engine.StubPrefix) {
		getStubProxy().HandleDelete(kv.Key())
	}
}
