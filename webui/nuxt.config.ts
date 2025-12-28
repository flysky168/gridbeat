// https://nuxt.com/docs/api/configuration/nuxt-config

import tailwindcss from '@tailwindcss/vite'
import { resolve } from 'pathe'

export default defineNuxtConfig({
  css: ['~/assets/css/main.css'],
  vite: {
    plugins: [tailwindcss()],
  },
  ssr: false,
  nitro: {
    output: {
      // 生成静态资源输出到项目根目录的 dist/
      publicDir: resolve('./dist'),
    },
  },
  modules: [
    '@element-plus/nuxt',
    '@pinia/nuxt',
    '@nuxtjs/i18n',
    '@nuxt/ui',
    'nuxt-echarts',
    '@nuxt/image',
    '@nuxt/icon',
    '@sidebase/nuxt-auth',
  ],
  runtimeConfig: {
    public: {
      apiBase: '/api/v1',
      authMock: process.env.NUXT_PUBLIC_AUTH_MOCK === 'true',
    },
  },
  // i18n（中/英/日）
  i18n: {
    locales: [
      { code: 'zh', language: 'zh-CN', name: '中文' },
      { code: 'en', language: 'en-US', name: 'English' },
      { code: 'ja', language: 'ja-JP', name: '日本語' },
    ],
    defaultLocale: 'zh',
    strategy: 'no_prefix',
    detectBrowserLanguage: {
      useCookie: true,
      cookieKey: 'gw_locale',
      fallbackLocale: 'zh',
    },
  },
  devtools: { enabled: true },
  compatibilityDate: '2024-04-03',

  // mock=false 时，把 /api/v1 代理到真实后端（例如 8080）
  routeRules: (process.env.NUXT_PUBLIC_MOCK === 'true')
    ? {}
    : {
        '/api/v1/**': {
          // 例如：NUXT_API_TARGET=http://localhost:8080
          proxy: `${process.env.NUXT_API_TARGET || 'http://localhost:8080'}/api/v1/**`,
        },
      },
})