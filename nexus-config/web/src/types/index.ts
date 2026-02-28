// 命名空间
export interface Namespace {
  id: string;
  name: string;
  description: string;
  created_at?: string;
  updated_at?: string;
}

// 配置格式
export type ConfigFormat = 'yaml' | 'json' | 'toml' | 'properties';

// 配置版本
export interface ConfigVersion {
  namespace: string;
  key: string;
  value: string;
  format: ConfigFormat;
  md5: string;
  version?: number;
  created_at?: string;
  updated_at?: string;
}

// 配置项
export interface ConfigItem {
  namespace: string;
  key: string;
  format: ConfigFormat;
  draft_value?: string;
  draft_md5?: string;
  published_value?: string;
  published_md5?: string;
  has_draft: boolean;
  has_published: boolean;
  created_at?: string;
  updated_at?: string;
}

// 灰度规则
export interface GrayRule {
  id?: number;
  namespace: string;
  key: string;
  percentage: number;
  enabled: boolean;
  created_at?: string;
  updated_at?: string;
}

// API 请求/响应类型
export interface CreateNamespaceRequest {
  id: string;
  name: string;
  description: string;
}

export interface SaveDraftRequest {
  namespace: string;
  key: string;
  value: string;
  format: ConfigFormat;
}

export interface PublishConfigRequest {
  namespace: string;
  key: string;
}

export interface SaveGrayRuleRequest {
  namespace: string;
  key: string;
  percentage: number;
  enabled: boolean;
}

export interface ApiResponse<T = any> {
  code: number;
  message: string;
  data: T;
}
