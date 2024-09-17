package engine

import (
	"errors"
	"go.uber.org/atomic"
	"net"
	"rpg/engine/engine/RingBuffer"
	"sync"
	"time"
)

const (
	connectStatusDisconnected  = 0
	connectStatusConnecting    = 1
	connectStatusConnected     = 2
	connectStatusDisconnecting = 3
)

const (
	connActionNone      = 0
	connActionClose     = 1
	connActionReconnect = 2
)

const (
	handlerActionNone         = 0
	handlerActionConnected    = 1
	handlerActionDisconnected = 2
)

type ITcpClientHandler interface {
	OnConnect(*TcpClient)
	OnDisconnect(*TcpClient)
	OnMessage(*TcpClient, []byte) error
}

type ITcpClientCodec interface {
	Encode([]byte) ([]byte, error)
	Decode([]byte) (int, []byte, error)
}

func WithTcpClientCodec(codec ITcpClientCodec) Option {
	return func(opts *Options) {
		opts.Codec = codec
	}
}

func WithTcpClientHandle(handler ITcpClientHandler) Option {
	return func(opts *Options) {
		opts.Handler = handler
	}
}

func WithTcpClientContext(ctx interface{}) Option {
	return func(opts *Options) {
		opts.Context = ctx
	}
}

type defaultCodec struct {
}

func (m *defaultCodec) Encode(buf []byte) ([]byte, error) {
	return buf, nil
}

func (m *defaultCodec) Decode(buf []byte) (int, []byte, error) {
	return len(buf), buf, nil
}

type defaultHandler struct {
}

func (m *defaultHandler) OnConnect(*TcpClient) {
}

func (m *defaultHandler) OnDisconnect(*TcpClient) {
}

func (m *defaultHandler) OnMessage(*TcpClient, []byte) error {
	return nil
}

type Options struct {
	Codec   ITcpClientCodec
	Handler ITcpClientHandler
	Context interface{}
}

type Option func(opts *Options)

func NewTcpClient(opts ...Option) *TcpClient {
	c := new(TcpClient)
	c.init(opts...)
	return c
}

type TcpClient struct {
	inBufferLock  sync.Mutex
	inBuffer      *RingBuffer.RingBuffer
	outBufferLock sync.Mutex
	outBuffer     *RingBuffer.RingBuffer
	conn          *net.TCPConn
	codec         ITcpClientCodec
	handler       ITcpClientHandler
	ctx           interface{}
	addr          string
	connAction    atomic.Int32
	connectStatus atomic.Int32
	handlerAction atomic.Int32
	autoReconnect bool
}

func (m *TcpClient) init(opts ...Option) {
	m.connectStatus.Store(connectStatusDisconnected)
	m.inBuffer = RingBuffer.New(0)
	m.outBuffer = RingBuffer.New(0)
	m.connAction.Store(connActionNone)
	m.handlerAction.Store(handlerActionNone)
	options := m.loadOptions(opts...)
	if options.Codec != nil {
		m.codec = options.Codec
	} else {
		m.codec = &defaultCodec{}
	}
	if options.Handler != nil {
		m.handler = options.Handler
	} else {
		m.handler = &defaultHandler{}
	}
	if options.Context != nil {
		m.ctx = options.Context
	}
}

func (m *TcpClient) loadOptions(options ...Option) *Options {
	opts := new(Options)
	for _, option := range options {
		option(opts)
	}
	return opts
}

func (m *TcpClient) recv() {
	for {
		buf := make([]byte, 2048)
		//read blocks until data comes
		if n, err := m.conn.Read(buf); err != nil {
			log.Infof("read from %s error: %s", m.conn.RemoteAddr(), err.Error())
			break
		} else {
			m.inBufferLock.Lock()
			_, _ = m.inBuffer.Write(buf[:n])
			log.Tracef("receive %d bytes from %s, total buf length: %d", n, m.conn.RemoteAddr(), m.inBuffer.Length())
			m.inBufferLock.Unlock()
		}
	}
	if m.autoReconnect {
		m.markReconnect()
	} else {
		m.Disconnect()
	}
}

func (m *TcpClient) SetContext(ctx interface{}) {
	m.ctx = ctx
}

func (m *TcpClient) Context() interface{} {
	return m.ctx
}

func (m *TcpClient) RemoteAddr() string {
	if m.conn != nil {
		return m.conn.RemoteAddr().String()
	}
	return ""
}

func (m *TcpClient) connect() error {
	status := m.connectStatus.Load()
	if status == connectStatusConnected || status == connectStatusConnecting {
		return nil
	}

	//log.Infof("TcpClient try connect to %s", m.addr)
	m.connectStatus.Store(connectStatusConnecting)
	tcpAddr, err := net.ResolveTCPAddr("tcp", m.addr)
	if err != nil {
		m.connectStatus.Store(connectStatusDisconnected)
		return err
	}

	m.conn, err = net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		m.connectStatus.Store(connectStatusDisconnected)
		return err
	}

	m.connectStatus.Store(connectStatusConnected)
	m.handlerAction.Store(handlerActionConnected)
	go m.recv()

	return nil
}

