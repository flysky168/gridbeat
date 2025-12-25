// src/api/system.ts
import axios from 'axios'
import http from './http'
import type { UserInfo } from '@/store/auth'
import { DEFAULT_SESSION_TIMEOUT_MINUTES } from '@/config/auth'

const mockEnabled = import.meta.env.VITE_USE_MOCK === 'true'

/** 版本信息，给版本弹窗用 */
export interface VersionInfo {
  productName?: string
  Version?: string
  buildTime?: string
  gitCommit?: string
  copyright?: string
  extra?: string
}

/** 和后端 /auth/me 或 /system/userinfo 对齐的返回结构 */
export interface SystemUserInfoResponse extends UserInfo {
  // 会话超时时长（分钟），和 Basic 设置里的 sessionTimeoutMinutes 保持一致
  sessionTimeoutMinutes?: number
}

/** 获取版本信息 */
export async function getVersionInfoApi(): Promise<VersionInfo> {
  if (mockEnabled) {
    return Promise.resolve({
      productName: 'SmartLogger5000A',
      Version: 'V100R025C10B604',
      buildTime: '2025-11-23 10:00:00',
      gitCommit: 'mock-commit-hash',
      copyright: '© 2025 Your Company. All rights reserved.',
      extra: 'Mock 环境下的版本信息',
    })
  }

  return http.get('/system/version', { })
}

/** 获取当前登录用户信息（含角色 + 会话超时配置） */
export async function getUserInfoApi(): Promise<SystemUserInfoResponse> {
  if (mockEnabled) {
    // mock：给一个 admin 用户 + 默认超时
    return Promise.resolve({
      username: 'root',
      token: '',
      jti: '',
      idle_timeout_seconds: 2000,
      type: 'web',
      roles: ['admin'],
      sessionTimeoutMinutes: DEFAULT_SESSION_TIMEOUT_MINUTES,
    })
  }

  // 实际接口：按你后端定义的路径改，比如 /api/auth/me 或 /api/system/userinfo
  const { data } = await axios.get<SystemUserInfoResponse>('/api/system/userinfo')

  // 给超时时长一个兜底
  if (!data.sessionTimeoutMinutes || data.sessionTimeoutMinutes <= 0) {
    data.sessionTimeoutMinutes = DEFAULT_SESSION_TIMEOUT_MINUTES
  }

  return data
}