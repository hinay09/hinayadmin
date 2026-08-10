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
