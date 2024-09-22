package main

import (
	"fmt"
	"github.com/panjf2000/gnet"
	"rpg/engine/engine"
	"strconv"
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
	clientMap map[engine.ConnectIdType]gnet.Conn //clientConnectId -> conn
	metricMap map[engine.ConnectIdType]*Metrics
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

	m.clientMap[clientConnectId] = c
	log.Infof("add client conn[%s] with ctx: %+v", c.RemoteAddr(), c.Context())

	metric := newMetrics(clientConnectId)
	m.metricMap[clientConnectId] = metric
	m.startActiveCheck(metric.clientId)
}

func (m *ClientProxy) removeConn(clientId engine.ConnectIdType) {
	if _, find := m.clientMap[clientId]; find {
		log.Info(m.getMetricsInfo(clientId))
		m.stopActiveCheck(clientId)
		delete(m.clientMap, clientId)
		log.Infof("remove client conn, clientId: %d", clientId)
	} else {
		log.Warnf("remove client conn but clientId: %d not found", clientId)
	}
}

func (m *ClientProxy) client(clientId engine.ConnectIdType) gnet.Conn {
	if c, ok := m.clientMap[clientId]; ok {
		return c
	} else {
		return nil
	}
}

func (m *ClientProxy) updateActive(clientId engine.ConnectIdType, msgType uint8) bool {
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
	if c, ok := m.metricMap[clientId]; ok {
		heartbeatDuration := engine.GetConfig().HeartBeatInterval
		if heartbeatDuration <= 0 {
			heartbeatDuration = engine.HeartbeatTick
		}
		if diff := time.Now().Sub(c.metrics.active.lastActiveTime); int32(diff.Seconds()) > 3*heartbeatDuration {
			return false
		} else {
			return true
		}
	}

	return false
}

func (m *ClientProxy) getActive(clientId engine.ConnectIdType) *ClientMetricsActive {
	if c, ok := m.metricMap[clientId]; ok {
		return c.metrics.active
	}

	return nil
}

func (m *ClientProxy) setBindEntity(clientId engine.ConnectIdType, entityId engine.EntityIdType) {
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
	if c, ok := m.metricMap[clientId]; ok {
		if c.metrics.active.activeTimerId == 0 {
			c.metrics.active.activeTimerId = engine.GetTimer().AddTimer(time.Second, 2*time.Second, m.checkActiveTimerCb, clientConnectId)
			log.Debugf("start active check timer clientId: %d, timerId: %d", clientId, c.metrics.active.activeTimerId)
		}
	}
}

func (m *ClientProxy) stopActiveCheck(clientId engine.ConnectIdType) {
	if c, ok := m.metricMap[clientId]; ok {
		if c.metrics.active.activeTimerId > 0 {
			engine.GetTimer().Cancel(c.metrics.active.activeTimerId)
			c.metrics.active.activeTimerId = 0
			log.Debugf("stop active check timer clientId: %d", clientId)
		}
	}
}

func (m *ClientProxy) rpcMetrics(clientId engine.ConnectIdType, name string) int {
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
