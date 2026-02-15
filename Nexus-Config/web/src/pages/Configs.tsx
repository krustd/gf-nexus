import React, { useEffect, useState } from 'react';
import {
  Button,
  Table,
  Modal,
  Form,
  Input,
  Select,
  message,
  Space,
  Popconfirm,
  Tag,
  Drawer,
} from 'antd';
import {
  PlusOutlined,
  DeleteOutlined,
  EditOutlined,
  EyeOutlined,
  SendOutlined,
} from '@ant-design/icons';
import { useSearchParams } from 'react-router-dom';
import type { ConfigItem, ConfigFormat } from '@/types';
import {
  listConfigs,
  saveDraft,
  publishConfig,
  deleteConfig,
  getDraft,
  getPublished,
} from '@/api';
import { ConfigEditor, DiffViewer } from '@/components';

const { Option } = Select;

const Configs: React.FC = () => {
  const [searchParams] = useSearchParams();
  const namespace = searchParams.get('namespace') || '';

  const [configs, setConfigs] = useState<ConfigItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [createModalVisible, setCreateModalVisible] = useState(false);
  const [editDrawerVisible, setEditDrawerVisible] = useState(false);
  const [diffDrawerVisible, setDiffDrawerVisible] = useState(false);

  const [currentConfig, setCurrentConfig] = useState<ConfigItem | null>(null);
  const [draftValue, setDraftValue] = useState('');
  const [publishedValue, setPublishedValue] = useState('');

  const [form] = Form.useForm();

  // 加载配置列表
  const loadConfigs = async () => {
    if (!namespace) {
      message.warning('请先选择命名空间');
      return;
    }
    setLoading(true);
    try {
      const data = await listConfigs(namespace);
      setConfigs(data);
    } catch (error) {
      console.error('加载配置列表失败:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadConfigs();
  }, [namespace]);

  // 创建配置（保存草稿）
  const handleCreate = async () => {
    try {
      const values = await form.validateFields();
      await saveDraft({
        namespace,
        key: values.key,
        value: values.value,
        format: values.format,
      });
      message.success('配置草稿创建成功');
      setCreateModalVisible(false);
      form.resetFields();
      loadConfigs();
    } catch (error) {
      console.error('创建配置失败:', error);
    }
  };

  // 编辑配置（保存草稿）
  const handleEdit = async (config: ConfigItem) => {
    setCurrentConfig(config);
    try {
      // 加载草稿或已发布的配置
      let value = '';
      if (config.has_draft) {
        const draft = await getDraft(namespace, config.key);
        value = draft.value;
      } else if (config.has_published) {
        const published = await getPublished(namespace, config.key);
        value = published.value;
      }
      setDraftValue(value);
      setEditDrawerVisible(true);
    } catch (error) {
      console.error('加载配置失败:', error);
    }
  };

  // 保存草稿
  const handleSaveDraft = async () => {
    if (!currentConfig) return;
    try {
      await saveDraft({
        namespace,
        key: currentConfig.key,
        value: draftValue,
        format: currentConfig.format,
      });
      message.success('草稿保存成功');
      loadConfigs();
    } catch (error) {
      console.error('保存草稿失败:', error);
    }
  };

  // 发布配置
  const handlePublish = async (key: string) => {
    try {
      await publishConfig({ namespace, key });
      message.success('配置发布成功');
      loadConfigs();
    } catch (error) {
      console.error('发布配置失败:', error);
    }
  };

  // 对比视图
  const handleViewDiff = async (config: ConfigItem) => {
    setCurrentConfig(config);
    try {
      const [draft, published] = await Promise.all([
        config.has_draft ? getDraft(namespace, config.key) : null,
        config.has_published ? getPublished(namespace, config.key) : null,
      ]);
      setDraftValue(draft?.value || '');
      setPublishedValue(published?.value || '');
      setDiffDrawerVisible(true);
    } catch (error) {
      console.error('加载配置对比失败:', error);
    }
  };

  // 删除配置
  const handleDelete = async (key: string) => {
    try {
      await deleteConfig(namespace, key);
      message.success('配置删除成功');
      loadConfigs();
    } catch (error) {
      console.error('删除配置失败:', error);
    }
  };

  const columns = [
    {
      title: '配置键',
      dataIndex: 'key',
      key: 'key',
    },
    {
      title: '格式',
      dataIndex: 'format',
      key: 'format',
      render: (format: ConfigFormat) => format.toUpperCase(),
    },
    {
      title: '状态',
      key: 'status',
      render: (_: any, record: ConfigItem) => (
        <Space>
          {record.has_published && <Tag color="green">已发布</Tag>}
          {record.has_draft && <Tag color="orange">有草稿</Tag>}
        </Space>
      ),
    },
    {
      title: '更新时间',
      dataIndex: 'updated_at',
      key: 'updated_at',
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: ConfigItem) => (
        <Space>
          <Button
            type="link"
            icon={<EditOutlined />}
            onClick={() => handleEdit(record)}
          >
            编辑
          </Button>
          <Button
            type="link"
            icon={<EyeOutlined />}
            onClick={() => handleViewDiff(record)}
            disabled={!record.has_draft && !record.has_published}
          >
            对比
          </Button>
          <Button
            type="link"
            icon={<SendOutlined />}
            onClick={() => handlePublish(record.key)}
            disabled={!record.has_draft}
          >
            发布
          </Button>
          <Popconfirm
            title="确定要删除该配置吗？"
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
      <div style={{ marginBottom: 16 }}>
        <Space>
          <span>命名空间: <strong>{namespace}</strong></span>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setCreateModalVisible(true)}
          >
            创建配置
          </Button>
        </Space>
      </div>

      <Table
        columns={columns}
        dataSource={configs}
        rowKey="key"
        loading={loading}
      />

      {/* 创建配置对话框 */}
      <Modal
        title="创建配置"
        open={createModalVisible}
        onOk={handleCreate}
        onCancel={() => {
          setCreateModalVisible(false);
          form.resetFields();
        }}
        width={800}
        okText="创建"
        cancelText="取消"
      >
        <Form form={form} layout="vertical">
          <Form.Item
            label="配置键"
            name="key"
            rules={[{ required: true, message: '请输入配置键' }]}
          >
            <Input placeholder="例如: app.yaml" />
          </Form.Item>
          <Form.Item
            label="格式"
            name="format"
            initialValue="yaml"
            rules={[{ required: true, message: '请选择配置格式' }]}
          >
            <Select>
              <Option value="yaml">YAML</Option>
              <Option value="json">JSON</Option>
              <Option value="toml">TOML</Option>
              <Option value="properties">Properties</Option>
            </Select>
          </Form.Item>
          <Form.Item
            label="配置内容"
            name="value"
            initialValue=""
          >
            <Input.TextArea rows={10} placeholder="请输入配置内容" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 编辑配置抽屉 */}
      <Drawer
        title={`编辑配置: ${currentConfig?.key}`}
        placement="right"
        width="60%"
        open={editDrawerVisible}
        onClose={() => setEditDrawerVisible(false)}
        extra={
          <Space>
            <Button onClick={() => setEditDrawerVisible(false)}>取消</Button>
            <Button type="primary" onClick={handleSaveDraft}>
              保存草稿
            </Button>
          </Space>
        }
      >
        {currentConfig && (
          <ConfigEditor
            value={draftValue}
            format={currentConfig.format}
            onChange={(value) => setDraftValue(value || '')}
            height="calc(100vh - 150px)"
          />
        )}
      </Drawer>

      {/* 对比视图抽屉 */}
      <Drawer
        title={`配置对比: ${currentConfig?.key}`}
        placement="right"
        width="80%"
        open={diffDrawerVisible}
        onClose={() => setDiffDrawerVisible(false)}
      >
        {currentConfig && (
          <DiffViewer
            originalValue={publishedValue}
            modifiedValue={draftValue}
            format={currentConfig.format}
            originalTitle="已发布版本"
            modifiedTitle="草稿版本"
            height="calc(100vh - 150px)"
          />
        )}
      </Drawer>
    </div>
  );
};

export default Configs;
