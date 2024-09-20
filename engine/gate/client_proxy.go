package main

import (
	"fmt"
	"github.com/panjf2000/gnet"
	"rpg/engine/engine"
	"strconv"
	"sync"
	"time"
)

const (
	ctxKeyConnId = "connId"
)

type connCtxType map[string]engine.ConnectIdType

var clientConnectId engine.ConnectIdType = 0 //客户端连接ID
var clientProxy *ClientProxy

type ClientMetricsActive struct {
	createTime     time.Time
	lastActiveTime time.Time
	activeTimerId  int64
	bindEntityId   engine.EntityIdType
	bindEntityTime time.Time
}

func (m *ClientMetricsActive) toString() string {
	return fmt.Sprintf("createTime: %s, lastActiveTime: %s, activeTimerId:%d, bindEntityId: %d, bindEntityTime: %s",
		m.createTime.Format(time.RFC3339), m.lastActiveTime.Format(time.RFC3339), m.activeTimerId, m.bindEntityId, m.bindEntityTime.Format(time.RFC3339))
}

type ClientMetricsRpc struct {
	rpcName       string
	callCount     int       //总调用次数
	lastCallTime  time.Time //上次调用时间
	busyCallCount int       //连续频繁调用次数
}

func (m *ClientMetricsRpc) toString() string {
	return fmt.Sprintf("rpcName: %s, callCount: %d, lastCallTime: %s", m.rpcName, m.callCount, m.lastCallTime.Format(time.RFC3339))
}

var messageIdToName = map[uint8]string{
	2: "RPC",
	4: "LOGIN",
	8: "HEARTBEAT",
}

type ClientMetricMsg struct {
	msgId        uint8
	callCount    int
	lastCallTime time.Time
}

func (m *ClientMetricMsg) toString() string {
	name := ""
	if messageName, ok := messageIdToName[m.msgId]; ok {
		name = messageName
	} else {
		name = strconv.FormatInt(int64(m.msgId), 10)
	}
	return fmt.Sprintf("msg: %s, callCount: %d, lastCallTime: %s", name, m.callCount, m.lastCallTime.Format(time.RFC3339))
}

type ClientMetrics struct {
	rpc    map[string]*ClientMetricsRpc
	active *ClientMetricsActive
	msg    map[uint8]*ClientMetricMsg
}

type Metrics struct {
	clientId engine.ConnectIdType
	metrics  *ClientMetrics
}

func newMetrics(clientId engine.ConnectIdType) *Metrics {
	return &Metrics{
		clientId: clientId,
		metrics: &ClientMetrics{
			rpc: make(map[string]*ClientMetricsRpc),
			active: &ClientMetricsActive{
				createTime:     time.Now(),
				lastActiveTime: time.Now(),
				activeTimerId:  0,
			},
			msg: make(map[uint8]*ClientMetricMsg),
		},
	}
}

func getClientProxy() *ClientProxy {
	if clientProxy == nil {
		clientProxy = new(ClientProxy)
		clientProxy.init()
	}
	return clientProxy
}

type ClientProxy struct {
	clientMutex sync.Mutex
	clientMap   map[engine.ConnectIdType]gnet.Conn //clientConnectId -> conn

	metricsMutex sync.Mutex
	metricMap    map[engine.ConnectIdType]*Metrics
}

func (m *ClientProxy) init() {
	m.clientMap = make(map[engine.ConnectIdType]gnet.Conn)
	m.metricMap = make(map[engine.ConnectIdType]*Metrics)
}

func (m *ClientProxy) addConn(c gnet.Conn) {
	clientConnectId += 1
	ctxMap := connCtxType{
		ctxKeyConnId: clientConnectId,
	}
	c.SetContext(ctxMap)

	m.clientMutex.Lock()
	m.clientMap[clientConnectId] = c
	m.clientMutex.Unlock()
	log.Infof("add client conn[%s] with ctx: %+v", c.RemoteAddr(), c.Context())

	metric := newMetrics(clientConnectId)
	m.metricsMutex.Lock()
	m.metricMap[clientConnectId] = metric
	m.metricsMutex.Unlock()
	m.startActiveCheck(metric.clientId)
}

func (m *ClientProxy) removeConn(c gnet.Conn) {
	if ctx, ok := c.Context().(connCtxType); ok {
		if id, ok := ctx[ctxKeyConnId]; ok {
			m.clientMutex.Lock()
			if _, find := m.clientMap[id]; find {
				m.clientMutex.Unlock()
				log.Info(m.getMetricsInfo(id))
				m.stopActiveCheck(id)
				delete(m.clientMap, id)
				log.Infof("remove client conn[%s] with ctx: %+v", c.RemoteAddr(), c.Context())
			} else {
				m.clientMutex.Unlock()
			}
		}
	}
}

func (m *ClientProxy) client(clientId engine.ConnectIdType) gnet.Conn {
	m.clientMutex.Lock()
	defer m.clientMutex.Unlock()
	if c, ok := m.clientMap[clientId]; ok {
		return c
	} else {
		return nil
	}
}

