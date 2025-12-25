// src/api/auth.ts
import axios from 'axios'
import type { UserInfo } from '@/store/auth'

const mockEnabled = import.meta.env.VITE_USE_MOCK === 'true'

export interface LoginPayload {
  username: string
  password: string
}

export interface LoginResponse {
  code: number
  message: string
  data: UserInfo
}

/**
 * 登录接口
 * 后端建议返回结构：
 * {
 *   "token": "9944b09199c62bcf9418ad846dd0e4bbdfc6ee4b",
 *   "user": { "id": "84a35e05-531f-4d96-8d5b-bc8a7a358493", "username": "root" },
 * }
 */
export async function loginApi(payload: LoginPayload): Promise<LoginResponse> {
  if (mockEnabled) {
    // mock 模式：本地假数据
    return Promise.resolve({
      
    "code": 0,
    "message": "ok",
    "data": {
        "username": "root",
        "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJncmlkYmVhdCIsInN1YiI6IiUhcyh1aW50PTEpIiwibmJmIjoxNzY2MTkxNDIyLCJpYXQiOjE3NjYxOTE0MjIsImp0aSI6ImVlOWQ1YWUxLTdmYWEtNGQ5OS1iYWFmLWI0Nzk1ZDE0MTQ2MyIsInVzZXJuYW1lIjoicm9vdCIsImlzX3Jvb3QiOnRydWUsInR5cCI6IndlYiJ9.j-S52GeCRQNKC_amcxwb0VVbK0s0pTC2lyBRGE7WQpk",
        "jti": "ee9d5ae1-7faa-4d99-baaf-b4795d141463",
        "idle_timeout_seconds": 1800,
        "type": "web"
      }
    })
  }

  const { data } = await axios.post<LoginResponse>('/api/v1/auth/login', payload)
  return data
}

/**
 * 登出接口（可选）
 * 如果后端有登出逻辑可以实现；没有的话前端直接清 auth 即可。
 */
export async function logoutApi(): Promise<void> {
  if (mockEnabled) {
    return Promise.resolve()
  }
  await axios.post('/api/auth/logout')
}