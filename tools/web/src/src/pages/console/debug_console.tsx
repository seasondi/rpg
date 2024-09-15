import React, {useEffect, useRef, useState} from 'react';
import {Space, Checkbox, Input, Button, Alert, Tag, Skeleton} from 'antd';
import ProForm, {ProFormInstance, ProFormTextArea} from "@ant-design/pro-form";
import {CheckboxChangeEvent} from "antd/es/checkbox";
import ProList from '@ant-design/pro-list';
import moment from "moment";
import {ActionType} from "@ant-design/pro-table";
import InfiniteScroll from "react-infinite-scroll-component";

export type ConsoleValueType = {
  name: string;
  data: string;
}

export type ConsoleProps = {
  name: string;
  onCommandSend: ((target: string, data: string) => void);
  response?: ConsoleValueType|undefined;
  onHotfix?: (() => void);
}

const DebugConsole: React.FC<ConsoleProps> = (props) => {
  const formRef = useRef<ProFormInstance>()
  const listActionRef = useRef<ActionType>()

  const [inputValue, setInputValue] = useState<string>()

  const [multiMode, setMultiMode] = useState<boolean>(false)

  const [showData, setShowData] = useState<any[]>([])

  const addNewContent = (action: string, val: string) => {
    if(val === undefined) {
      return
    }
    if(val === "" && action === "SEND") {
        return
    }
    const color = action === "SEND" && "#5BD8A6" || "green";
    const data = showData;
    const info = [{
      title: moment().format("YYYY-MM-DD HH:mm:ss"),
      subTitle: <Tag color={color}>{action}</Tag>,
      content: <pre>{val}</pre>,
    }];
    // setShowData(data.concat(info));
    setShowData(info.concat(data));
    listActionRef.current?.reloadAndRest?.();
    // if(action == "RECV") {
    //   setTimeout(
    //     () => {
    //       document.getElementById("historyScrollDiv")?.scroll({
    //         behavior: "smooth",
    //         left: 0,
    //         top: 100000,
    //       });
    //     }, 100)
    // }
  }

  const onMultiModeChange = (e: CheckboxChangeEvent) => {
    setMultiMode(e.target.checked)
  }

  const onInputContentChange = (e: any) => {
    setInputValue(e.target.value)
  }

  const onSubmitMultiCommand = () => {
    const val = formRef.current?.getFieldValue(props.name)
    if(val === undefined || val === "") {
      return
    }
    addNewContent("SEND", val)
    formRef.current?.resetFields()
    props.onCommandSend(props.name, val)
  }

  const onSubmitSingleCommand = (e: any) => {
    if(e.target.value === undefined || e.target.value === "") {
      return
    }
    addNewContent("SEND", e.target.value)
    setInputValue("")
    props.onCommandSend(props.name, e.target.value)
  }

  const onClickClearListButton = () => {
    setShowData([])
    listActionRef.current?.reloadAndRest?.()
  }

  const onClickHotfix = () => {
    props.onHotfix?.();
  }

  useEffect(() => {
    if(props.response && props.response.name == props.name) {
      addNewContent("RECV", props.response.data)
    }
  }, [props.response])

  return (
    <Space direction={"vertical"} style={{width: "100%"}}>
      <Space direction={"horizontal"} size={20}>
        <Button
          type={"primary"}
          onClick={onClickHotfix}
        >
          热更
        </Button>
        <Alert message={`当前进程: ${props.name}`} type={"success"}/>
        <Checkbox
          name={props.name}
          onChange={onMultiModeChange}
        >
          多行模式
        </Checkbox>
        <Button
          type={"primary"}
          onClick={onClickClearListButton}
        >
          清空历史记录
        </Button>
        {multiMode && (<Button type={"primary"} onClick={onSubmitMultiCommand}>
          提交
        </Button>)}

      </Space>
      {!multiMode && (<Input
        allowClear={true}
        placeholder={"输入命令后回车提交命令"}
        onPressEnter={onSubmitSingleCommand}
        value={inputValue}
        onChange={onInputContentChange}
      />) ||
      (<ProForm
        formRef={formRef}
        submitter={{
          render: () => { return [] }
        }}
      >
        <ProFormTextArea
          initialValue={""}
          allowClear={true}
          fieldProps={{rows: 5}}
          placeholder={"输入命令后点击提交按钮提交命令"}
          name={props.name}
        >
        </ProFormTextArea>
      </ProForm> )
      }

      <div>
        <div
          id="historyScrollDiv"
          style={{
            height: 600,
            overflow: "auto",
            padding: "0 0",
          }}
        >
          <InfiniteScroll
            next={() => {}}
            hasMore={false}
            loader={<Skeleton avatar paragraph={{rows: 1}} active/>}
            dataLength={showData.length}
            scrollableTarget={"historyScrollDiv"}
          >
            <ProList<any>
              style={{height: "auto"}}
              dataSource={showData}
              actionRef={listActionRef}
              grid={{ gutter: 16, column: 1}}
              metas={{
                title: {
                  dataIndex: "title",
                },
                subTitle: {
                  dataIndex: "subTitle",
                },
                content: {
                  dataIndex: "content",
                },
              }}
            />
          </InfiniteScroll>
        </div>
      </div>
    </Space>
  );
};

export default DebugConsole;
