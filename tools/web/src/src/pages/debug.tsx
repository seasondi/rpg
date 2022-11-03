import React from 'react';
import { PageContainer } from '@ant-design/pro-layout';
import {Tabs} from 'antd';
import ConsoleTabPanes from "@/pages/console/consoles";

const Debug: React.FC = () => {
  const { TabPane } = Tabs

  return (
    <PageContainer>
      <Tabs defaultActiveKey={"console"}>
        <TabPane tab="控制台" key="console">
          <ConsoleTabPanes/>
        </TabPane>
      </Tabs>
    </PageContainer>
  );
};

export default Debug;
