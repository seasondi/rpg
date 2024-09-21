package main

import "rpg/engine/engine"

type adminHandler struct {
}

func (m *adminHandler) Encode(data []byte) ([]byte, error) {
	return data, nil
}

func (m *adminHandler) Decode(data []byte) (int, []byte, error) {
	return len(data), data, nil
}

func (m *adminHandler) OnConnect(conn *engine.TcpClient) {
	log.Infof("connected to [%s]", conn.RemoteAddr())
}

func (m *adminHandler) OnDisconnect(conn *engine.TcpClient) {
	log.Infof("disconnect from [%s]", conn.RemoteAddr())
	serverConn.Delete(conn.Context())
}

func (m *adminHandler) OnMessage(_ *engine.TcpClient, buf []byte) error {
	commandResponseChan <- string(buf)
	return nil
}
