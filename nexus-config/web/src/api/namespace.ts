import request from './request';
import type { Namespace, CreateNamespaceRequest } from '@/types';

// 创建命名空间
export const createNamespace = (data: CreateNamespaceRequest) => {
  return request.post<any, Namespace>('/namespaces/', data);
};

// 获取命名空间列表
export const listNamespaces = () => {
  return request.get<any, Namespace[]>('/namespaces/');
};

// 获取命名空间详情
export const getNamespace = (id: string) => {
  return request.get<any, Namespace>(`/namespaces/${id}`);
};

// 删除命名空间
export const deleteNamespace = (id: string) => {
  return request.delete(`/namespaces/${id}`);
};
