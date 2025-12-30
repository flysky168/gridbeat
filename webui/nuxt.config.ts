// https://nuxt.com/docs/api/configuration/nuxt-config
import tailwindcss from '@tailwindcss/vite'
import { resolve } from 'pathe'

export default defineNuxtConfig({
  modules: ['@nuxt/eslint', '@nuxt/ui', '@sidebase/nuxt-auth', '@nuxtjs/i18n'],
  ssr: false,
  devtools: {
    enabled: true
  },
  vite: {
    plugins: [tailwindcss()],
  },
  nitro: {
    output: {
      // 生成静态资源输出到项目根目录的 dist/
      publicDir: resolve('./dist'),
    },
  },
  css: ['~/assets/css/main.css'],
  routeRules: {
    '/': { prerender: true }
  },
  compatibilityDate: '2025-01-15',
  eslint: {
    config: {
      stylistic: {
        commaDangle: 'never',
        braceStyle: '1tbs'
      }
    }
  },
  // i18n（中/英/日）
  i18n: {
    locales: [
      { code: 'en', language: 'en-US', name: 'English', file: 'en.json' },
      { code: 'zh', language: 'zh-CN', name: '中文', file: 'zh.json' },
      { code: 'ja', language: 'ja-JP', name: '日本語', file: 'ja.json' },
    ],
    defaultLocale: 'zh',
    strategy: 'no_prefix',
    langDir: 'locales', // 语言文件存放目录 (默认 'locales')
    detectBrowserLanguage: {
      useCookie: true,
      cookieKey: 'gw_locale',
      fallbackLocale: 'zh',
    },
  },
  runtimeConfig: {
    baseURL: '/api/v1',
    upstream: process.env.NUXT_API_UPSTREAM || 'http://localhost:8080',
    public: {
      apiMode: process.env.NUXT_PUBLIC_API_MODE || 'mock',
    },
  },
  auth: {
    baseURL: '/api/v1',
    originEnvKey: 'NUXT_API_UPSTREAM',
    provider: {
      type: 'local',
      endpoints: {
        signIn: { path: '/auth/login', method: 'post' },
        signOut: { path: '/auth/logout', method: 'post' },
        signUp: false,
        // 不能禁用 getSession；它用于判断登录态  [oai_citation:2‡NuxtAuth](https://auth.sidebase.io/guide/local/quick-start)
        getSession: { path: '/me', method: 'get' },
      },
      token: {
        // signIn 响应里 token 的 JSON Pointer，默认 /token  [oai_citation:3‡NuxtAuth](https://auth.sidebase.io/guide/local/quick-start)
        signInResponseTokenPointer: '/data/token',
        type: 'Bearer',
        headerName: 'Authorization',
        cookieName: 'auth.token',
        maxAgeInSeconds: 3600,
        sameSiteAttribute: 'lax',
        // SPA 下通常保持 false（否则 JS 无法读 token 来拼 Authorization header）
        httpOnlyCookieAttribute: false,
      },

      // 可选：如你有 refresh token，再打开这一块（示例字段见官方文档） [oai_citation:4‡NuxtAuth](https://auth.sidebase.io/guide/local/quick-start)
      // refresh: { isEnabled: true, endpoint: { path: '/refresh', method: 'post' }, ... }
    },

    // 全站默认受保护：开启后，登录页要显式 auth:false  [oai_citation:5‡NuxtAuth](https://auth.sidebase.io/guide/application-side/configuration)
    globalAppMiddleware: true,

    // 让 session 在窗口聚焦时刷新（可按需关掉） [oai_citation:6‡NuxtAuth](https://auth.sidebase.io/guide/application-side/configuration)
    sessionRefresh: {
      enableOnWindowFocus: true,
      enablePeriodically: false,
    },
  },
})