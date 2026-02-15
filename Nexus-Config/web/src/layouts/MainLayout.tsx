import React, { useState, useEffect } from 'react';
import { Layout, Menu, theme } from 'antd';
import {
  AppstoreOutlined,
  FileTextOutlined,
  BranchesOutlined,
} from '@ant-design/icons';
import { Outlet, useNavigate, useLocation } from 'react-router-dom';
import type { Namespace } from '@/types';
import { listNamespaces } from '@/api';

const { Header, Sider, Content } = Layout;

const MainLayout: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const {
    token: { colorBgContainer },
  } = theme.useToken();

  const [collapsed, setCollapsed] = useState(false);
  const [namespaces, setNamespaces] = useState<Namespace[]>([]);
  const [selectedKey, setSelectedKey] = useState<string>('namespaces');

  // 加载命名空间列表
  useEffect(() => {
    const loadNamespaces = async () => {
      try {
        const data = await listNamespaces();
        setNamespaces(data);
      } catch (error) {
        console.error('加载命名空间失败:', error);
      }
    };
    loadNamespaces();
  }, []);

  // 根据路由更新选中的菜单项
  useEffect(() => {
    const path = location.pathname;
    if (path === '/') {
      setSelectedKey('namespaces');
    } else if (path.startsWith('/configs')) {
      setSelectedKey('configs');
    } else if (path.startsWith('/gray')) {
      setSelectedKey('gray');
    }
  }, [location]);

  const menuItems = [
    {
      key: 'namespaces',
      icon: <AppstoreOutlined />,
      label: '命名空间管理',
      onClick: () => navigate('/'),
    },
    {
      key: 'configs',
      icon: <FileTextOutlined />,
      label: '配置管理',
      onClick: () => {
        const namespace = namespaces[0]?.id || '';
        navigate(`/configs?namespace=${namespace}`);
      },
    },
    {
      key: 'gray',
      icon: <BranchesOutlined />,
      label: '灰度发布',
      onClick: () => {
        const namespace = namespaces[0]?.id || '';
        navigate(`/gray?namespace=${namespace}`);
      },
    },
  ];

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Header style={{ display: 'flex', alignItems: 'center', color: '#fff' }}>
        <div style={{ fontSize: '20px', fontWeight: 'bold', marginRight: '20px' }}>
          Nexus-Config 管理中心
        </div>
      </Header>
      <Layout>
        <Sider
          collapsible
          collapsed={collapsed}
          onCollapse={setCollapsed}
          width={200}
          style={{ background: colorBgContainer }}
        >
          <Menu
            mode="inline"
            selectedKeys={[selectedKey]}
            style={{ height: '100%', borderRight: 0 }}
            items={menuItems}
          />
        </Sider>
        <Layout style={{ padding: '24px' }}>
          <Content
            style={{
              padding: 24,
              margin: 0,
              minHeight: 280,
              background: colorBgContainer,
            }}
          >
            <Outlet />
          </Content>
        </Layout>
      </Layout>
    </Layout>
  );
};

export default MainLayout;
