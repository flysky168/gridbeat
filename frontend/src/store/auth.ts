// src/store/auth.ts
import { defineStore } from 'pinia'
import { ElMessage } from 'element-plus'
import { DEFAULT_SESSION_TIMEOUT_MINUTES } from '@/config/auth'

export interface UserInfo {
  username: string
  token: string
  jti: string
  idle_timeout_seconds: number
  type: string
}

interface AuthState {
  token: string | null
  user: UserInfo | null

  // 登录超时时长（分钟）
  sessionTimeoutMinutes: number

  // 最近一次活跃时间戳（ms）
  lastActivityAt: number | null

  // 超时定时器 id
  timeoutTimer: number | null
}

// 本地持久化 key
const AUTH_STORAGE_KEY = 'gateway_auth_v1'

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({
    token: null,
    user: null,
    sessionTimeoutMinutes: DEFAULT_SESSION_TIMEOUT_MINUTES,
    lastActivityAt: null,
    timeoutTimer: null,
  }),

  getters: {
    isAuthenticated: (state) => !!state.token,
  },

  actions: {
    /**
     * 登录：login API 直接传 token + user
     */
    async login(payload: {
      token: string
      user: UserInfo
    }) {
      const { token, user } = payload

      this.token = token
      this.user = user

      this.startSessionTimer()
    },

    logout() {
      this.token = null
      this.user = null
      this.lastActivityAt = null
      this.clearSessionTimer()
      this.clearStorage()
    },

    /**
     * “基本设置”页面调用：更新当前会话的超时时间
     */
    setSessionTimeout(minutes: number) {
      if (!minutes || minutes <= 0) {
        this.sessionTimeoutMinutes = DEFAULT_SESSION_TIMEOUT_MINUTES
      } else {
        this.sessionTimeoutMinutes = minutes
      }

      if (this.token) {
        this.startSessionTimer()
      }
    },

    /**
     * 每次路由跳转 / 重要操作时调用，重置超时计时器
     */
    touch() {
      if (!this.token) return
      this.startSessionTimer()
    },

    /**
     * 启动/重启会话超时定时器，并写入 localStorage
     */
    startSessionTimer() {
      this.clearSessionTimer()
      if (!this.token) return

      const minutes =
        this.sessionTimeoutMinutes > 0
          ? this.sessionTimeoutMinutes
          : DEFAULT_SESSION_TIMEOUT_MINUTES

      const timeoutMs = minutes * 60 * 1000
      this.lastActivityAt = Date.now()

      // 持久化当前会话状态
      this.saveToStorage()

      this.timeoutTimer = window.setTimeout(() => {
        this.token = null
        this.user = null
        this.lastActivityAt = null
        this.timeoutTimer = null
        this.clearStorage()
        ElMessage.warning('登录已超时，请重新登录')
      }, timeoutMs) as unknown as number
    },

    clearSessionTimer() {
      if (this.timeoutTimer != null) {
        clearTimeout(this.timeoutTimer)
        this.timeoutTimer = null
      }
    },

    /**
     * 持久化当前 auth 状态到 localStorage
     */
    saveToStorage() {
      if (typeof window === 'undefined') return

      const payload = {
        token: this.token,
        user: this.user,
        sessionTimeoutMinutes: this.sessionTimeoutMinutes,
        lastActivityAt: this.lastActivityAt,
      }

      try {
        localStorage.setItem(AUTH_STORAGE_KEY, JSON.stringify(payload))
      } catch (e) {
        console.warn('save auth to storage failed', e)
      }
    },

    /**
     * 清除本地存储
     */
    clearStorage() {
      if (typeof window === 'undefined') return
      try {
        localStorage.removeItem(AUTH_STORAGE_KEY)
      } catch (e) {
        console.warn('clear auth storage failed', e)
      }
    },

    /**
     * 应用启动时调用：从 localStorage 恢复登录状态
     */
    initFromStorage() {
      if (typeof window === 'undefined') return

      const raw = localStorage.getItem(AUTH_STORAGE_KEY)
      if (!raw) return

      try {
        const data = JSON.parse(raw) as {
          token: string | null
          user: UserInfo | null
          sessionTimeoutMinutes?: number
          lastActivityAt?: number | null
        }

        if (!data.token) return

        this.token = data.token
        this.user = data.user || null
        this.sessionTimeoutMinutes =
          data.sessionTimeoutMinutes && data.sessionTimeoutMinutes > 0
            ? data.sessionTimeoutMinutes
            : DEFAULT_SESSION_TIMEOUT_MINUTES

        const now = Date.now()
        const timeoutMs = this.sessionTimeoutMinutes * 60 * 1000

        if (!data.lastActivityAt) {
          // 没记录最近活动时间，当成刚登录
          this.startSessionTimer()
          return
        }

        const elapsed = now - data.lastActivityAt
        if (elapsed >= timeoutMs) {
          // 已经过期
          this.token = null
          this.user = null
          this.lastActivityAt = null
          this.clearStorage()
          return
        }

        // 还有剩余时间，按剩余时间设置定时器
        this.lastActivityAt = data.lastActivityAt
        this.clearSessionTimer()
        const remaining = timeoutMs - elapsed

        this.timeoutTimer = window.setTimeout(() => {
          this.token = null
          this.user = null
          this.lastActivityAt = null
          this.timeoutTimer = null
          this.clearStorage()
          ElMessage.warning('登录已超时，请重新登录')
        }, remaining) as unknown as number
      } catch (e) {
        console.warn('restore auth from storage failed', e)
        this.token = null
        this.user = null
        this.lastActivityAt = null
        this.clearStorage()
      }
    },
  },
})