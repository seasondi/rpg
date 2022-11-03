package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientV3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/atomic"
	"strconv"
	"strings"
	"time"
)

func initEtcd() error {
	etcdMgr = new(etcd)
	if err := etcdMgr.init(); err != nil {
		return err
	}

	log.Infof("etcd inited. config: %v", cfg.Etcd)
	return nil
}

func etcdEndPoints() []string {
	return strings.Split(cfg.Etcd.EndPoints, ",")
}

func GetEtcd() *etcd {
	return etcdMgr
}

type etcdLeaseResult struct {
	lease   clientV3.Lease
	stopped atomic.Bool
}

func (m *etcdLeaseResult) Close() {
	m.stopped.Store(true)
	if m.lease != nil {
		_ = m.lease.Close()
		m.lease = nil
	}
}

type EtcdWatchHandle interface {
	Key() string
	OnUpdated(*EtcdKV)
	OnDelete(*EtcdKV)
}

type EtcdValue map[string]interface{}

func parseEtcdValue(v []byte) EtcdValue {
	r := EtcdValue{}
	_ = json.Unmarshal(v, &r)
	return r
}

type EtcdKV struct {
	key   string
	value EtcdValue
}

func (m *EtcdKV) Key() string {
	return m.key
}

func (m *EtcdKV) Value() EtcdValue {
	return m.value
}

func (m *EtcdKV) ValueJson() string {
	r, _ := json.Marshal(m.value)
	return string(r)
}

func NewEtcdKV(key string, value EtcdValue) *EtcdKV {
	return &EtcdKV{
		key:   key,
		value: value,
	}
}

type etcd struct {
	cli *clientV3.Client
}

