package main

import "github.com/panjf2000/gnet"

type connContext struct {
	serverName        string
	lastHeartBeatTime int64
}

func setCtxServiceName(conn gnet.Conn, name string) {
	ctx, ok := conn.Context().(*connContext)
	if ok {
		ctx.serverName = name
	} else {
		ctx = &connContext{serverName: name}
	}
	conn.SetContext(ctx)
}

func getCtxServiceName(conn gnet.Conn) string {
	if ctx, ok := conn.Context().(*connContext); ok {
		return ctx.serverName
	} else {
		return ""
	}
}
