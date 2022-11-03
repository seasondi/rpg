package main

import (
	"rpg/engine/engine"
	"fmt"
	"github.com/sirupsen/logrus"
	"html/template"
	"net"
	"net/http"
)

var log *logrus.Entry

func web(w http.ResponseWriter, _ *http.Request) {
	if t, err := template.ParseFiles("tools/web/dist/index.html"); err == nil {
		_ = t.Execute(w, nil)
	} else {
		log.Errorf("parse web html error: %s", err.Error())
	}
}

func StartWebConsole() {
	if !engine.GetConfig().ServerConfig().EnableWeb {
		log.Infof("web console is disabled")
		return
	}
	addr := engine.GetConfig().ServerConfig().Addr
	server := http.Server{Addr: addr}
	http.Handle("/dist/", http.StripPrefix("/dist/", http.FileServer(http.Dir("tools/web/dist"))))
	http.HandleFunc("/debug", web)
	log.Infof("命令行调试 listen at http://%s/dist/debug", addr)
	log.Infof("GM指令 listen at http://%s/dist/gm", addr)
	log.Infof("导表 listen at http://%s/dist/exportTable", addr)
	_ = server.ListenAndServe()
}

func StartWebSocket() {
	http.HandleFunc("/telnet", NewWebSocketHandler)
	http.HandleFunc("/gm", NewWebSocketHandler)
	http.HandleFunc("/exportTable", NewWebSocketHandler)
	//修改该端口需同步修改tools/web/src/src/pages目录中所有的websocket端口,然后重新编译web
	_ = http.ListenAndServe(":9000", nil)
}

func main() {
	if err := engine.Init(engine.STAdmin); err != nil {
		fmt.Println("engine init error: ", err.Error())
		return
	}
	log = engine.GetLogger()
	defer engine.Close()

	serverConn = make(map[string]*net.TCPConn)

	go StartWebConsole()
	StartWebSocket()
}