func (m *etcd) init() error {
	var err error
	m.cli, err = clientV3.New(clientV3.Config{
		Endpoints:   etcdEndPoints(),
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return err
	}
	return m.registerServer()
}

func (m *etcd) close() {
	if m.cli != nil {
		_ = m.cli.Close()
		m.cli = nil
	}
}

func (m *etcd) Register(ctx context.Context, ttl int64, data *EtcdKV) (*etcdLeaseResult, error) {
	if data == nil {
		return nil, errors.New("etcd register data nil")
	}
	if r := m.Get(ctx, data.Key()); len(r) > 0 {
		return nil, fmt.Errorf("register to etcd but key[%s] already exist", data.Key())
	}
	log.Infof("try register to etcd, ttl: %d, info: %+v", ttl, *data)
	r, err := m.PutWithLease(ctx, clientV3.NewLease(m.cli), ttl, data)
	if err == nil {
		log.Infof("register to etcd success, ttl: %d, info: %+v", ttl, *data)
	}
	return r, err
}

func (m *etcd) PutWithLease(ctx context.Context, lease clientV3.Lease, ttl int64, data *EtcdKV) (*etcdLeaseResult, error) {
	rsp, err := lease.Grant(ctx, ttl)
	if err != nil {
		return nil, err
	} else {
		kv := clientV3.NewKV(m.cli)
		if _, err = kv.Put(ctx, data.Key(), data.ValueJson(), clientV3.WithLease(rsp.ID)); err != nil {
			return nil, err
		}
	}
	if keepChan, err := lease.KeepAlive(context.TODO(), rsp.ID); err != nil {
		return nil, err
	} else {
		result := &etcdLeaseResult{lease: lease}
		go m.autoKeepLease(ttl, data, keepChan, result)
		return result, nil
	}
}

func (m *etcd) registerWithLeaseResult(result *etcdLeaseResult, ttl int64, data *EtcdKV) error {
	result.stopped.Store(false)
	rsp, err := result.lease.Grant(context.TODO(), ttl)
	if err != nil {
		return err
	} else {
		kv := clientV3.NewKV(m.cli)
		if _, err = kv.Put(context.TODO(), data.Key(), data.ValueJson(), clientV3.WithLease(rsp.ID)); err != nil {
			return err
		}
	}
	if keepChan, err := result.lease.KeepAlive(context.TODO(), rsp.ID); err != nil {
		return err
	} else {
		go m.autoKeepLease(ttl, data, keepChan, result)
	}
	return nil
}

func (m *etcd) autoKeepLease(ttl int64, data *EtcdKV, keepChan <-chan *clientV3.LeaseKeepAliveResponse, result *etcdLeaseResult) {
Loop:
	for {
		select {
		case v := <-keepChan:
			if v == nil {
				if result.stopped.Load() == true {
					return
				} else {
					break Loop
				}
			}
		}
	}

	for {
		time.Sleep(time.Second)
		if err := m.registerWithLeaseResult(result, ttl, data); err == nil {
			log.Infof("reRegister to etcd success. data: %+v", data)
			return
		}
	}
}

func (m *etcd) Watch(handle EtcdWatchHandle, opts ...clientV3.OpOption) {
	log.Info("start watch key: ", handle.Key())
	watcher := clientV3.NewWatcher(m.cli)
	watcherChan := watcher.Watch(context.TODO(), handle.Key(), opts...)
	for watchRsp := range watcherChan {
		for _, ev := range watchRsp.Events {
			switch ev.Type {
			case mvccpb.PUT:
				handle.OnUpdated(NewEtcdKV(string(ev.Kv.Key), parseEtcdValue(ev.Kv.Value)))
			case mvccpb.DELETE:
				handle.OnDelete(NewEtcdKV(string(ev.Kv.Key), parseEtcdValue(ev.Kv.Value)))
			default:
				log.Warnf("etcd watch %s recevied unknown event %d", handle.Key(), ev.Type)
			}
		}
	}
	log.Info("stop watch key: ", handle.Key())
}

func (m *etcd) Get(ctx context.Context, key string, opts ...clientV3.OpOption) []EtcdKV {
	r := make([]EtcdKV, 0)
	rsp, err := m.cli.Get(ctx, key, opts...)
	if err != nil {
		log.Warnf("Get key[%s] from etcd error: %s", key, err.Error())
		return r
	}
	for _, kv := range rsp.Kvs {
		r = append(r, EtcdKV{key: string(kv.Key), value: parseEtcdValue(kv.Value)})
	}

	return r
}

func (m *etcd) Put(ctx context.Context, data *EtcdKV) error {
	log.Infof("put to etcd, info: %+v", *data)
	_, err := m.cli.Put(ctx, data.Key(), data.ValueJson())
	return err
}

func (m *etcd) Delete(ctx context.Context, key string, opts ...clientV3.OpOption) error {
	log.Infof("delete from etcd, key: %s", key)
	_, err := m.cli.Delete(ctx, key, opts...)
	if err != nil {
		return err
	}
	return nil
}

func (m *etcd) registerServer() error {
	ctx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
	defer cancel()

	val := EtcdValue{
		EtcdValueAddr:   GetConfig().ServerConfig().Addr,
		EtcdValueType:   EtcdTypeServer,
		EtcdValueIsStub: GetConfig().ServerConfig().IsStub,
	}
	kv := NewEtcdKV(ServiceName(), val)
	retryCount := 0
	for _, err := m.Register(ctx, EtcdServerLeaseTTL, kv); err != nil; _, err = m.Register(ctx, EtcdServerLeaseTTL, kv) {
		if strings.Contains(err.Error(), "already exist") {
			log.Infof("wait for server register to etcd")
			time.Sleep(time.Second)
			retryCount += 1
			if retryCount >= 5 {
				return err
			}
		} else {
			log.Errorf("register to etcd error: %s", err.Error())
			return err
		}
	}
	return nil
}

func GetEtcdPrefixWithServer(prefix string) string {
	if !strings.HasSuffix(prefix, ".") {
		prefix += "."
	}
	return fmt.Sprintf("%s%d.", prefix, GetConfig().ServerId)
}

//ParseEtcdServerKey 返回值: 前缀,服务器ID,服务器编号,err
func ParseEtcdServerKey(s string) (string, ServerIdType, ServerTagType, error) {
	r := strings.Split(s, ".")
	if len(r) != 3 {
		return "", 0, 0, fmt.Errorf("invalid server key: %s", s)
	}
	serverId, err := strconv.ParseInt(r[1], 10, 64)
	if err != nil {
		return "", 0, 0, err
	}
	serverTag, err := strconv.ParseInt(r[2], 10, 64)
	if err != nil {
		return "", 0, 0, err
	}
	return r[0] + ".", ServerIdType(serverId), ServerTagType(serverTag), nil
}

func GetEtcdStubKey(id EntityIdType) string {
	return fmt.Sprintf("%s%d.%d", StubPrefix, GetConfig().ServerId, id)
}

//ParseEtcdStubKey 返回值: 前缀,服务器ID,entityId,err
func ParseEtcdStubKey(s string) (string, ServerIdType, EntityIdType, error) {
	r := strings.Split(s, ".")
	if len(r) != 3 {
		return "", 0, 0, fmt.Errorf("invalid stub key: %s", s)
	}
	serverId, err := strconv.ParseInt(r[1], 10, 64)
	if err != nil {
		return "", 0, 0, err
	}
	entityId, err := strconv.ParseInt(r[2], 10, 64)
	if err != nil {
		return "", 0, 0, err
	}
	return r[0] + ".", ServerIdType(serverId), EntityIdType(entityId), nil
}

func GetEtcdEntityKey(id EntityIdType) string {
	return fmt.Sprintf("%s%d.%d", EntityPrefix, GetConfig().ServerId, id)
}

//ParseEtcdEntityKey 返回值: 前缀,服务器ID,entityId,err
func ParseEtcdEntityKey(s string) (string, ServerIdType, EntityIdType, error) {
	r := strings.Split(s, ".")
	if len(r) != 3 {
		return "", 0, 0, fmt.Errorf("invalid entity key: %s", s)
	}
	serverId, err := strconv.ParseInt(r[1], 10, 64)
	if err != nil {
		return "", 0, 0, err
	}
	entityId, err := strconv.ParseInt(r[2], 10, 64)
	if err != nil {
		return "", 0, 0, err
	}
	return r[0] + ".", ServerIdType(serverId), EntityIdType(entityId), nil
}
