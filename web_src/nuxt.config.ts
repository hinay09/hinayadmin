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
    // Element Plus 暗黑模式变量 (<html> 挂 dark 类即生效), 由布局设置的暗黑开关驱动
    'element-plus/theme-chalk/dark/css-vars.css',
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
      script: [
        {
          // 暗黑模式防闪白: 水合前读 localStorage 给 <html> 加 dark 类。
          // 读取的键/字段与 stores/settings.ts 的持久化格式保持一致。
          innerHTML: `(function(){try{var s=JSON.parse(localStorage.getItem('hinay_layout_setting')||'{}');if(s&&s.isDark){var d=document.documentElement;d.classList.add('dark');d.style.colorScheme='dark';}}catch(e){}})();`,
        },
      ],
    },
  },
})
