import React, {useEffect, useState} from 'react';
import {message, Modal, Result, Tabs} from 'antd';
import {useRequest} from "@@/plugin-request/request";
import DebugConsole, {ConsoleValueType} from "@/pages/console/debug_console";

export type ConsoleTextAreaProps = {
  name: string;
}

export type serverPaneInfo = {
  name: string;
}

const ConsoleTabPanes: React.FC = () => {
  const { TabPane } = Tabs
  const { confirm } = Modal
  const [client, setClient] = useState<WebSocket>()

  const [serverPanes, setServerPanes] = useState<serverPaneInfo[]>([])

  const [consoleResponse, setConsoleResponse] = useState<ConsoleValueType>()

  const onWebSocketMessage = async (e: {data: any}) => {
    const ab = await new Response(e.data).arrayBuffer()
    const msg = JSON.parse(Buffer.from(ab).toString())
    if(msg.type == "servers") {
      const panes: serverPaneInfo[] = [];
      for(let i = 0; i < msg.data.length; i++) {
        panes.push({name: msg.data[i]})
      }
      setServerPanes(panes)
    } else if(msg.type == "message") {
      setConsoleResponse({name: msg.target, data: msg.data})
    } else if(msg.type == "error") {
      message.error(msg.data);
    } else if(msg.type == "reload") {
      setConsoleResponse({name: msg.target, data: msg.data})
    }
  }

  const startWebSocket = () => {
    const c = new WebSocket("ws://localhost:9000/telnet")
    c.onopen = () => {
      // message.success("已连接至调试后台").then()
      setClient(c)
    }
    c.onclose = () => {
      setClient(undefined)
      startWebSocket()
    }
    c.onmessage = onWebSocketMessage
  }

  useRequest(async() => {
    startWebSocket()
  })

  useEffect(() => {
    if(client === undefined) {
      return
    }
    //load servers
    client.send(JSON.stringify({"type": "servers"}))
  }, [client])

  const onConsoleCommand = (target: string, data: string) => {
    if(!client) {
      message.warn("连接已断开，请刷新页面重试").then()
      return
    }
    const msg = {
      type: "message",
      target: target,
      data: data,
    }
    client.send(JSON.stringify(msg))
  }

  const onHotfix = () => {
    confirm({
      title: "热更确认",
      onOk() {
        client?.send(JSON.stringify({type: "reload" }));
      },
      onCancel() {
      },
    })
  }

  return (
    <Tabs
      type="card"
      tabPosition="left"
      tabBarGutter={10}
      tabBarStyle={{width: "15%"}}
    >
      {serverPanes.length > 0 && (serverPanes.map(pane => (
        <TabPane tab={pane.name} key={pane.name}>
          <DebugConsole
            name={pane.name}
            response={consoleResponse}
            onCommandSend={onConsoleCommand}
            onHotfix={onHotfix}
          />
        </TabPane>
      ))) || (
        <TabPane tab={`无`} key={"empty"}>
          <Result
            status="404"
            title="404"
            subTitle={"未获取到服务器列表"}/>
        </TabPane>
      )
      }
    </Tabs>
  );
};

export default ConsoleTabPanes;
