package main

import (
	"encoding/json"
	"errors"
	"github.com/gorilla/websocket"
	"net/http"
	"sync"
)

var serverConn sync.Map

type messageProcessor func(ws *webSocketConnection, req *webSocketMessage) (*webSocketMessage, error)

type webSocketMessage struct {
	Type    string      `json:"type"`    //与web页面交互的消息类型
	Target  string      `json:"target"`  //目标进程
	Command string      `json:"command"` //gm命令的名称,不做处理直接返回给页面
	Data    interface{} `json:"data"`    //消息内容
}

var gMsgProcess messageProcessor = dispatcher

var wsUpGrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type webSocketConnection struct {
	sync.Mutex
	conn      *websocket.Conn
	inChan    chan *webSocketMessage
	outChan   chan *webSocketMessage
	isClosed  bool
	closeChan chan byte
}

func (m *webSocketConnection) loopRead() {
	for {
		_, data, err := m.conn.ReadMessage()
		if err != nil {
			goto error
		}
		req := &webSocketMessage{}
		if err = json.Unmarshal(data, &req); err != nil {
			goto error
		}
		select {
		case m.inChan <- req:
		case <-m.closeChan:
			goto closed
		}
	}

error:
	m.close()
closed:
}

func (m *webSocketConnection) loopWrite() {
	for {
		select {
		case msg := <-m.outChan:
			if err := m.conn.WriteJSON(msg); err != nil {
				goto error
			}
		case <-m.closeChan:
			goto closed
		}
	}
error:
	m.close()
closed:
}

func (m *webSocketConnection) loop() {
	for {
		msg, err := m.read()
		if err != nil {
			break
		}
		rsp, err := gMsgProcess(m, msg)
		if err != nil {
			_ = m.write(&webSocketMessage{
				Type: "error",
				Data: err.Error(),
			})
			continue
		} else if rsp == nil {
			continue
		}
		if err = m.write(rsp); err != nil {
			break
		}
	}
}

func (m *webSocketConnection) write(data *webSocketMessage) error {
	select {
	case m.outChan <- data:
	case <-m.closeChan:
		return errors.New("websocket closed")
	}
	return nil
}

func (m *webSocketConnection) read() (*webSocketMessage, error) {
	select {
	case msg := <-m.inChan:
		return msg, nil
	case <-m.closeChan:
	}
	return nil, errors.New("websocket closed")
}

func (m *webSocketConnection) close() {
	_ = m.conn.Close()
	m.Lock()
	defer m.Unlock()
	if !m.isClosed {
		m.isClosed = true
		close(m.closeChan)
	}
}

func NewWebSocketHandler(rsp http.ResponseWriter, req *http.Request) {
	conn, err := wsUpGrader.Upgrade(rsp, req, nil)
	if err != nil {
		return
	}

	c := &webSocketConnection{
		conn:      conn,
		inChan:    make(chan *webSocketMessage, 1000),
		outChan:   make(chan *webSocketMessage, 1000),
		closeChan: make(chan byte),
		isClosed:  false,
	}

	go c.loop()
	go c.loopWrite()
	go c.loopRead()
}
