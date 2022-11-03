package main

import (
	"rpg/engine/engine"
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	clientV3 "go.etcd.io/etcd/client/v3"
	"golang.org/x/text/encoding/simplifiedchinese"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const exportTableConfigFile = "./config/export_table_config.txt"

func dispatcher(ws *webSocketConnection, req *webSocketMessage) (*webSocketMessage, error) {
	switch req.Type {
	case "servers": //请求服务器列表
		return onQueryServerList(req)
	case "message": //调试消息
		return onConsoleMessage(ws, req)
	case "gmList": //gm列表
		return onGetGMList(ws, req)
	case "gmCommand": //gm命令
		return onGMCommand(ws, req)
	case "reload": //热更
		return onReloadCommand(ws, req)
	case "tableConfig": //导表配置
		return onGetExportTableConfig(ws, req)
	case "setTableConfig": //设置导表配置
		return onSetExportTableConfig(ws, req)
	case "exportTable": //导表
		return onExportTable(ws, req)
	case "findSheet": //查找表格文件
		return onFindSheet(ws, req)
	}
	return nil, errors.New("unknown message type")
}

func connectTarget(_ *webSocketConnection, target string) *net.TCPConn {
	//target格式: game.1001.1
	conn := serverConn[target]
	if conn == nil {
		info := strings.Split(target, ".")
		if len(info) >= 3 {
			name := info[0] + "_" + info[2]
			if config := engine.GetConfig().GetServerConfigByName(name); config != nil {
				if tcpAddr, err := net.ResolveTCPAddr("tcp", config.Telnet); err == nil {
					if conn, err = net.DialTCP("tcp", nil, tcpAddr); err == nil {
						_ = conn.SetKeepAlive(true)
						serverConn[target] = conn
					}
				}
			}
		}
	}
	return conn
}

func sendCommandToTarget(ws *webSocketConnection, ty int, target string, command string, retry ...bool) string {
	conn := connectTarget(ws, target)
	response := ""
	if conn == nil {
		response = fmt.Sprintf("cannot connect to target: %s", target)
	} else {
		msg := engine.TelnetMessage{
			Type: ty,
			Data: strings.ReplaceAll(command, "\n", "\t"),
		}
		data, _ := json.Marshal(msg)
		if _, err := conn.Write([]byte(string(data) + "\n")); err != nil {
			delete(serverConn, target)
			if len(retry) == 0 {
				return sendCommandToTarget(ws, ty, target, command, true)
			} else {
				return err.Error()
			}
		} else {
			buf := make([]byte, 2048)
			_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
			n, _ := conn.Read(buf)
			response = strings.TrimRight(string(buf[:n]), "\r\n")
		}
	}
	return response
}

func onQueryServerList(req *webSocketMessage) (*webSocketMessage, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()

	servers := make([]string, 0)
	results := engine.GetEtcd().Get(ctx, engine.GetEtcdPrefixWithServer(engine.ServiceGamePrefix), clientV3.WithPrefix())
	for _, r := range results {
		servers = append(servers, r.Key())
	}

	data := &webSocketMessage{Type: req.Type, Data: servers}
	return data, nil
}

func onConsoleMessage(ws *webSocketConnection, req *webSocketMessage) (*webSocketMessage, error) {
	response := sendCommandToTarget(ws, engine.TelnetMessageTypeWebCmd, req.Target, req.Data.(string))

	return &webSocketMessage{
		Type:   req.Type,
		Target: req.Target,
		Data:   response,
	}, nil
}

func onGetGMList(ws *webSocketConnection, req *webSocketMessage) (*webSocketMessage, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()

	response := "{}"
	results := engine.GetEtcd().Get(ctx, engine.StubPrefix, clientV3.WithPrefix())
	for _, r := range results {
		values := r.Value()
		if name, ok := values[engine.EtcdValueName].(string); ok && name == "GMStub" {
			if target, ok := values[engine.EtcdValueServer].(string); ok {
				response = sendCommandToTarget(ws, engine.TelnetMessageTypeGMList, target, "")
			}
			break
		}
	}

	return &webSocketMessage{
		Type:   req.Type,
		Target: req.Target,
		Data:   response,
	}, nil
}

func onGMCommand(ws *webSocketConnection, req *webSocketMessage) (*webSocketMessage, error) {
	jsonData, err := json.Marshal(req.Data)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()

	targetServer := ""
	if data, ok := req.Data.(map[string]interface{}); ok {
		if entityId := engine.InterfaceToInt(data["entity_id"]); entityId > 0 {
			result := engine.EtcdValue{}
			if err = engine.GetRedisMgr().Get(ctx, engine.GetEtcdEntityKey(engine.EntityIdType(entityId)), &result); err == nil {
				if target, ok := result[engine.EtcdValueServer].(string); ok {
					targetServer = target
				}
			}
		} else {
			results := engine.GetEtcd().Get(ctx, engine.StubPrefix, clientV3.WithPrefix())
			for _, r := range results {
				values := r.Value()
				if name, ok := values[engine.EtcdValueName].(string); ok && name == "GMStub" {
					if target, ok := values[engine.EtcdValueServer].(string); ok {
						targetServer = target
					}
					break
				}
			}
		}
	} else {
		log.Warnf("do gm command request data is not map type, data: %+v", req.Data)
	}

	response := "target not found"
	if targetServer != "" {
		response = sendCommandToTarget(ws, engine.TelnetMessageTypeGMCmd, targetServer, string(jsonData))
	}

	return &webSocketMessage{
		Type:    req.Type,
		Target:  req.Target,
		Command: req.Command,
		Data:    response,
	}, nil
}

func onReloadCommand(ws *webSocketConnection, req *webSocketMessage) (*webSocketMessage, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
	defer cancel()

	results := engine.GetEtcd().Get(ctx, engine.GetEtcdPrefixWithServer(engine.ServiceGamePrefix), clientV3.WithPrefix())
	for _, r := range results {
		response := sendCommandToTarget(ws, engine.TelnetMessageTypeReload, r.Key(), "")
		_ = ws.write(&webSocketMessage{Type: req.Type, Target: r.Key(), Data: "热更结果: " + response})
	}
	return nil, nil
}

func onGetExportTableConfig(_ *webSocketConnection, req *webSocketMessage) (*webSocketMessage, error) {
	f, err := os.OpenFile(exportTableConfigFile, os.O_CREATE|os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}
	response := ""
	b := make([]byte, 2048)
	if n, _ := f.Read(b); n > 0 {
		data := b[:n]
		if err = json.Unmarshal(data, &map[string]interface{}{}); err != nil {
			d, _ := json.Marshal(map[string]interface{}{})
			response = string(d)
		} else {
			response = string(b[:n])
		}
	} else {
		d, _ := json.Marshal(map[string]interface{}{})
		response = string(d)
	}

	return &webSocketMessage{
		Type:   req.Type,
		Target: req.Target,
		Data:   response,
	}, nil
}

func exportTableKeyName(key string) string {
	switch key {
	case "export_cmd":
		return "导表命令"
	case "excel_path":
		return "excel路径"
	case "export_client_path":
		return "客户端表格输出路径"
	case "export_server_path":
		return "服务端表格输出路径"
	default:
		return key
	}
}

func checkPath(name, path string) error {
	if fi, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%s: %s 不存在,请检查配置", exportTableKeyName(name), path)
		}
	} else if !fi.IsDir() {
		return fmt.Errorf("%s: %s 不是目录,请检查配置", exportTableKeyName(name), path)
	}
	return nil
}

