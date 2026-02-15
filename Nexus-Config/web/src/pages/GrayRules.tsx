import React, { useEffect, useState } from 'react';
import {
  Button,
  Table,
  Modal,
  Form,
  Input,
  Slider,
  Switch,
  message,
  Space,
  Popconfirm,
  Tag,
  Card,
  Progress,
} from 'antd';
import { PlusOutlined, DeleteOutlined, EditOutlined } from '@ant-design/icons';
import { useSearchParams } from 'react-router-dom';
import type { GrayRule } from '@/types';
import { listGrayRules, saveGrayRule, deleteGrayRule } from '@/api';

const GrayRules: React.FC = () => {
  const [searchParams] = useSearchParams();
  const namespace = searchParams.get('namespace') || '';

  const [rules, setRules] = useState<GrayRule[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingRule, setEditingRule] = useState<GrayRule | null>(null);

  const [form] = Form.useForm();

  // 加载灰度规则列表
  const loadRules = async () => {
    if (!namespace) {
      message.warning('请先选择命名空间');
      return;
    }
    setLoading(true);
    try {
      const data = await listGrayRules(namespace);
      setRules(data);
    } catch (error) {
      console.error('加载灰度规则失败:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadRules();
  }, [namespace]);

  // 打开创建/编辑对话框
  const handleOpenModal = (rule?: GrayRule) => {
    if (rule) {
      setEditingRule(rule);
      form.setFieldsValue({
        key: rule.key,
        percentage: rule.percentage,
        enabled: rule.enabled,
      });
    } else {
      setEditingRule(null);
      form.resetFields();
    }
    setModalVisible(true);
  };

  // 保存灰度规则
  const handleSave = async () => {
    try {
      const values = await form.validateFields();
      await saveGrayRule({
        namespace,
        key: values.key,
        percentage: values.percentage,
        enabled: values.enabled,
      });
      message.success('灰度规则保存成功');
      setModalVisible(false);
      form.resetFields();
      setEditingRule(null);
      loadRules();
    } catch (error) {
      console.error('保存灰度规则失败:', error);
    }
  };

  // 删除灰度规则
  const handleDelete = async (key: string) => {
    try {
      await deleteGrayRule(namespace, key);
      message.success('灰度规则删除成功');
      loadRules();
    } catch (error) {
      console.error('删除灰度规则失败:', error);
    }
  };

  // 快速切换启用状态
  const handleToggleEnabled = async (rule: GrayRule) => {
    try {
      await saveGrayRule({
        namespace: rule.namespace,
        key: rule.key,
        percentage: rule.percentage,
        enabled: !rule.enabled,
      });
      message.success('灰度规则状态已更新');
      loadRules();
    } catch (error) {
      console.error('更新灰度规则失败:', error);
    }
  };

  const columns = [
    {
      title: '配置键',
      dataIndex: 'key',
      key: 'key',
    },
    {
      title: '灰度百分比',
      dataIndex: 'percentage',
      key: 'percentage',
      render: (percentage: number) => (
        <div style={{ width: 200 }}>
          <Progress
            percent={percentage}
            size="small"
            status={percentage > 0 ? 'active' : 'normal'}
          />
        </div>
      ),
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      key: 'enabled',
      render: (enabled: boolean) => (
        <Tag color={enabled ? 'green' : 'default'}>
          {enabled ? '已启用' : '已禁用'}
        </Tag>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: GrayRule) => (
        <Space>
          <Switch
            checked={record.enabled}
            onChange={() => handleToggleEnabled(record)}
            checkedChildren="启用"
            unCheckedChildren="禁用"
          />
          <Button
            type="link"
            icon={<EditOutlined />}
            onClick={() => handleOpenModal(record)}
          >
            编辑
          </Button>
          <Popconfirm
            title="确定要删除该灰度规则吗？"
            onConfirm={() => handleDelete(record.key)}
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

  if (!namespace) {
    return (
      <div style={{ textAlign: 'center', padding: '50px' }}>
        <p>请先从左侧选择命名空间</p>
      </div>
    );
  }

  return (
    <div>
      <Card style={{ marginBottom: 16 }}>
        <h3>灰度发布说明</h3>
        <p>
          灰度发布允许您将新配置版本（草稿）按百分比逐步发布给客户端。
          系统根据客户端 ID 的哈希值自动分流：
        </p>
        <ul>
          <li>命中灰度的客户端将收到草稿版本</li>
          <li>未命中的客户端将继续使用已发布版本</li>
          <li>灰度百分比范围: 0-100，0 表示关闭灰度，100 表示全量灰度</li>
        </ul>
      </Card>

      <div style={{ marginBottom: 16 }}>
        <Space>
          <span>命名空间: <strong>{namespace}</strong></span>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => handleOpenModal()}
          >
            创建灰度规则
          </Button>
        </Space>
      </div>

      <Table
        columns={columns}
        dataSource={rules}
        rowKey="key"
        loading={loading}
      />

      <Modal
        title={editingRule ? '编辑灰度规则' : '创建灰度规则'}
        open={modalVisible}
        onOk={handleSave}
        onCancel={() => {
          setModalVisible(false);
          form.resetFields();
          setEditingRule(null);
        }}
        okText="保存"
        cancelText="取消"
      >
        <Form form={form} layout="vertical">
          <Form.Item
            label="配置键"
            name="key"
            rules={[{ required: true, message: '请输入配置键' }]}
          >
            <Input
              placeholder="例如: app.yaml"
              disabled={!!editingRule}
            />
          </Form.Item>
          <Form.Item
            label="灰度百分比"
            name="percentage"
            initialValue={0}
            rules={[{ required: true, message: '请设置灰度百分比' }]}
          >
            <Slider
              min={0}
              max={100}
              marks={{
                0: '0%',
                25: '25%',
                50: '50%',
                75: '75%',
                100: '100%',
              }}
            />
          </Form.Item>
          <Form.Item
            label="启用状态"
            name="enabled"
            valuePropName="checked"
            initialValue={true}
          >
            <Switch checkedChildren="启用" unCheckedChildren="禁用" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default GrayRules;
