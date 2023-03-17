import React, {useRef, useState} from 'react';
import {gmCommandArg} from "@/pages/gm";
import {Card, Col, Empty, Row, Space} from "antd";
import ProForm, {ProFormDigit, ProFormInstance, ProFormSwitch, ProFormText} from "@ant-design/pro-form";

export type GMCommandProps = {
  command?: string;
  name?: string;
  args?: gmCommandArg[];
  response?: string;
  onSubmitCommand?: ((command: string, args: any[]) => void);
}

const GMCommand: React.FC<GMCommandProps> = (props) => {
  const formRef = useRef<ProFormInstance>()

  const [formValue, setFormValue] = useState<Record<string, any>>({})

  const formItemLayout = {
    labelCol: {span: 4},
    wrapperCol: {span: 14},
  }

  const onSubmit = (command: string, values: any) => {
    if(props.onSubmitCommand) {
      props.onSubmitCommand(command, values)
    }
  }

  return (
    <Space direction={"vertical"} style={{width: "100%"}}>
      <Card style={{height: "100%"}} title={props.name}>
        {props.command === undefined && (
          <Empty description={"未选择GM指令"}/>
        ) ||
        (
          <ProForm<any>
            {...formItemLayout}
            layout={"horizontal"}
            key={props.command}
            formRef={formRef}
            submitter={{
              render: (_, dom) => (
                <Row>
                  <Col span={14} offset={4}>
                    <Space>{dom}</Space>
                  </Col>
                </Row>
              )
            }}
            onFinish={async (values) => {onSubmit(props.command || "", values)}}
            onValuesChange={async (values) => {
              const allValues = formValue
              const newValues = formValue[props.command || ""] || {}
              for(const k in values) {
                newValues[k] = values[k]
              }
              allValues[props.command || ""] = newValues
              setFormValue(allValues)
            }}
            initialValues={formValue[props.command || ""] || {}}
            onReset={async() => {
              const allValues = formValue
              allValues[props.command || ""] = {}
              setFormValue(allValues)
              formRef.current?.resetFields()
            }}
          >
            {
              props.args !== undefined && props.args.map(arg => (
                arg.type === "number" && (<ProFormDigit
                  key={arg.index}
                  width="md"
                  name={arg.index}
                  label={arg.name}
                />) ||
                arg.type === "string" && (<ProFormText
                  key={arg.index}
                  width="md"
                  name={arg.index}
                  label={arg.name}
                />) ||
                arg.type === "bool" && (<ProFormSwitch
                  key={arg.index}
                  name={arg.index}
                  label={arg.name}
                />)
              ))
            }
          </ProForm>
        )}
      </Card>
      {props.command && <Card style={{width: "100%"}} title={"输出结果"}>
        <pre>{props.response}</pre>
      </Card>}
    </Space>
  );
};

export default GMCommand;