func onSetExportTableConfig(_ *webSocketConnection, req *webSocketMessage) (*webSocketMessage, error) {
	if s, ok := req.Data.(string); ok {
		r := make(map[string]string)
		if err := json.Unmarshal([]byte(s), &r); err != nil {
			return nil, err
		}
		for name, path := range r {
			if err := checkPath(name, path); err != nil {
				return nil, err
			}
		}
	}

	f, err := os.OpenFile(exportTableConfigFile, os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	_ = f.Truncate(0)
	if _, err = f.WriteString(req.Data.(string)); err != nil {
		return nil, err
	}

	return &webSocketMessage{
		Type:   req.Type,
		Target: req.Target,
	}, nil
}

func onExportTable(ws *webSocketConnection, req *webSocketMessage) (*webSocketMessage, error) {
	f, err := os.OpenFile(exportTableConfigFile, os.O_CREATE|os.O_RDONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("导表配置文件打开失败:%s", err.Error())
	}

	readOutput := func(wg *sync.WaitGroup, in io.ReadCloser) {
		defer wg.Done()
		reader := bufio.NewReader(in)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			r, _ := simplifiedchinese.GB18030.NewDecoder().Bytes([]byte(line))
			_ = ws.write(&webSocketMessage{
				Type: req.Type,
				Data: string(r),
			})
		}
	}

	b := make([]byte, 2048)
	if n, _ := f.Read(b); n > 0 {
		data := b[:n]
		r := make(map[string]string)
		if err = json.Unmarshal(data, &r); err != nil {
			return nil, errors.New("导表配置数据读取失败")
		} else {
			if r["export_cmd"] == "" {
				return nil, errors.New("导表命令错误,请检查配置")
			}
			if err = checkPath("excel_path", r["excel_path"]); err != nil {
				return nil, err
			}
			if r["export_client_path"] == "" {
				return nil, errors.New(exportTableKeyName("export_client_path") + "不存在")
			}
			if r["export_server_path"] == "" {
				return nil, errors.New(exportTableKeyName("export_server_path") + "不存在")
			}
			exportCmd := strings.ReplaceAll(r["export_cmd"], "\\", "/")
			excelPath := strings.ReplaceAll(r["excel_path"], "\\", "/")
			exportClientPath := strings.ReplaceAll(r["export_client_path"], "\\", "/")
			exportServerPath := strings.ReplaceAll(r["export_server_path"], "\\", "/")
			cmd := fmt.Sprintf("%s --excel_path=%s --client_output=%s --server_output=%s", exportCmd, excelPath, exportClientPath, exportServerPath)
			c := exec.Command("cmd", "/C", cmd)
			c.Env = os.Environ()

			wg := &sync.WaitGroup{}
			if stdout, err := c.StdoutPipe(); err != nil {
				return nil, err
			} else {
				wg.Add(1)
				go readOutput(wg, stdout)
			}
			if stderr, err := c.StderrPipe(); err != nil {
				return nil, err
			} else {
				wg.Add(1)
				go readOutput(wg, stderr)
			}

			if err := c.Start(); err != nil {
				return nil, err
			}
			wg.Wait()
			_ = c.Wait()
			return &webSocketMessage{Type: req.Type, Data: ""}, nil
		}
	} else {
		return nil, errors.New("请先进行导表配置")
	}
}

