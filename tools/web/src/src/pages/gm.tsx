import React, {useEffect, useState} from 'react';
import {PageContainer} from "@ant-design/pro-layout";
import {Col, Menu, message, Result, Row, Select, Space} from "antd";
import {MailOutlined} from "@ant-design/icons";
import type {MenuInfo} from "rc-menu/lib/interface";
import GMCommand, {GMCommandProps} from "@/pages/gm/command";
import {useRequest} from "@@/plugin-request/request";

export type selectOption = {
  label: any;
  value: any;
}

export type gmCommandArg = {
  type: string;
  name: string;
  index: string;
  min?: number;
  max?: number;
}

export type gmCommand = {
  command: string;
  args: gmCommandArg[];
}

export type gmListCategory = {
  name: string;
  value: gmCommand[];
}

const GMPane: React.FC = () => {
  const { SubMenu } = Menu

  //websocket信息
  const [client, setClient] = useState<WebSocket>()

  //下来筛选框信息
  const [selectOptions, setSelectOptions] = useState<selectOption[]>([])
  const [currSelectOption, setCurrSelectOption] = useState<string|undefined>(undefined)
  //所有指令信息
  const [GMData, setGMData] = useState<gmListCategory[]>([])
  //指令所有分类信息
  const [categories, setCategories] = useState<string[]>([])
  //指令名称->指令信息的映射
  const [nameToCommand, setNameToCommand] = useState<Record<string, GMCommandProps>>({})
  //当前指令信息
  const [gmCommandProps, setGmCommandProps] = useState<GMCommandProps>({})
  //每个指令对应的返回值
  const [allGmCommandResponse, setAllGmCommandResponse] = useState<Record<string, any>>({})
  //当前选择的菜单
  const [currSelectedMenu, setCurrSelectedMenu] = useState<string[]>([])

  // 设置指令的返回值数据
  const setCommandResponse = (command: string, data: any) => {
    if(command === undefined || command === "") {
      return
    }
    const info = allGmCommandResponse || {}
    info[command] = data
    setAllGmCommandResponse({}) //先置空再赋值,触发刷新
    setAllGmCommandResponse(info)
  }

  const onWebSocketMessage = async (e: {data: any}) => {
    const ab = await new Response(e.data).arrayBuffer()
    const msg = JSON.parse(Buffer.from(ab).toString())
    if(msg.type == "gmList") {
      const r = JSON.parse(msg.data)
      const info: gmListCategory[] = []
      const allCategory: string[] = []
      const commandMap: Record<string, GMCommandProps> = {}

      for(let i = 0; i < r.length; i++) {
        const commands: gmCommand[] = [];
        for(const cmd in r[i].value) {
          const gm: gmCommand = {
            command: r[i].value[cmd].name + "(" + cmd + ")",
            args: [],
          }
          const args: gmCommandArg[] = []
          for(let t = 0; t < r[i].value[cmd].args.length; t++) {
            args.push(r[i].value[cmd].args[t] as gmCommandArg)
          }
          gm.args = args
          commands.push(gm)
          commandMap[gm.command] = {command: cmd, args: args, name: gm.command}
        }
        const category = {
          name: r[i].name,
          value: commands,
        }
        info.push(category)
        allCategory.push(r[i].name)
      }

      setGMData(info)
      setCategories(allCategory)
      setNameToCommand(commandMap)
      const opts: selectOption[] = []
      for(const cmd in commandMap) {
        opts.push({label: cmd, value: cmd})
      }
      setSelectOptions(opts)
    } else if(msg.type === "gmCommand") {
      setCommandResponse(msg.command, msg.data)
    } else if (msg.type == "error") {
      message.error(msg.data);
    }
  }

  //启动websocket
  const startWebSocket = () => {
    const c = new WebSocket("ws://localhost:9000/gm")
    c.onopen = () => {
      // message.success("已连接至GM后台").then()
      setClient(c)
    }
    c.onclose = () => {
      setClient(undefined)
    }
    c.onmessage = onWebSocketMessage
    return c
  }

  useRequest(async() => {
    startWebSocket()
  })

  useEffect(() => {
    if(client === undefined) {
      return
    }
    //load gm list
    client.send(JSON.stringify({"type": "gmList"}))
  }, [client])

  const onClickGmCommand = (info: MenuInfo) => {
    setCurrSelectedMenu([info.key])
    setCurrSelectOption(undefined)
  }

  const onMenuSelected = (info: MenuInfo) => {
    const props = nameToCommand[info.key]
    setGmCommandProps(props)
  }

  const onSubmitGMCommand = (command: string, values: any) => {
    values.command = command
    setCommandResponse(command, undefined)
    if(client === undefined) {
      const c = startWebSocket()
      setTimeout(() => {
        c.send(JSON.stringify({type: "gmCommand", data: values}))
      }, 500)
    } else {
      client.send(JSON.stringify({type: "gmCommand", data: values, command: command}))
    }
  }

  const onCommandSelected = (value: string) => {
    setCurrSelectedMenu([value])
    setCurrSelectOption(value)
    if(value !== undefined) {
      setGmCommandProps(nameToCommand[value])
    } else {
      setGmCommandProps({})
    }
  }

  return (
    <PageContainer>
      {GMData.length > 0 && categories.length > 0 &&
      (<Row gutter={[30, 8]}>
        <Col style={{width: "20%", height: "100%"}}>
          <Space direction={'vertical'} style={{width: "100%"}}>
            <Select
              allowClear={true}
              showSearch
              style={{width: "100%"}}
              placeholder={"输入要查询的GM指令"}
              options={selectOptions}
              onChange={onCommandSelected}
              value={currSelectOption}
            />
            <Menu
              mode="inline"
              defaultOpenKeys={categories}
              onClick={onClickGmCommand}
              onSelect={onMenuSelected}
              forceSubMenuRender={true}
              selectedKeys={currSelectedMenu}
            >
              {GMData.map(category => (
                <SubMenu key={category.name} icon={<MailOutlined/>} title={category.name}>
                  {category.value.length > 0 && category.value.map(command => (
                    <Menu.Item key={command.command}>{command.command}</Menu.Item>
                  ))}
                </SubMenu>
              ))}
            </Menu>
          </Space>
        </Col>
        <Col style={{width: "80%", height: "100%"}}>
          <GMCommand
            command={gmCommandProps.command}
            args={gmCommandProps.args}
            name={gmCommandProps.name}
            onSubmitCommand={onSubmitGMCommand}
            response={allGmCommandResponse?.[gmCommandProps.command || ""]}
          />
        </Col>
      </Row>
      ) || (
        <Result
          status="404"
          title="404"
          subTitle={"未获取到GM命令列表"}/>
      )
      }
    </PageContainer>
  );
};

export default GMPane;
