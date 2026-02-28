import request from './request';
import type { ConfigVersion, ConfigItem, SaveDraftRequest, PublishConfigRequest } from '@/types';

// 保存草稿
export const saveDraft = (data: SaveDraftRequest) => {
  return request.post<any, ConfigVersion>('/configs/draft', data);
};

// 获取草稿
export const getDraft = (namespace: string, key: string) => {
  return request.get<any, ConfigVersion>('/configs/draft', {
    params: { namespace, key },
  });
};

// 发布配置
export const publishConfig = (data: PublishConfigRequest) => {
  return request.post<any, ConfigVersion>('/configs/publish', data);
};

// 获取已发布配置
export const getPublished = (namespace: string, key: string) => {
  return request.get<any, ConfigVersion>('/configs/published', {
    params: { namespace, key },
  });
};

// 获取配置列表
export const listConfigs = (namespace: string) => {
  return request.get<any, ConfigItem[]>('/configs/list', {
    params: { namespace },
  });
};

// 删除配置
export const deleteConfig = (namespace: string, key: string) => {
  return request.delete('/configs/', {
    params: { namespace, key },
  });
};
