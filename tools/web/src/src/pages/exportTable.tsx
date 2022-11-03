import React, {useEffect, useRef, useState} from 'react';
import {Button, Card, Input, message, Modal, Result, Skeleton, Space} from "antd";
import {ModalForm, ProFormText} from "@ant-design/pro-form";
import {useRequest} from "@@/plugin-request/request";
import ProList from '@ant-design/pro-list';
import {ActionType} from "@ant-design/pro-table";
import InfiniteScroll from "react-infinite-scroll-component";
import moment from "moment";

const exportCmdName = "export_cmd";
const excelPathName = "excel_path";
const exportClientPathName = "export_client_path";
const exportServerPathName = "export_server_path";

type exportTableConfig = {
  export_cmd: string;
  excel_path: string;
  export_client_path: string;
  export_server_path: string;
}

const cmdSetTableConfig = "setTableConfig" //设置导表配置
const cmdGetTableConfig = "tableConfig" //获取导表配置
const cmdExportTable = "exportTable" //进行导表
const cmdError = "error" //错误信息
const cmdFindSheet = "findSheet" //查询表格

let tempMsg: any = [];

const ExportToLua: React.FC = () => {
  const [client, setClient] = useState<WebSocket>()
  const [configVisible, setConfigVisible] = useState<boolean>(false);
  const [exportConfig, setExportConfig] = useState<exportTableConfig|undefined>(undefined);

  const [resultData, setResultData] = useState<any[]>([]);

  const listActionRef = useRef<ActionType>()

  const { confirm } = Modal;
  const { Search } = Input;

  const onWebSocketMessage = async (e: {data: any}) => {
    const ab = await new Response(e.data).arrayBuffer()
    const msg = JSON.parse(Buffer.from(ab).toString())
    if (msg.type == cmdGetTableConfig) {
      const r = JSON.parse(msg.data);
      setExportConfig({
        export_cmd: r.export_cmd || "",
        excel_path: r.excel_path || "",
        export_client_path: r.export_client_path || "",
        export_server_path: r.export_server_path || "",
      });
    } else if (msg.type == cmdSetTableConfig) {
      setConfigVisible(false);
    } else if (msg.type == cmdError) {
      message.error(msg.data);
    } else if (msg.type == cmdExportTable) {
      if(msg.data === undefined || msg.data === "") {
        return
      }
      const data = [{
        title: moment().format("YYYY-MM-DD HH:mm:ss"),
        content: msg.data,
      }]
      const r = tempMsg.concat(data);
      tempMsg = r;
      setResultData(r);
      listActionRef.current?.reloadAndRest?.();
      document.getElementById("historyScrollDiv")?.scroll({
        behavior: "auto",
        left: 0,
        top: 100000,
      });
    } else if(msg.type == cmdFindSheet) {
      confirm({
        width: 700,
        title: msg.data,
        autoFocusButton: "ok",
      })
    }
  }

  const startWebSocket = () => {
    const c = new WebSocket("ws://localhost:9000/exportTable")
    c.onopen = () => {
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
    client.send(JSON.stringify({"type": cmdGetTableConfig}))
  }, [client])

  const onSubmitConfig = async (values: any) => {
    const msg = {
      "type": cmdSetTableConfig,
      "data": JSON.stringify(values),
    }
    client?.send(JSON.stringify(msg))
  }

  const doExportTable = async() => {
    tempMsg = [];
    setResultData([]);
    client?.send(JSON.stringify({"type": cmdExportTable}))
  }

  const doFindSheet = async(value: string) => {
    if (value === "") {
      message.info("请输入表格名称")
      return
    }
    client?.send(JSON.stringify({"type": cmdFindSheet, "data": value}))
  }

  return (
    <Space direction={"vertical"} style={{width: "100%"}} >
      {exportConfig !== undefined &&
      (<Space direction={"vertical"} style={{width: "100%"}}>
          <Card
            title="导表"
            key="card"
            extra={
              [
                <Space key="space" direction={"horizontal"}>
                  <Search
                    placeholder="请输入要查询的表格名称"
                    onSearch={doFindSheet}
                    style={{width: 300 }}
                    enterButton="查询"
                  />
                  <Button
                    key="config"
                    type="primary"
                    onClick={() => {setConfigVisible(!configVisible)}}
                  >
                    配置
                  </Button>
                  <Button
                    key="export"
                    type="primary"
                    onClick={doExportTable}
                  >
                    导表
                  </Button>

                </Space>
              ]
            }
          />
          <div
            id="historyScrollDiv"
            style={{
              height: 700,
              overflow: "auto",
              padding: "0 0",
              marginLeft: "20%",
              marginRight: "20%",
            }}
          >
            <InfiniteScroll
              next={() => {}}
              hasMore={false}
              loader={<Skeleton avatar paragraph={{rows: 1}} active/>}
              dataLength={resultData.length}
              scrollableTarget={"historyScrollDiv"}
            >
              <ProList<any>
                headerTitle="导表结果"
                style={{height: "auto"}}
                dataSource={resultData}
                actionRef={listActionRef}
                metas={{
                  // title: {
                  //   dataIndex: "title",
                  // },
                  description: {
                    dataIndex: "content",
                  },
                }}
              />
            </InfiniteScroll>
          </div>
        </Space>
      ) || (
        <Result
          status="404"
          title="404"
          subTitle={"连接调试后台失败,无法获取配置表信息"}/>
      )
      }
      <ModalForm
        visible={configVisible}
        onVisibleChange={setConfigVisible}
        onFinish={onSubmitConfig}
        name="导表配置"
      >
        <ProFormText
          label="导表命令"
          name={exportCmdName}
          initialValue={exportConfig?.export_cmd}
        />
        <ProFormText
          label="excel路径"
          name={excelPathName}
          initialValue={exportConfig?.excel_path}
        />
        <ProFormText
          label="客户端表格输出路径"
          name={exportClientPathName}
          initialValue={exportConfig?.export_client_path}
        />
        <ProFormText
          label="服务端表格输出路径"
          name={exportServerPathName}
          initialValue={exportConfig?.export_server_path}
        />
      </ModalForm>
    </Space>
  );
};

export default ExportToLua;
