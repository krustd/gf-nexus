import request from './request';
import type { GrayRule, SaveGrayRuleRequest } from '@/types';

// 保存灰度规则
export const saveGrayRule = (data: SaveGrayRuleRequest) => {
  return request.post<any, GrayRule>('/gray/', data);
};

// 获取灰度规则
export const getGrayRule = (namespace: string, key: string) => {
  return request.get<any, GrayRule>('/gray/', {
    params: { namespace, key },
  });
};

// 删除灰度规则
export const deleteGrayRule = (namespace: string, key: string) => {
  return request.delete('/gray/', {
    params: { namespace, key },
  });
};

// 获取灰度规则列表
export const listGrayRules = (namespace: string) => {
  return request.get<any, GrayRule[]>('/gray/list', {
    params: { namespace },
  });
};
