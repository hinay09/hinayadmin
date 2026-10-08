// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  devtools: { enabled: true },

  modules: [
    '@element-plus/nuxt',
    '@pinia/nuxt',
  ],

  css: [
    '~/assets/styles/global.css',
  ],

  elementPlus: {
    importStyle: 'css',
    defaultLocale: 'zh-cn',
  },

  runtimeConfig: {
    public: {
      apiBase: process.env.NUXT_PUBLIC_API_BASE || '/api/v1',
    },
  },

  nitro: {
    devProxy: {
      '/api': {
        target: 'http://127.0.0.1:8000/api',
        changeOrigin: true,
      },
      '/upload': {
        target: 'http://127.0.0.1:8000/upload',
        changeOrigin: true,
      },
      // 接口文档: GoFrame 内置 Redoc 页面与 OpenAPI 描述文件 (系统工具-接口文档 iframe 内嵌)
      // 生产部署需在 nginx 将 /swagger 与 /api.json 反代到后端 (与 /api 同理)
      '/swagger': {
        target: 'http://127.0.0.1:8000/swagger',
        changeOrigin: true,
      },
      '/api.json': {
        target: 'http://127.0.0.1:8000/api.json',
        changeOrigin: true,
      },
    },
  },

  app: {
    head: {
      title: 'Hinay Admin',
      meta: [
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
      ],
    },
  },
})
