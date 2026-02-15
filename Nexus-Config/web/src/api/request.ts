import axios, { AxiosInstance, AxiosResponse } from 'axios';
import { message } from 'antd';

// 创建 axios 实例
const request: AxiosInstance = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// 请求拦截器
request.interceptors.request.use(
  (config) => {
    return config;
  },
  (error) => {
    console.error('Request error:', error);
    return Promise.reject(error);
  }
);

// 响应拦截器
request.interceptors.response.use(
  (response: AxiosResponse) => {
    const res = response.data;

    // 后端返回格式: { code: 0, msg: "success", data: ... }
    if (res.code !== undefined && res.code !== 0) {
      // 业务错误
      message.error(res.msg || '请求失败');
      return Promise.reject(new Error(res.msg || '请求失败'));
    }

    // 返回实际数据
    return res.data;
  },
  (error) => {
    if (error.response) {
      const { status, data } = error.response;
      const errorMessage = data?.msg || data?.message || `请求失败 (${status})`;
      message.error(errorMessage);
    } else if (error.request) {
      message.error('网络错误，请检查网络连接');
    } else {
      message.error('请求配置错误');
    }
    return Promise.reject(error);
  }
);

export default request;