func (m *ClientProxy) updateActive(clientId engine.ConnectIdType, msgType uint8) bool {
	m.metricsMutex.Lock()
	defer m.metricsMutex.Unlock()

	busy := false
	if c, ok := m.metricMap[clientId]; ok {
		now := time.Now()
		c.metrics.active.lastActiveTime = now

		var metricMsg *ClientMetricMsg
		if msg, find := c.metrics.msg[msgType]; find {
			metricMsg = msg
			if msgType != engine.ClientMsgTypeEntityRpc && now.Sub(metricMsg.lastCallTime).Milliseconds() <= 100 {
				busy = true
			}
		} else {
			metricMsg = &ClientMetricMsg{
				msgId: msgType,
			}
			c.metrics.msg[msgType] = metricMsg
		}

		metricMsg.callCount += 1
		metricMsg.lastCallTime = now
	}

	return busy
}

func (m *ClientProxy) checkActive(clientId engine.ConnectIdType) bool {
	m.metricsMutex.Lock()
	defer m.metricsMutex.Unlock()
	if c, ok := m.metricMap[clientId]; ok {
		if diff := time.Now().Sub(c.metrics.active.lastActiveTime); diff.Seconds() > 10 {
			return false
		} else {
			return true
		}
	}

	return false
}

func (m *ClientProxy) getActive(clientId engine.ConnectIdType) *ClientMetricsActive {
	m.metricsMutex.Lock()
	defer m.metricsMutex.Unlock()
	if c, ok := m.metricMap[clientId]; ok {
		return c.metrics.active
	}

	return nil
}

func (m *ClientProxy) setBindEntity(clientId engine.ConnectIdType, entityId engine.EntityIdType) {
	m.metricsMutex.Lock()
	defer m.metricsMutex.Unlock()
	if c, ok := m.metricMap[clientId]; ok {
		c.metrics.active.bindEntityId = entityId
		c.metrics.active.bindEntityTime = time.Now()
	}
}

func (m *ClientProxy) checkActiveTimerCb(params ...interface{}) {
	cliId := params[0].(engine.ConnectIdType)
	if !m.checkActive(cliId) {
		if conn := m.client(cliId); conn != nil {
			log.Infof("client active check timeout, clientId: %d, active: %s", cliId, m.getActive(cliId).toString())
			_ = conn.Close()
		}
	}
}

func (m *ClientProxy) startActiveCheck(clientId engine.ConnectIdType) {
	m.metricsMutex.Lock()
	defer m.metricsMutex.Unlock()
	if c, ok := m.metricMap[clientId]; ok {
		if c.metrics.active.activeTimerId == 0 {
			c.metrics.active.activeTimerId = engine.GetTimer().AddTimer(time.Second, 2*time.Second, m.checkActiveTimerCb, clientConnectId)
			log.Debugf("start active check timer clientId: %d, timerId: %d", clientId, c.metrics.active.activeTimerId)
		}
	}
}

func (m *ClientProxy) stopActiveCheck(clientId engine.ConnectIdType) {
	m.metricsMutex.Lock()
	defer m.metricsMutex.Unlock()
	if c, ok := m.metricMap[clientId]; ok {
		if c.metrics.active.activeTimerId > 0 {
			engine.GetTimer().Cancel(c.metrics.active.activeTimerId)
			c.metrics.active.activeTimerId = 0
			log.Debugf("stop active check timer clientId: %d", clientId)
		}
	}
}

func (m *ClientProxy) rpcMetrics(clientId engine.ConnectIdType, name string) int {
	m.metricsMutex.Lock()
	defer m.metricsMutex.Unlock()
	if c, ok := m.metricMap[clientId]; ok {
		now := time.Now()
		var rpc *ClientMetricsRpc
		if info, find := c.metrics.rpc[name]; find {
			rpc = info
			//0.1秒内多次调用
			diff := now.Sub(rpc.lastCallTime).Milliseconds()
			if diff <= 100 {
				rpc.busyCallCount += 1
			} else {
				rpc.busyCallCount = 0
			}
		} else {
			rpc = &ClientMetricsRpc{
				rpcName: name,
			}
			c.metrics.rpc[name] = rpc
		}
		rpc.lastCallTime = now
		rpc.callCount += 1
		return rpc.busyCallCount
	}
	return 0
}

func (m *ClientProxy) getMetricsInfo(clientId engine.ConnectIdType) string {
	m.metricsMutex.Lock()
	defer m.metricsMutex.Unlock()
	msg := "\n=============================Metrics Begin============================================\n"
	if c, ok := m.metricMap[clientId]; ok {
		msg += fmt.Sprintf("[Metrics clientId %d]\n", clientId)
		msg += fmt.Sprintf("[Metrics active] %s\n", c.metrics.active.toString())
		for _, rpcInfo := range c.metrics.rpc {
			msg += fmt.Sprintf("[Metrics rpc] %s\n", rpcInfo.toString())
		}
		for _, msgInfo := range c.metrics.msg {
			msg += fmt.Sprintf("[Metrics msg] %s\n", msgInfo.toString())
		}
	}
	msg += "=============================Metrics End============================================\n"

	return msg
}
