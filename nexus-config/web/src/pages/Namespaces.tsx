import React, { useEffect, useState } from 'react';
import { Button, Table, Modal, Form, Input, message, Space, Popconfirm } from 'antd';
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons';
import type { Namespace } from '@/types';
import { listNamespaces, createNamespace, deleteNamespace } from '@/api';
import { useNavigate } from 'react-router-dom';

const Namespaces: React.FC = () => {
  const [namespaces, setNamespaces] = useState<Namespace[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [form] = Form.useForm();
  const navigate = useNavigate();

  // 加载命名空间列表
  const loadNamespaces = async () => {
    setLoading(true);
    try {
      const data = await listNamespaces();
      setNamespaces(data);
    } catch (error) {
      console.error('加载命名空间失败:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadNamespaces();
  }, []);

  // 创建命名空间
  const handleCreate = async () => {
    try {
      const values = await form.validateFields();
      await createNamespace(values);
      message.success('命名空间创建成功');
      setModalVisible(false);
      form.resetFields();
      loadNamespaces();
    } catch (error) {
      console.error('创建命名空间失败:', error);
    }
  };

  // 删除命名空间
  const handleDelete = async (id: string) => {
    try {
      await deleteNamespace(id);
      message.success('命名空间删除成功');
      loadNamespaces();
    } catch (error) {
      console.error('删除命名空间失败:', error);
    }
  };

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      key: 'id',
    },
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: Namespace) => (
        <Space>
          <Button
            type="link"
            onClick={() => navigate(`/configs?namespace=${record.id}`)}
          >
            管理配置
          </Button>
          <Popconfirm
            title="确定要删除该命名空间吗？"
            onConfirm={() => handleDelete(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Button type="link" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div style={{ marginBottom: 16 }}>
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => setModalVisible(true)}
        >
          创建命名空间
        </Button>
      </div>

      <Table
        columns={columns}
        dataSource={namespaces}
        rowKey="id"
        loading={loading}
      />

      <Modal
        title="创建命名空间"
        open={modalVisible}
        onOk={handleCreate}
        onCancel={() => {
          setModalVisible(false);
          form.resetFields();
        }}
        okText="创建"
        cancelText="取消"
      >
        <Form form={form} layout="vertical">
          <Form.Item
            label="ID"
            name="id"
            rules={[
              { required: true, message: '请输入命名空间 ID' },
              { pattern: /^[a-z0-9-]+$/, message: 'ID 只能包含小写字母、数字和连字符' },
            ]}
          >
            <Input placeholder="例如: myapp" />
          </Form.Item>
          <Form.Item
            label="名称"
            name="name"
            rules={[{ required: true, message: '请输入命名空间名称' }]}
          >
            <Input placeholder="例如: 我的应用" />
          </Form.Item>
          <Form.Item
            label="描述"
            name="description"
          >
            <Input.TextArea placeholder="命名空间描述" rows={3} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default Namespaces;
