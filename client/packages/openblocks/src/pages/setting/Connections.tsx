import { Form, Input } from "antd";
import { EditPopover, PointIcon, TacoButton, CustomModal } from "openblocks-design";
import { useEffect, useState } from "react";
import { Table } from "components/Table";
import { trans } from "i18n";
import styled from "styled-components";
import ConnectionApi, { Connection } from "api/connectionApi";
import {
  Level1SettingPageContentWithList,
  Level1SettingPageTitleWithBtn,
} from "./styled";

const OperationWrapper = styled.div`
  display: flex;
  justify-content: flex-end;
`;

export function ConnectionsSetting() {
  const [list, setList] = useState<Connection[]>([]);
  const [visible, setVisible] = useState(false);
  const [current, setCurrent] = useState<Connection | null>(null);
  const [form] = Form.useForm();

  const fetchList = () => {
    ConnectionApi.list().then((resp) => {
      if (resp.data?.items) {
        setList(resp.data.items);
      }
    });
  };

  useEffect(() => {
    fetchList();
  }, []);

  const openModal = (item?: Connection) => {
    setCurrent(item || null);
    if (item) {
      try {
        const cfg = JSON.parse(item.config);
        form.setFieldsValue({ name: item.name, ...cfg });
      } catch {
        form.setFieldsValue({ name: item.name });
      }
    } else {
      form.resetFields();
      form.setFieldsValue({ port: 1433 });
    }
    setVisible(true);
  };

  const handleSubmit = () => {
    form.validateFields().then((values) => {
      const payload = {
        name: values.name,
        type: "mssql",
        config: JSON.stringify({
          host: values.host,
          port: Number(values.port),
          user: values.user,
          password: values.password,
          database: values.database,
        }),
      };
      const req = current
        ? ConnectionApi.update(current.id, payload)
        : ConnectionApi.create(payload);
      req.then(() => {
        setVisible(false);
        fetchList();
      });
    });
  };

  const columns = [
    { title: trans("query.datasourceName"), dataIndex: "name" },
    { title: trans("query.connectionType"), dataIndex: "type" },
    {
      title: " ",
      dataIndex: "operation",
      width: "72px",
      render: (_: any, record: Connection) => (
        <OperationWrapper>
          <EditPopover
            items={[
              { text: trans("edit"), onClick: () => openModal(record) },
              {
                text: trans("delete"),
                type: "delete",
                onClick: () => {
                  ConnectionApi.delete(record.id).then(fetchList);
                },
              },
            ]}
          >
            <PointIcon tabIndex={-1} />
          </EditPopover>
        </OperationWrapper>
      ),
    },
  ];

  return (
    <Level1SettingPageContentWithList>
      <Level1SettingPageTitleWithBtn>
        {trans("settings.connections")}
        <TacoButton buttonType="primary" onClick={() => openModal()}>
          {trans("add")}
        </TacoButton>
      </Level1SettingPageTitleWithBtn>
      <Table
        tableLayout="auto"
        scroll={{ x: "100%" }}
        pagination={false}
        columns={columns}
        dataSource={list.map((item, i) => ({ ...item, key: i }))}
      />
      <CustomModal
        width="448px"
        visible={visible}
        title={trans("settings.connections")}
        onOk={handleSubmit}
        onCancel={() => setVisible(false)}
        destroyOnClose
        draggable
      >
        <Form layout="vertical" form={form}>
          <Form.Item
            name="name"
            label={trans("query.datasourceName")}
            rules={[{ required: true }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="host"
            label={trans("query.host")}
            rules={[{ required: true }]}
          >
            <Input />
          </Form.Item>
          <Form.Item name="port" label={trans("query.port")}> 
            <Input />
          </Form.Item>
          <Form.Item name="user" label={trans("query.userName")}> 
            <Input />
          </Form.Item>
          <Form.Item name="password" label={trans("query.password")}> 
            <Input.Password />
          </Form.Item>
          <Form.Item name="database" label={trans("query.databaseName")}> 
            <Input />
          </Form.Item>
        </Form>
      </CustomModal>
    </Level1SettingPageContentWithList>
  );
}

export default ConnectionsSetting;