func (m *TcpClient) Connect(addr string, autoReconnect bool) {
	m.addr = addr
	m.autoReconnect = autoReconnect

	go func() {
		if m.autoReconnect {
			tryTimes := 0
			for err := m.connect(); err != nil && tryTimes < 60; err = m.connect() {
				tryTimes += 1
				log.Warnf("connect to %s error: %s, tryTimes: %d", addr, err.Error(), tryTimes)
				time.Sleep(5 * time.Second)
			}
		} else {
			if err := m.connect(); err != nil {
				log.Warnf("connect to %s error: %s", addr, err.Error())
			}
		}
	}()
}

func (m *TcpClient) IsDisconnected() bool {
	return m.connectStatus.Load() == connectStatusDisconnected
}

func (m *TcpClient) Disconnect() {
	m.connAction.Store(connActionClose)
}

func (m *TcpClient) markReconnect() {
	m.connAction.Store(connActionReconnect)
}

func (m *TcpClient) Status() int32 {
	return m.connectStatus.Load()
}

func (m *TcpClient) Send(buf []byte) (int, error) {
	if m.conn == nil {
		return 0, errors.New("connection is nil")
	}
	data, err := m.codec.Encode(buf)
	if err != nil {
		return 0, err
	}
	m.outBufferLock.Lock()
	defer m.outBufferLock.Unlock()
	return m.outBuffer.Write(data)
}

func (m *TcpClient) Tick() {
	if m.conn == nil {
		return
	}

	m.inBufferLock.Lock()
	var buf []byte
	head, tail := m.inBuffer.LazyReadAll()
	if tail != nil {
		buf = RingBuffer.CombatBytes(head, tail)
	} else {
		buf = head
	}
	//if m.inBuffer.Length() > 0 {
	//	log.Tracef("inBuffSize: %d conn: %s, buf: %v", m.inBuffer.Length(), m.conn.RemoteAddr(), len(buf))
	//}
	if shiftN, data, err := m.codec.Decode(buf); err != nil {
		log.Errorf("decode from %s error: %s, buf len: %d", m.conn.RemoteAddr(), err.Error(), len(buf))
		m.markReconnect()
	} else if shiftN > 0 {
		if err = m.handler.OnMessage(m, data); err != nil {
			m.markReconnect()
		}
		//oldLen := m.inBuffer.Length()
		m.inBuffer.Shift(shiftN)
		//log.Tracef("shift inBuff %d, before: %d, left: %d, conn: %s", shiftN, oldLen, m.inBuffer.Length(), m.conn.RemoteAddr())
	}
	m.inBufferLock.Unlock()

	m.outBufferLock.Lock()
	if m.outBuffer.Length() > 0 {
		outHead, outTail := m.outBuffer.LazyReadAll()
		if outTail != nil {
			buf = RingBuffer.CombatBytes(outHead, outTail)
		} else {
			buf = outHead
		}
		//log.Tracef("outBuffSize: %d conn: %s, buf: %v", m.outBuffer.Length(), m.conn.RemoteAddr(), len(buf))
		if n, err := m.conn.Write(buf); err != nil {
			m.markReconnect()
		} else {
			m.outBuffer.Shift(n)
			log.Tracef("write %d bytes to [%s:%v]", n, m.conn.RemoteAddr(), m.ctx)
		}
	}
	m.outBufferLock.Unlock()
	m.doAction()
}

func (m *TcpClient) doAction() {
	m.doHandlerAction()
	m.doConnAction()
}

func (m *TcpClient) doHandlerAction() {
	switch m.handlerAction.Load() {
	case handlerActionConnected:
		m.handler.OnConnect(m)
	case handlerActionDisconnected:
		m.handler.OnDisconnect(m)
	}
	m.handlerAction.Store(handlerActionNone)
}

func (m *TcpClient) doConnAction() {
	action := m.connAction.Load()
	if action == connActionReconnect || action == connActionClose {
		connectStatus := m.connectStatus.Load()
		if connectStatus != connectStatusDisconnecting && connectStatus != connectStatusDisconnected {
			m.connectStatus.Store(connectStatusDisconnecting)

			m.inBufferLock.Lock()
			m.inBuffer.Reset()
			m.inBufferLock.Unlock()

			m.outBufferLock.Lock()
			m.outBuffer.Reset()
			m.outBufferLock.Unlock()

			_ = m.conn.Close()
			m.connectStatus.Store(connectStatusDisconnected)
			m.handlerAction.Store(handlerActionDisconnected)
		}
		if action == connActionReconnect {
			m.Connect(m.addr, m.autoReconnect)
		}
	}
	m.connAction.Store(connActionNone)
	m.doHandlerAction()
}