func onFindSheet(_ *webSocketConnection, req *webSocketMessage) (*webSocketMessage, error) {
	f, err := os.OpenFile(exportTableConfigFile, os.O_CREATE|os.O_RDONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("导表配置文件打开失败:%s", err.Error())
	}

	b := make([]byte, 2048)
	if n, _ := f.Read(b); n > 0 {
		data := b[:n]
		r := make(map[string]string)
		if err = json.Unmarshal(data, &r); err != nil {
			return nil, errors.New("导表配置数据读取失败")
		} else {
			if r["export_cmd"] == "" {
				return nil, errors.New("导表命令错误,请检查配置")
			}
			if err = checkPath("excel_path", r["excel_path"]); err != nil {
				return nil, err
			}
			exportCmd := strings.ReplaceAll(r["export_cmd"], "\\", "/")
			excelPath := strings.ReplaceAll(r["excel_path"], "\\", "/")
			cmd := fmt.Sprintf("%s --excel_path=%s --find_sheet=%s", exportCmd, excelPath, req.Data)
			c := exec.Command("cmd", "/C", cmd)
			c.Env = os.Environ()
			output, err := c.CombinedOutput()
			if err != nil {
				return nil, err
			} else {
				result, _ := simplifiedchinese.GB18030.NewDecoder().Bytes(output)
				return &webSocketMessage{
					Type: req.Type,
					Data: string(result),
				}, nil
			}
		}
	} else {
		return nil, errors.New("请先进行导表配置")
	}
}
